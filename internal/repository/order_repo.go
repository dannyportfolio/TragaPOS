package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/uid"
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

	// Check if this order was previously parked/held
	var existingStatus string
	checkQ := r.rebind(`SELECT status FROM orders WHERE id = ?`)
	err := r.exec(ctx).QueryRowContext(ctx, checkQ, order.ID).Scan(&existingStatus)
	if err == nil {
		if domain.OrderStatus(existingStatus) == domain.OrderStatusPaid {
			return domain.ErrOrderAlreadyPaid
		}

		// Update the existing order record
		updateQ := r.rebind(`
			UPDATE orders
			SET status = ?, subtotal = ?, tax_amount = ?, discount_amount = ?,
			    total_amount = ?, notes = ?, metadata = ?, updated_at = ?
			WHERE id = ?
		`)
		metaStr := string(order.Metadata)
		if metaStr == "" {
			metaStr = "{}"
		}
		_, err = r.exec(ctx).ExecContext(
			ctx, updateQ,
			string(order.Status), order.Subtotal, order.TaxAmount, order.DiscountAmount,
			order.TotalAmount, order.Notes, metaStr, formatSQLTime(order.UpdatedAt), order.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update held order: %w", err)
		}

		// Clean up previous order items
		delQ := r.rebind(`DELETE FROM order_items WHERE order_id = ?`)
		if _, err := r.exec(ctx).ExecContext(ctx, delQ, order.ID); err != nil {
			return fmt.Errorf("failed to clear previous order items: %w", err)
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		// Insert brand new order
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
	} else {
		return fmt.Errorf("failed to query existing order: %w", err)
	}

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
	return r.ListOrdersWithFilter(ctx, domain.OrderFilter{
		TenantID: tenantID,
		OutletID: outletID,
		Limit:    limit,
		Offset:   offset,
	})
}

func (r *OrderRepo) ListOrdersWithFilter(ctx context.Context, filter domain.OrderFilter) ([]domain.Order, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	whereClause := "WHERE tenant_id = ?"
	args := []any{filter.TenantID}
	if filter.OutletID != "" {
		whereClause += " AND outlet_id = ?"
		args = append(args, filter.OutletID)
	}
	if filter.Status != "" {
		whereClause += " AND status = ?"
		args = append(args, filter.Status)
	}

	query := r.rebind(fmt.Sprintf(`
		SELECT id, tenant_id, outlet_id, order_number, status, subtotal,
		       tax_amount, discount_amount, total_amount, currency, notes,
		       metadata, created_at, updated_at
		FROM orders
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause))
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.exec(ctx).QueryContext(ctx, query, args...)
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

	// Populate items and payments for listed orders
	for i := range orders {
		qItems := r.rebind(`
			SELECT id, order_id, item_id, sku, name, sector, quantity, uom,
			       unit_price, subtotal, tax_amount, total_amount,
			       selected_modifiers, sector_payload
			FROM order_items
			WHERE order_id = ?
		`)
		iRows, err := r.exec(ctx).QueryContext(ctx, qItems, orders[i].ID)
		if err == nil {
			for iRows.Next() {
				var it domain.OrderItem
				var modStr, secStr string
				if err := iRows.Scan(
					&it.ID, &it.OrderID, &it.ItemID, &it.SKU, &it.Name, &it.Sector,
					&it.Quantity, &it.UOM, &it.UnitPrice, &it.Subtotal, &it.TaxAmount, &it.TotalAmount,
					&modStr, &secStr,
				); err == nil {
					it.SelectedModifiers = []byte(modStr)
					it.SectorPayload = []byte(secStr)
					orders[i].Items = append(orders[i].Items, it)
				}
			}
			iRows.Close()
		}

		qPayments := r.rebind(`
			SELECT id, order_id, method, amount, reference, paid_at
			FROM payments
			WHERE order_id = ?
		`)
		pRows, err := r.exec(ctx).QueryContext(ctx, qPayments, orders[i].ID)
		if err == nil {
			for pRows.Next() {
				var p domain.Payment
				var paidAtStr string
				if err := pRows.Scan(&p.ID, &p.OrderID, &p.Method, &p.Amount, &p.Reference, &paidAtStr); err == nil {
					p.PaidAt = parseSQLTime(paidAtStr)
					orders[i].Payments = append(orders[i].Payments, p)
				}
			}
			pRows.Close()
		}
	}

	return orders, nil
}

func (r *OrderRepo) ListKdsOrders(ctx context.Context, tenantID, outletID string) ([]domain.Order, error) {
	query := r.rebind(`
		SELECT id, tenant_id, outlet_id, order_number, status, subtotal,
		       tax_amount, discount_amount, total_amount, currency, notes,
		       metadata, created_at, updated_at
		FROM orders
		WHERE tenant_id = ? AND outlet_id = ? AND status IN ('PAID', 'PREPARING', 'READY')
		ORDER BY created_at ASC
	`)

	rows, err := r.exec(ctx).QueryContext(ctx, query, tenantID, outletID)
	if err != nil {
		return nil, fmt.Errorf("failed to list kds orders: %w", err)
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

	for i := range orders {
		itemQ := r.rebind(`
			SELECT id, order_id, item_id, sku, name, sector, quantity, uom,
			       unit_price, subtotal, tax_amount, total_amount,
			       selected_modifiers, sector_payload
			FROM order_items
			WHERE order_id = ?
		`)
		iRows, err := r.exec(ctx).QueryContext(ctx, itemQ, orders[i].ID)
		if err == nil {
			for iRows.Next() {
				var it domain.OrderItem
				var modStr, payloadStr string
				if err := iRows.Scan(
					&it.ID, &it.OrderID, &it.ItemID, &it.SKU, &it.Name, &it.Sector,
					&it.Quantity, &it.UOM, &it.UnitPrice, &it.Subtotal, &it.TaxAmount, &it.TotalAmount,
					&modStr, &payloadStr,
				); err == nil {
					it.SelectedModifiers = []byte(modStr)
					it.SectorPayload = []byte(payloadStr)
					orders[i].Items = append(orders[i].Items, it)
				}
			}
			iRows.Close()
		}
	}

	return orders, nil
}

func (r *OrderRepo) ListHeldOrders(ctx context.Context, tenantID, outletID string) ([]domain.Order, error) {
	query := r.rebind(`
		SELECT id, tenant_id, outlet_id, order_number, status, subtotal,
		       tax_amount, discount_amount, total_amount, currency, notes,
		       metadata, created_at, updated_at
		FROM orders
		WHERE tenant_id = ? AND outlet_id = ? AND status = 'HELD'
		ORDER BY created_at DESC
	`)

	rows, err := r.exec(ctx).QueryContext(ctx, query, tenantID, outletID)
	if err != nil {
		return nil, fmt.Errorf("failed to list held orders: %w", err)
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

	for i := range orders {
		itemQ := r.rebind(`
			SELECT id, order_id, item_id, sku, name, sector, quantity, uom,
			       unit_price, subtotal, tax_amount, total_amount,
			       selected_modifiers, sector_payload
			FROM order_items
			WHERE order_id = ?
		`)
		iRows, err := r.exec(ctx).QueryContext(ctx, itemQ, orders[i].ID)
		if err == nil {
			for iRows.Next() {
				var it domain.OrderItem
				var modStr, payloadStr string
				if err := iRows.Scan(
					&it.ID, &it.OrderID, &it.ItemID, &it.SKU, &it.Name, &it.Sector,
					&it.Quantity, &it.UOM, &it.UnitPrice, &it.Subtotal, &it.TaxAmount, &it.TotalAmount,
					&modStr, &payloadStr,
				); err == nil {
					it.SelectedModifiers = []byte(modStr)
					it.SectorPayload = []byte(payloadStr)
					orders[i].Items = append(orders[i].Items, it)
				}
			}
			iRows.Close()
		}
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

func (r *OrderRepo) VoidOrder(ctx context.Context, tenantID, orderID string, reason string) error {
	return r.txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
		order, err := r.GetOrderByID(txCtx, orderID)
		if err != nil {
			return err
		}
		if order.TenantID != tenantID {
			return domain.ErrOrderNotFound
		}
		if order.Status == domain.OrderStatusCancelled {
			return fmt.Errorf("order is already cancelled/voided")
		}

		now := time.Now().UTC()
		voidNote := "VOID: " + reason
		query := r.rebind(`
			UPDATE orders
			SET status = ?, notes = CASE WHEN notes IS NULL OR notes = '' THEN ? ELSE notes || ' | ' || ? END, updated_at = ?
			WHERE id = ? AND tenant_id = ?
		`)
		if _, err := r.exec(txCtx).ExecContext(txCtx, query, string(domain.OrderStatusCancelled), voidNote, voidNote, formatSQLTime(now), orderID, tenantID); err != nil {
			return fmt.Errorf("failed to void order: %w", err)
		}

		// Reverse stock movements
		insertEventQ := r.rebind(`
			INSERT INTO stock_movements (id, tenant_id, outlet_id, sku, delta_qty, reason, reference_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`)
		var updateBalQ string
		if r.txMgr.Driver() == "postgres" {
			updateBalQ = `
				INSERT INTO inventory_balances (tenant_id, outlet_id, sku, on_hand_qty, allocated_qty, updated_at)
				VALUES ($1, $2, $3, $4, 0, $5)
				ON CONFLICT (tenant_id, outlet_id, sku) DO UPDATE SET
					on_hand_qty = inventory_balances.on_hand_qty + EXCLUDED.on_hand_qty,
					updated_at = EXCLUDED.updated_at
			`
		} else {
			updateBalQ = `
				INSERT INTO inventory_balances (tenant_id, outlet_id, sku, on_hand_qty, allocated_qty, updated_at)
				VALUES (?, ?, ?, ?, 0, ?)
				ON CONFLICT(tenant_id, outlet_id, sku) DO UPDATE SET
					on_hand_qty = inventory_balances.on_hand_qty + excluded.on_hand_qty,
					updated_at = excluded.updated_at
			`
		}

		for _, it := range order.Items {
			evtID := uid.NewV7()
			if _, err := r.exec(txCtx).ExecContext(
				txCtx, insertEventQ,
				evtID, order.TenantID, order.OutletID, it.SKU, it.Quantity, "ORDER_VOID_REVERSAL", order.ID,
				formatSQLTime(now),
			); err != nil {
				return fmt.Errorf("failed to record void stock movement for %s: %w", it.SKU, err)
			}

			if _, err := r.exec(txCtx).ExecContext(txCtx, updateBalQ, order.TenantID, order.OutletID, it.SKU, it.Quantity, formatSQLTime(now)); err != nil {
				return fmt.Errorf("failed to restore inventory balance for %s: %w", it.SKU, err)
			}
		}

		return nil
	})
}

func (r *OrderRepo) ListFolios(ctx context.Context, tenantID string, openOnly bool, limit, offset int) ([]domain.GuestFolio, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	whereClause := "WHERE tenant_id = ?"
	args := []any{tenantID}
	if openOnly {
		whereClause += " AND is_open = ?"
		if r.txMgr.Driver() == "postgres" {
			args = append(args, true)
		} else {
			args = append(args, 1)
		}
	}
	query := r.rebind(fmt.Sprintf(`
		SELECT id, tenant_id, outlet_id, room_number, guest_name, credit_limit,
		       current_total, is_open, check_in_at, check_out_at
		FROM guest_folios
		%s
		ORDER BY room_number ASC
		LIMIT ? OFFSET ?
	`, whereClause))
	args = append(args, limit, offset)

	rows, err := r.exec(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list folios: %w", err)
	}
	defer rows.Close()

	var folios []domain.GuestFolio
	for rows.Next() {
		var f domain.GuestFolio
		var inStr, outStr string
		if err := rows.Scan(
			&f.ID, &f.TenantID, &f.OutletID, &f.RoomNumber, &f.GuestName,
			&f.CreditLimit, &f.CurrentTotal, &f.IsOpen, &inStr, &outStr,
		); err != nil {
			return nil, err
		}
		f.CheckInAt = parseSQLTime(inStr)
		f.CheckOutAt = parseSQLTime(outStr)
		folios = append(folios, f)
	}
	return folios, nil
}

func (r *OrderRepo) CloseFolio(ctx context.Context, tenantID, folioID string) error {
	var query string
	if r.txMgr.Driver() == "postgres" {
		query = r.rebind("UPDATE guest_folios SET is_open = false WHERE id = ? AND tenant_id = ?")
	} else {
		query = r.rebind("UPDATE guest_folios SET is_open = 0 WHERE id = ? AND tenant_id = ?")
	}
	res, err := r.exec(ctx).ExecContext(ctx, query, folioID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to close folio: %w", err)
	}
	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return domain.ErrFolioNotFound
	}
	return nil
}
