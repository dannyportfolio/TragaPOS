package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"pos-engine/internal/domain"
)

type OrderRepo struct {
	txMgr *TxManager
}

func NewOrderRepository(txMgr *TxManager) *OrderRepo {
	return &OrderRepo{txMgr: txMgr}
}

func (r *OrderRepo) exec(ctx context.Context) DBTX {
	return r.txMgr.GetExecutor(ctx)
}

func (r *OrderRepo) rebind(q string) string {
	return Rebind(r.txMgr.Driver(), q)
}

func (r *OrderRepo) CreateOrder(ctx context.Context, order *domain.Order) error {
	now := time.Now().UTC()
	if order.CreatedAt.IsZero() {
		order.CreatedAt = now
	}
	order.UpdatedAt = now

	// 1. Insert order header
	queryOrder := r.rebind(`
		INSERT INTO orders (
			id, tenant_id, outlet_id, order_number, status, subtotal,
			tax_amount, discount_amount, total_amount, currency, notes,
			metadata, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)

	metaStr := string(order.Metadata)
	if metaStr == "" {
		metaStr = "{}"
	}

	_, err := r.exec(ctx).ExecContext(
		ctx, queryOrder,
		order.ID, order.TenantID, order.OutletID, order.OrderNumber, string(order.Status),
		order.Subtotal, order.TaxAmount, order.DiscountAmount, order.TotalAmount,
		order.Currency, order.Notes, metaStr,
		formatSQLTime(order.CreatedAt), formatSQLTime(order.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// 2. Insert order items
	queryItem := r.rebind(`
		INSERT INTO order_items (
			id, order_id, item_id, sku, name, sector, quantity, uom,
			unit_price, subtotal, tax_amount, total_amount,
			selected_modifiers, sector_payload
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)

	for _, item := range order.Items {
		modStr := string(item.SelectedModifiers)
		if modStr == "" {
			modStr = "[]"
		}
		secStr := string(item.SectorPayload)
		if secStr == "" {
			secStr = "{}"
		}

		_, err := r.exec(ctx).ExecContext(
			ctx, queryItem,
			item.ID, item.OrderID, item.ItemID, item.SKU, item.Name, string(item.Sector),
			item.Quantity, item.UOM, item.UnitPrice, item.Subtotal, item.TaxAmount, item.TotalAmount,
			modStr, secStr,
		)
		if err != nil {
			return fmt.Errorf("failed to insert order item %s: %w", item.ID, err)
		}
	}

	// 3. Insert initial payments if any
	for _, p := range order.Payments {
		if err := r.AddPayment(ctx, &p); err != nil {
			return err
		}
	}

	return nil
}

func (r *OrderRepo) GetOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	queryOrder := r.rebind(`
		SELECT id, tenant_id, outlet_id, order_number, status, subtotal,
		       tax_amount, discount_amount, total_amount, currency, notes,
		       metadata, created_at, updated_at
		FROM orders
		WHERE id = ?
	`)

	var order domain.Order
	var metaStr string
	var createdAtStr, updatedAtStr string

	err := r.exec(ctx).QueryRowContext(ctx, queryOrder, id).Scan(
		&order.ID, &order.TenantID, &order.OutletID, &order.OrderNumber, &order.Status,
		&order.Subtotal, &order.TaxAmount, &order.DiscountAmount, &order.TotalAmount,
		&order.Currency, &order.Notes, &metaStr,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	order.Metadata = []byte(metaStr)
	order.CreatedAt = parseSQLTime(createdAtStr)
	order.UpdatedAt = parseSQLTime(updatedAtStr)

	// Fetch Items
	queryItems := r.rebind(`
		SELECT id, order_id, item_id, sku, name, sector, quantity, uom,
		       unit_price, subtotal, tax_amount, total_amount,
		       selected_modifiers, sector_payload
		FROM order_items
		WHERE order_id = ?
	`)
	rows, err := r.exec(ctx).QueryContext(ctx, queryItems, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var it domain.OrderItem
		var modStr, secStr string
		if err := rows.Scan(
			&it.ID, &it.OrderID, &it.ItemID, &it.SKU, &it.Name, &it.Sector,
			&it.Quantity, &it.UOM, &it.UnitPrice, &it.Subtotal, &it.TaxAmount, &it.TotalAmount,
			&modStr, &secStr,
		); err != nil {
			return nil, err
		}
		it.SelectedModifiers = []byte(modStr)
		it.SectorPayload = []byte(secStr)
		order.Items = append(order.Items, it)
	}

	// Fetch Payments
	queryPayments := r.rebind(`
		SELECT id, order_id, method, amount, reference, paid_at
		FROM payments
		WHERE order_id = ?
	`)
	pRows, err := r.exec(ctx).QueryContext(ctx, queryPayments, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query payments: %w", err)
	}
	defer pRows.Close()

	for pRows.Next() {
		var p domain.Payment
		var paidAtStr string
		if err := pRows.Scan(&p.ID, &p.OrderID, &p.Method, &p.Amount, &p.Reference, &paidAtStr); err != nil {
			return nil, err
		}
		p.PaidAt = parseSQLTime(paidAtStr)
		order.Payments = append(order.Payments, p)
	}

	return &order, nil
}

func (r *OrderRepo) ListOrders(ctx context.Context, tenantID, outletID string, limit, offset int) ([]domain.Order, error) {
	query := r.rebind(`
		SELECT id, tenant_id, outlet_id, order_number, status, subtotal,
		       tax_amount, discount_amount, total_amount, currency, notes,
		       metadata, created_at, updated_at
		FROM orders
		WHERE tenant_id = ? AND outlet_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`)

	rows, err := r.exec(ctx).QueryContext(ctx, query, tenantID, outletID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var order domain.Order
		var metaStr, createdAtStr, updatedAtStr string
		if err := rows.Scan(
			&order.ID, &order.TenantID, &order.OutletID, &order.OrderNumber, &order.Status,
			&order.Subtotal, &order.TaxAmount, &order.DiscountAmount, &order.TotalAmount,
			&order.Currency, &order.Notes, &metaStr,
			&createdAtStr, &updatedAtStr,
		); err != nil {
			return nil, err
		}
		order.Metadata = []byte(metaStr)
		order.CreatedAt = parseSQLTime(createdAtStr)
		order.UpdatedAt = parseSQLTime(updatedAtStr)
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *OrderRepo) AddPayment(ctx context.Context, payment *domain.Payment) error {
	now := time.Now().UTC()
	if payment.PaidAt.IsZero() {
		payment.PaidAt = now
	}

	query := r.rebind(`
		INSERT INTO payments (id, order_id, method, amount, reference, paid_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	_, err := r.exec(ctx).ExecContext(
		ctx, query,
		payment.ID, payment.OrderID, string(payment.Method), payment.Amount,
		payment.Reference, formatSQLTime(payment.PaidAt),
	)
	if err != nil {
		return fmt.Errorf("failed to add payment: %w", err)
	}
	return nil
}

func (r *OrderRepo) UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	now := time.Now().UTC()
	query := r.rebind(`
		UPDATE orders
		SET status = ?, updated_at = ?
		WHERE id = ?
	`)
	_, err := r.exec(ctx).ExecContext(ctx, query, string(status), formatSQLTime(now), orderID)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}

// Hospitality Folio Implementations

func (r *OrderRepo) GetFolioByRoom(ctx context.Context, tenantID, roomNumber string) (*domain.GuestFolio, error) {
	query := r.rebind(`
		SELECT id, tenant_id, outlet_id, room_number, guest_name, credit_limit,
		       current_total, is_open, check_in_at, check_out_at
		FROM guest_folios
		WHERE tenant_id = ? AND room_number = ? AND is_open = TRUE
	`)

	var f domain.GuestFolio
	var inStr, outStr string
	err := r.exec(ctx).QueryRowContext(ctx, query, tenantID, roomNumber).Scan(
		&f.ID, &f.TenantID, &f.OutletID, &f.RoomNumber, &f.GuestName, &f.CreditLimit,
		&f.CurrentTotal, &f.IsOpen, &inStr, &outStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrFolioNotFound
		}
		return nil, fmt.Errorf("failed to get folio by room: %w", err)
	}
	f.CheckInAt = parseSQLTime(inStr)
	f.CheckOutAt = parseSQLTime(outStr)
	return &f, nil
}

func (r *OrderRepo) GetFolioByID(ctx context.Context, id string) (*domain.GuestFolio, error) {
	query := r.rebind(`
		SELECT id, tenant_id, outlet_id, room_number, guest_name, credit_limit,
		       current_total, is_open, check_in_at, check_out_at
		FROM guest_folios
		WHERE id = ?
	`)

	var f domain.GuestFolio
	var inStr, outStr string
	err := r.exec(ctx).QueryRowContext(ctx, query, id).Scan(
		&f.ID, &f.TenantID, &f.OutletID, &f.RoomNumber, &f.GuestName, &f.CreditLimit,
		&f.CurrentTotal, &f.IsOpen, &inStr, &outStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrFolioNotFound
		}
		return nil, fmt.Errorf("failed to get folio by id: %w", err)
	}
	f.CheckInAt = parseSQLTime(inStr)
	f.CheckOutAt = parseSQLTime(outStr)
	return &f, nil
}

func (r *OrderRepo) CreateFolio(ctx context.Context, folio *domain.GuestFolio) error {
	query := r.rebind(`
		INSERT INTO guest_folios (
			id, tenant_id, outlet_id, room_number, guest_name,
			credit_limit, current_total, is_open, check_in_at, check_out_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	_, err := r.exec(ctx).ExecContext(
		ctx, query,
		folio.ID, folio.TenantID, folio.OutletID, folio.RoomNumber, folio.GuestName,
		folio.CreditLimit, folio.CurrentTotal, folio.IsOpen,
		formatSQLTime(folio.CheckInAt), formatSQLTime(folio.CheckOutAt),
	)
	if err != nil {
		return fmt.Errorf("failed to create folio: %w", err)
	}
	return nil
}

func (r *OrderRepo) ChargeFolio(ctx context.Context, charge *domain.FolioCharge) error {
	// 1. Fetch current folio state
	folio, err := r.GetFolioByID(ctx, charge.FolioID)
	if err != nil {
		return err
	}
	if !folio.IsOpen {
		return domain.ErrFolioNotFound
	}

	newTotal := folio.CurrentTotal + charge.Amount
	if folio.CreditLimit > 0 && newTotal > folio.CreditLimit {
		return domain.ErrFolioLimitExceeded
	}

	// 2. Insert charge item
	now := time.Now().UTC()
	if charge.CreatedAt.IsZero() {
		charge.CreatedAt = now
	}
	insertChargeQ := r.rebind(`
		INSERT INTO folio_charges (id, folio_id, order_id, amount, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	_, err = r.exec(ctx).ExecContext(
		ctx, insertChargeQ,
		charge.ID, charge.FolioID, charge.OrderID, charge.Amount, charge.Notes,
		formatSQLTime(charge.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("failed to record folio charge: %w", err)
	}

	// 3. Update folio total
	updateFolioQ := r.rebind(`
		UPDATE guest_folios
		SET current_total = ?
		WHERE id = ?
	`)
	_, err = r.exec(ctx).ExecContext(ctx, updateFolioQ, newTotal, charge.FolioID)
	if err != nil {
		return fmt.Errorf("failed to update folio balance: %w", err)
	}

	return nil
}
