package domain

import (
	"context"
	"errors"
	"time"
)

type ShiftStatus string

const (
	ShiftStatusOpen   ShiftStatus = "OPEN"
	ShiftStatusClosed ShiftStatus = "CLOSED"
)

type CashMovementType string

const (
	CashMovementIn          CashMovementType = "CASH_IN"
	CashMovementOut         CashMovementType = "CASH_OUT"
	CashMovementFloatAdjust CashMovementType = "FLOAT_ADJUST"
)

var (
	ErrShiftNotFound      = errors.New("shift session not found")
	ErrActiveShiftExists  = errors.New("an active shift session already exists for this cashier")
	ErrNoActiveShift      = errors.New("no active shift session found")
	ErrShiftAlreadyClosed = errors.New("shift session is already closed")
)

type ShiftSession struct {
	ID           string      `json:"id"`
	TenantID     string      `json:"tenant_id"`
	OutletID     string      `json:"outlet_id"`
	CashierID    string      `json:"cashier_id"`
	CashierName  string      `json:"cashier_name"`
	Status       ShiftStatus `json:"status"`
	OpeningFloat int64       `json:"opening_float"`
	ExpectedCash int64       `json:"expected_cash"`
	ActualCash   int64       `json:"actual_cash"`
	Variance     int64       `json:"variance"`
	Notes        string      `json:"notes,omitempty"`
	OpenedAt     time.Time   `json:"opened_at"`
	ClosedAt     *time.Time  `json:"closed_at,omitempty"`
}

type CashMovement struct {
	ID        string           `json:"id"`
	ShiftID   string           `json:"shift_id"`
	Type      CashMovementType `json:"type"`
	Amount    int64            `json:"amount"`
	Reason    string           `json:"reason"`
	CreatedAt time.Time        `json:"created_at"`
}

type ShiftSummary struct {
	ShiftID           string       `json:"shift_id"`
	TenantID          string       `json:"tenant_id"`
	OutletID          string       `json:"outlet_id"`
	CashierID         string       `json:"cashier_id"`
	CashierName       string       `json:"cashier_name"`
	Status            ShiftStatus  `json:"status"`
	OpeningFloat      int64        `json:"opening_float"`
	CashInTotal       int64        `json:"cash_in_total"`
	CashOutTotal      int64        `json:"cash_out_total"`
	CashSalesTotal    int64        `json:"cash_sales_total"`
	NonCashSalesTotal int64        `json:"non_cash_sales_total"`
	TotalSales        int64        `json:"total_sales"`
	ExpectedCash      int64        `json:"expected_cash"`
	ActualCash        int64        `json:"actual_cash"`
	Variance          int64        `json:"variance"`
	Notes             string       `json:"notes,omitempty"`
	OpenedAt          time.Time    `json:"opened_at"`
	ClosedAt          *time.Time   `json:"closed_at,omitempty"`
	Movements         []CashMovement `json:"movements,omitempty"`
}

type ShiftRepository interface {
	CreateShift(ctx context.Context, shift *ShiftSession) error
	GetActiveShift(ctx context.Context, tenantID, outletID, cashierID string) (*ShiftSession, error)
	GetShiftByID(ctx context.Context, id string) (*ShiftSession, error)
	ListShifts(ctx context.Context, tenantID, outletID string, limit, offset int) ([]ShiftSession, error)
	CloseShift(ctx context.Context, shiftID string, actualCash, expectedCash, variance int64, notes string, closedAt time.Time) error
	CreateCashMovement(ctx context.Context, movement *CashMovement) error
	ListCashMovements(ctx context.Context, shiftID string) ([]CashMovement, error)
	GetShiftSalesBreakdown(ctx context.Context, tenantID, outletID string, fromTime time.Time, toTime time.Time) (cashSales int64, nonCashSales int64, err error)
}
