package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"pos-engine/internal/config"
	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/uid"
)

var (
	ErrInvalidSKU       = errors.New("sku cannot be empty")
	ErrInvalidItemName  = errors.New("item name cannot be empty")
	ErrInvalidPrice     = errors.New("item base price cannot be negative")
	ErrInvalidTaxRate   = errors.New("tax rate must be between 0.0 and 1.0")
	ErrInvalidItemSector = errors.New("invalid sector type (must be FB, RETAIL, TOURISM, or HOSPITALITY)")
)

type CatalogUsecase struct {
	cfg         *config.Config
	catalogRepo domain.CatalogRepository
	broadcaster EventBroadcaster
}

func NewCatalogUsecase(cfg *config.Config, catalogRepo domain.CatalogRepository) *CatalogUsecase {
	return &CatalogUsecase{
		cfg:         cfg,
		catalogRepo: catalogRepo,
	}
}

func (uc *CatalogUsecase) SetBroadcaster(b EventBroadcaster) {
	uc.broadcaster = b
}

func (uc *CatalogUsecase) ValidateItem(item *domain.Item) error {
	item.SKU = strings.TrimSpace(item.SKU)
	if item.SKU == "" {
		return ErrInvalidSKU
	}
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return ErrInvalidItemName
	}
	if item.BasePrice < 0 {
		return ErrInvalidPrice
	}
	if item.TaxRate < 0.0 || item.TaxRate > 1.0 {
		return ErrInvalidTaxRate
	}

	switch item.Sector {
	case domain.SectorFB, domain.SectorRetail, domain.SectorTourism, domain.SectorHospitality:
	default:
		return ErrInvalidItemSector
	}

	if item.Currency == "" {
		item.Currency = "IDR"
	}
	if len(item.SectorTraits) == 0 {
		item.SectorTraits = []byte("{}")
	}
	return nil
}

func (uc *CatalogUsecase) CreateItem(ctx context.Context, item *domain.Item) error {
	if err := uc.ValidateItem(item); err != nil {
		return err
	}

	if item.ID == "" {
		item.ID = uid.NewV7()
	}
	if item.TenantID == "" {
		item.TenantID = uc.cfg.TenantID
	}
	if item.OutletID == "" {
		item.OutletID = uc.cfg.OutletID
	}
	item.Version = time.Now().UTC().UnixNano()
	item.IsActive = true

	if err := uc.catalogRepo.UpsertItem(ctx, item); err != nil {
		return err
	}

	if uc.broadcaster != nil {
		uc.broadcaster.BroadcastEvent("catalog", "ITEM_CREATED", item)
	}
	return nil
}

func (uc *CatalogUsecase) UpdateItem(ctx context.Context, id string, updated *domain.Item) error {
	existing, err := uc.catalogRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if updated.SKU == "" {
		updated.SKU = existing.SKU
	}
	if updated.Sector == "" {
		updated.Sector = existing.Sector
	}
	if updated.Currency == "" {
		updated.Currency = existing.Currency
	}

	if err := uc.ValidateItem(updated); err != nil {
		return err
	}

	existing.Name = updated.Name
	existing.Barcode = updated.Barcode
	existing.BasePrice = updated.BasePrice
	existing.Currency = updated.Currency
	existing.TaxRate = updated.TaxRate
	existing.Sector = updated.Sector
	if len(updated.SectorTraits) > 0 {
		existing.SectorTraits = updated.SectorTraits
	}
	existing.IsActive = updated.IsActive
	nowNano := time.Now().UTC().UnixNano()
	if nowNano <= existing.Version {
		nowNano = existing.Version + 1
	}
	existing.Version = nowNano
	existing.UpdatedAt = time.Now().UTC()

	if err := uc.catalogRepo.UpsertItem(ctx, existing); err != nil {
		return err
	}

	if uc.broadcaster != nil {
		uc.broadcaster.BroadcastEvent("catalog", "ITEM_UPDATED", existing)
	}
	return nil
}

func (uc *CatalogUsecase) DeleteItem(ctx context.Context, tenantID, id string) error {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}

	if err := uc.catalogRepo.DeleteItem(ctx, tenantID, id); err != nil {
		return err
	}

	if uc.broadcaster != nil {
		uc.broadcaster.BroadcastEvent("catalog", "ITEM_DELETED", map[string]string{"id": id, "tenant_id": tenantID})
	}
	return nil
}

func (uc *CatalogUsecase) GetItemByID(ctx context.Context, id string) (*domain.Item, error) {
	return uc.catalogRepo.GetByID(ctx, id)
}

func (uc *CatalogUsecase) UpsertItem(ctx context.Context, item *domain.Item) error {
	if item.ID == "" {
		item.ID = uid.NewV7()
	}
	if item.TenantID == "" {
		item.TenantID = uc.cfg.TenantID
	}
	if item.OutletID == "" {
		item.OutletID = uc.cfg.OutletID
	}
	if item.Version <= 0 {
		item.Version = time.Now().UTC().UnixNano()
	}

	return uc.catalogRepo.UpsertItem(ctx, item)
}

func (uc *CatalogUsecase) ListItems(ctx context.Context, tenantID string, limit, offset int) ([]domain.Item, error) {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return uc.catalogRepo.ListItems(ctx, tenantID, limit, offset)
}

func (uc *CatalogUsecase) GetItemBySKU(ctx context.Context, tenantID, sku string) (*domain.Item, error) {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	return uc.catalogRepo.GetBySKU(ctx, tenantID, sku)
}

func (uc *CatalogUsecase) GetItemByBarcode(ctx context.Context, tenantID, barcode string) (*domain.Item, error) {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	return uc.catalogRepo.GetByBarcode(ctx, tenantID, barcode)
}

func (uc *CatalogUsecase) GetStock(ctx context.Context, tenantID, outletID, sku string) (*domain.InventoryBalance, error) {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	if outletID == "" {
		outletID = uc.cfg.OutletID
	}
	return uc.catalogRepo.GetStock(ctx, tenantID, outletID, sku)
}

func (uc *CatalogUsecase) AdjustStock(ctx context.Context, tenantID, outletID, sku string, delta float64, reason, refID string) (*domain.InventoryBalance, error) {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	if outletID == "" {
		outletID = uc.cfg.OutletID
	}
	if sku == "" {
		return nil, ErrInvalidSKU
	}
	if delta == 0 {
		return nil, fmt.Errorf("adjustment delta cannot be zero")
	}
	if reason == "" {
		reason = "MANUAL_ADJUSTMENT"
	}

	event := &domain.StockMovementEvent{
		ID:          uid.NewV7(),
		TenantID:    tenantID,
		OutletID:    outletID,
		SKU:         sku,
		DeltaQty:    delta,
		Reason:      reason,
		ReferenceID: refID,
		CreatedAt:   time.Now().UTC(),
	}

	if err := uc.catalogRepo.RecordStockMovement(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to record stock movement: %w", err)
	}

	updatedBal, err := uc.catalogRepo.GetStock(ctx, tenantID, outletID, sku)
	if err != nil {
		return nil, err
	}

	if uc.broadcaster != nil {
		uc.broadcaster.BroadcastEvent("inventory", "STOCK_ADJUSTED", updatedBal)
	}
	return updatedBal, nil
}

func (uc *CatalogUsecase) ListUpdatedSince(ctx context.Context, tenantID string, sinceVersion int64, limit int) ([]domain.Item, error) {
	if tenantID == "" {
		tenantID = uc.cfg.TenantID
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return uc.catalogRepo.ListUpdatedSince(ctx, tenantID, sinceVersion, limit)
}
