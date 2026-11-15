package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusPaid      OrderStatus = "PAID"
	OrderStatusHeld      OrderStatus = "HELD"
	OrderStatusPreparing OrderStatus = "PREPARING"
	OrderStatusReady     OrderStatus = "READY"
	OrderStatusCompleted OrderStatus = "COMPLETED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusRefunded  OrderStatus = "REFUNDED"
)

type PaymentMethod string

const (
	PaymentMethodCash       PaymentMethod = "CASH"
	PaymentMethodCard       PaymentMethod = "CARD"
	PaymentMethodQRIS       PaymentMethod = "QRIS"
	PaymentMethodRoomCharge PaymentMethod = "ROOM_CHARGE"
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrOrderAlreadyPaid   = errors.New("order is already paid")
	ErrInvalidOrderTotal  = errors.New("invalid order total calculation")
	ErrFolioNotFound      = errors.New("room folio not found or closed")
	ErrFolioLimitExceeded = errors.New("room folio credit limit exceeded")
)

type Order struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	OutletID       string          `json:"outlet_id"`
	OrderNumber    string          `json:"order_number"`
	Status         OrderStatus     `json:"status"`
	Subtotal       int64           `json:"subtotal"`
	TaxAmount      int64           `json:"tax_amount"`
	DiscountAmount int64           `json:"discount_amount"`
	TotalAmount    int64           `json:"total_amount"`
	Currency       string          `json:"currency"`
	Notes          string          `json:"notes,omitempty"`
	Items          []OrderItem     `json:"items"`
	Payments       []Payment       `json:"payments,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type OrderItem struct {
	ID                string          `json:"id"`
	OrderID           string          `json:"order_id"`
	ItemID            string          `json:"item_id"`
	SKU               string          `json:"sku"`
	Name              string          `json:"name"`
	Sector            ItemSector      `json:"sector"`
	Quantity          float64         `json:"quantity"`
	UOM               string          `json:"uom"`
	UnitPrice         int64           `json:"unit_price"`
	Subtotal          int64           `json:"subtotal"`
	TaxAmount         int64           `json:"tax_amount"`
	TotalAmount       int64           `json:"total_amount"`
	SelectedModifiers json.RawMessage `json:"selected_modifiers,omitempty"`
	SectorPayload     json.RawMessage `json:"sector_payload,omitempty"`
}

type Payment struct {
	ID        string        `json:"id"`
	OrderID   string        `json:"order_id"`
	Method    PaymentMethod `json:"method"`
	Amount    int64         `json:"amount"`
	Reference string        `json:"reference,omitempty"`
	PaidAt    time.Time     `json:"paid_at"`
}

type GuestFolio struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	OutletID     string    `json:"outlet_id"`
	RoomNumber   string    `json:"room_number"`
	GuestName    string    `json:"guest_name"`
	CreditLimit  int64     `json:"credit_limit"`
	CurrentTotal int64     `json:"current_total"`
	IsOpen       bool      `json:"is_open"`
	CheckInAt    time.Time `json:"check_in_at"`
	CheckOutAt   time.Time `json:"check_out_at"`
}

type FolioCharge struct {
	ID        string    `json:"id"`
	FolioID   string    `json:"folio_id"`
	OrderID   string    `json:"order_id"`
	Amount    int64     `json:"amount"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

type OrderFilter struct {
	TenantID string
	OutletID string
	Status   string
	Limit    int
	Offset   int
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *Order) error
	GetOrderByID(ctx context.Context, id string) (*Order, error)
	ListOrders(ctx context.Context, tenantID, outletID string, limit, offset int) ([]Order, error)
	ListOrdersWithFilter(ctx context.Context, filter OrderFilter) ([]Order, error)
	ListHeldOrders(ctx context.Context, tenantID, outletID string) ([]Order, error)
	ListKdsOrders(ctx context.Context, tenantID, outletID string) ([]Order, error)
	AddPayment(ctx context.Context, payment *Payment) error
	UpdateOrderStatus(ctx context.Context, orderID string, status OrderStatus) error
	VoidOrder(ctx context.Context, tenantID, orderID string, reason string) error
	GetFolioByRoom(ctx context.Context, tenantID, roomNumber string) (*GuestFolio, error)
	GetFolioByID(ctx context.Context, id string) (*GuestFolio, error)
	ListFolios(ctx context.Context, tenantID string, openOnly bool, limit, offset int) ([]GuestFolio, error)
	CreateFolio(ctx context.Context, folio *GuestFolio) error
	CloseFolio(ctx context.Context, tenantID, folioID string) error
	ChargeFolio(ctx context.Context, charge *FolioCharge) error
}
