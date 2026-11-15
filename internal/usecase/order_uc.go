package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"pos-engine/internal/config"
	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/uid"
)

type OrderUsecase struct {
	cfg         *config.Config
	orderRepo   domain.OrderRepository
	broadcaster EventBroadcaster
}

func NewOrderUsecase(cfg *config.Config, orderRepo domain.OrderRepository) *OrderUsecase {
	return &OrderUsecase{
		cfg:       cfg,
		orderRepo: orderRepo,
	}
}

func (uc *OrderUsecase) SetBroadcaster(b EventBroadcaster) {
	uc.broadcaster = b
}

func (uc *OrderUsecase) UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) (*domain.Order, error) {
	if err := uc.orderRepo.UpdateOrderStatus(ctx, orderID, status); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	order, err := uc.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if uc.broadcaster != nil {
		uc.broadcaster.BroadcastEvent("kds", "ORDER_STATUS_CHANGED", order)
		uc.broadcaster.BroadcastEvent("orders", "ORDER_STATUS_CHANGED", order)
	}

	return order, nil
}

func (uc *OrderUsecase) ListOrders(ctx context.Context, tenantID, outletID, status string, limit, offset int) ([]domain.Order, error) {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	if outletID == "" {
		outletID = uc.cfg.OutletID
	}
	return uc.orderRepo.ListOrdersWithFilter(ctx, domain.OrderFilter{
		TenantID: tenantID,
		OutletID: outletID,
		Status:   status,
		Limit:    limit,
		Offset:   offset,
	})
}

func (uc *OrderUsecase) VoidOrder(ctx context.Context, tenantID, orderID string, reason string) error {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	if reason == "" {
		reason = "CUSTOMER_REQUEST"
	}
	if err := uc.orderRepo.VoidOrder(ctx, tenantID, orderID, reason); err != nil {
		return err
	}

	if uc.broadcaster != nil {
		uc.broadcaster.BroadcastEvent("orders", "ORDER_VOIDED", map[string]string{
			"order_id":  orderID,
			"tenant_id": tenantID,
			"reason":    reason,
		})
	}
	return nil
}

func (uc *OrderUsecase) ListKdsOrders(ctx context.Context, tenantID, outletID string) ([]domain.Order, error) {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	if outletID == "" {
		outletID = uc.cfg.OutletID
	}
	return uc.orderRepo.ListKdsOrders(ctx, tenantID, outletID)
}

func (uc *OrderUsecase) GetOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	return uc.orderRepo.GetOrderByID(ctx, id)
}

func (uc *OrderUsecase) ListHeldOrders(ctx context.Context, tenantID, outletID string) ([]domain.Order, error) {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	if outletID == "" {
		outletID = uc.cfg.OutletID
	}
	return uc.orderRepo.ListHeldOrders(ctx, tenantID, outletID)
}

func (uc *OrderUsecase) CreateFolio(ctx context.Context, folio *domain.GuestFolio) error {
	if folio.ID == "" {
		folio.ID = uid.NewV7()
	}
	if folio.TenantID == "" {
		folio.TenantID = uc.cfg.TenantID
	}
	if folio.OutletID == "" {
		folio.OutletID = uc.cfg.OutletID
	}
	if folio.CheckInAt.IsZero() {
		folio.CheckInAt = time.Now().UTC()
	}
	if folio.CheckOutAt.IsZero() {
		folio.CheckOutAt = time.Now().UTC().Add(24 * time.Hour)
	}
	folio.IsOpen = true

	return uc.orderRepo.CreateFolio(ctx, folio)
}

func (uc *OrderUsecase) GetFolioByRoom(ctx context.Context, tenantID, roomNumber string) (*domain.GuestFolio, error) {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	return uc.orderRepo.GetFolioByRoom(ctx, tenantID, roomNumber)
}

func (uc *OrderUsecase) ListFolios(ctx context.Context, tenantID string, openOnly bool, limit, offset int) ([]domain.GuestFolio, error) {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	return uc.orderRepo.ListFolios(ctx, tenantID, openOnly, limit, offset)
}

func (uc *OrderUsecase) CloseFolio(ctx context.Context, tenantID, folioID string) error {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	return uc.orderRepo.CloseFolio(ctx, tenantID, folioID)
}

func (uc *OrderUsecase) GenerateReceipt(ctx context.Context, orderID string) (string, error) {
	order, err := uc.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	writeLine := func(s string) {
		sb.WriteString(s)
		sb.WriteByte('\n')
	}
	width := 42
	divider := strings.Repeat("-", width)
	doubleDiv := strings.Repeat("=", width)

	writeLine(doubleDiv)
	writeLine(centerText("COMMERCE & POS ENGINE", width))
	writeLine(centerText("OFFLINE-FIRST OUTLET HUB", width))
	writeLine(doubleDiv)
	fmt.Fprintf(&sb, "Order No : %s\n", order.OrderNumber)
	fmt.Fprintf(&sb, "Outlet   : %s\n", order.OutletID)
	fmt.Fprintf(&sb, "Date     : %s\n", order.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(&sb, "Status   : %s\n", order.Status)
	writeLine(divider)
	fmt.Fprintf(&sb, "%-22s %5s %12s\n", "ITEM", "QTY", "PRICE")
	writeLine(divider)

	for _, item := range order.Items {
		name := item.Name
		if len(name) > 22 {
			name = name[:19] + "..."
		}
		fmt.Fprintf(&sb, "%-22s %5.0f %12s\n", name, item.Quantity, formatCurrency(item.TotalAmount))
		if len(item.SelectedModifiers) > 2 && string(item.SelectedModifiers) != "[]" {
			fmt.Fprintf(&sb, "  + Modifiers: %s\n", string(item.SelectedModifiers))
		}
	}

	writeLine(divider)
	fmt.Fprintf(&sb, "%-28s %12s\n", "Subtotal:", formatCurrency(order.Subtotal))
	if order.TaxAmount > 0 {
		fmt.Fprintf(&sb, "%-28s %12s\n", "Tax:", formatCurrency(order.TaxAmount))
	}
	if order.DiscountAmount > 0 {
		fmt.Fprintf(&sb, "%-28s %12s\n", "Discount:", formatCurrency(order.DiscountAmount))
	}
	fmt.Fprintf(&sb, "%-28s %12s\n", "TOTAL AMOUNT:", formatCurrency(order.TotalAmount))
	writeLine(divider)

	if len(order.Payments) > 0 {
		writeLine("PAYMENTS:")
		var totalPaid int64
		for _, p := range order.Payments {
			fmt.Fprintf(&sb, "  %-24s %14s\n", p.Method, formatCurrency(p.Amount))
			totalPaid += p.Amount
		}
		if totalPaid > order.TotalAmount {
			fmt.Fprintf(&sb, "%-28s %12s\n", "Change:", formatCurrency(totalPaid-order.TotalAmount))
		}
	}

	writeLine(doubleDiv)
	writeLine(centerText("Thank you for your visit!", width))
	writeLine(centerText("Please retain for your records", width))
	writeLine(doubleDiv)

	return sb.String(), nil
}

func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	pad := (width - len(text)) / 2
	return strings.Repeat(" ", pad) + text
}

func formatCurrency(amount int64) string {
	// Simple comma formatting for thousands
	s := fmt.Sprintf("%d", amount)
	if len(s) <= 3 {
		return s
	}
	var res []string
	for len(s) > 3 {
		res = append([]string{s[len(s)-3:]}, res...)
		s = s[:len(s)-3]
	}
	if len(s) > 0 {
		res = append([]string{s}, res...)
	}
	return strings.Join(res, ",")
}
