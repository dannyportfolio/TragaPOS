package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

type txKey struct{}

// DBTX is the common interface satisfied by both *sql.DB and *sql.Tx.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type TxManager struct {
	db     *sql.DB
	driver string
}

func NewTxManager(db *sql.DB, driver string) *TxManager {
	return &TxManager{
		db:     db,
		driver: driver,
	}
}

func (m *TxManager) DB() *sql.DB {
	return m.db
}

func (m *TxManager) Driver() string {
	return m.driver
}

// WithTransaction runs the given function inside a database transaction.
// If fn returns an error, the transaction is rolled back; otherwise committed.
func (m *TxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// If already in a transaction, reuse it (nested transaction boundary)
	if _, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return fn(ctx)
	}

	tx, err := m.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txKey{}, tx)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
			return fmt.Errorf("tx err: %v, rollback err: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetExecutor extracts the active *sql.Tx from context, or falls back to *sql.DB.
func (m *TxManager) GetExecutor(ctx context.Context) DBTX {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok && tx != nil {
		return tx
	}
	return m.db
}

// Rebind adapts standard '?' placeholders to PostgreSQL '$1, $2, ...' placeholders if needed.
func Rebind(driver string, query string) string {
	if driver != "postgres" {
		return query
	}

	var b strings.Builder
	b.Grow(len(query) + 20)
	paramIdx := 1

	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '?' {
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(paramIdx))
			paramIdx++
		} else {
			b.WriteByte(ch)
		}
	}
	return b.String()
}
