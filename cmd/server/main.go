package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pos-engine/internal/config"
	"pos-engine/internal/database"
	deliveryHttp "pos-engine/internal/delivery/http"
	"pos-engine/internal/delivery/ws"
	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/crypto"
	"pos-engine/internal/pkg/uid"
	"pos-engine/internal/repository"
	"pos-engine/internal/usecase"
	"pos-engine/internal/usecase/strategies"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	log.Println("[Engine] Initializing Headless Multi-Industry POS & Commerce Engine...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[Engine] Config load error: %v", err)
	}

	log.Printf("[Engine] Running in %s Mode on Port %d", cfg.AppMode, cfg.Port)

	ensureCryptoKeys(cfg)

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("[Engine] Database initialization error: %v", err)
	}
	defer db.Close()

	txMgr := repository.NewTxManager(db, cfg.DBDriver)
	catalogRepo := repository.NewCatalogRepository(txMgr)
	orderRepo := repository.NewOrderRepository(txMgr)
	ticketRepo := repository.NewTicketRepository(txMgr)
	shiftRepo := repository.NewShiftRepository(txMgr)

	seedDemoData(cfg, catalogRepo, orderRepo)

	var outboxRepo domain.OutboxRepository
	if cfg.AppMode == config.ModeEdge {
		outboxRepo = repository.NewOutboxRepository(txMgr)
	}

	strategyRegistry := strategies.NewRegistry()
	wsHub := ws.NewHub()
	wsHub.SetAllowedOrigins(cfg.AllowedOrigins)
	go wsHub.Run()

	catalogUC := usecase.NewCatalogUsecase(cfg, catalogRepo)
	orderUC := usecase.NewOrderUsecase(cfg, orderRepo)
	orderUC.SetBroadcaster(wsHub)
	shiftUC := usecase.NewShiftUsecase(shiftRepo, txMgr)

	checkoutUC := usecase.NewCheckoutUsecase(
		cfg,
		txMgr,
		catalogRepo,
		orderRepo,
		outboxRepo,
		strategyRegistry,
		wsHub,
	)

	ticketUC := usecase.NewTicketUsecase(
		cfg,
		txMgr,
		ticketRepo,
		outboxRepo,
		wsHub,
	)

	var syncWorker *usecase.SyncWorker
	if cfg.AppMode == config.ModeEdge {
		syncWorker = usecase.NewSyncWorker(
			cfg,
			txMgr,
			outboxRepo,
			catalogRepo,
			orderRepo,
			ticketRepo,
		)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if syncWorker != nil {
		syncWorker.Start(ctx)
		defer syncWorker.Stop()
	}

	healthChecker := database.NewDatabaseHealthChecker(db)
	handler := deliveryHttp.NewHandler(
		cfg,
		healthChecker,
		catalogUC,
		orderUC,
		checkoutUC,
		ticketUC,
		shiftUC,
		syncWorker,
		txMgr,
		catalogRepo,
		orderRepo,
		ticketRepo,
		shiftRepo,
	)

	router := deliveryHttp.SetupRouter(handler, wsHub, cfg)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("[Engine] HTTP REST API & WS listening on http://0.0.0.0:%d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		log.Fatalf("[Engine] Critical HTTP server failure: %v", err)
	case <-ctx.Done():
		log.Println("[Engine] Shutdown signal received. Initiating graceful shutdown...")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Engine] Error during server shutdown: %v", err)
	} else {
		log.Println("[Engine] HTTP connections drained successfully.")
	}

	log.Println("[Engine] Process terminated cleanly.")
}

func ensureCryptoKeys(cfg *config.Config) {
	if cfg.Ed25519PrivKey != "" && cfg.Ed25519PubKey != "" {
		return
	}
	if cfg.AppEnv == "production" && cfg.AppMode == config.ModeCloud {
		log.Fatal("[Engine] Production CLOUD mode requires explicit ED25519_PRIVATE_KEY and ED25519_PUBLIC_KEY")
	}

	pubKey, privKey, err := crypto.GenerateKeyPair()
	if err != nil {
		return
	}

	if cfg.Ed25519PrivKey == "" {
		cfg.Ed25519PrivKey = privKey
		log.Printf("[Engine] Generated temporary ED25519_PRIVATE_KEY for %s mode", cfg.AppMode)
	}
	if cfg.Ed25519PubKey == "" {
		cfg.Ed25519PubKey = pubKey
		log.Printf("[Engine] Generated temporary ED25519_PUBLIC_KEY for %s mode", cfg.AppMode)
	}
}

func seedDemoData(cfg *config.Config, catRepo domain.CatalogRepository, orderRepo domain.OrderRepository) {
	ctx := context.Background()
	existing, err := catRepo.ListItems(ctx, cfg.TenantID, 1, 0)
	if err == nil && len(existing) > 0 {
		return
	}

	log.Printf("[Engine] Seeding initial multi-industry catalog & guest folios for tenant '%s'...", cfg.TenantID)

	fbModifiers, _ := json.Marshal(map[string]any{
		"modifiers": []map[string]any{
			{"name": "Extra Shot", "price_delta": 5000},
			{"name": "Oat Milk Sub", "price_delta": 8000},
			{"name": "Vanilla Syrup", "price_delta": 4000},
			{"name": "Less Sugar (Normal)", "price_delta": 0},
			{"name": "Less Ice (70%)", "price_delta": 0},
		},
	})

	demoItems := []*domain.Item{
		{
			ID:           uid.NewV7(),
			TenantID:     cfg.TenantID,
			OutletID:     cfg.OutletID,
			SKU:          "COF-LATTE-01",
			Barcode:      "8991001",
			Name:         "Caffe Latte",
			Sector:       domain.SectorFB,
			BasePrice:    32000,
			Currency:     "IDR",
			TaxRate:      0.1,
			SectorTraits: fbModifiers,
			IsActive:     true,
		},
		{
			ID:           uid.NewV7(),
			TenantID:     cfg.TenantID,
			OutletID:     cfg.OutletID,
			SKU:          "COF-AME-01",
			Barcode:      "8991002",
			Name:         "Iced Americano",
			Sector:       domain.SectorFB,
			BasePrice:    25000,
			Currency:     "IDR",
			TaxRate:      0.1,
			SectorTraits: fbModifiers,
			IsActive:     true,
		},
		{
			ID:           uid.NewV7(),
			TenantID:     cfg.TenantID,
			OutletID:     cfg.OutletID,
			SKU:          "FOD-NDL-01",
			Barcode:      "8991003",
			Name:         "Indomie Goreng Special",
			Sector:       domain.SectorFB,
			BasePrice:    18000,
			Currency:     "IDR",
			TaxRate:      0.1,
			SectorTraits: []byte(`{}`),
			IsActive:     true,
		},
		{
			ID:           uid.NewV7(),
			TenantID:     cfg.TenantID,
			OutletID:     cfg.OutletID,
			SKU:          "SNK-CHIPS-01",
			Barcode:      "899123456001",
			Name:         "Artisan Truffle Chips 80g",
			Sector:       domain.SectorRetail,
			BasePrice:    15000,
			Currency:     "IDR",
			TaxRate:      0.1,
			SectorTraits: []byte(`{}`),
			IsActive:     true,
		},
		{
			ID:           uid.NewV7(),
			TenantID:     cfg.TenantID,
			OutletID:     cfg.OutletID,
			SKU:          "BEV-WATER-01",
			Barcode:      "899123456002",
			Name:         "Pure Mineral Water 600ml",
			Sector:       domain.SectorRetail,
			BasePrice:    6000,
			Currency:     "IDR",
			TaxRate:      0.1,
			SectorTraits: []byte(`{}`),
			IsActive:     true,
		},
		{
			ID:           uid.NewV7(),
			TenantID:     cfg.TenantID,
			OutletID:     cfg.OutletID,
			SKU:          "PASS-ADULT-01",
			Barcode:      "8992001",
			Name:         "Theme Park Full Day Pass",
			Sector:       domain.SectorTourism,
			BasePrice:    125000,
			Currency:     "IDR",
			TaxRate:      0.1,
			SectorTraits: []byte(`{"pax_per_ticket": 1, "valid_duration_hours": 24}`),
			IsActive:     true,
		},
		{
			ID:           uid.NewV7(),
			TenantID:     cfg.TenantID,
			OutletID:     cfg.OutletID,
			SKU:          "ROOM-DELUXE-01",
			Barcode:      "8993001",
			Name:         "Deluxe Villa Stay",
			Sector:       domain.SectorHospitality,
			BasePrice:    750000,
			Currency:     "IDR",
			TaxRate:      0.1,
			SectorTraits: []byte(`{"room_type": "DELUXE"}`),
			IsActive:     true,
		},
	}

	for _, it := range demoItems {
		_ = catRepo.UpsertItem(ctx, it)
		_ = catRepo.RecordStockMovement(ctx, &domain.StockMovementEvent{
			ID:        uid.NewV7(),
			TenantID:  cfg.TenantID,
			OutletID:  cfg.OutletID,
			SKU:       it.SKU,
			DeltaQty:  1000,
			Reason:    "INITIAL_DEMO_SEED",
			CreatedAt: time.Now().UTC(),
		})
	}

	// Seed Sample Guest Folios
	_ = orderRepo.CreateFolio(ctx, &domain.GuestFolio{
		ID:           uid.NewV7(),
		TenantID:     cfg.TenantID,
		OutletID:     cfg.OutletID,
		RoomNumber:   "101",
		GuestName:    "Budi Santoso",
		CreditLimit:  2000000,
		CurrentTotal: 0,
		IsOpen:       true,
		CheckInAt:    time.Now().UTC(),
		CheckOutAt:   time.Now().UTC().Add(48 * time.Hour),
	})

	_ = orderRepo.CreateFolio(ctx, &domain.GuestFolio{
		ID:           uid.NewV7(),
		TenantID:     cfg.TenantID,
		OutletID:     cfg.OutletID,
		RoomNumber:   "102",
		GuestName:    "Sarah Jenkins",
		CreditLimit:  1500000,
		CurrentTotal: 0,
		IsOpen:       true,
		CheckInAt:    time.Now().UTC(),
		CheckOutAt:   time.Now().UTC().Add(24 * time.Hour),
	})

	log.Printf("[Engine] Successfully seeded %d demo items and 2 guest room folios.", len(demoItems))
}

