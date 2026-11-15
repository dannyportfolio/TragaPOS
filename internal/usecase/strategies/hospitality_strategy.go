package strategies

import (
	"context"
	"encoding/json"
	"fmt"

	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/uid"
)

type HospitalityStrategy struct{}

func NewHospitalityStrategy() *HospitalityStrategy {
	return &HospitalityStrategy{}
}

func (s *HospitalityStrategy) Sector() domain.ItemSector {
	return domain.SectorHospitality
}

func (s *HospitalityStrategy) ValidateAndPriceItem(ctx context.Context, item *domain.Item, orderItem *domain.OrderItem) error {
	var trait domain.HospitalityTrait
	if len(item.SectorTraits) > 0 {
		if err := json.Unmarshal(item.SectorTraits, &trait); err != nil {
			return fmt.Errorf("%w: failed to parse Hospitality traits: %v", domain.ErrInvalidSectorTrait, err)
		}
	}

	orderItem.UnitPrice = item.BasePrice
	orderItem.Subtotal = int64(float64(orderItem.UnitPrice) * orderItem.Quantity)
	orderItem.TaxAmount = int64(float64(orderItem.Subtotal) * item.TaxRate)
	orderItem.TotalAmount = orderItem.Subtotal + orderItem.TaxAmount

	meta, _ := json.Marshal(map[string]any{
		"room_facility_id":   trait.RoomFacilityID,
		"room_type":          trait.RoomType,
		"housekeeping_state": trait.HousekeepingState,
		"max_occupancy":      trait.MaxOccupancy,
	})
	orderItem.SectorPayload = meta

	return nil
}

func (s *HospitalityStrategy) ProcessFulfillment(
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
			Reason:      "FACILITY_OCCUPIED",
			ReferenceID: execCtx.OrderID,
		},
	}, nil
}
