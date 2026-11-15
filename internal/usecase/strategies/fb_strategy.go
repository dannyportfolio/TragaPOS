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

	unitPrice := item.BasePrice

	if len(orderItem.SelectedModifiers) > 0 {
		var selected []domain.Modifier
		if err := json.Unmarshal(orderItem.SelectedModifiers, &selected); err != nil {
			return fmt.Errorf("invalid modifier format: %w", err)
		}

		allowedByID := make(map[string]domain.Modifier)
		allowedByName := make(map[string]domain.Modifier)
		for _, grp := range trait.ModifierGroups {
			for _, m := range grp.Modifiers {
				if m.ID != "" {
					allowedByID[m.ID] = m
				}
				if m.Name != "" {
					allowedByName[m.Name] = m
				}
			}
		}

		for i, sel := range selected {
			if sel.AddPrice < 0 {
				return fmt.Errorf("modifier %q has invalid negative price", sel.Name)
			}
			if len(trait.ModifierGroups) > 0 {
				catalogMod, ok := allowedByID[sel.ID]
				if !ok {
					catalogMod, ok = allowedByName[sel.Name]
				}
				if !ok {
					return fmt.Errorf("unknown modifier %q for item %s", sel.Name, item.SKU)
				}
				// Always enforce server-side catalog modifier price
				selected[i].AddPrice = catalogMod.AddPrice
				unitPrice += catalogMod.AddPrice
			} else {
				unitPrice += sel.AddPrice
			}
		}

		if normalized, err := json.Marshal(selected); err == nil {
			orderItem.SelectedModifiers = normalized
		}
	}

	orderItem.UnitPrice = unitPrice
	orderItem.Subtotal = int64(float64(orderItem.UnitPrice) * orderItem.Quantity)
	orderItem.TaxAmount = int64(float64(orderItem.Subtotal) * item.TaxRate)
	orderItem.TotalAmount = orderItem.Subtotal + orderItem.TaxAmount

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
