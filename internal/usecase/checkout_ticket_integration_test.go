package usecase_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"pos-engine/internal/config"
	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/crypto"
	"pos-engine/internal/pkg/uid"
	"pos-engine/internal/repository"
	"pos-engine/internal/usecase"
	"pos-engine/internal/usecase/strategies"
)

type dummyBroadcaster struct {
	events []string
}

func (d *dummyBroadcaster) BroadcastEvent(topic, event string, payload any) {
	d.events = append(d.events, fmt.Sprintf("%s:%s", topic, event))
}

func setupTestDB(t *testing.T) (*sql.DB, *repository.TxManager) {
	dbName := fmt.Sprintf("file:test_%s?mode=memory&cache=shared", uid.NewV7())
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
		id TEXT PRIMARY KEY, tenant_id TEXT, outlet_id TEXT, room_number TEXT, guest_name TEXT,
		credit_limit INTEGER, current_total INTEGER, is_open INTEGER, check_in_at TEXT, check_out_at TEXT
	);
	CREATE TABLE folio_charges (
		id TEXT PRIMARY KEY, folio_id TEXT, order_id TEXT, amount INTEGER, notes TEXT, created_at TEXT
	);
	CREATE TABLE redeemed_tickets (
		id TEXT PRIMARY KEY, ticket_id TEXT, tenant_id TEXT, outlet_id TEXT, gate_zone TEXT,
		pax_redeemed INTEGER, redeemed_at TEXT, device_id TEXT, UNIQUE (tenant_id, ticket_id)
	);
	CREATE TABLE issued_tickets (
		ticket_id TEXT PRIMARY KEY, tenant_id TEXT, category TEXT, pax INTEGER, valid_until INTEGER,
		issued_at INTEGER, order_id TEXT, signature TEXT
	);
	CREATE TABLE sync_outbox (
		id TEXT PRIMARY KEY, tenant_id TEXT, outlet_id TEXT, event_type TEXT, aggregate_type TEXT,
		aggregate_id TEXT, payload TEXT, status TEXT, retry_count INTEGER, last_attempt_at TEXT,
		error_message TEXT, created_at TEXT
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to apply test schema: %v", err)
	}

	txMgr := repository.NewTxManager(db, "sqlite")
	return db, txMgr
}

func TestCheckout_FulfillmentAndOutbox(t *testing.T) {
	db, txMgr := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	catalogRepo := repository.NewCatalogRepository(txMgr)
	orderRepo := repository.NewOrderRepository(txMgr)
	outboxRepo := repository.NewOutboxRepository(txMgr)
	broadcaster := &dummyBroadcaster{}

	cfg := &config.Config{
		AppMode:  config.ModeEdge,
		TenantID: "tenant-park-01",
		OutletID: "outlet-cafe-gate",
	}

	reg := strategies.NewRegistry()
	checkoutUC := usecase.NewCheckoutUsecase(
		cfg, txMgr, catalogRepo, orderRepo, outboxRepo, reg, broadcaster,
	)

	fbTrait := domain.FBTrait{
		KitchenStation: "BAR",
		Recipe: []domain.RecipeIngredient{
			{RawMaterialSKU: "RAW_BEANS_G", Quantity: 20, UOM: "g"},
		},
	}
	traitBytes, _ := json.Marshal(fbTrait)

	coffeeItem := &domain.Item{
		ID:           uid.NewV7(),
		TenantID:     cfg.TenantID,
		OutletID:     cfg.OutletID,
		SKU:          "AMERICANO",
		Name:         "Iced Americano",
		Sector:       domain.SectorFB,
		BasePrice:    25000,
		TaxRate:      0.10,
		SectorTraits: traitBytes,
		IsActive:     true,
	}
	if err := catalogRepo.UpsertItem(ctx, coffeeItem); err != nil {
		t.Fatalf("failed to seed item: %v", err)
	}

	req := usecase.CheckoutRequest{
		TenantID: cfg.TenantID,
		OutletID: cfg.OutletID,
		Items: []usecase.CheckoutItemInput{
			{SKU: "AMERICANO", Quantity: 2},
		},
		Payment: &usecase.PaymentInput{
			Method: domain.PaymentMethodCash,
			Amount: 60000,
		},
	}

	order, err := checkoutUC.Checkout(ctx, req)
	if err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	if order.Status != domain.OrderStatusPaid {
		t.Errorf("expected order status PAID, got %s", order.Status)
	}
	if order.TotalAmount != 55000 {
		t.Errorf("expected total amount 55000, got %d", order.TotalAmount)
	}

	bal, err := catalogRepo.GetStock(ctx, cfg.TenantID, cfg.OutletID, "RAW_BEANS_G")
	if err != nil {
		t.Fatalf("GetStock failed: %v", err)
	}
	if bal.OnHandQty != -40 {
		t.Errorf("expected on_hand_qty -40, got %v", bal.OnHandQty)
	}

	outboxItems, err := outboxRepo.FetchPendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("FetchPendingBatch failed: %v", err)
	}
	if len(outboxItems) < 3 {
		t.Fatalf("expected at least 3 outbox records (ORDER, PAYMENT, STOCK), got %d", len(outboxItems))
	}
}

func TestHospitalityFolioCharge(t *testing.T) {
	db, txMgr := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	catalogRepo := repository.NewCatalogRepository(txMgr)
	orderRepo := repository.NewOrderRepository(txMgr)
	outboxRepo := repository.NewOutboxRepository(txMgr)
	broadcaster := &dummyBroadcaster{}

	cfg := &config.Config{
		AppMode:  config.ModeEdge,
		TenantID: "resort-01",
		OutletID: "outlet-lounge",
	}

	reg := strategies.NewRegistry()
	checkoutUC := usecase.NewCheckoutUsecase(
		cfg, txMgr, catalogRepo, orderRepo, outboxRepo, reg, broadcaster,
	)

	folio := &domain.GuestFolio{
		ID:           uid.NewV7(),
		TenantID:     cfg.TenantID,
		OutletID:     cfg.OutletID,
		RoomNumber:   "101",
		GuestName:    "John Doe",
		CreditLimit:  500000,
		CurrentTotal: 0,
		IsOpen:       true,
		CheckInAt:    time.Now().UTC(),
		CheckOutAt:   time.Now().UTC().Add(48 * time.Hour),
	}
	if err := orderRepo.CreateFolio(ctx, folio); err != nil {
		t.Fatalf("failed to create folio: %v", err)
	}

	item := &domain.Item{
		ID:        uid.NewV7(),
		TenantID:  cfg.TenantID,
		OutletID:  cfg.OutletID,
		SKU:       "SNACK_PLATTER",
		Name:      "Lounge Snack Platter",
		Sector:    domain.SectorRetail,
		BasePrice: 150000,
		IsActive:  true,
	}
	_ = catalogRepo.UpsertItem(ctx, item)

	req := usecase.CheckoutRequest{
		TenantID: cfg.TenantID,
		OutletID: cfg.OutletID,
		Items: []usecase.CheckoutItemInput{
			{SKU: "SNACK_PLATTER", Quantity: 1},
		},
		Payment: &usecase.PaymentInput{
			Method:     domain.PaymentMethodRoomCharge,
			Amount:     150000,
			RoomNumber: "101",
		},
	}

	order, err := checkoutUC.Checkout(ctx, req)
	if err != nil {
		t.Fatalf("Checkout with Room Charge failed: %v", err)
	}

	if order.Status != domain.OrderStatusPaid {
		t.Errorf("expected order status PAID, got %s", order.Status)
	}

	updatedFolio, err := orderRepo.GetFolioByRoom(ctx, cfg.TenantID, "101")
	if err != nil {
		t.Fatalf("GetFolioByRoom failed: %v", err)
	}
	if updatedFolio.CurrentTotal != 150000 {
		t.Errorf("expected folio total 150000, got %d", updatedFolio.CurrentTotal)
	}
}

func TestCryptographicOfflineTicketingAndRedemption(t *testing.T) {
	db, txMgr := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	outboxRepo := repository.NewOutboxRepository(txMgr)
	ticketRepo := repository.NewTicketRepository(txMgr)
	broadcaster := &dummyBroadcaster{}

	pubKey, privKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate keypair: %v", err)
	}

	cloudCfg := &config.Config{
		AppMode:        config.ModeCloud,
		TenantID:       "park-central",
		Ed25519PrivKey: privKey,
		Ed25519PubKey:  pubKey,
	}
	cloudTicketUC := usecase.NewTicketUsecase(cloudCfg, txMgr, ticketRepo, nil, broadcaster)

	signedTicket, qrString, err := cloudTicketUC.IssueTicket(ctx, usecase.IssueTicketRequest{
		TenantID:   cloudCfg.TenantID,
		Category:   "VIP_PASS",
		Pax:        2,
		ValidHours: 24,
	})
	if err != nil {
		t.Fatalf("IssueTicket failed: %v", err)
	}
	if qrString == "" || signedTicket.Signature == "" {
		t.Fatal("expected non-empty qr string and signature")
	}

	edgeCfg := &config.Config{
		AppMode:       config.ModeEdge,
		TenantID:      "park-central",
		OutletID:      "gate-turnstile-01",
		Ed25519PubKey: pubKey,
	}
	edgeTicketUC := usecase.NewTicketUsecase(edgeCfg, txMgr, ticketRepo, outboxRepo, broadcaster)

	redemption, err := edgeTicketUC.RedeemTicketOffline(ctx, usecase.RedeemTicketRequest{
		QRString: qrString,
		TenantID: edgeCfg.TenantID,
		OutletID: edgeCfg.OutletID,
		GateZone: "MAIN_GATE",
	})
	if err != nil {
		t.Fatalf("offline redemption failed: %v", err)
	}
	if redemption.PaxRedeemed != 2 {
		t.Errorf("expected 2 pax redeemed, got %d", redemption.PaxRedeemed)
	}

	_, err = edgeTicketUC.RedeemTicketOffline(ctx, usecase.RedeemTicketRequest{
		QRString: qrString,
		TenantID: edgeCfg.TenantID,
		OutletID: edgeCfg.OutletID,
		GateZone: "MAIN_GATE",
	})
	if err != domain.ErrTicketAlreadyUsed {
		t.Fatalf("expected ErrTicketAlreadyUsed on double scan, got %v", err)
	}
}

func TestSyncIngestor_PhysicalRealityWins(t *testing.T) {
	db, txMgr := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	catalogRepo := repository.NewCatalogRepository(txMgr)
	orderRepo := repository.NewOrderRepository(txMgr)
	ticketRepo := repository.NewTicketRepository(txMgr)

	eventID := uid.NewV7()
	stockMov := domain.StockMovementEvent{
		ID:          uid.NewV7(),
		TenantID:    "tenant-01",
		OutletID:    "outlet-offline",
		SKU:         "POPCORN_LARGE",
		DeltaQty:    -10,
		Reason:      "OFFLINE_PAID_SALE",
		ReferenceID: "ord-offline-999",
		CreatedAt:   time.Now().UTC(),
	}
	movBytes, _ := json.Marshal(stockMov)

	batch := domain.SyncPushBatch{
		TenantID:       "tenant-01",
		OutletID:       "outlet-offline",
		IdempotencyKey: "push-test-01",
		Events: []domain.SyncOutbox{
			{
				ID:            eventID,
				TenantID:      "tenant-01",
				OutletID:      "outlet-offline",
				EventType:     domain.EventStockMoved,
				AggregateType: "stock",
				AggregateID:   stockMov.ID,
				Payload:       movBytes,
				Status:        domain.OutboxStatusPending,
			},
		},
	}

	resp, err := usecase.IngestUpstreamBatch(ctx, txMgr, catalogRepo, orderRepo, ticketRepo, batch)
	if err != nil {
		t.Fatalf("IngestUpstreamBatch failed: %v", err)
	}

	if len(resp.AckEventIDs) != 1 || resp.AckEventIDs[0] != eventID {
		t.Fatalf("expected ACK for event %s, got %+v", eventID, resp.AckEventIDs)
	}

	if len(resp.Discrepancies) == 0 {
		t.Fatal("expected STOCK_DISCREPANCY_ALERT for negative stock, got none")
	}

	bal, err := catalogRepo.GetStock(ctx, "tenant-01", "outlet-offline", "POPCORN_LARGE")
	if err != nil {
		t.Fatalf("GetStock failed: %v", err)
	}
	if bal.OnHandQty != -10 {
		t.Errorf("expected on-hand stock -10, got %v", bal.OnHandQty)
	}
}
