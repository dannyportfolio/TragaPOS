package strategies

import (
	"context"
	"pos-engine/internal/domain"
)

type ExecutionContext struct {
	TenantID string
	OutletID string
	OrderID  string
}

type FulfillmentStrategy interface {
	Sector() domain.ItemSector
	ValidateAndPriceItem(ctx context.Context, item *domain.Item, orderItem *domain.OrderItem) error
	ProcessFulfillment(ctx context.Context, execCtx ExecutionContext, item *domain.Item, orderItem *domain.OrderItem) ([]domain.StockMovementEvent, error)
}
