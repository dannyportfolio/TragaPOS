package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/uid"
	"pos-engine/internal/repository"
)

func setupShiftTestDB(t *testing.T) (*sql.DB, *repository.TxManager) {
	dbName := fmt.Sprintf("file:test_shift_%s?mode=memory&cache=shared", uid.NewV7())
	db, err := sql.Open("sqlite", dbName)
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}

	schema := `
	CREATE TABLE shift_sessions (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		outlet_id TEXT NOT NULL,
		cashier_id TEXT NOT NULL,
		cashier_name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'OPEN',
		opening_float INTEGER NOT NULL DEFAULT 0,
		expected_cash INTEGER NOT NULL DEFAULT 0,
		actual_cash INTEGER NOT NULL DEFAULT 0,
		variance INTEGER NOT NULL DEFAULT 0,
		notes TEXT,
		opened_at TEXT NOT NULL,
		closed_at TEXT
	);
	CREATE TABLE cash_movements (
		id TEXT PRIMARY KEY,
		shift_id TEXT NOT NULL,
		type TEXT NOT NULL,
		amount INTEGER NOT NULL,
		reason TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE orders (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		outlet_id TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE payments (
		id TEXT PRIMARY KEY,
		order_id TEXT NOT NULL,
		method TEXT NOT NULL,
		amount INTEGER NOT NULL,
		paid_at TEXT NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to exec schema: %v", err)
	}

	txMgr := repository.NewTxManager(db, "sqlite")
	return db, txMgr
}

func TestShiftRepository(t *testing.T) {
	db, txMgr := setupShiftTestDB(t)
	defer db.Close()

	repo := repository.NewShiftRepository(txMgr)
	ctx := context.Background()

	shiftID := uid.NewV7()
	shift := &domain.ShiftSession{
		ID:           shiftID,
		TenantID:     "tenant-1",
		OutletID:     "outlet-1",
		CashierID:    "cashier-101",
		CashierName:  "Alice",
		Status:       domain.ShiftStatusOpen,
		OpeningFloat: 100000,
		OpenedAt:     time.Now().UTC().Add(-2 * time.Hour),
	}

	// 1. Create Shift
	if err := repo.CreateShift(ctx, shift); err != nil {
		t.Fatalf("CreateShift failed: %v", err)
	}

	// 2. Get Active Shift
	active, err := repo.GetActiveShift(ctx, "tenant-1", "outlet-1", "cashier-101")
	if err != nil {
		t.Fatalf("GetActiveShift failed: %v", err)
	}
	if active.ID != shiftID {
		t.Errorf("expected shift ID %s, got %s", shiftID, active.ID)
	}
	if active.OpeningFloat != 100000 {
		t.Errorf("expected float 100000, got %d", active.OpeningFloat)
	}

	// 3. Record Cash Movements
	movement1 := &domain.CashMovement{
		ID:        uid.NewV7(),
		ShiftID:   shiftID,
		Type:      domain.CashMovementIn,
		Amount:    50000,
		Reason:    "Extra change requested",
		CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
	}
	movement2 := &domain.CashMovement{
		ID:        uid.NewV7(),
		ShiftID:   shiftID,
		Type:      domain.CashMovementOut,
		Amount:    20000,
		Reason:    "Bought office coffee supplies",
		CreatedAt: time.Now().UTC().Add(-30 * time.Minute),
	}
	if err := repo.CreateCashMovement(ctx, movement1); err != nil {
		t.Fatalf("CreateCashMovement in failed: %v", err)
	}
	if err := repo.CreateCashMovement(ctx, movement2); err != nil {
		t.Fatalf("CreateCashMovement out failed: %v", err)
	}

	movements, err := repo.ListCashMovements(ctx, shiftID)
	if err != nil {
		t.Fatalf("ListCashMovements failed: %v", err)
	}
	if len(movements) != 2 {
		t.Fatalf("expected 2 movements, got %d", len(movements))
	}

	// 4. Simulate Orders & Payments for Sales Breakdown
	order1ID := uid.NewV7()
	order2ID := uid.NewV7()
	now := time.Now().UTC()
	timeFmt := "2006-01-02 15:04:05.000000"
	_, _ = db.Exec(`INSERT INTO orders (id, tenant_id, outlet_id, status, created_at) VALUES (?, ?, ?, 'PAID', ?)`,
		order1ID, "tenant-1", "outlet-1", now.Add(-45*time.Minute).UTC().Format(timeFmt))
	_, _ = db.Exec(`INSERT INTO orders (id, tenant_id, outlet_id, status, created_at) VALUES (?, ?, ?, 'PAID', ?)`,
		order2ID, "tenant-1", "outlet-1", now.Add(-15*time.Minute).UTC().Format(timeFmt))

	_, _ = db.Exec(`INSERT INTO payments (id, order_id, method, amount, paid_at) VALUES (?, ?, 'CASH', 75000, ?)`,
		uid.NewV7(), order1ID, now.UTC().Format(timeFmt))
	_, _ = db.Exec(`INSERT INTO payments (id, order_id, method, amount, paid_at) VALUES (?, ?, 'QRIS', 50000, ?)`,
		uid.NewV7(), order2ID, now.UTC().Format(timeFmt))

	cashSales, nonCashSales, err := repo.GetShiftSalesBreakdown(
		ctx, "tenant-1", "outlet-1",
		now.Add(-3*time.Hour), now.Add(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("GetShiftSalesBreakdown failed: %v", err)
	}
	if cashSales != 75000 {
		t.Errorf("expected cash sales 75000, got %d", cashSales)
	}
	if nonCashSales != 50000 {
		t.Errorf("expected non-cash sales 50000, got %d", nonCashSales)
	}

	// 5. Close Shift
	// Expected cash = 100000 (float) + 50000 (in) - 20000 (out) + 75000 (sales) = 205000
	expectedCash := int64(205000)
	actualCash := int64(200000) // 5000 short
	variance := actualCash - expectedCash // -5000
	closedAt := time.Now().UTC()

	err = repo.CloseShift(ctx, shiftID, actualCash, expectedCash, variance, "Slight short on closing", closedAt)
	if err != nil {
		t.Fatalf("CloseShift failed: %v", err)
	}

	// 6. Verify closed state
	closed, err := repo.GetShiftByID(ctx, shiftID)
	if err != nil {
		t.Fatalf("GetShiftByID failed: %v", err)
	}
	if closed.Status != domain.ShiftStatusClosed {
		t.Errorf("expected CLOSED, got %s", closed.Status)
	}
	if closed.Variance != -5000 {
		t.Errorf("expected variance -5000, got %d", closed.Variance)
	}

	// 7. Verify no active shift remains
	_, err = repo.GetActiveShift(ctx, "tenant-1", "outlet-1", "cashier-101")
	if err != domain.ErrNoActiveShift {
		t.Errorf("expected ErrNoActiveShift, got %v", err)
	}
}
