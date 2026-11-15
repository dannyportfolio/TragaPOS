package usecase_test

import (
	"context"
	"encoding/json"
	"testing"

	"pos-engine/internal/domain"
	"pos-engine/internal/usecase/strategies"
)

func TestFBStrategy_RecipeBOM(t *testing.T) {
	st := strategies.NewFBStrategy()
	ctx := context.Background()

	trait := domain.FBTrait{
		KitchenStation: "BAR",
		Recipe: []domain.RecipeIngredient{
			{RawMaterialSKU: "RAW_BEANS_G", Quantity: 18, UOM: "g"},
			{RawMaterialSKU: "RAW_MILK_ML", Quantity: 200, UOM: "ml"},
		},
		ModifierGroups: []domain.ModifierGroup{
			{
				GroupID:   "EXTRAS",
				GroupName: "Extras",
				Modifiers: []domain.Modifier{
					{ID: "EXTRA_SHOT", Name: "Extra Shot", AddPrice: 5000},
				},
			},
		},
	}
	traitBytes, _ := json.Marshal(trait)

	item := &domain.Item{
		SKU:          "LATTE_HOT",
		Name:         "Hot Caffe Latte",
		Sector:       domain.SectorFB,
		BasePrice:    30000,
		TaxRate:      0.11,
		SectorTraits: traitBytes,
	}

	selectedModBytes, _ := json.Marshal([]domain.Modifier{
		{ID: "EXTRA_SHOT", Name: "Extra Shot", AddPrice: 5000},
	})

	orderItem := domain.OrderItem{
		Quantity:          2,
		SelectedModifiers: selectedModBytes,
	}

	err := st.ValidateAndPriceItem(ctx, item, &orderItem)
	if err != nil {
		t.Fatalf("ValidateAndPriceItem failed: %v", err)
	}

	if orderItem.UnitPrice != 35000 {
		t.Errorf("expected unit price 35000, got %d", orderItem.UnitPrice)
	}
	if orderItem.Subtotal != 70000 {
		t.Errorf("expected subtotal 70000, got %d", orderItem.Subtotal)
	}

	execCtx := strategies.ExecutionContext{
		TenantID: "tenant-01",
		OutletID: "outlet-cafe-01",
		OrderID:  "ord-001",
	}

	movements, err := st.ProcessFulfillment(ctx, execCtx, item, &orderItem)
	if err != nil {
		t.Fatalf("ProcessFulfillment failed: %v", err)
	}

	if len(movements) != 2 {
		t.Fatalf("expected 2 raw ingredient deductions, got %d", len(movements))
	}

	if movements[0].SKU != "RAW_BEANS_G" || movements[0].DeltaQty != -36 {
		t.Errorf("unexpected beans deduction: %+v", movements[0])
	}
	if movements[1].SKU != "RAW_MILK_ML" || movements[1].DeltaQty != -400 {
		t.Errorf("unexpected milk deduction: %+v", movements[1])
	}
}

func TestRetailStrategy_MultiUOMAndWholesaleTier(t *testing.T) {
	st := strategies.NewRetailStrategy()
	ctx := context.Background()

	trait := domain.RetailTrait{
		GTIN:    "8992345678901",
		BaseUOM: "PCS",
		UOMConversions: []domain.UOMConversion{
			{UOM: "BOX", Multiplier: 24, Price: 72000},
		},
		TieredPricings: []domain.TieredPricing{
			{MinQuantity: 10, UnitPrice: 68000},
		},
	}
	traitBytes, _ := json.Marshal(trait)

	item := &domain.Item{
		SKU:          "MIE_GORENG",
		Name:         "Indomie Goreng",
		Sector:       domain.SectorRetail,
		BasePrice:    3500,
		TaxRate:      0.0,
		SectorTraits: traitBytes,
	}

	orderItem := domain.OrderItem{
		Quantity: 10,
		UOM:      "BOX",
	}

	err := st.ValidateAndPriceItem(ctx, item, &orderItem)
	if err != nil {
		t.Fatalf("ValidateAndPriceItem failed: %v", err)
	}

	if orderItem.UnitPrice != 68000 {
		t.Errorf("expected unit price 68000, got %d", orderItem.UnitPrice)
	}
	if orderItem.Subtotal != 680000 {
		t.Errorf("expected subtotal 680000, got %d", orderItem.Subtotal)
	}

	execCtx := strategies.ExecutionContext{
		TenantID: "tenant-01",
		OutletID: "outlet-swalayan",
		OrderID:  "ord-002",
	}
	movements, err := st.ProcessFulfillment(ctx, execCtx, item, &orderItem)
	if err != nil {
		t.Fatalf("ProcessFulfillment failed: %v", err)
	}

	if len(movements) != 1 || movements[0].DeltaQty != -240 {
		t.Errorf("expected deduction of -240 pcs, got %+v", movements)
	}
}
