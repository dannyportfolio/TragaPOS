package usecase

import (
	"context"
	"fmt"
	"time"

	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/uid"
)

type ShiftUsecase struct {
	shiftRepo domain.ShiftRepository
	txMgr     domain.TransactionManager
}

func NewShiftUsecase(shiftRepo domain.ShiftRepository, txMgr domain.TransactionManager) *ShiftUsecase {
	return &ShiftUsecase{
		shiftRepo: shiftRepo,
		txMgr:     txMgr,
	}
}

type OpenShiftRequest struct {
	TenantID     string `json:"tenant_id"`
	OutletID     string `json:"outlet_id"`
	CashierID    string `json:"cashier_id"`
	CashierName  string `json:"cashier_name"`
	OpeningFloat int64  `json:"opening_float"`
	Notes        string `json:"notes,omitempty"`
}

type CashMovementRequest struct {
	TenantID  string                  `json:"tenant_id"`
	OutletID  string                  `json:"outlet_id"`
	CashierID string                  `json:"cashier_id"`
	Type      domain.CashMovementType `json:"type"`
	Amount    int64                   `json:"amount"`
	Reason    string                  `json:"reason"`
}

type CloseShiftRequest struct {
	TenantID   string `json:"tenant_id"`
	OutletID   string `json:"outlet_id"`
	CashierID  string `json:"cashier_id"`
	ActualCash int64  `json:"actual_cash"`
	Notes      string `json:"notes,omitempty"`
}

func (uc *ShiftUsecase) OpenShift(ctx context.Context, req OpenShiftRequest) (*domain.ShiftSession, error) {
	if req.CashierID == "" {
		return nil, fmt.Errorf("cashier_id is required")
	}
	if req.OpeningFloat < 0 {
		return nil, fmt.Errorf("opening_float cannot be negative")
	}

	existing, err := uc.shiftRepo.GetActiveShift(ctx, req.TenantID, req.OutletID, req.CashierID)
	if err == nil && existing != nil {
		return nil, domain.ErrActiveShiftExists
	}

	now := time.Now().UTC()
	shift := &domain.ShiftSession{
		ID:           uid.NewV7(),
		TenantID:     req.TenantID,
		OutletID:     req.OutletID,
		CashierID:    req.CashierID,
		CashierName:  req.CashierName,
		Status:       domain.ShiftStatusOpen,
		OpeningFloat: req.OpeningFloat,
		ExpectedCash: req.OpeningFloat,
		ActualCash:   0,
		Variance:     0,
		Notes:        req.Notes,
		OpenedAt:     now,
	}

	if err := uc.shiftRepo.CreateShift(ctx, shift); err != nil {
		return nil, fmt.Errorf("failed to open shift: %w", err)
	}

	return shift, nil
}

func (uc *ShiftUsecase) GetCurrentShift(ctx context.Context, tenantID, outletID, cashierID string) (*domain.ShiftSummary, error) {
	active, err := uc.shiftRepo.GetActiveShift(ctx, tenantID, outletID, cashierID)
	if err != nil {
		return nil, err
	}

	movements, err := uc.shiftRepo.ListCashMovements(ctx, active.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cash movements: %w", err)
	}

	var cashInTotal, cashOutTotal int64
	for _, m := range movements {
		switch m.Type {
		case domain.CashMovementIn, domain.CashMovementFloatAdjust:
			cashInTotal += m.Amount
		case domain.CashMovementOut:
			cashOutTotal += m.Amount
		}
	}

	now := time.Now().UTC()
	cashSales, nonCashSales, err := uc.shiftRepo.GetShiftSalesBreakdown(
		ctx, tenantID, outletID, active.OpenedAt, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate shift sales: %w", err)
	}

	expectedCash := active.OpeningFloat + cashInTotal - cashOutTotal + cashSales

	return &domain.ShiftSummary{
		ShiftID:           active.ID,
		TenantID:          active.TenantID,
		OutletID:          active.OutletID,
		CashierID:         active.CashierID,
		CashierName:       active.CashierName,
		Status:            active.Status,
		OpeningFloat:      active.OpeningFloat,
		CashInTotal:       cashInTotal,
		CashOutTotal:      cashOutTotal,
		CashSalesTotal:    cashSales,
		NonCashSalesTotal: nonCashSales,
		TotalSales:        cashSales + nonCashSales,
		ExpectedCash:      expectedCash,
		ActualCash:        active.ActualCash,
		Variance:          active.Variance,
		Notes:             active.Notes,
		OpenedAt:          active.OpenedAt,
		ClosedAt:          active.ClosedAt,
		Movements:         movements,
	}, nil
}

func (uc *ShiftUsecase) RecordCashMovement(ctx context.Context, req CashMovementRequest) (*domain.CashMovement, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	active, err := uc.shiftRepo.GetActiveShift(ctx, req.TenantID, req.OutletID, req.CashierID)
	if err != nil {
		return nil, err
	}

	movement := &domain.CashMovement{
		ID:        uid.NewV7(),
		ShiftID:   active.ID,
		Type:      req.Type,
		Amount:    req.Amount,
		Reason:    req.Reason,
		CreatedAt: time.Now().UTC(),
	}

	if err := uc.shiftRepo.CreateCashMovement(ctx, movement); err != nil {
		return nil, fmt.Errorf("failed to record cash movement: %w", err)
	}

	return movement, nil
}

func (uc *ShiftUsecase) CloseShift(ctx context.Context, req CloseShiftRequest) (*domain.ShiftSummary, error) {
	summary, err := uc.GetCurrentShift(ctx, req.TenantID, req.OutletID, req.CashierID)
	if err != nil {
		return nil, err
	}

	variance := req.ActualCash - summary.ExpectedCash
	closedAt := time.Now().UTC()

	err = uc.shiftRepo.CloseShift(
		ctx, summary.ShiftID, req.ActualCash, summary.ExpectedCash, variance, req.Notes, closedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to close shift: %w", err)
	}

	summary.Status = domain.ShiftStatusClosed
	summary.ActualCash = req.ActualCash
	summary.Variance = variance
	summary.ClosedAt = &closedAt
	if req.Notes != "" {
		summary.Notes = req.Notes
	}

	return summary, nil
}

func (uc *ShiftUsecase) ListShifts(ctx context.Context, tenantID, outletID string, limit, offset int) ([]domain.ShiftSession, error) {
	return uc.shiftRepo.ListShifts(ctx, tenantID, outletID, limit, offset)
}
