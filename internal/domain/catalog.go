package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type ItemSector string

const (
	SectorFB          ItemSector = "FB"
	SectorRetail      ItemSector = "RETAIL"
	SectorTourism     ItemSector = "TOURISM"
	SectorHospitality ItemSector = "HOSPITALITY"
)

var (
	ErrItemNotFound       = errors.New("catalog item not found")
	ErrInactiveItem       = errors.New("catalog item is inactive")
	ErrInvalidSectorTrait = errors.New("invalid sector trait data")
	ErrInvalidQuantity    = errors.New("invalid item quantity")
	ErrInventoryShortage  = errors.New("insufficient inventory")
)

type Item struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	OutletID     string          `json:"outlet_id"`
	SKU          string          `json:"sku"`
	Barcode      string          `json:"barcode,omitempty"`
	Name         string          `json:"name"`
	Sector       ItemSector      `json:"sector"`
	BasePrice    int64           `json:"base_price"`
	Currency     string          `json:"currency"`
	TaxRate      float64         `json:"tax_rate"`
	SectorTraits json.RawMessage `json:"sector_traits"`
	Version      int64           `json:"version"`
	IsActive     bool            `json:"is_active"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type FBTrait struct {
	KitchenStation string             `json:"kitchen_station"`
	Recipe         []RecipeIngredient `json:"recipe,omitempty"`
	ModifierGroups []ModifierGroup    `json:"modifier_groups,omitempty"`
}

type RecipeIngredient struct {
	RawMaterialSKU string  `json:"raw_material_sku"`
	Quantity       float64 `json:"quantity"`
	UOM            string  `json:"uom"`
}

type ModifierGroup struct {
	GroupID   string     `json:"group_id"`
	GroupName string     `json:"group_name"`
	MinSelect int        `json:"min_select"`
	MaxSelect int        `json:"max_select"`
	Modifiers []Modifier `json:"modifiers"`
}

type Modifier struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	AddPrice int64  `json:"add_price"`
}

type RetailTrait struct {
	GTIN           string          `json:"gtin"`
	BaseUOM        string          `json:"base_uom"`
	UOMConversions []UOMConversion `json:"uom_conversions,omitempty"`
	TieredPricings []TieredPricing `json:"tiered_pricings,omitempty"`
	RequiresBatch  bool            `json:"requires_batch"`
	TrackExpiry    bool            `json:"track_expiry"`
}

type UOMConversion struct {
	UOM        string  `json:"uom"`
	Multiplier float64 `json:"multiplier"`
	Price      int64   `json:"price"`
}

type TieredPricing struct {
	MinQuantity int   `json:"min_qty"`
	UnitPrice   int64 `json:"unit_price"`
}

type TourismTrait struct {
	PassCategory       string   `json:"pass_category"`
	ValidDurationHours int      `json:"valid_duration_hours"`
	PaxPerTicket       int      `json:"pax_per_ticket"`
	AllowedGateZones   []string `json:"allowed_gate_zones"`
	DailyQuota         int      `json:"daily_quota,omitempty"`
}

type HospitalityTrait struct {
	RoomFacilityID    string `json:"room_facility_id"`
	RoomType          string `json:"room_type"`
	HousekeepingState string `json:"housekeeping_state"`
	MaxOccupancy      int    `json:"max_occupancy"`
	RequiresCheckIn   bool   `json:"requires_check_in"`
}

type InventoryBalance struct {
	TenantID     string    `json:"tenant_id"`
	OutletID     string    `json:"outlet_id"`
	SKU          string    `json:"sku"`
	OnHandQty    float64   `json:"on_hand_qty"`
	AllocatedQty float64   `json:"allocated_qty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type StockMovementEvent struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	OutletID    string    `json:"outlet_id"`
	SKU         string    `json:"sku"`
	DeltaQty    float64   `json:"delta_qty"`
	Reason      string    `json:"reason"`
	ReferenceID string    `json:"reference_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type CatalogRepository interface {
	GetByID(ctx context.Context, id string) (*Item, error)
	GetBySKU(ctx context.Context, tenantID, sku string) (*Item, error)
	GetByBarcode(ctx context.Context, tenantID, barcode string) (*Item, error)
	ListItems(ctx context.Context, tenantID string, limit, offset int) ([]Item, error)
	ListUpdatedSince(ctx context.Context, tenantID string, sinceVersion int64, limit int) ([]Item, error)
	UpsertItem(ctx context.Context, item *Item) error
	DeleteItem(ctx context.Context, tenantID, id string) error
	GetStock(ctx context.Context, tenantID, outletID, sku string) (*InventoryBalance, error)
	RecordStockMovement(ctx context.Context, event *StockMovementEvent) error
}
