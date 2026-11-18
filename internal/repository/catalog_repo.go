package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"pos-engine/internal/domain"
)

type CatalogRepo struct {
	txMgr *TxManager
}

func NewCatalogRepository(txMgr *TxManager) *CatalogRepo {
	return &CatalogRepo{txMgr: txMgr}
}

func (r *CatalogRepo) exec(ctx context.Context) DBTX {
	return r.txMgr.GetExecutor(ctx)
}

func (r *CatalogRepo) rebind(q string) string {
	return Rebind(r.txMgr.Driver(), q)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanItem(row rowScanner) (*domain.Item, error) {
	var item domain.Item
	var traitsStr string
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&item.ID, &item.TenantID, &item.OutletID, &item.SKU, &item.Barcode,
		&item.Name, &item.Sector, &item.BasePrice, &item.Currency, &item.TaxRate,
		&traitsStr, &item.Version, &item.IsActive, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	item.SectorTraits = []byte(traitsStr)
	item.CreatedAt = parseSQLTime(createdAtStr)
	item.UpdatedAt = parseSQLTime(updatedAtStr)
	return &item, nil
}

func (r *CatalogRepo) GetByID(ctx context.Context, id string) (*domain.Item, error) {
	query := r.rebind(`
		SELECT id, tenant_id, outlet_id, sku, barcode, name, sector, base_price,
		       currency, tax_rate, sector_traits, version, is_active, created_at, updated_at
		FROM items
		WHERE id = ?
	`)

	item, err := scanItem(r.exec(ctx).QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get item by id: %w", err)
	}
	return item, nil
}

func (r *CatalogRepo) GetBySKU(ctx context.Context, tenantID, sku string) (*domain.Item, error) {
	query := r.rebind(`
		SELECT id, tenant_id, outlet_id, sku, barcode, name, sector, base_price,
		       currency, tax_rate, sector_traits, version, is_active, created_at, updated_at
		FROM items
		WHERE tenant_id = ? AND sku = ?
	`)

	item, err := scanItem(r.exec(ctx).QueryRowContext(ctx, query, tenantID, sku))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get item by sku: %w", err)
	}
	return item, nil
}

func (r *CatalogRepo) GetByBarcode(ctx context.Context, tenantID, barcode string) (*domain.Item, error) {
	query := r.rebind(`
		SELECT id, tenant_id, outlet_id, sku, barcode, name, sector, base_price,
		       currency, tax_rate, sector_traits, version, is_active, created_at, updated_at
		FROM items
		WHERE tenant_id = ? AND barcode = ?
	`)

	item, err := scanItem(r.exec(ctx).QueryRowContext(ctx, query, tenantID, barcode))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get item by barcode: %w", err)
	}
	return item, nil
}

func (r *CatalogRepo) ListItems(ctx context.Context, tenantID string, limit, offset int) ([]domain.Item, error) {
	query := r.rebind(`
		SELECT id, tenant_id, outlet_id, sku, barcode, name, sector, base_price,
		       currency, tax_rate, sector_traits, version, is_active, created_at, updated_at
		FROM items
		WHERE tenant_id = ? AND is_active
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`)

	rows, err := r.exec(ctx).QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	defer rows.Close()

	var items []domain.Item
	for rows.Next() {
		var item domain.Item
		var traitsStr string
		var createdAtStr, updatedAtStr string

		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.OutletID, &item.SKU, &item.Barcode,
			&item.Name, &item.Sector, &item.BasePrice, &item.Currency, &item.TaxRate,
			&traitsStr, &item.Version, &item.IsActive, &createdAtStr, &updatedAtStr,
		); err != nil {
			return nil, err
		}
		item.SectorTraits = []byte(traitsStr)
		item.CreatedAt = parseSQLTime(createdAtStr)
		item.UpdatedAt = parseSQLTime(updatedAtStr)
		items = append(items, item)
	}
	return items, nil
}

func (r *CatalogRepo) ListUpdatedSince(ctx context.Context, tenantID string, sinceVersion int64, limit int) ([]domain.Item, error) {
	query := r.rebind(`
		SELECT id, tenant_id, outlet_id, sku, barcode, name, sector, base_price,
		       currency, tax_rate, sector_traits, version, is_active, created_at, updated_at
		FROM items
		WHERE tenant_id = ? AND version > ?
		ORDER BY version ASC
		LIMIT ?
	`)

	rows, err := r.exec(ctx).QueryContext(ctx, query, tenantID, sinceVersion, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list updated items: %w", err)
	}
	defer rows.Close()

	var items []domain.Item
	for rows.Next() {
		var item domain.Item
		var traitsStr string
		var createdAtStr, updatedAtStr string

		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.OutletID, &item.SKU, &item.Barcode,
			&item.Name, &item.Sector, &item.BasePrice, &item.Currency, &item.TaxRate,
			&traitsStr, &item.Version, &item.IsActive, &createdAtStr, &updatedAtStr,
		); err != nil {
			return nil, err
		}
		item.SectorTraits = []byte(traitsStr)
		item.CreatedAt = parseSQLTime(createdAtStr)
		item.UpdatedAt = parseSQLTime(updatedAtStr)
		items = append(items, item)
	}
	return items, nil
}

func (r *CatalogRepo) UpsertItem(ctx context.Context, item *domain.Item) error {
	now := time.Now().UTC()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now

	var query string
	if r.txMgr.Driver() == "postgres" {
		query = `
			INSERT INTO items (
				id, tenant_id, outlet_id, sku, barcode, name, sector, base_price,
				currency, tax_rate, sector_traits, version, is_active, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				barcode = EXCLUDED.barcode,
				base_price = EXCLUDED.base_price,
				tax_rate = EXCLUDED.tax_rate,
				sector_traits = EXCLUDED.sector_traits,
				version = EXCLUDED.version,
				is_active = EXCLUDED.is_active,
				updated_at = EXCLUDED.updated_at
		`
	} else {
		query = `
			INSERT INTO items (
				id, tenant_id, outlet_id, sku, barcode, name, sector, base_price,
				currency, tax_rate, sector_traits, version, is_active, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				name = excluded.name,
				barcode = excluded.barcode,
				base_price = excluded.base_price,
				tax_rate = excluded.tax_rate,
				sector_traits = excluded.sector_traits,
				version = excluded.version,
				is_active = excluded.is_active,
				updated_at = excluded.updated_at
		`
	}

	_, err := r.exec(ctx).ExecContext(
		ctx, query,
		item.ID, item.TenantID, item.OutletID, item.SKU, item.Barcode,
		item.Name, string(item.Sector), item.BasePrice, item.Currency, item.TaxRate,
		string(item.SectorTraits), item.Version, item.IsActive,
		formatSQLTime(item.CreatedAt), formatSQLTime(item.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("failed to upsert item: %w", err)
	}
	return nil
}

func (r *CatalogRepo) DeleteItem(ctx context.Context, tenantID, id string) error {
	now := time.Now().UTC()
	version := now.UnixNano()
	query := r.rebind(`
		UPDATE items
		SET is_active = ?, version = ?, updated_at = ?
		WHERE id = ? AND tenant_id = ?
	`)
	res, err := r.exec(ctx).ExecContext(ctx, query, false, version, formatSQLTime(now), id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}
	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return domain.ErrItemNotFound
	}
	return nil
}

func (r *CatalogRepo) GetStock(ctx context.Context, tenantID, outletID, sku string) (*domain.InventoryBalance, error) {
	query := r.rebind(`
		SELECT tenant_id, outlet_id, sku, on_hand_qty, allocated_qty, updated_at
		FROM inventory_balances
		WHERE tenant_id = ? AND outlet_id = ? AND sku = ?
	`)

	var bal domain.InventoryBalance
	var updatedAtStr string
	err := r.exec(ctx).QueryRowContext(ctx, query, tenantID, outletID, sku).Scan(
		&bal.TenantID, &bal.OutletID, &bal.SKU, &bal.OnHandQty, &bal.AllocatedQty, &updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &domain.InventoryBalance{
				TenantID:     tenantID,
				OutletID:     outletID,
				SKU:          sku,
				OnHandQty:    0,
				AllocatedQty: 0,
				UpdatedAt:    time.Now().UTC(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get inventory stock: %w", err)
	}
	bal.UpdatedAt = parseSQLTime(updatedAtStr)
	return &bal, nil
}

func (r *CatalogRepo) RecordStockMovement(ctx context.Context, event *domain.StockMovementEvent) error {
	now := time.Now().UTC()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}

	insertEventQ := r.rebind(`
		INSERT INTO stock_movements (id, tenant_id, outlet_id, sku, delta_qty, reason, reference_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	_, err := r.exec(ctx).ExecContext(
		ctx, insertEventQ,
		event.ID, event.TenantID, event.OutletID, event.SKU, event.DeltaQty, event.Reason, event.ReferenceID,
		formatSQLTime(event.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("failed to record stock movement: %w", err)
	}

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

	_, err = r.exec(ctx).ExecContext(
		ctx, updateBalQ,
		event.TenantID, event.OutletID, event.SKU, event.DeltaQty, formatSQLTime(now),
	)
	if err != nil {
		return fmt.Errorf("failed to update inventory balance: %w", err)
	}

	return nil
}

func formatSQLTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05.000000")
}

func parseSQLTime(s string) time.Time {
	layouts := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Now().UTC()
}
