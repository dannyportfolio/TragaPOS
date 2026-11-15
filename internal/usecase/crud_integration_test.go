package usecase_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"pos-engine/internal/config"
	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/uid"
	"pos-engine/internal/repository"
	"pos-engine/internal/usecase"
	"pos-engine/internal/usecase/strategies"
)

func setupCRUDTestDB(t *testing.T) (*sql.DB, *repository.TxManager) {
	dbName := fmt.Sprintf("file:test_crud_%s?mode=memory&cache=shared", uid.NewV7())
	db, err := sql.Open("sqlite", dbName)
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}

	schema := `
	CREATE TABLE items (
		id TEXT PRIMARY KEY, tenant_id TEXT, outlet_id TEXT, sku TEXT, barcode TEXT,
		name TEXT, sector TEXT, base_price INTEGER, currency TEXT, tax_rate REAL,
		sector_traits TEXT, version INTEGER, is_active INTEGER, created_at TEXT, updated_at TEXT,
		UNIQUE(tenant_id, sku)
	);
	CREATE TABLE inventory_balances (
		tenant_id TEXT, outlet_id TEXT, sku TEXT, on_hand_qty REAL, allocated_qty REAL, updated_at TEXT,
		PRIMARY KEY(tenant_id, outlet_id, sku)
	);
	CREATE TABLE stock_movements (
		id TEXT PRIMARY KEY, tenant_id TEXT, outlet_id TEXT, sku TEXT, delta_qty REAL,
		reason TEXT, reference_id TEXT, created_at TEXT
	);
	CREATE TABLE orders (
		id TEXT PRIMARY KEY, tenant_id TEXT, outlet_id TEXT, order_number TEXT, status TEXT,
		subtotal INTEGER, tax_amount INTEGER, discount_amount INTEGER, total_amount INTEGER,
		currency TEXT, notes TEXT, metadata TEXT, created_at TEXT, updated_at TEXT
	);
	CREATE TABLE order_items (
		id TEXT PRIMARY KEY, order_id TEXT, item_id TEXT, sku TEXT, name TEXT, sector TEXT,
		quantity REAL, uom TEXT, unit_price INTEGER, subtotal INTEGER, tax_amount INTEGER, total_amount INTEGER,
		selected_modifiers TEXT, sector_payload TEXT
	);
	CREATE TABLE payments (
		id TEXT PRIMARY KEY, order_id TEXT, method TEXT, amount INTEGER, reference TEXT, paid_at TEXT
	);
	CREATE TABLE guest_folios (
		id TEXT PRIMARY KEY, tenant_id TEXT, outlet_id TEXT, room_number TEXT,
		guest_name TEXT, credit_limit INTEGER, current_total INTEGER,
		is_open INTEGER, check_in_at TEXT, check_out_at TEXT
	);
	CREATE TABLE folio_charges (
		id TEXT PRIMARY KEY, folio_id TEXT, order_id TEXT, amount INTEGER, notes TEXT, created_at TEXT
	);
	CREATE TABLE sync_outbox (
		id TEXT PRIMARY KEY, tenant_id TEXT, outlet_id TEXT, event_type TEXT, aggregate_type TEXT,
		aggregate_id TEXT, payload TEXT, status TEXT, retry_count INTEGER, last_attempt_at TEXT, error_message TEXT, created_at TEXT
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	txMgr := repository.NewTxManager(db, "sqlite")
	return db, txMgr
}

func TestCatalogCRUD_FullLifecycle(t *testing.T) {
	db, txMgr := setupCRUDTestDB(t)
	defer db.Close()

	ctx := context.Background()
	cfg := &config.Config{
		AppMode:  config.ModeEdge,
		TenantID: "tenant-crud",
		OutletID: "outlet-crud",
	}

	catRepo := repository.NewCatalogRepository(txMgr)
	catUC := usecase.NewCatalogUsecase(cfg, catRepo)

	// 1. Validation check: Missing SKU should fail
	badItem := &domain.Item{
		Name:      "No SKU Item",
		Sector:    domain.SectorRetail,
		BasePrice: 10000,
	}
	if err := catUC.CreateItem(ctx, badItem); err == nil {
		t.Fatalf("expected error creating item without SKU, got nil")
	}

	// 2. Create valid item
	newItem := &domain.Item{
		SKU:          "TEST-PRODUCT-1",
		Barcode:      "8991234567890",
		Name:         "Organic Coffee Beans 250g",
		Sector:       domain.SectorRetail,
		BasePrice:    75000,
		Currency:     "IDR",
		TaxRate:      0.11,
		SectorTraits: []byte(`{"uom_conversions":[{"from_uom":"BAG","to_uom":"PCS","factor":1}]}`),
	}
	if err := catUC.CreateItem(ctx, newItem); err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}

	if newItem.ID == "" {
		t.Errorf("expected generated UUID for item ID")
	}
	if newItem.Version == 0 {
		t.Errorf("expected non-zero version timestamp")
	}

	// 3. Read by ID and Read by SKU
	fetchedByID, err := catUC.GetItemByID(ctx, newItem.ID)
	if err != nil {
		t.Fatalf("GetItemByID failed: %v", err)
	}
	if fetchedByID.Name != "Organic Coffee Beans 250g" {
		t.Errorf("expected item name 'Organic Coffee Beans 250g', got '%s'", fetchedByID.Name)
	}

	fetchedBySKU, err := catUC.GetItemBySKU(ctx, cfg.TenantID, newItem.SKU)
	if err != nil {
		t.Fatalf("GetItemBySKU failed: %v", err)
	}
	if fetchedBySKU.ID != newItem.ID {
		t.Errorf("expected fetched SKU ID %s to match %s", fetchedBySKU.ID, newItem.ID)
	}

	// 4. Update item
	updatePayload := &domain.Item{
		Name:      "Organic Coffee Beans 250g (Premium Dark Roast)",
		BasePrice: 85000,
	}
	if err := catUC.UpdateItem(ctx, newItem.ID, updatePayload); err != nil {
		t.Fatalf("UpdateItem failed: %v", err)
	}

	updatedItem, err := catUC.GetItemByID(ctx, newItem.ID)
	if err != nil {
		t.Fatalf("GetItemByID after update failed: %v", err)
	}
	if updatedItem.BasePrice != 85000 {
		t.Errorf("expected updated price 85000, got %d", updatedItem.BasePrice)
	}
	if updatedItem.Name != "Organic Coffee Beans 250g (Premium Dark Roast)" {
		t.Errorf("expected updated name, got %s", updatedItem.Name)
	}
	if updatedItem.Version <= newItem.Version {
		t.Errorf("expected updated version (%d) to be greater than initial (%d)", updatedItem.Version, newItem.Version)
	}

	// 5. Stock Adjustment (Restock +50 units)
	bal, err := catUC.AdjustStock(ctx, cfg.TenantID, cfg.OutletID, newItem.SKU, 50, "RESTOCK", "PO-1001")
	if err != nil {
		t.Fatalf("AdjustStock RESTOCK failed: %v", err)
	}
	if bal.OnHandQty != 50 {
		t.Errorf("expected stock on hand 50, got %f", bal.OnHandQty)
	}

	// Stock Adjustment (Damage -5 units)
	balAfterDamage, err := catUC.AdjustStock(ctx, cfg.TenantID, cfg.OutletID, newItem.SKU, -5, "DAMAGE", "INCIDENT-01")
	if err != nil {
		t.Fatalf("AdjustStock DAMAGE failed: %v", err)
	}
	if balAfterDamage.OnHandQty != 45 {
		t.Errorf("expected stock on hand 45 after damage, got %f", balAfterDamage.OnHandQty)
	}

	// 6. Delete item (Soft delete)
	if err := catUC.DeleteItem(ctx, cfg.TenantID, newItem.ID); err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}

	// Verify active items list does not contain deleted item
	activeItems, err := catRepo.ListItems(ctx, cfg.TenantID, 100, 0)
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	for _, it := range activeItems {
		if it.ID == newItem.ID {
			t.Errorf("deleted item %s should not appear in active ListItems", newItem.ID)
		}
	}
}

func TestOrderCRUD_CheckoutListAndVoidWithInventoryReversal(t *testing.T) {
	db, txMgr := setupCRUDTestDB(t)
	defer db.Close()

	ctx := context.Background()
	cfg := &config.Config{
		AppMode:  config.ModeEdge,
		TenantID: "tenant-crud-order",
		OutletID: "outlet-crud-order",
	}

	catRepo := repository.NewCatalogRepository(txMgr)
	orderRepo := repository.NewOrderRepository(txMgr)
	outboxRepo := repository.NewOutboxRepository(txMgr)
	stratRegistry := strategies.NewRegistry()
	broadcaster := &dummyBroadcaster{}

	checkoutUC := usecase.NewCheckoutUsecase(cfg, txMgr, catRepo, orderRepo, outboxRepo, stratRegistry, broadcaster)
	orderUC := usecase.NewOrderUsecase(cfg, orderRepo)

	// Seed item and stock
	sku := "SKU-TOWEL-01"
	item := &domain.Item{
		ID:           uid.NewV7(),
		TenantID:     cfg.TenantID,
		OutletID:     cfg.OutletID,
		SKU:          sku,
		Name:         "Microfiber Sports Towel",
		Sector:       domain.SectorRetail,
		BasePrice:    50000,
		Currency:     "IDR",
		TaxRate:      0.11,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		SectorTraits: []byte(`{"uom_conversions":[{"from_uom":"PCS","to_uom":"PCS","factor":1}]}`),
	}
	if err := catRepo.UpsertItem(ctx, item); err != nil {
		t.Fatalf("UpsertItem failed: %v", err)
	}
	if err := catRepo.RecordStockMovement(ctx, &domain.StockMovementEvent{
		ID:          uid.NewV7(),
		TenantID:    cfg.TenantID,
		OutletID:    cfg.OutletID,
		SKU:         sku,
		DeltaQty:    100,
		Reason:      "INITIAL_STOCK",
		ReferenceID: "INIT",
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("RecordStockMovement failed: %v", err)
	}

	// 1. Checkout Order (Buy 4 towels = 200,000 + tax)
	req := usecase.CheckoutRequest{
		TenantID: cfg.TenantID,
		OutletID: cfg.OutletID,
		Items: []usecase.CheckoutItemInput{
			{
				SKU:      sku,
				Quantity: 4,
				UOM:      "PCS",
			},
		},
		Payments: []usecase.PaymentInput{
			{
				Method: domain.PaymentMethodCash,
				Amount: 250000,
			},
		},
	}

	order, err := checkoutUC.Checkout(ctx, req)
	if err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	// Verify inventory reduced by 4 (100 - 4 = 96)
	bal, err := catRepo.GetStock(ctx, cfg.TenantID, cfg.OutletID, sku)
	if err != nil {
		t.Fatalf("GetStock failed: %v", err)
	}
	if bal.OnHandQty != 96 {
		t.Errorf("expected stock 96 after checkout, got %f", bal.OnHandQty)
	}

	// 2. Read / List Orders with Filter
	orders, err := orderUC.ListOrders(ctx, cfg.TenantID, cfg.OutletID, string(domain.OrderStatusPaid), 10, 0)
	if err != nil {
		t.Fatalf("ListOrders failed: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 paid order, got %d", len(orders))
	}
	if orders[0].ID != order.ID {
		t.Errorf("expected order ID %s, got %s", order.ID, orders[0].ID)
	}
	if len(orders[0].Items) != 1 {
		t.Fatalf("expected 1 order item eager loaded, got %d", len(orders[0].Items))
	}

	// 3. Void / Cancel Order with inventory reversal
	voidReason := "Customer returned items immediately due to billing issue"
	if err := orderUC.VoidOrder(ctx, cfg.TenantID, order.ID, voidReason); err != nil {
		t.Fatalf("VoidOrder failed: %v", err)
	}

	// 4. Verify order status is CANCELLED and audit notes recorded
	voidedOrder, err := orderRepo.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetOrderByID failed: %v", err)
	}
	if voidedOrder.Status != domain.OrderStatusCancelled {
		t.Errorf("expected order status CANCELLED, got %s", voidedOrder.Status)
	}
	if voidedOrder.Notes == "" || !containsSubstring(voidedOrder.Notes, "VOID:") {
		t.Errorf("expected void note in order notes, got '%s'", voidedOrder.Notes)
	}

	// 5. Verify inventory restored (96 + 4 = 100)
	balRestored, err := catRepo.GetStock(ctx, cfg.TenantID, cfg.OutletID, sku)
	if err != nil {
		t.Fatalf("GetStock failed: %v", err)
	}
	if balRestored.OnHandQty != 100 {
		t.Errorf("expected inventory restored to 100 after void, got %f", balRestored.OnHandQty)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && len(sub) > 0 && stringContains(s, sub)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
