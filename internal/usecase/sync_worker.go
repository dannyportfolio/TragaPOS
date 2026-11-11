package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"pos-engine/internal/config"
	"pos-engine/internal/domain"
	"pos-engine/internal/repository"
)

type SyncWorker struct {
	cfg               *config.Config
	txMgr             *repository.TxManager
	outboxRepo        domain.OutboxRepository
	catalogRepo       domain.CatalogRepository
	orderRepo         domain.OrderRepository
	ticketRepo        domain.TicketRepository
	httpClient        *http.Client
	lastSyncedVersion int64
	mu                sync.Mutex
	stopCh            chan struct{}
}

func NewSyncWorker(
	cfg *config.Config,
	txMgr *repository.TxManager,
	outboxRepo domain.OutboxRepository,
	catalogRepo domain.CatalogRepository,
	orderRepo domain.OrderRepository,
	ticketRepo domain.TicketRepository,
) *SyncWorker {
	return &SyncWorker{
		cfg:         cfg,
		txMgr:       txMgr,
		outboxRepo:  outboxRepo,
		catalogRepo: catalogRepo,
		orderRepo:   orderRepo,
		ticketRepo:  ticketRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		stopCh: make(chan struct{}),
	}
}

// Start launches background syncing in EDGE mode:
// 1. Upstream Pusher: pushes pending outbox batches to Cloud /v1/sync/push
// 2. Downstream Puller: fetches updated catalog delta from Cloud /v1/sync/pull
func (w *SyncWorker) Start(ctx context.Context) {
	if w.cfg.AppMode != config.ModeEdge {
		log.Println("[SyncWorker] Not running in EDGE mode, background sync worker disabled.")
		return
	}

	log.Printf("[SyncWorker] Started for Tenant=%s Outlet=%s (polling every %v to %s)",
		w.cfg.TenantID, w.cfg.OutletID, w.cfg.SyncInterval, w.cfg.CloudSyncURL)

	ticker := time.NewTicker(w.cfg.SyncInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("[SyncWorker] Context cancelled, shutting down sync loop.")
				return
			case <-w.stopCh:
				log.Println("[SyncWorker] Stop signal received.")
				return
			case <-ticker.C:
				w.syncCycle(ctx)
			}
		}
	}()
}

func (w *SyncWorker) Stop() {
	select {
	case <-w.stopCh:
	default:
		close(w.stopCh)
	}
}

func (w *SyncWorker) syncCycle(ctx context.Context) {
	// A. Push Upstream Transactional Events
	if err := w.PushOutboxBatch(ctx); err != nil {
		log.Printf("[SyncWorker] Push error: %v", err)
	}

	// B. Pull Downstream Master Data Delta
	if err := w.PullCatalogDelta(ctx); err != nil {
		log.Printf("[SyncWorker] Pull error: %v", err)
	}
}

func (w *SyncWorker) PushOutboxBatch(ctx context.Context) error {
	batch, err := w.outboxRepo.FetchPendingBatch(ctx, w.cfg.SyncBatchSize)
	if err != nil {
		return fmt.Errorf("failed to fetch pending outbox: %w", err)
	}
	if len(batch) == 0 {
		return nil // Nothing to push
	}

	idempotencyKey := fmt.Sprintf("push-%s-%d", w.cfg.OutletID, time.Now().UTC().UnixNano())
	payload := domain.SyncPushBatch{
		TenantID:       w.cfg.TenantID,
		OutletID:       w.cfg.OutletID,
		IdempotencyKey: idempotencyKey,
		Events:         batch,
		SentAt:         time.Now().UTC(),
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal push batch: %w", err)
	}

	pushURL := fmt.Sprintf("%s/v1/sync/push", w.cfg.CloudSyncURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pushURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.cfg.EdgeAuthToken)
	req.Header.Set("X-Idempotency-Key", idempotencyKey)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		for _, ev := range batch {
			_ = w.outboxRepo.MarkFailed(ctx, ev.ID, err.Error())
		}
		return fmt.Errorf("http push error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		errStr := fmt.Sprintf("cloud rejected batch (status %d): %s", resp.StatusCode, string(respBody))
		for _, ev := range batch {
			_ = w.outboxRepo.MarkFailed(ctx, ev.ID, errStr)
		}
		return errors.New(errStr)
	}

	var syncResp domain.SyncPushResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return fmt.Errorf("failed to decode cloud sync response: %w", err)
	}

	if len(syncResp.AckEventIDs) > 0 {
		if err := w.outboxRepo.MarkSent(ctx, syncResp.AckEventIDs); err != nil {
			return fmt.Errorf("failed to mark sent outbox records: %w", err)
		}
		log.Printf("[SyncWorker] Successfully pushed and ACKed %d events to Cloud Hub", len(syncResp.AckEventIDs))
	}

	return nil
}

func (w *SyncWorker) PullCatalogDelta(ctx context.Context) error {
	w.mu.Lock()
	currentVer := w.lastSyncedVersion
	w.mu.Unlock()

	pullURL := fmt.Sprintf("%s/v1/sync/pull?tenant_id=%s&last_synced_version=%d&limit=100",
		w.cfg.CloudSyncURL, w.cfg.TenantID, currentVer)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to build pull request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+w.cfg.EdgeAuthToken)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute pull: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloud sync pull error (%d): %s", resp.StatusCode, string(body))
	}

	var pullResp domain.SyncPullResponse
	if err := json.NewDecoder(resp.Body).Decode(&pullResp); err != nil {
		return fmt.Errorf("failed to decode pull response: %w", err)
	}

	if len(pullResp.Items) > 0 {
		log.Printf("[SyncWorker] Received %d delta catalog items from Cloud", len(pullResp.Items))
		for _, item := range pullResp.Items {
			it := item
			if err := w.catalogRepo.UpsertItem(ctx, &it); err != nil {
				log.Printf("[SyncWorker] Failed to upsert pulled catalog item %s: %v", it.SKU, err)
			}
		}

		w.mu.Lock()
		if pullResp.LatestVersion > w.lastSyncedVersion {
			w.lastSyncedVersion = pullResp.LatestVersion
		}
		w.mu.Unlock()
	}

	return nil
}

// IngestUpstreamBatch processes incoming Edge sync batches on the Cloud server.
// Enforces "Physical Reality Wins": paid offline sales are accepted even if stock is negative.
func IngestUpstreamBatch(
	ctx context.Context,
	txMgr *repository.TxManager,
	catalogRepo domain.CatalogRepository,
	orderRepo domain.OrderRepository,
	ticketRepo domain.TicketRepository,
	batch domain.SyncPushBatch,
) (*domain.SyncPushResponse, error) {
	var ackIDs []string
	var discrepancies []domain.DiscrepancyNotice

	err := txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, event := range batch.Events {
			switch event.EventType {
			case domain.EventOrderCreated:
				var order domain.Order
				if err := json.Unmarshal(event.Payload, &order); err == nil {
					_ = orderRepo.CreateOrder(txCtx, &order)
				}

			case domain.EventPaymentReceived:
				var payment domain.Payment
				if err := json.Unmarshal(event.Payload, &payment); err == nil {
					_ = orderRepo.AddPayment(txCtx, &payment)
				}

			case domain.EventStockMoved:
				var mov domain.StockMovementEvent
				if err := json.Unmarshal(event.Payload, &mov); err == nil {
					// Physical Reality Wins: record stock movement directly
					_ = catalogRepo.RecordStockMovement(txCtx, &mov)

					// Check if stock became negative and flag alert
					bal, _ := catalogRepo.GetStock(txCtx, mov.TenantID, mov.OutletID, mov.SKU)
					if bal != nil && bal.OnHandQty < 0 {
						discrepancies = append(discrepancies, domain.DiscrepancyNotice{
							SKU:            mov.SKU,
							RequestedDelta: mov.DeltaQty,
							CurrentStock:   bal.OnHandQty,
							Message:        fmt.Sprintf("PHYSICAL REALITY WINS: Stock for %s became negative (%v) due to paid offline order", mov.SKU, bal.OnHandQty),
						})
					}
				}

			case domain.EventTicketRedeemed:
				var red domain.TicketRedemption
				if err := json.Unmarshal(event.Payload, &red); err == nil {
					_ = ticketRepo.RecordRedemption(txCtx, &red)
				}
			}

			ackIDs = append(ackIDs, event.ID)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &domain.SyncPushResponse{
		ProcessedCount: len(ackIDs),
		AckEventIDs:    ackIDs,
		Discrepancies:  discrepancies,
	}, nil
}
