# Headless Multi-Industry POS & Commerce Engine

A production-ready, distributed, offline-first commerce engine written in Go. Compiled and configured from a single unified codebase into two runtime targets via `APP_MODE`:

1. **`CLOUD` Mode (Central Hub):** Runs on Managed Kubernetes with PostgreSQL. Acts as Master Data Manager, Multi-Tenant Coordinator, Online Ticket Issuer (Ed25519 Signer), and Sync Ingestor.
2. **`EDGE` Mode (Local Outlet Hub):** Runs on local Mini PCs with Local SQLite (`modernc.org/sqlite` in WAL mode, `CGO_ENABLED=0`). Single source of truth for daily store/gate operations over Local LAN, capable of running 100% offline.

---

## Architectural Highlights

```mermaid
flowchart TD
    subgraph Cloud["CLOUD RUNTIME (Central Hub)"]
        CP[PostgreSQL Master Data]
        CS[Ed25519 Private Key Signer]
        CI[Sync Ingestor /v1/sync/push]
        CD[Delta Catalog /v1/sync/pull]
    end

    subgraph Edge["EDGE RUNTIME (Local Outlet Mini PC)"]
        direction TB
        ES[SQLite WAL mode - CGO_ENABLED=0]
        TX[Transactional Outbox Pattern]
        UC[Unified Checkout Engine]
        TK[100% Offline Ed25519 Verifier]
        SW[Background Sync Worker]
        WS[WebSocket Hub: KDS & Gate Turnstile]
    end

    %% Sync Flows
    CD -- "Delta Sync (Catalog & Pricing)" --> SW
    SW -- "Append-Only Outbox Events (Idempotent)" --> CI
    CS -. "Issued Cryptographic QR Tickets" .-> TK
    UC --> TX --> ES
    TK --> ES
    UC --> WS
```

### 1. Clean Architecture & DDD
- **Presentation / Delivery (`internal/delivery/http`, `ws`):** REST API with Gin, Idempotency middleware (`X-Idempotency-Key`), permissive LAN CORS, and real-time WebSocket Hub for Kitchen Display Systems (KDS) and Gate Turnstiles.
- **Usecase (`internal/usecase`):** Orchestrates business workflows inside atomic transaction boundaries (`WithTransaction`), handles payment allocations, and dispatches domain events.
- **Domain (`internal/domain`):** Pure Go structs and domain interfaces with zero external framework dependencies.
- **Repository / Infrastructure (`internal/repository`):** Unified SQL abstraction supporting PostgreSQL (`$1, $2, ...`) and SQLite (`?`) transparently.

### 2. Polymorphic Multi-Sector Catalog
A single sellable items catalog (`items` table + `sector_traits` JSON) driven by Go strategy interfaces (`FulfillmentStrategy`):
- **F&B (Cafe/Resto):** Composite BOM (Bill of Materials) recipe deduction (e.g., deducting 18g coffee beans and 200ml milk per latte), item modifiers, and KDS routing tags.
- **Retail (Swalayan):** High-velocity barcode/GTIN lookup, multi-UOM conversions (e.g., BOX = 24 PCS), batch/expiry tracking, and tiered wholesale pricing.
- **Tourism (Park/Zoo):** Quota/date-bound passes with Ed25519 Cryptographic Offline QR verification.
- **Hospitality (Resort/Hotel):** Duration-bound room/facility bookings, housekeeping status (`CLEAN`, `DIRTY`), and Open Folio / Room Charge (charging cafe/park tabs to a guest's room).

### 3. Offline-First & Sync Engine
- **Collision-Free Identity:** Time-ordered UUIDv7 (`uid.NewV7()`) across all distributed nodes.
- **Transactional Outbox:** Every write on Edge (Checkout, Stock Movement, Ticket Redemption) atomically writes business records and an outbox event in a single database transaction.
- **Physical Reality Wins:** Cloud never rejects paid offline transactions due to negative stock. Events are accepted, negative stock is recorded, and a `STOCK_DISCREPANCY_ALERT` is triggered for administrative reconciliation.

### 4. Cryptographic Offline Ticketing
- **Signer (Cloud):** Signs payload (`TicketID`, `ValidUntil`, `Pax`, `Category`) using Ed25519 private key.
- **Verifier (Edge):** Verifies QR mathematical signature and expiration time locally without internet. Atomic insertion into `redeemed_tickets` prevents local double-scans.

---

## Documentation & Architecture Guides

Detailed documentation, architecture diagrams, API contracts, and domain specifications are organized in the [`docs/`](file:///docs) directory:
- [Architecture & Layers](file:///docs/architecture.md)
- [API Reference & Contracts](file:///docs/api_reference.md)
- [Domain Models & Sector Fulfillment Strategies](file:///docs/domain_and_strategies.md)
- [Database & Migrations Guide](file:///docs/database_and_migrations.md)
- [Clean Code Standards & Engineering Policies](file:///docs/clean_code_standards.md)

---

## Directory Structure

```text
pos-engine/
├── cmd/
│   └── server/
│       └── main.go                 # Lean Composition Root & Graceful Shutdown
├── docs/                           # Dedicated documentation directory
│   ├── architecture.md
│   ├── api_reference.md
│   ├── domain_and_strategies.md
│   ├── database_and_migrations.md
│   └── clean_code_standards.md
├── internal/
│   ├── config/                     # Environment configuration loader
│   ├── database/                   # DB connection setup & migration runner
│   ├── domain/                     # Entities, Value Objects, Repository & TxManager Interfaces
│   │   ├── catalog.go
│   │   ├── order.go
│   │   ├── sync.go
│   │   ├── ticket.go
│   │   └── tx.go                   # TransactionManager abstraction
│   ├── usecase/                    # Clean Architecture Application Logic
│   │   ├── catalog_uc.go           # Catalog querying and upserting usecase
│   │   ├── checkout_uc.go          # Unified checkout with Strategy hooks + Outbox
│   │   ├── order_uc.go             # Order and Folio management usecase
│   │   ├── ticket_uc.go            # Cloud Sign & Edge Offline Verify/Redeem
│   │   ├── sync_worker.go          # Background Outbox Pusher & Delta Puller
│   │   └── strategies/             # F&B, Retail, Tourism, Hospitality implementations
│   ├── repository/                 # SQL implementations (PostgreSQL & SQLite compatible)
│   │   ├── db_tx.go                # Atomic Transaction Manager
│   │   ├── catalog_repo.go
│   │   ├── order_repo.go
│   │   ├── ticket_repo.go
│   │   └── outbox_repo.go
│   ├── delivery/
│   │   ├── http/                   # REST Handlers, Idempotency Middleware, Router
│   │   └── ws/                     # Real-time WebSocket Hub for KDS & Gate
│   └── pkg/
│       ├── crypto/                 # Ed25519 Keygen, Signer, Verifier
│       └── uid/                    # UUIDv7 Generator wrapper
├── migrations/
│   ├── cloud_postgres/             # DDL for Cloud Central DB
│   └── edge_sqlite/                # DDL for Local Edge DB (WAL mode)
├── deploy/
│   ├── Dockerfile                  # Multi-stage Distroless build (CGO_ENABLED=0)
│   └── k8s/
│       ├── cloud-deployment.yaml   # Cloud K8s + HPA
│       └── edge-k3s.yaml           # Edge K3s + PVC
└── go.mod
```

---

## Quick Start & Verification

### Run in EDGE Mode (Default)
```bash
# Set environment
export APP_MODE=EDGE
export SQLITE_PATH=./data/edge.db
export PORT=8081

# Run server
go run cmd/server/main.go
```

### Run in CLOUD Mode
```bash
export APP_MODE=CLOUD
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/pos_central?sslmode=disable"
export PORT=8080

go run cmd/server/main.go
```

### Execute Tests
```bash
go test -v ./...
```
