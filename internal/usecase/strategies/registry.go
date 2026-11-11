package strategies

import (
	"fmt"
	"pos-engine/internal/domain"
)

type Registry struct {
	strategies map[domain.ItemSector]FulfillmentStrategy
}

func NewRegistry() *Registry {
	r := &Registry{
		strategies: make(map[domain.ItemSector]FulfillmentStrategy),
	}
	r.Register(NewFBStrategy())
	r.Register(NewRetailStrategy())
	r.Register(NewTourismStrategy())
	r.Register(NewHospitalityStrategy())
	return r
}

func (r *Registry) Register(strategy FulfillmentStrategy) {
	r.strategies[strategy.Sector()] = strategy
}

func (r *Registry) Get(sector domain.ItemSector) (FulfillmentStrategy, error) {
	st, exists := r.strategies[sector]
	if !exists {
		return nil, fmt.Errorf("no fulfillment strategy registered for sector: %s", sector)
	}
	return st, nil
}
