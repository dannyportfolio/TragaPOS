package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"pos-engine/internal/domain"
)

type TicketRepo struct {
	txMgr *TxManager
}

func NewTicketRepository(txMgr *TxManager) *TicketRepo {
	return &TicketRepo{txMgr: txMgr}
}

func (r *TicketRepo) exec(ctx context.Context) DBTX {
	return r.txMgr.GetExecutor(ctx)
}

func (r *TicketRepo) rebind(q string) string {
	return Rebind(r.txMgr.Driver(), q)
}

func (r *TicketRepo) IsTicketRedeemed(ctx context.Context, tenantID, ticketID string) (bool, error) {
	query := r.rebind(`
		SELECT COUNT(1)
		FROM redeemed_tickets
		WHERE tenant_id = ? AND ticket_id = ?
	`)
	var count int
	err := r.exec(ctx).QueryRowContext(ctx, query, tenantID, ticketID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check redeemed ticket: %w", err)
	}
	return count > 0, nil
}

func (r *TicketRepo) RecordRedemption(ctx context.Context, red *domain.TicketRedemption) error {
	now := time.Now().UTC()
	if red.RedeemedAt.IsZero() {
		red.RedeemedAt = now
	}

	query := r.rebind(`
		INSERT INTO redeemed_tickets (id, ticket_id, tenant_id, outlet_id, gate_zone, pax_redeemed, redeemed_at, device_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	_, err := r.exec(ctx).ExecContext(
		ctx, query,
		red.ID, red.TicketID, red.TenantID, red.OutletID, red.GateZone,
		red.PaxRedeemed, formatSQLTime(red.RedeemedAt), red.DeviceID,
	)
	if err != nil {
		return fmt.Errorf("failed to record ticket redemption: %w", err)
	}
	return nil
}

func (r *TicketRepo) GetRedemption(ctx context.Context, tenantID, ticketID string) (*domain.TicketRedemption, error) {
	query := r.rebind(`
		SELECT id, ticket_id, tenant_id, outlet_id, gate_zone, pax_redeemed, redeemed_at, device_id
		FROM redeemed_tickets
		WHERE tenant_id = ? AND ticket_id = ?
	`)

	var red domain.TicketRedemption
	var redAtStr string
	err := r.exec(ctx).QueryRowContext(ctx, query, tenantID, ticketID).Scan(
		&red.ID, &red.TicketID, &red.TenantID, &red.OutletID, &red.GateZone,
		&red.PaxRedeemed, &redAtStr, &red.DeviceID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query redemption: %w", err)
	}
	red.RedeemedAt = parseSQLTime(redAtStr)
	return &red, nil
}

func (r *TicketRepo) SaveIssuedTicket(ctx context.Context, payload *domain.TicketPayload, signature string) error {
	query := r.rebind(`
		INSERT INTO issued_tickets (ticket_id, tenant_id, category, pax, valid_until, issued_at, order_id, signature)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	_, err := r.exec(ctx).ExecContext(
		ctx, query,
		payload.TicketID, payload.TenantID, payload.Category, payload.Pax,
		payload.ValidUntil, payload.IssuedAt, payload.OrderID, signature,
	)
	if err != nil {
		return fmt.Errorf("failed to save issued ticket: %w", err)
	}
	return nil
}

func (r *TicketRepo) GetIssuedTicket(ctx context.Context, ticketID string) (*domain.SignedTicketQR, error) {
	query := r.rebind(`
		SELECT ticket_id, tenant_id, category, pax, valid_until, issued_at, order_id, signature
		FROM issued_tickets
		WHERE ticket_id = ?
	`)

	var p domain.TicketPayload
	var sig string
	err := r.exec(ctx).QueryRowContext(ctx, query, ticketID).Scan(
		&p.TicketID, &p.TenantID, &p.Category, &p.Pax, &p.ValidUntil, &p.IssuedAt, &p.OrderID, &sig,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvalidTicketCode
		}
		return nil, fmt.Errorf("failed to get issued ticket: %w", err)
	}

	return &domain.SignedTicketQR{
		Payload:   p,
		Signature: sig,
	}, nil
}
