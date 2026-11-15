package usecase_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"pos-engine/internal/config"
	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/uid"
	"pos-engine/internal/repository"
	"pos-engine/internal/usecase"
	"pos-engine/internal/usecase/strategies"
)

func setupSplitTestDB(t *testing.T) (*sql.DB, *repository.TxManager) {
	dbName := fmt.Sprintf("file:test_split_%s?mode=memory&cache=shared", uid.NewV7())
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

func TestSplitCheckoutAndHoldOrders(t *testing.T) {
	db, txMgr := setupSplitTestDB(t)
	defer db.Close()

	ctx := context.Background()
	cfg := &config.Config{
		AppMode:  config.ModeEdge,
		TenantID: "tenant-split",
		OutletID: "outlet-split",
	}

	catRepo := repository.NewCatalogRepository(txMgr)
	orderRepo := repository.NewOrderRepository(txMgr)
	outboxRepo := repository.NewOutboxRepository(txMgr)
	stratRegistry := strategies.NewRegistry()

	broadcaster := &dummyBroadcaster{}
	checkoutUC := usecase.NewCheckoutUsecase(cfg, txMgr, catRepo, orderRepo, outboxRepo, stratRegistry, broadcaster)
	orderUC := usecase.NewOrderUsecase(cfg, orderRepo)

	// Seed items
	item1 := &domain.Item{
		ID:           uid.NewV7(),
		TenantID:     "tenant-split",
		OutletID:     "outlet-split",
		SKU:          "RETAIL-SNACK",
		Barcode:      "899001",
		Name:         "Potato Chips",
		Sector:       domain.SectorRetail,
		BasePrice:    20000,
		Currency:     "IDR",
		TaxRate:      0.1,
		SectorTraits: []byte(`{}`),
		IsActive:     true,
	}
	_ = catRepo.UpsertItem(ctx, item1)

	item2 := &domain.Item{
		ID:           uid.NewV7(),
		TenantID:     "tenant-split",
		OutletID:     "outlet-split",
		SKU:          "FB-COFFEE",
		Barcode:      "899002",
		Name:         "Iced Americano",
		Sector:       domain.SectorFB,
		BasePrice:    30000,
		Currency:     "IDR",
		TaxRate:      0.1,
		SectorTraits: []byte(`{}`),
		IsActive:     true,
	}
	_ = catRepo.UpsertItem(ctx, item2)

	// Test 1: Underpayment rejection
	underpayReq := usecase.CheckoutRequest{
		TenantID: "tenant-split",
		OutletID: "outlet-split",
		Items: []usecase.CheckoutItemInput{
			{SKU: "RETAIL-SNACK", Quantity: 1},
			{SKU: "FB-COFFEE", Quantity: 1},
		},
		Payments: []usecase.PaymentInput{
			{Method: domain.PaymentMethodCash, Amount: 30000}, // Total is 22k + 33k = 55k
		},
	}
	_, err := checkoutUC.Checkout(ctx, underpayReq)
	if err == nil {
		t.Fatal("expected underpayment error, got nil")
	}

	// Test 2: Successful Split Payment Checkout (30,000 Cash + 25,000 QRIS = 55,000)
	splitReq := usecase.CheckoutRequest{
		TenantID: "tenant-split",
		OutletID: "outlet-split",
		Items: []usecase.CheckoutItemInput{
			{SKU: "RETAIL-SNACK", Quantity: 1},
			{SKU: "FB-COFFEE", Quantity: 1},
		},
		Payments: []usecase.PaymentInput{
			{Method: domain.PaymentMethodCash, Amount: 30000},
			{Method: domain.PaymentMethodQRIS, Amount: 25000, Reference: "QRIS-REF-99"},
		},
	}
	order, err := checkoutUC.Checkout(ctx, splitReq)
	if err != nil {
		t.Fatalf("split checkout failed: %v", err)
	}
	if order.Status != domain.OrderStatusPaid {
		t.Errorf("expected PAID status, got %s", order.Status)
	}
	if order.TotalAmount != 55000 {
		t.Errorf("expected 55000 total, got %d", order.TotalAmount)
	}
	if len(order.Payments) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(order.Payments))
	}

	// Test 3: Generate Receipt
	receipt, err := orderUC.GenerateReceipt(ctx, order.ID)
	if err != nil {
		t.Fatalf("GenerateReceipt failed: %v", err)
	}
	if !strings.Contains(receipt, "Potato Chips") || !strings.Contains(receipt, "Iced Americano") {
		t.Errorf("receipt missing item names: %s", receipt)
	}
	if !strings.Contains(receipt, "CASH") || !strings.Contains(receipt, "QRIS") {
		t.Errorf("receipt missing split payments breakdown: %s", receipt)
	}

	// Test 4: Hold Order & List Held Orders
	holdReq := usecase.CheckoutRequest{
		TenantID: "tenant-split",
		OutletID: "outlet-split",
		Notes:    "Table 4 waiting for friend",
		Items: []usecase.CheckoutItemInput{
			{SKU: "FB-COFFEE", Quantity: 2},
		},
	}
	heldOrder, err := checkoutUC.HoldOrder(ctx, holdReq)
	if err != nil {
		t.Fatalf("HoldOrder failed: %v", err)
	}
	if heldOrder.Status != domain.OrderStatusHeld {
		t.Errorf("expected HELD status, got %s", heldOrder.Status)
	}

	heldList, err := orderUC.ListHeldOrders(ctx, "tenant-split", "outlet-split")
	if err != nil {
		t.Fatalf("ListHeldOrders failed: %v", err)
	}
	if len(heldList) != 1 {
		t.Fatalf("expected 1 held order, got %d", len(heldList))
	}
	if heldList[0].ID != heldOrder.ID {
		t.Errorf("expected held order ID %s, got %s", heldOrder.ID, heldList[0].ID)
	}
	if len(heldList[0].Items) != 1 {
		t.Errorf("expected 1 item in held order, got %d", len(heldList[0].Items))
	}
}
