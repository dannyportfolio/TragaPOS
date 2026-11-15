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
	Method     domain.PaymentMethod `json:"method"`
	Amount     int64                `json:"amount"`
	Reference  string               `json:"reference,omitempty"`
	RoomNumber string               `json:"room_number,omitempty"`
}

type CheckoutRequest struct {
	OrderID  string              `json:"order_id,omitempty"`
	TenantID string              `json:"tenant_id"`
	OutletID string              `json:"outlet_id"`
	Notes    string              `json:"notes,omitempty"`
	Metadata json.RawMessage     `json:"metadata,omitempty"`
	Items    []CheckoutItemInput `json:"items"`
	Payment  *PaymentInput       `json:"payment,omitempty"`
	Payments []PaymentInput      `json:"payments,omitempty"`
}

type CheckoutUsecase struct {
	cfg         *config.Config
	txMgr       domain.TransactionManager
	catalogRepo domain.CatalogRepository
	orderRepo   domain.OrderRepository
	outboxRepo  domain.OutboxRepository
	strategies  *strategies.Registry
	broadcaster EventBroadcaster
}

func NewCheckoutUsecase(
	cfg *config.Config,
	txMgr domain.TransactionManager,
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

	orderID := req.OrderID
	if orderID == "" {
		orderID = uid.NewV7()
	}
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

	if err := uc.populateOrderItems(ctx, req, order); err != nil {
		return nil, err
	}

	paymentRecords, err := uc.preparePayments(req, order)
	if err != nil {
		return nil, err
	}

	err = uc.txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.orderRepo.CreateOrder(txCtx, order); err != nil {
			return fmt.Errorf("failed to save order: %w", err)
		}

		if err := uc.processRoomCharges(txCtx, req, order, paymentRecords); err != nil {
			return err
		}

		if err := uc.processFulfillmentsAndMovements(txCtx, req, order); err != nil {
			return err
		}

		return uc.enqueueOrderOutboxEvents(txCtx, req, order, paymentRecords)
	})

	if err != nil {
		return nil, err
	}

	uc.dispatchOrderEvents(order, paymentRecords)
	return order, nil
}

func (uc *CheckoutUsecase) HoldOrder(ctx context.Context, req CheckoutRequest) (*domain.Order, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("cannot hold order with empty items")
	}

	orderID := req.OrderID
	if orderID == "" {
		orderID = uid.NewV7()
	}
	orderNumber := fmt.Sprintf("HOLD-%s-%s", req.OutletID, orderID[len(orderID)-6:])

	order := &domain.Order{
		ID:          orderID,
		TenantID:    req.TenantID,
		OutletID:    req.OutletID,
		OrderNumber: orderNumber,
		Status:      domain.OrderStatusHeld,
		Currency:    "IDR",
		Notes:       req.Notes,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := uc.populateOrderItems(ctx, req, order); err != nil {
		return nil, err
	}

	if err := uc.orderRepo.CreateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to hold order: %w", err)
	}

	if uc.broadcaster != nil {
		uc.broadcaster.BroadcastEvent("orders", "ORDER_HELD", order)
	}

	return order, nil
}

func (uc *CheckoutUsecase) populateOrderItems(ctx context.Context, req CheckoutRequest, order *domain.Order) error {
	var totalSubtotal int64
	var totalTax int64

	for _, itInput := range req.Items {
		item, err := uc.findCatalogItem(ctx, req.TenantID, itInput)
		if err != nil {
			return err
		}
		if !item.IsActive {
			return fmt.Errorf("%w: %s", domain.ErrInactiveItem, item.SKU)
		}

		strategy, err := uc.strategies.Get(item.Sector)
		if err != nil {
			return fmt.Errorf("failed to resolve fulfillment strategy for sector %s: %w", item.Sector, err)
		}

		orderItem := domain.OrderItem{
			ID:                uid.NewV7(),
			OrderID:           order.ID,
			ItemID:            item.ID,
			SKU:               item.SKU,
			Name:              item.Name,
			Sector:            item.Sector,
			Quantity:          itInput.Quantity,
			UOM:               itInput.UOM,
			SelectedModifiers: itInput.SelectedModifiers,
			SectorPayload:     itInput.SectorPayload,
		}

		if err := strategy.ValidateAndPriceItem(ctx, item, &orderItem); err != nil {
			return fmt.Errorf("strategy error on item %s: %w", item.SKU, err)
		}

		totalSubtotal += orderItem.Subtotal
		totalTax += orderItem.TaxAmount
		order.Items = append(order.Items, orderItem)
	}

	order.Subtotal = totalSubtotal
	order.TaxAmount = totalTax
	order.TotalAmount = totalSubtotal + totalTax
	return nil
}

func (uc *CheckoutUsecase) findCatalogItem(ctx context.Context, tenantID string, input CheckoutItemInput) (*domain.Item, error) {
	if input.Barcode != "" {
		item, err := uc.catalogRepo.GetByBarcode(ctx, tenantID, input.Barcode)
		if err != nil {
			return nil, fmt.Errorf("item lookup failed by barcode %s: %w", input.Barcode, err)
		}
		return item, nil
	}
	if input.SKU != "" {
		item, err := uc.catalogRepo.GetBySKU(ctx, tenantID, input.SKU)
		if err != nil {
			return nil, fmt.Errorf("item lookup failed by sku %s: %w", input.SKU, err)
		}
		return item, nil
	}
	return nil, errors.New("each item requires either 'sku' or 'barcode'")
}

func (uc *CheckoutUsecase) preparePayments(req CheckoutRequest, order *domain.Order) ([]*domain.Payment, error) {
	inputs := req.Payments
	if len(inputs) == 0 && req.Payment != nil {
		inputs = []PaymentInput{*req.Payment}
	}

	if len(inputs) == 0 {
		return nil, nil
	}

	var totalPaid int64
	for _, p := range inputs {
		if p.Amount <= 0 {
			return nil, errors.New("each payment allocation must have an amount greater than zero")
		}
		totalPaid += p.Amount
	}

	if totalPaid < order.TotalAmount {
		return nil, fmt.Errorf("total payments (%d) is less than required order total (%d)", totalPaid, order.TotalAmount)
	}

	now := time.Now().UTC()
	var paymentRecords []*domain.Payment
	for _, p := range inputs {
		record := &domain.Payment{
			ID:        uid.NewV7(),
			OrderID:   order.ID,
			Method:    p.Method,
			Amount:    p.Amount,
			Reference: p.Reference,
			PaidAt:    now,
		}
		order.Payments = append(order.Payments, *record)
		paymentRecords = append(paymentRecords, record)
	}

	order.Status = domain.OrderStatusPaid
	return paymentRecords, nil
}

func (uc *CheckoutUsecase) processRoomCharges(ctx context.Context, req CheckoutRequest, order *domain.Order, paymentRecords []*domain.Payment) error {
	inputs := req.Payments
	if len(inputs) == 0 && req.Payment != nil {
		inputs = []PaymentInput{*req.Payment}
	}

	for i, record := range paymentRecords {
		if record.Method != domain.PaymentMethodRoomCharge {
			continue
		}

		var roomNum string
		if i < len(inputs) {
			roomNum = inputs[i].RoomNumber
		}
		if roomNum == "" && req.Payment != nil {
			roomNum = req.Payment.RoomNumber
		}
		if roomNum == "" {
			return errors.New("room_number is required for ROOM_CHARGE payment")
		}

		folio, err := uc.orderRepo.GetFolioByRoom(ctx, req.TenantID, roomNum)
		if err != nil {
			return fmt.Errorf("failed to find active folio for room %s: %w", roomNum, err)
		}

		charge := &domain.FolioCharge{
			ID:        uid.NewV7(),
			FolioID:   folio.ID,
			OrderID:   order.ID,
			Amount:    record.Amount,
			Notes:     fmt.Sprintf("Room Charge for %s (part %d)", order.OrderNumber, i+1),
			CreatedAt: time.Now().UTC(),
		}

		if err := uc.orderRepo.ChargeFolio(ctx, charge); err != nil {
			return fmt.Errorf("failed to charge room folio: %w", err)
		}

		record.Reference = folio.ID
	}

	return nil
}

func (uc *CheckoutUsecase) processFulfillmentsAndMovements(ctx context.Context, req CheckoutRequest, order *domain.Order) error {
	execContext := strategies.ExecutionContext{
		TenantID: req.TenantID,
		OutletID: req.OutletID,
		OrderID:  order.ID,
	}

	for idx, orderItem := range order.Items {
		item, err := uc.catalogRepo.GetByID(ctx, orderItem.ItemID)
		if err != nil {
			return fmt.Errorf("failed to load catalog item %s: %w", orderItem.ItemID, err)
		}
		strategy, err := uc.strategies.Get(orderItem.Sector)
		if err != nil {
			return err
		}

		movements, err := strategy.ProcessFulfillment(ctx, execContext, item, &order.Items[idx])
		if err != nil {
			return fmt.Errorf("fulfillment error: %w", err)
		}

		for _, mov := range movements {
			if err := uc.catalogRepo.RecordStockMovement(ctx, &mov); err != nil {
				return fmt.Errorf("stock deduction error: %w", err)
			}

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
				if err := uc.outboxRepo.Enqueue(ctx, outboxRecord); err != nil {
					return fmt.Errorf("failed to enqueue stock outbox: %w", err)
				}
			}
		}
	}
	return nil
}

func (uc *CheckoutUsecase) enqueueOrderOutboxEvents(ctx context.Context, req CheckoutRequest, order *domain.Order, paymentRecords []*domain.Payment) error {
	if uc.cfg.AppMode != config.ModeEdge || uc.outboxRepo == nil {
		return nil
	}

	orderBytes, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order outbox: %w", err)
	}

	outboxOrder := &domain.SyncOutbox{
		ID:            uid.NewV7(),
		TenantID:      req.TenantID,
		OutletID:      req.OutletID,
		EventType:     domain.EventOrderCreated,
		AggregateType: "order",
		AggregateID:   order.ID,
		Payload:       orderBytes,
		Status:        domain.OutboxStatusPending,
	}
	if err := uc.outboxRepo.Enqueue(ctx, outboxOrder); err != nil {
		return fmt.Errorf("failed to enqueue order outbox: %w", err)
	}

	for _, paymentRecord := range paymentRecords {
		payBytes, err := json.Marshal(paymentRecord)
		if err != nil {
			return fmt.Errorf("failed to marshal payment outbox: %w", err)
		}
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
		if err := uc.outboxRepo.Enqueue(ctx, outboxPayment); err != nil {
			return fmt.Errorf("failed to enqueue payment outbox: %w", err)
		}
	}
	return nil
}

func (uc *CheckoutUsecase) dispatchOrderEvents(order *domain.Order, paymentRecords []*domain.Payment) {
	if uc.broadcaster == nil {
		return
	}

	uc.broadcaster.BroadcastEvent("orders", "ORDER_CREATED", order)

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

	for _, paymentRecord := range paymentRecords {
		if paymentRecord.Method == domain.PaymentMethodQRIS {
			uc.broadcaster.BroadcastEvent("qris", "PAYMENT_SETTLED", map[string]any{
				"order_id":  order.ID,
				"amount":    paymentRecord.Amount,
				"reference": paymentRecord.Reference,
			})
		}
	}
}
