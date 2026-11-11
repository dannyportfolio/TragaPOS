package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"pos-engine/internal/config"
	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/uid"
	"pos-engine/internal/repository"
	"pos-engine/internal/usecase/strategies"
)

type EventBroadcaster interface {
	BroadcastEvent(topic, event string, payload any)
}

type CheckoutItemInput struct {
	SKU               string          `json:"sku"`
	Barcode           string          `json:"barcode,omitempty"`
	Quantity          float64         `json:"quantity"`
	UOM               string          `json:"uom,omitempty"`
	SelectedModifiers json.RawMessage `json:"selected_modifiers,omitempty"`
	SectorPayload     json.RawMessage `json:"sector_payload,omitempty"`
}

type PaymentInput struct {
	Method    domain.PaymentMethod `json:"method"`
	Amount    int64                `json:"amount"`
	Reference string               `json:"reference,omitempty"`
	RoomNumber string              `json:"room_number,omitempty"` // For ROOM_CHARGE
}

type CheckoutRequest struct {
	TenantID string              `json:"tenant_id"`
	OutletID string              `json:"outlet_id"`
	Notes    string              `json:"notes,omitempty"`
	Metadata json.RawMessage     `json:"metadata,omitempty"`
	Items    []CheckoutItemInput `json:"items"`
	Payment  *PaymentInput       `json:"payment,omitempty"`
}

type CheckoutUsecase struct {
	cfg          *config.Config
	txMgr        *repository.TxManager
	catalogRepo  domain.CatalogRepository
	orderRepo    domain.OrderRepository
	outboxRepo   domain.OutboxRepository
	strategies   *strategies.Registry
	broadcaster  EventBroadcaster
}

func NewCheckoutUsecase(
	cfg *config.Config,
	txMgr *repository.TxManager,
	catalogRepo domain.CatalogRepository,
	orderRepo domain.OrderRepository,
	outboxRepo domain.OutboxRepository,
	strategies *strategies.Registry,
	broadcaster EventBroadcaster,
) *CheckoutUsecase {
	return &CheckoutUsecase{
		cfg:         cfg,
		txMgr:       txMgr,
		catalogRepo: catalogRepo,
		orderRepo:   orderRepo,
		outboxRepo:  outboxRepo,
		strategies:  strategies,
		broadcaster: broadcaster,
	}
}

func (uc *CheckoutUsecase) Checkout(ctx context.Context, req CheckoutRequest) (*domain.Order, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("cannot checkout with empty items")
	}

	orderID := uid.NewV7()
	orderNumber := fmt.Sprintf("ORD-%s-%s", req.OutletID, orderID[len(orderID)-6:])

	order := &domain.Order{
		ID:          orderID,
		TenantID:    req.TenantID,
		OutletID:    req.OutletID,
		OrderNumber: orderNumber,
		Status:      domain.OrderStatusPending,
		Currency:    "IDR",
		Notes:       req.Notes,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	var totalSubtotal int64
	var totalTax int64

	// 1. Resolve each catalog item and validate through its sector strategy
	for _, itInput := range req.Items {
		var item *domain.Item
		var err error

		if itInput.Barcode != "" {
			item, err = uc.catalogRepo.GetByBarcode(ctx, req.TenantID, itInput.Barcode)
		} else if itInput.SKU != "" {
			item, err = uc.catalogRepo.GetBySKU(ctx, req.TenantID, itInput.SKU)
		} else {
			return nil, errors.New("each item requires either 'sku' or 'barcode'")
		}

		if err != nil {
			return nil, fmt.Errorf("item lookup failed: %w", err)
		}
		if !item.IsActive {
			return nil, fmt.Errorf("%w: %s", domain.ErrInactiveItem, item.SKU)
		}

		strategy, err := uc.strategies.Get(item.Sector)
		if err != nil {
			return nil, err
		}

		orderItem := domain.OrderItem{
			ID:                uid.NewV7(),
			OrderID:           orderID,
			ItemID:            item.ID,
			SKU:               item.SKU,
			Name:              item.Name,
			Sector:            item.Sector,
			Quantity:          itInput.Quantity,
			UOM:               itInput.UOM,
			SelectedModifiers: itInput.SelectedModifiers,
			SectorPayload:     itInput.SectorPayload,
		}

		// Apply sector pricing & validation
		if err := strategy.ValidateAndPriceItem(ctx, item, &orderItem); err != nil {
			return nil, fmt.Errorf("strategy error on item %s: %w", item.SKU, err)
		}

		totalSubtotal += orderItem.Subtotal
		totalTax += orderItem.TaxAmount
		order.Items = append(order.Items, orderItem)
	}

	order.Subtotal = totalSubtotal
	order.TaxAmount = totalTax
	order.TotalAmount = totalSubtotal + totalTax

	// 2. Prepare Payment if included
	var paymentRecord *domain.Payment
	if req.Payment != nil {
		if req.Payment.Amount < order.TotalAmount {
			return nil, fmt.Errorf("payment amount %d is less than required total %d", req.Payment.Amount, order.TotalAmount)
		}

		paymentRecord = &domain.Payment{
			ID:        uid.NewV7(),
			OrderID:   orderID,
			Method:    req.Payment.Method,
			Amount:    req.Payment.Amount,
			Reference: req.Payment.Reference,
			PaidAt:    time.Now().UTC(),
		}

		order.Status = domain.OrderStatusPaid
		order.Payments = append(order.Payments, *paymentRecord)
	}

	// 3. Execute in Atomic Database Transaction (WithTransaction)
	err := uc.txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
		// A. Insert Order & Items
		if err := uc.orderRepo.CreateOrder(txCtx, order); err != nil {
			return fmt.Errorf("failed to save order: %w", err)
		}

		// B. Handle Hospitality Room Charge if applicable
		if paymentRecord != nil && paymentRecord.Method == domain.PaymentMethodRoomCharge {
			roomNum := req.Payment.RoomNumber
			if roomNum == "" {
				return errors.New("room_number is required for ROOM_CHARGE payment")
			}
			folio, err := uc.orderRepo.GetFolioByRoom(txCtx, req.TenantID, roomNum)
			if err != nil {
				return fmt.Errorf("failed to find active folio for room %s: %w", roomNum, err)
			}
			charge := &domain.FolioCharge{
				ID:        uid.NewV7(),
				FolioID:   folio.ID,
				OrderID:   orderID,
				Amount:    order.TotalAmount,
				Notes:     fmt.Sprintf("Room Charge for %s", orderNumber),
				CreatedAt: time.Now().UTC(),
			}
			if err := uc.orderRepo.ChargeFolio(txCtx, charge); err != nil {
				return fmt.Errorf("failed to charge room folio: %w", err)
			}
			paymentRecord.Reference = folio.ID
		}

		// C. Sector Fulfillment Stock Deductions & Events
		execContext := strategies.ExecutionContext{
			TenantID: req.TenantID,
			OutletID: req.OutletID,
			OrderID:  orderID,
		}

		for idx, orderItem := range order.Items {
			item, _ := uc.catalogRepo.GetByID(txCtx, orderItem.ItemID)
			strategy, _ := uc.strategies.Get(orderItem.Sector)

			movements, err := strategy.ProcessFulfillment(txCtx, execContext, item, &order.Items[idx])
			if err != nil {
				return fmt.Errorf("fulfillment error: %w", err)
			}

			for _, mov := range movements {
				if err := uc.catalogRepo.RecordStockMovement(txCtx, &mov); err != nil {
					return fmt.Errorf("stock deduction error: %w", err)
				}

				// If in EDGE mode, enqueue stock movement into outbox
				if uc.cfg.AppMode == config.ModeEdge && uc.outboxRepo != nil {
					payloadBytes, _ := json.Marshal(mov)
					outboxRecord := &domain.SyncOutbox{
						ID:            uid.NewV7(),
						TenantID:      req.TenantID,
						OutletID:      req.OutletID,
						EventType:     domain.EventStockMoved,
						AggregateType: "stock",
						AggregateID:   mov.ID,
						Payload:       payloadBytes,
						Status:        domain.OutboxStatusPending,
					}
					if err := uc.outboxRepo.Enqueue(txCtx, outboxRecord); err != nil {
						return fmt.Errorf("failed to enqueue stock outbox: %w", err)
					}
				}
			}
		}

		// D. Transactional Outbox Pattern for EDGE Mode
		if uc.cfg.AppMode == config.ModeEdge && uc.outboxRepo != nil {
			orderBytes, _ := json.Marshal(order)
			outboxOrder := &domain.SyncOutbox{
				ID:            uid.NewV7(),
				TenantID:      req.TenantID,
				OutletID:      req.OutletID,
				EventType:     domain.EventOrderCreated,
				AggregateType: "order",
				AggregateID:   orderID,
				Payload:       orderBytes,
				Status:        domain.OutboxStatusPending,
			}
			if err := uc.outboxRepo.Enqueue(txCtx, outboxOrder); err != nil {
				return fmt.Errorf("failed to enqueue order outbox: %w", err)
			}

			if paymentRecord != nil {
				payBytes, _ := json.Marshal(paymentRecord)
				outboxPayment := &domain.SyncOutbox{
					ID:            uid.NewV7(),
					TenantID:      req.TenantID,
					OutletID:      req.OutletID,
					EventType:     domain.EventPaymentReceived,
					AggregateType: "payment",
					AggregateID:   paymentRecord.ID,
					Payload:       payBytes,
					Status:        domain.OutboxStatusPending,
				}
				if err := uc.outboxRepo.Enqueue(txCtx, outboxPayment); err != nil {
					return fmt.Errorf("failed to enqueue payment outbox: %w", err)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 4. Post-Commit Real-Time Event Dispatch
	if uc.broadcaster != nil {
		uc.broadcaster.BroadcastEvent("orders", "ORDER_CREATED", order)

		// Check if any F&B items exist to route to Kitchen Display System (KDS)
		var kdsItems []domain.OrderItem
		for _, item := range order.Items {
			if item.Sector == domain.SectorFB {
				kdsItems = append(kdsItems, item)
			}
		}
		if len(kdsItems) > 0 {
			uc.broadcaster.BroadcastEvent("kds", "NEW_KITCHEN_TICKET", map[string]any{
				"order_id":     order.ID,
				"order_number": order.OrderNumber,
				"items":        kdsItems,
				"notes":        order.Notes,
				"created_at":   order.CreatedAt,
			})
		}

		if paymentRecord != nil && paymentRecord.Method == domain.PaymentMethodQRIS {
			uc.broadcaster.BroadcastEvent("qris", "PAYMENT_SETTLED", map[string]any{
				"order_id":  order.ID,
				"amount":    paymentRecord.Amount,
				"reference": paymentRecord.Reference,
			})
		}
	}

	return order, nil
}
