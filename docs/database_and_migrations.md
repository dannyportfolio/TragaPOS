# Database & Migrations

## Multi-Engine Support

The engine supports dual database backends depending on the deployment target:

1. **PostgreSQL (`CLOUD` Mode):**
   - High concurrency, multi-tenant indexing, JSONB support.
   - Parameter placeholders use `$1, $2, ...` syntax (handled transparently via `repository.Rebind`).
   - Tables use UUIDv7 strings for distributed ID generation without sequence bottlenecks.

2. **SQLite (`EDGE` Mode):**
   - Pure Go implementation (`modernc.org/sqlite`, `CGO_ENABLED=0`).
   - Runs in Write-Ahead Logging (`WAL`) mode with `busy_timeout=5000` and `synchronous=NORMAL`.
   - Single-writer safety enforced with `MaxOpenConns(1)`.

---

## Migration Architecture

Schema migrations reside under:
- `migrations/cloud_postgres/001_init.sql`
- `migrations/edge_sqlite/001_init.sql`

The database initialization package (`internal/database`) loads and applies these schema definitions automatically upon engine startup, verifying table structures before HTTP listeners activate.

---

## Core Tables

| Table | Purpose |
| :--- | :--- |
| `items` | Polymorphic catalog of sellable products, services, and passes. |
| `inventory_balances` | Materialized current stock level per SKU and outlet. |
| `stock_movements` | Immutable append-only audit trail of stock deductions and additions. |
| `orders` | Header record containing financial totals, tax, status, and metadata. |
| `order_items` | Detailed line items with pricing, modifier snapshots, and sector payloads. |
| `payments` | Settlement allocations against an order. |
| `guest_folios` | Open account folios for hospitality rooms. |
| `folio_charges` | Charges attached to guest folios. |
| `issued_tickets` | Cryptographically signed tickets issued on Cloud. |
| `redeemed_tickets` | Turnstile gate redemptions recorded offline on Edge. |
| `sync_outbox` | Edge transactional event queue pending upstream synchronization. |
