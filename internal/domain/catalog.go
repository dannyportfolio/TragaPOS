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

// Item represents a polymorphic sellable item across all industries.
type Item struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	OutletID     string          `json:"outlet_id"`
	SKU          string          `json:"sku"`
	Barcode      string          `json:"barcode,omitempty"`
	Name         string          `json:"name"`
	Sector       ItemSector      `json:"sector"`
	BasePrice    int64           `json:"base_price"` // Lowest currency unit (e.g. cents, IDR)
	Currency     string          `json:"currency"`
	TaxRate      float64         `json:"tax_rate"` // e.g. 0.11 for 11% VAT
	SectorTraits json.RawMessage `json:"sector_traits"`
	Version      int64           `json:"version"` // Incremental version for Delta Sync
	IsActive     bool            `json:"is_active"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// F&B Traits: BOM ingredients, kitchen routing, modifiers
type FBTrait struct {
	KitchenStation string             `json:"kitchen_station"` // e.g., "KITCHEN", "BAR", "GRILL"
	Recipe         []RecipeIngredient `json:"recipe,omitempty"`
	ModifierGroups []ModifierGroup    `json:"modifier_groups,omitempty"`
}

type RecipeIngredient struct {
	RawMaterialSKU string  `json:"raw_material_sku"`
	Quantity       float64 `json:"quantity"`
	UOM            string  `json:"uom"` // e.g., "g", "ml", "pcs"
}

type ModifierGroup struct {
	GroupID     string     `json:"group_id"`
	GroupName   string     `json:"group_name"`
	MinSelect   int        `json:"min_select"`
	MaxSelect   int        `json:"max_select"`
	Modifiers   []Modifier `json:"modifiers"`
}

type Modifier struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	AddPrice int64  `json:"add_price"`
}

// Retail Traits: High-velocity barcode, Multi-UOM, Expiry/Batch, Wholesale Tiers
type RetailTrait struct {
	GTIN             string           `json:"gtin"`
	BaseUOM          string           `json:"base_uom"` // e.g., "PCS"
	UOMConversions   []UOMConversion  `json:"uom_conversions,omitempty"`
	TieredPricings   []TieredPricing  `json:"tiered_pricings,omitempty"`
	RequiresBatch    bool             `json:"requires_batch"`
	TrackExpiry      bool             `json:"track_expiry"`
}

type UOMConversion struct {
	UOM        string  `json:"uom"`        // e.g., "BOX", "PACK"
	Multiplier float64 `json:"multiplier"` // e.g., 24 (1 BOX = 24 PCS)
	Price      int64   `json:"price"`
}

type TieredPricing struct {
	MinQuantity int   `json:"min_qty"`
	UnitPrice   int64 `json:"unit_price"`
}

// Tourism Traits: Quota/date-bound pass, gate redemptions
type TourismTrait struct {
	PassCategory       string `json:"pass_category"`        // "REGULAR", "VIP", "WEEKEND_FAST_TRACK"
	ValidDurationHours int    `json:"valid_duration_hours"` // e.g., 24
	PaxPerTicket       int    `json:"pax_per_ticket"`       // e.g., 1 or 4 (Family)
	AllowedGateZones   []string `json:"allowed_gate_zones"` // e.g., ["MAIN_GATE", "WATERPARK"]
	DailyQuota         int    `json:"daily_quota,omitempty"`
}

// Hospitality Traits: Duration-bound facility, housekeeping status, room folio integration
type HospitalityTrait struct {
	RoomFacilityID   string `json:"room_facility_id"`
	RoomType         string `json:"room_type"` // e.g., "DELUXE_VILLA", "STANDARD"
	HousekeepingState string `json:"housekeeping_state"` // "CLEAN", "DIRTY", "INSPECTING"
	MaxOccupancy     int    `json:"max_occupancy"`
	RequiresCheckIn  bool   `json:"requires_check_in"`
}

// InventoryBalance models local or central stock per SKU.
type InventoryBalance struct {
	TenantID     string    `json:"tenant_id"`
	OutletID     string    `json:"outlet_id"`
	SKU          string    `json:"sku"`
	OnHandQty    float64   `json:"on_hand_qty"`
	AllocatedQty float64   `json:"allocated_qty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// StockMovementEvent represents an append-only delta event.
type StockMovementEvent struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	OutletID       string    `json:"outlet_id"`
	SKU            string    `json:"sku"`
	DeltaQty       float64   `json:"delta_qty"` // Can be negative (deduction) or positive (restock)
	Reason         string    `json:"reason"`    // "SALE", "BOM_DEDUCTION", "DAMAGE", "RESTOCK"
	ReferenceID    string    `json:"reference_id"` // OrderID, TicketID, etc.
	CreatedAt      time.Time `json:"created_at"`
}

// CatalogRepository abstracts persistence for catalog items and stock.
type CatalogRepository interface {
	GetByID(ctx context.Context, id string) (*Item, error)
	GetBySKU(ctx context.Context, tenantID, sku string) (*Item, error)
	GetByBarcode(ctx context.Context, tenantID, barcode string) (*Item, error)
	ListItems(ctx context.Context, tenantID string, limit, offset int) ([]Item, error)
	ListUpdatedSince(ctx context.Context, tenantID string, sinceVersion int64, limit int) ([]Item, error)
	UpsertItem(ctx context.Context, item *Item) error

	// Stock and Inventory
	GetStock(ctx context.Context, tenantID, outletID, sku string) (*InventoryBalance, error)
	RecordStockMovement(ctx context.Context, event *StockMovementEvent) error
}
