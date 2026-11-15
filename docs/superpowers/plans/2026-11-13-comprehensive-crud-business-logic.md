# Implementation Plan - Full Business Logic CRUD Overhaul

**Target Architecture:** Full Multi-Sector CRUD across Catalog, Orders, Inventory, Folios, and Shifts  
**Components:** Backend Core (Go 1.24) + Frontend Client (React 19 Vite)  
**Execution Environment:** WSL2 Kali Linux + Docker + Local Host Dev  
**Created:** 2026-11-13T13:19:00+07:00  

---

## 1. Problem Statement & Scope

The user requested:
> *"stop run the docker, fix main business logic... I want to this project worked CRUD!!!!!"*

Currently:
1. Docker stack has been stopped cleanly (`docker compose down`).
2. The core business engine was designed with asymmetric read-heavy edge synchronization and write-ahead checkouts, but lacked standard CRUD APIs and UI capabilities:
   - **Catalog Items**: Has Create/Upsert and List, but lacks Delete, GetByID, and UI management.
   - **Inventory**: Has GetStock and Outbox deduction, but lacks manual stock adjustments (`RESTOCK`, `AUDIT`, `DAMAGE`).
   - **Orders**: Has Checkout, Hold, and UpdateStatus, but lacks `ListOrders` route, Cancel/Void order with inventory rollback, and UI order history.
   - **Hospitality Folios**: Has Create and GetByRoom, but lacks List, Update, and Settle/Close.
   - **Frontend UI**: Only provides POS register, KDS, and Gate Scanner. Has NO Catalog Management (Items CRUD) or Order Management (Orders CRUD) interfaces.

---

## 2. 5-Stage Implementation Plan

### Stage 1: Domain & Repository Expansion (Go)
1. **Catalog Repository**:
   - Add `GetByID(ctx, id string) (*domain.Item, error)`
   - Add `DeleteItem(ctx, tenantID, id string) error` (soft-delete `is_active = false`, increments version for multi-edge sync).
   - Add `AdjustStock(ctx, tenantID, outletID, sku string, delta float64, reason, refID string) error`.
2. **Order Repository**:
   - Expose `ListOrders(ctx, tenantID, outletID, status string, limit, offset int) ([]domain.Order, error)` with optional status filter.
   - Implement `VoidOrder(ctx, tenantID, orderID string, reason string) error` that updates status to `CANCELLED` and reverses stock movements.
   - Implement `ListFolios(ctx, tenantID string, openOnly bool, limit, offset int) ([]domain.GuestFolio, error)`.
   - Implement `CloseFolio(ctx, tenantID, folioID string) error`.
3. **Shift Repository**:
   - Implement `ListShifts(ctx, tenantID, outletID string, limit, offset int) ([]domain.ShiftSession, error)`.

### Stage 2: Usecase Business Validation (Go)
1. **Catalog Usecase**:
   - Validate Item Create/Update: non-empty SKU, name, price >= 0, valid Sector.
   - Implement Delete with version bumping.
   - Implement Stock Adjustment with `StockMovementEvent`.
2. **Order Usecase**:
   - Implement `ListOrders` with pagination and status filter.
   - Implement `VoidOrder` with transaction safety and WebSocket broadcast (`ORDER_VOIDED`).
   - Implement `ListFolios` and `CloseFolio`.

### Stage 3: HTTP API & Routing (Gin)
Expose standard RESTful endpoints under `/v1`:
- **Catalog**:
  - `POST /v1/catalog/items` (Create / Upsert)
  - `GET /v1/catalog/items` (List with `search`, `sector`, `active_only`, `limit`, `offset`)
  - `GET /v1/catalog/items/:id` (Get by ID)
  - `GET /v1/catalog/items/sku/:sku` (Get by SKU)
  - `PUT /v1/catalog/items/:id` (Update item)
  - `DELETE /v1/catalog/items/:id` (Delete/Deactivate item)
  - `POST /v1/catalog/stock/adjust` (Manual stock adjustment)
- **Orders**:
  - `GET /v1/orders` (List orders with pagination & status filter)
  - `GET /v1/orders/:id` (Get order details)
  - `PUT /v1/orders/:id/status` (Update order status)
  - `POST /v1/orders/:id/void` or `DELETE /v1/orders/:id` (Void/Cancel order)
- **Hospitality Folios**:
  - `GET /v1/folios` (List folios)
  - `POST /v1/folios/:id/settle` (Settle/Close folio)
- **Shifts**:
  - `GET /v1/shifts` (List past shift history)

### Stage 4: Frontend Management UI (React 19 + TypeScript)
1. **API Client (`web/src/services/api.ts`)**:
   - Add: `createItem`, `updateItem`, `deleteItem`, `adjustStock`, `getOrders`, `voidOrder`, `getFolios`, `settleFolio`, `getShiftHistory`.
2. **Header Navigation (`web/src/components/layout/Header.tsx`)**:
   - Add `ITEMS` (Catalog & Stock Management) and `ORDERS` (Order History & Management) tabs.
3. **Catalog Management View (`web/src/components/catalog/ItemsView.tsx`)**:
   - Searchable, filterable table of catalog items.
   - Add/Edit Item modal with sector-specific traits.
   - Stock Adjustment modal with reason dropdown.
   - Delete/Deactivate button with confirmation.
4. **Order Management View (`web/src/components/orders/OrdersView.tsx`)**:
   - Filterable table of past orders by status (`PAID`, `HELD`, `PREPARING`, `READY`, `COMPLETED`, `CANCELLED`).
   - View order details modal.
   - Void/Cancel order button.
   - Thermal receipt preview modal.

### Stage 5: Verification & Quality Gate
1. Execute backend unit tests (`go test ./...`) inside WSL Kali Linux.
2. Execute frontend unit tests (`pnpm test` / Vitest) inside WSL Kali Linux.
3. Validate type check (`tsc -b`).
4. Perform end-to-end operational verification across all CRUD endpoints.
