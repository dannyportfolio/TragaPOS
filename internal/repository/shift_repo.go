package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"pos-engine/internal/domain"
)

type ShiftRepo struct {
	txMgr *TxManager
}

func NewShiftRepository(txMgr *TxManager) *ShiftRepo {
	return &ShiftRepo{txMgr: txMgr}
}

func (r *ShiftRepo) exec(ctx context.Context) DBTX {
	return r.txMgr.GetExecutor(ctx)
}

func (r *ShiftRepo) rebind(q string) string {
	return Rebind(r.txMgr.Driver(), q)
}

func (r *ShiftRepo) CreateShift(ctx context.Context, shift *domain.ShiftSession) error {
	now := time.Now().UTC()
	if shift.OpenedAt.IsZero() {
		shift.OpenedAt = now
	}

	q := r.rebind(`
		INSERT INTO shift_sessions (
			id, tenant_id, outlet_id, cashier_id, cashier_name,
			status, opening_float, expected_cash, actual_cash, variance,
			notes, opened_at, closed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)

	var closedAtStr *string
	if shift.ClosedAt != nil {
		s := formatSQLTime(*shift.ClosedAt)
		closedAtStr = &s
	}

	_, err := r.exec(ctx).ExecContext(
		ctx, q,
		shift.ID, shift.TenantID, shift.OutletID, shift.CashierID, shift.CashierName,
		string(shift.Status), shift.OpeningFloat, shift.ExpectedCash, shift.ActualCash, shift.Variance,
		shift.Notes, formatSQLTime(shift.OpenedAt), closedAtStr,
	)
	if err != nil {
		return fmt.Errorf("failed to insert shift session: %w", err)
	}

	return nil
}

func (r *ShiftRepo) GetActiveShift(ctx context.Context, tenantID, outletID, cashierID string) (*domain.ShiftSession, error) {
	q := r.rebind(`
		SELECT id, tenant_id, outlet_id, cashier_id, cashier_name,
		       status, opening_float, expected_cash, actual_cash, variance,
		       notes, opened_at, closed_at
		FROM shift_sessions
		WHERE tenant_id = ? AND outlet_id = ? AND cashier_id = ? AND status = 'OPEN'
		ORDER BY opened_at DESC
		LIMIT 1
	`)

	var s domain.ShiftSession
	var statusStr, openedAtStr string
	var notes sql.NullString
	var closedAtStr sql.NullString

	err := r.exec(ctx).QueryRowContext(ctx, q, tenantID, outletID, cashierID).Scan(
		&s.ID, &s.TenantID, &s.OutletID, &s.CashierID, &s.CashierName,
		&statusStr, &s.OpeningFloat, &s.ExpectedCash, &s.ActualCash, &s.Variance,
		&notes, &openedAtStr, &closedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNoActiveShift
		}
		return nil, fmt.Errorf("failed to query active shift: %w", err)
	}

	s.Status = domain.ShiftStatus(statusStr)
	s.Notes = notes.String
	s.OpenedAt = parseSQLTime(openedAtStr)
	if closedAtStr.Valid && closedAtStr.String != "" {
		t := parseSQLTime(closedAtStr.String)
		s.ClosedAt = &t
	}

	return &s, nil
}

func (r *ShiftRepo) GetShiftByID(ctx context.Context, id string) (*domain.ShiftSession, error) {
	q := r.rebind(`
		SELECT id, tenant_id, outlet_id, cashier_id, cashier_name,
		       status, opening_float, expected_cash, actual_cash, variance,
		       notes, opened_at, closed_at
		FROM shift_sessions
		WHERE id = ?
	`)

	var s domain.ShiftSession
	var statusStr, openedAtStr string
	var notes sql.NullString
	var closedAtStr sql.NullString

	err := r.exec(ctx).QueryRowContext(ctx, q, id).Scan(
		&s.ID, &s.TenantID, &s.OutletID, &s.CashierID, &s.CashierName,
		&statusStr, &s.OpeningFloat, &s.ExpectedCash, &s.ActualCash, &s.Variance,
		&notes, &openedAtStr, &closedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrShiftNotFound
		}
		return nil, fmt.Errorf("failed to query shift by id: %w", err)
	}

	s.Status = domain.ShiftStatus(statusStr)
	s.Notes = notes.String
	s.OpenedAt = parseSQLTime(openedAtStr)
	if closedAtStr.Valid && closedAtStr.String != "" {
		t := parseSQLTime(closedAtStr.String)
		s.ClosedAt = &t
	}

	return &s, nil
}

func (r *ShiftRepo) CloseShift(
	ctx context.Context,
	shiftID string,
	actualCash, expectedCash, variance int64,
	notes string,
	closedAt time.Time,
) error {
	q := r.rebind(`
		UPDATE shift_sessions
		SET status = 'CLOSED',
		    actual_cash = ?,
		    expected_cash = ?,
		    variance = ?,
		    notes = ?,
		    closed_at = ?
		WHERE id = ? AND status = 'OPEN'
	`)

	res, err := r.exec(ctx).ExecContext(
		ctx, q,
		actualCash, expectedCash, variance, notes, formatSQLTime(closedAt), shiftID,
	)
	if err != nil {
		return fmt.Errorf("failed to close shift: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrShiftAlreadyClosed
	}

	return nil
}

func (r *ShiftRepo) CreateCashMovement(ctx context.Context, movement *domain.CashMovement) error {
	now := time.Now().UTC()
	if movement.CreatedAt.IsZero() {
		movement.CreatedAt = now
	}

	q := r.rebind(`
		INSERT INTO cash_movements (id, shift_id, type, amount, reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)

	_, err := r.exec(ctx).ExecContext(
		ctx, q,
		movement.ID, movement.ShiftID, string(movement.Type), movement.Amount,
		movement.Reason, formatSQLTime(movement.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("failed to insert cash movement: %w", err)
	}

	return nil
}

func (r *ShiftRepo) ListCashMovements(ctx context.Context, shiftID string) ([]domain.CashMovement, error) {
	q := r.rebind(`
		SELECT id, shift_id, type, amount, reason, created_at
		FROM cash_movements
		WHERE shift_id = ?
		ORDER BY created_at ASC
	`)

	rows, err := r.exec(ctx).QueryContext(ctx, q, shiftID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cash movements: %w", err)
	}
	defer rows.Close()

	var movements []domain.CashMovement
	for rows.Next() {
		var m domain.CashMovement
		var typeStr, createdAtStr string
		if err := rows.Scan(&m.ID, &m.ShiftID, &typeStr, &m.Amount, &m.Reason, &createdAtStr); err != nil {
			return nil, fmt.Errorf("failed to scan cash movement: %w", err)
		}
		m.Type = domain.CashMovementType(typeStr)
		m.CreatedAt = parseSQLTime(createdAtStr)
		movements = append(movements, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return movements, nil
}

func (r *ShiftRepo) GetShiftSalesBreakdown(
	ctx context.Context,
	tenantID, outletID string,
	fromTime time.Time,
	toTime time.Time,
) (int64, int64, error) {
	q := r.rebind(`
		SELECT 
			COALESCE(SUM(CASE WHEN p.method = 'CASH' THEN p.amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN p.method != 'CASH' THEN p.amount ELSE 0 END), 0)
		FROM payments p
		JOIN orders o ON p.order_id = o.id
		WHERE o.tenant_id = ? AND o.outlet_id = ? 
		  AND o.created_at >= ? AND o.created_at <= ? 
		  AND o.status != 'CANCELLED'
	`)

	var cashSales, nonCashSales int64
	err := r.exec(ctx).QueryRowContext(
		ctx, q,
		tenantID, outletID,
		formatSQLTime(fromTime), formatSQLTime(toTime),
	).Scan(&cashSales, &nonCashSales)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query shift sales breakdown: %w", err)
	}

	return cashSales, nonCashSales, nil
}

func (r *ShiftRepo) ListShifts(ctx context.Context, tenantID, outletID string, limit, offset int) ([]domain.ShiftSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	whereClause := "WHERE tenant_id = ?"
	args := []any{tenantID}
	if outletID != "" {
		whereClause += " AND outlet_id = ?"
		args = append(args, outletID)
	}
	query := r.rebind(fmt.Sprintf(`
		SELECT id, tenant_id, outlet_id, cashier_id, cashier_name, status,
		       opening_float, expected_cash, actual_cash, variance, notes,
		       opened_at, closed_at
		FROM shift_sessions
		%s
		ORDER BY opened_at DESC
		LIMIT ? OFFSET ?
	`, whereClause))
	args = append(args, limit, offset)

	rows, err := r.exec(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list shifts: %w", err)
	}
	defer rows.Close()

	var sessions []domain.ShiftSession
	for rows.Next() {
		var s domain.ShiftSession
		var openStr string
		var closeStr sql.NullString
		var notes sql.NullString
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.OutletID, &s.CashierID, &s.CashierName, &s.Status,
			&s.OpeningFloat, &s.ExpectedCash, &s.ActualCash, &s.Variance, &notes,
			&openStr, &closeStr,
		); err != nil {
			return nil, err
		}
		s.Notes = notes.String
		s.OpenedAt = parseSQLTime(openStr)
		if closeStr.Valid && closeStr.String != "" {
			t := parseSQLTime(closeStr.String)
			s.ClosedAt = &t
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}
