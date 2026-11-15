package strategies

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/uid"
)

type RetailStrategy struct{}

func NewRetailStrategy() *RetailStrategy {
	return &RetailStrategy{}
}

func (s *RetailStrategy) Sector() domain.ItemSector {
	return domain.SectorRetail
}

func (s *RetailStrategy) ValidateAndPriceItem(ctx context.Context, item *domain.Item, orderItem *domain.OrderItem) error {
	var trait domain.RetailTrait
	if len(item.SectorTraits) > 0 {
		if err := json.Unmarshal(item.SectorTraits, &trait); err != nil {
			return fmt.Errorf("%w: failed to parse Retail traits: %v", domain.ErrInvalidSectorTrait, err)
		}
	}

	unitPrice := item.BasePrice
	multiplier := 1.0

	if orderItem.UOM != "" && trait.BaseUOM != "" && !strings.EqualFold(orderItem.UOM, trait.BaseUOM) {
		matched := false
		for _, conv := range trait.UOMConversions {
			if strings.EqualFold(conv.UOM, orderItem.UOM) {
				if conv.Price > 0 {
					unitPrice = conv.Price
				} else if conv.Multiplier > 0 {
					unitPrice = int64(float64(item.BasePrice) * conv.Multiplier)
				}
				multiplier = conv.Multiplier
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("unsupported UOM '%s' for item %s", orderItem.UOM, item.SKU)
		}
	} else if orderItem.UOM == "" {
		orderItem.UOM = trait.BaseUOM
		if orderItem.UOM == "" {
			orderItem.UOM = "PCS"
		}
	}

	intQty := int(orderItem.Quantity)
	for _, tier := range trait.TieredPricings {
		if intQty >= tier.MinQuantity && tier.UnitPrice > 0 {
			unitPrice = tier.UnitPrice
		}
	}

	orderItem.UnitPrice = unitPrice
	orderItem.Subtotal = int64(float64(orderItem.UnitPrice) * orderItem.Quantity)
	orderItem.TaxAmount = int64(float64(orderItem.Subtotal) * item.TaxRate)
	orderItem.TotalAmount = orderItem.Subtotal + orderItem.TaxAmount

	payloadData, _ := json.Marshal(map[string]any{
		"uom_multiplier": multiplier,
		"base_uom":       trait.BaseUOM,
		"gtin":           trait.GTIN,
	})
	orderItem.SectorPayload = payloadData

	return nil
}

func (s *RetailStrategy) ProcessFulfillment(
	ctx context.Context,
	execCtx ExecutionContext,
	item *domain.Item,
	orderItem *domain.OrderItem,
) ([]domain.StockMovementEvent, error) {
	multiplier := 1.0
	if len(orderItem.SectorPayload) > 0 {
		var payload map[string]any
		if err := json.Unmarshal(orderItem.SectorPayload, &payload); err == nil {
			if m, ok := payload["uom_multiplier"].(float64); ok && m > 0 {
				multiplier = m
			}
		}
	}

	baseDeductionQty := orderItem.Quantity * multiplier

	return []domain.StockMovementEvent{
		{
			ID:          uid.NewV7(),
			TenantID:    execCtx.TenantID,
			OutletID:    execCtx.OutletID,
			SKU:         item.SKU,
			DeltaQty:    -baseDeductionQty,
			Reason:      "RETAIL_SALE",
			ReferenceID: execCtx.OrderID,
		},
	}, nil
}
