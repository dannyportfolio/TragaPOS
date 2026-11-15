package strategies_test

import (
	"context"
	"encoding/json"
	"testing"

	"pos-engine/internal/domain"
	"pos-engine/internal/usecase/strategies"
)

func TestFBStrategy_EnforcesCatalogModifierPricing(t *testing.T) {
	st := strategies.NewFBStrategy()
	ctx := context.Background()

	trait := domain.FBTrait{
		KitchenStation: "BAR",
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
		Name:         "Hot Latte",
		Sector:       domain.SectorFB,
		BasePrice:    30000,
		TaxRate:      0.10,
		SectorTraits: traitBytes,
	}

	// Client attempts to tamper AddPrice down to 100
	tamperedMods, _ := json.Marshal([]domain.Modifier{
		{ID: "EXTRA_SHOT", Name: "Extra Shot", AddPrice: 100},
	})
	oi := domain.OrderItem{Quantity: 1, SelectedModifiers: tamperedMods}
	if err := st.ValidateAndPriceItem(ctx, item, &oi); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if oi.UnitPrice != 35000 {
		t.Fatalf("expected server-enforced unit price 35000, got %d", oi.UnitPrice)
	}

	// Client attempts negative modifier price
	negativeMods, _ := json.Marshal([]domain.Modifier{
		{ID: "EXTRA_SHOT", Name: "Extra Shot", AddPrice: -10000},
	})
	oiNeg := domain.OrderItem{Quantity: 1, SelectedModifiers: negativeMods}
	if err := st.ValidateAndPriceItem(ctx, item, &oiNeg); err == nil {
		t.Fatal("expected error for negative modifier price, got nil")
	}

	// Client attempts unknown modifier
	unknownMods, _ := json.Marshal([]domain.Modifier{
		{ID: "FAKE_MOD", Name: "Fake Mod", AddPrice: 0},
	})
	oiUnk := domain.OrderItem{Quantity: 1, SelectedModifiers: unknownMods}
	if err := st.ValidateAndPriceItem(ctx, item, &oiUnk); err == nil {
		t.Fatal("expected error for unknown modifier, got nil")
	}
}
