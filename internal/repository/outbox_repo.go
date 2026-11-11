package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"pos-engine/internal/domain"
)

type OutboxRepo struct {
	txMgr *TxManager
}

func NewOutboxRepository(txMgr *TxManager) *OutboxRepo {
	return &OutboxRepo{txMgr: txMgr}
}

func (r *OutboxRepo) exec(ctx context.Context) DBTX {
	return r.txMgr.GetExecutor(ctx)
}

func (r *OutboxRepo) rebind(q string) string {
	return Rebind(r.txMgr.Driver(), q)
}

func (r *OutboxRepo) Enqueue(ctx context.Context, item *domain.SyncOutbox) error {
	now := time.Now().UTC()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	if item.Status == "" {
		item.Status = domain.OutboxStatusPending
	}

	query := r.rebind(`
		INSERT INTO sync_outbox (
			id, tenant_id, outlet_id, event_type, aggregate_type, aggregate_id,
			payload, status, retry_count, error_message, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)

	payloadStr := string(item.Payload)
	if payloadStr == "" {
		payloadStr = "{}"
	}

	_, err := r.exec(ctx).ExecContext(
		ctx, query,
		item.ID, item.TenantID, item.OutletID, string(item.EventType), item.AggregateType, item.AggregateID,
		payloadStr, string(item.Status), item.RetryCount, item.ErrorMessage, formatSQLTime(item.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue outbox event: %w", err)
	}
	return nil
}

func (r *OutboxRepo) FetchPendingBatch(ctx context.Context, limit int) ([]domain.SyncOutbox, error) {
	query := r.rebind(`
		SELECT id, tenant_id, outlet_id, event_type, aggregate_type, aggregate_id,
		       payload, status, retry_count, last_attempt_at, error_message, created_at
		FROM sync_outbox
		WHERE status IN ('PENDING', 'FAILED') AND retry_count < 10
		ORDER BY created_at ASC
		LIMIT ?
	`)

	rows, err := r.exec(ctx).QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending outbox batch: %w", err)
	}
	defer rows.Close()

	var batch []domain.SyncOutbox
	for rows.Next() {
		var item domain.SyncOutbox
		var payloadStr, createdAtStr string
		var lastAttemptStr sql.NullString
		var errStr sql.NullString

		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.OutletID, &item.EventType, &item.AggregateType, &item.AggregateID,
			&payloadStr, &item.Status, &item.RetryCount, &lastAttemptStr, &errStr, &createdAtStr,
		); err != nil {
			return nil, err
		}

		item.Payload = []byte(payloadStr)
		item.CreatedAt = parseSQLTime(createdAtStr)
		if lastAttemptStr.Valid {
			t := parseSQLTime(lastAttemptStr.String)
			item.LastAttemptAt = &t
		}
		if errStr.Valid {
			item.ErrorMessage = errStr.String
		}

		batch = append(batch, item)
	}
	return batch, nil
}

func (r *OutboxRepo) MarkSent(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids)+1)
	args[0] = string(domain.OutboxStatusSent)

	for i, id := range ids {
		placeholders[i] = "?"
		args[i+1] = id
	}

	queryRaw := fmt.Sprintf(`
		UPDATE sync_outbox
		SET status = ?
		WHERE id IN (%s)
	`, strings.Join(placeholders, ", "))

	query := r.rebind(queryRaw)
	_, err := r.exec(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to mark outbox records sent: %w", err)
	}
	return nil
}

func (r *OutboxRepo) MarkFailed(ctx context.Context, id string, errMsg string) error {
	now := time.Now().UTC()
	query := r.rebind(`
		UPDATE sync_outbox
		SET status = ?, retry_count = retry_count + 1, last_attempt_at = ?, error_message = ?
		WHERE id = ?
	`)
	_, err := r.exec(ctx).ExecContext(
		ctx, query,
		string(domain.OutboxStatusFailed), formatSQLTime(now), errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("failed to mark outbox item failed: %w", err)
	}
	return nil
}

func (r *OutboxRepo) PruneSent(ctx context.Context, olderThan time.Duration) (int64, error) {
	threshold := time.Now().UTC().Add(-olderThan)
	query := r.rebind(`
		DELETE FROM sync_outbox
		WHERE status = 'SENT' AND created_at < ?
	`)
	res, err := r.exec(ctx).ExecContext(ctx, query, formatSQLTime(threshold))
	if err != nil {
		return 0, fmt.Errorf("failed to prune sent outbox records: %w", err)
	}
	return res.RowsAffected()
}
