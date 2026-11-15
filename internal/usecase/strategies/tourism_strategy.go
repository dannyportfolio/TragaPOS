package strategies

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/uid"
)

type TourismStrategy struct{}

func NewTourismStrategy() *TourismStrategy {
	return &TourismStrategy{}
}

func (s *TourismStrategy) Sector() domain.ItemSector {
	return domain.SectorTourism
}

func (s *TourismStrategy) ValidateAndPriceItem(ctx context.Context, item *domain.Item, orderItem *domain.OrderItem) error {
	var trait domain.TourismTrait
	if len(item.SectorTraits) > 0 {
		if err := json.Unmarshal(item.SectorTraits, &trait); err != nil {
			return fmt.Errorf("%w: failed to parse Tourism traits: %v", domain.ErrInvalidSectorTrait, err)
		}
	}

	orderItem.UnitPrice = item.BasePrice
	orderItem.Subtotal = int64(float64(orderItem.UnitPrice) * orderItem.Quantity)
	orderItem.TaxAmount = int64(float64(orderItem.Subtotal) * item.TaxRate)
	orderItem.TotalAmount = orderItem.Subtotal + orderItem.TaxAmount

	hours := trait.ValidDurationHours
	if hours <= 0 {
		hours = 24
	}
	validUntil := time.Now().UTC().Add(time.Duration(hours) * time.Hour).Unix()
	pax := trait.PaxPerTicket
	if pax <= 0 {
		pax = 1
	}

	meta, _ := json.Marshal(map[string]any{
		"pass_category":  trait.PassCategory,
		"pax_per_ticket": pax,
		"valid_until":    validUntil,
		"allowed_zones":  trait.AllowedGateZones,
	})
	orderItem.SectorPayload = meta

	return nil
}

func (s *TourismStrategy) ProcessFulfillment(
	ctx context.Context,
	execCtx ExecutionContext,
	item *domain.Item,
	orderItem *domain.OrderItem,
) ([]domain.StockMovementEvent, error) {
	return []domain.StockMovementEvent{
		{
			ID:          uid.NewV7(),
			TenantID:    execCtx.TenantID,
			OutletID:    execCtx.OutletID,
			SKU:         item.SKU,
			DeltaQty:    -orderItem.Quantity,
			Reason:      "TICKET_PASS_ISSUED",
			ReferenceID: execCtx.OrderID,
		},
	}, nil
}
