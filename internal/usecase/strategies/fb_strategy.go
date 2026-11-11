package strategies

import (
	"context"
	"encoding/json"
	"fmt"

	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/uid"
)

type FBStrategy struct{}

func NewFBStrategy() *FBStrategy {
	return &FBStrategy{}
}

func (s *FBStrategy) Sector() domain.ItemSector {
	return domain.SectorFB
}

func (s *FBStrategy) ValidateAndPriceItem(ctx context.Context, item *domain.Item, orderItem *domain.OrderItem) error {
	var trait domain.FBTrait
	if len(item.SectorTraits) > 0 {
		if err := json.Unmarshal(item.SectorTraits, &trait); err != nil {
			return fmt.Errorf("%w: failed to parse FB traits: %v", domain.ErrInvalidSectorTrait, err)
		}
	}

	// Calculate base unit price
	unitPrice := item.BasePrice

	// Validate modifiers and sum price additions
	if len(orderItem.SelectedModifiers) > 0 {
		var selected []domain.Modifier
		if err := json.Unmarshal(orderItem.SelectedModifiers, &selected); err != nil {
			return fmt.Errorf("invalid modifier format: %w", err)
		}

		for _, sel := range selected {
			unitPrice += sel.AddPrice
		}
	}

	orderItem.UnitPrice = unitPrice
	orderItem.Subtotal = int64(float64(orderItem.UnitPrice) * orderItem.Quantity)
	orderItem.TaxAmount = int64(float64(orderItem.Subtotal) * item.TaxRate)
	orderItem.TotalAmount = orderItem.Subtotal + orderItem.TaxAmount

	// Ensure station routing metadata is recorded
	station := trait.KitchenStation
	if station == "" {
		station = "KITCHEN"
	}
	routingData, _ := json.Marshal(map[string]any{
		"kitchen_station": station,
	})
	orderItem.SectorPayload = routingData

	return nil
}

func (s *FBStrategy) ProcessFulfillment(
	ctx context.Context,
	execCtx ExecutionContext,
	item *domain.Item,
	orderItem *domain.OrderItem,
) ([]domain.StockMovementEvent, error) {
	var trait domain.FBTrait
	if len(item.SectorTraits) > 0 {
		_ = json.Unmarshal(item.SectorTraits, &trait)
	}

	var movements []domain.StockMovementEvent

	// If BOM recipe is configured, deduct raw ingredients
	if len(trait.Recipe) > 0 {
		for _, ing := range trait.Recipe {
			totalDeduct := ing.Quantity * orderItem.Quantity
			movements = append(movements, domain.StockMovementEvent{
				ID:          uid.NewV7(),
				TenantID:    execCtx.TenantID,
				OutletID:    execCtx.OutletID,
				SKU:         ing.RawMaterialSKU,
				DeltaQty:    -totalDeduct,
				Reason:      "BOM_RECIPE_DEDUCTION",
				ReferenceID: execCtx.OrderID,
			})
		}
	} else {
		// Fallback: direct deduction of finished product SKU
		movements = append(movements, domain.StockMovementEvent{
			ID:          uid.NewV7(),
			TenantID:    execCtx.TenantID,
			OutletID:    execCtx.OutletID,
			SKU:         item.SKU,
			DeltaQty:    -orderItem.Quantity,
			Reason:      "SALE_DEDUCTION",
			ReferenceID: execCtx.OrderID,
		})
	}

	return movements, nil
}
