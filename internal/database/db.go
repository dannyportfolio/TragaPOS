package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"

	"pos-engine/internal/config"
)

type DatabaseHealthChecker struct {
	db *sql.DB
}

func NewDatabaseHealthChecker(db *sql.DB) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{db: db}
}

func (h *DatabaseHealthChecker) Ping(ctx context.Context) error {
	return h.db.PingContext(ctx)
}

func Open(cfg *config.Config) (*sql.DB, error) {
	if cfg.AppMode == config.ModeCloud {
		return openPostgres(cfg)
	}
	return openSQLite(cfg)
}

func openPostgres(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DBDSN)
	if err != nil {
		return nil, fmt.Errorf("postgres open error: %w", err)
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(10 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Printf("[Database] Warning: PostgreSQL not reachable yet at %s (%v)", cfg.DBDSN, err)
	} else {
		log.Println("[Database] Connected to PostgreSQL Cloud Database.")
		if err := ApplyMigrations(db, config.ModeCloud); err != nil {
			log.Printf("[Database] PostgreSQL migration warning: %v", err)
		}
	}

	return db, nil
}

func openSQLite(cfg *config.Config) (*sql.DB, error) {
	dir := filepath.Dir(cfg.DBDSN)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)", cfg.DBDSN)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open error: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("sqlite ping error: %w", err)
	}

	log.Println("[Database] SQLite database opened in WAL mode.")
	if err := ApplyMigrations(db, config.ModeEdge); err != nil {
		log.Printf("[Database] SQLite migration warning: %v", err)
	}

	return db, nil
}

func ApplyMigrations(db *sql.DB, mode config.AppMode) error {
	schema := EdgeSchemaSQL
	if mode == config.ModeCloud {
		schema = CloudSchemaSQL
	}

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed executing schema migration: %w", err)
	}

	log.Printf("[Database] Verified schema migrations for %s mode.", mode)
	return nil
}
