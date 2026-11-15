# Cashier POS Operations & React Client Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Front-of-House Cashier & Store Operations backend (Shift/Drawer management, Held Orders, Split Payments, Receipt generation) and a production-grade React client UI, staged for subsequent Order Lifecycle/KDS fulfillment execution.

**Architecture:** 
- **Backend (Go Clean Architecture):** Domain entities, repositories, and use cases for Shift Sessions, Held Orders, and Split Payments integrated into existing SQLite/Postgres persistence, with zero framework leaks into domain code.
- **Frontend (`web/` React + Vite + TypeScript + Tailwind):** Client clean architecture separated into Domain Types, API Client Gateway (with automatic UUIDv7 idempotency headers), Reactive State Store (Shift, Cart, Catalog), and Modular UI Components.
- **Fulfillment Bridge (WebSocket):** Real-time pub/sub connecting POS checkout to KDS screens and Turnstile verifiers.

**Tech Stack:** Go 1.24+, Gin, SQLite/PostgreSQL, React 19, Vite, TypeScript, Tailwind CSS, Lucide React, Vitest.

**Spec:** [docs/architecture.md](file:///docs/architecture.md), [docs/api_reference.md](file:///docs/api_reference.md), and [docs/domain_and_strategies.md](file:///docs/domain_and_strategies.md).

## Global Constraints
- Clean Architecture boundaries must be strictly preserved: domain has no external imports; delivery depends only on use cases.
- Edge SQLite mode must remain 100% offline-capable (`CGO_ENABLED=0`, `modernc.org/sqlite` in WAL mode).
- Every write transaction must maintain atomicity and record outbox events where appropriate.
- UI must follow antislop hygiene: no generic template slop, high-contrast accessible layouts, keyboard friendly, and responsive.
- All mutating API calls (`POST`, `PUT`) must support and test `X-Idempotency-Key`.

## Review Focus
1. **Split-tender balance mismatch:** Sum of split payments (Cash + Card + QRIS + Room Charge) must strictly equal total order amount; reject checkout if underpaid, calculate change only for cash overpayment.
2. **Shift cash discrepancy:** End-of-shift closing must accurately calculate: `expected_cash = initial_float + cash_sales + cash_in - cash_out`, and flag any `short / over` discrepancy.
3. **Cart & offline hold order corruption:** Parking/holding an order must serialize full sector traits (F&B modifiers, Retail UOM, Room info) so resuming preserves complete state without price drift.
4. **Network disconnect & idempotency retry:** If frontend loses connection mid-checkout, retrying with the same `X-Idempotency-Key` must return the existing order without double-charging or double-deducting stock.
5. **Guest Folio credit limit breach:** In split payments where part is charged to `ROOM_CHARGE`, ensure folio check strictly blocks if charge exceeds remaining available credit.

---

## Task Structure

### Task 1: Shift & Cash Drawer Management (Backend Domain & Repository)

**Files:**
- Create: `internal/domain/shift.go`
- Create: `internal/repository/shift_repo.go`
- Modify: `migrations/edge_sqlite/001_init.sql`
- Modify: `migrations/cloud_postgres/001_init.sql`
- Test: `internal/repository/shift_repo_test.go`

- [x] **Step 1: Write failing test for Shift repository**
  Create `internal/repository/shift_repo_test.go` testing `OpenShift`, `GetActiveShift`, `RecordCashMovement`, and `CloseShift`.
- [x] **Step 2: Run test to make sure it fails**
  Run `go test ./internal/repository -run TestShiftRepo` and verify compilation failure.
- [x] **Step 3: Define Shift domain models in `internal/domain/shift.go`**
  Add `ShiftSession`, `ShiftStatus` (`OPEN`, `CLOSED`), `CashMovement` (`CASH_IN`, `CASH_OUT`), and `ShiftRepository` interface.
- [x] **Step 4: Update migrations for SQLite and PostgreSQL**
  Add tables `shift_sessions` and `cash_movements` with tenant, outlet, cashier ID, float, actual cash, and timestamps.
- [x] **Step 5: Implement `internal/repository/shift_repo.go`**
  Implement cross-compatible SQLite/Postgres SQL queries with query rebound.
- [x] **Step 6: Run tests and ensure they pass**
  Run `go test -v ./internal/repository -run TestShiftRepo`.
- [x] **Step 7: Commit changes**
  Commit with message: `feat(shift): implement shift session domain and repository`

---

### Task 2: Shift Usecase & REST API Handlers (Option 1 Backend)

**Files:**
- Create: `internal/usecase/shift_uc.go`
- Modify: `internal/delivery/http/handlers.go`
- Modify: `internal/delivery/http/router.go`
- Test: `internal/usecase/shift_uc_test.go`

- [x] **Step 1: Write failing test for `ShiftUsecase`**
  Test opening shift, recording drops/pay-ins, calculating expected cash, preventing double active shifts, and closing shift.
- [x] **Step 2: Run test to verify failure**
  Run `go test ./internal/usecase -run TestShiftUsecase`.
- [x] **Step 3: Implement `ShiftUsecase` in `internal/usecase/shift_uc.go`**
  Implement business rules: ensure only 1 active shift per outlet/cashier, validate float amounts, and compute cash variance upon closing.
- [x] **Step 4: Implement HTTP Handlers in `internal/delivery/http/handlers.go`**
  Add `OpenShift`, `GetCurrentShift`, `RecordCashMovement`, `CloseShift` with idempotency middleware.
- [x] **Step 5: Register routes in `internal/delivery/http/router.go`**
  Wire `/v1/shifts/open`, `/v1/shifts/current`, `/v1/shifts/movements`, and `/v1/shifts/close`.
- [x] **Step 6: Run test suite**
  Run `go test -v ./internal/usecase -run TestShiftUsecase` and HTTP integration tests.
- [x] **Step 7: Commit changes**
  Commit with message: `feat(api): add shift management usecase and HTTP endpoints`

---

### Task 3: Multi-Payment Split Billing & Held Orders (Option 1 Backend)

**Files:**
- Modify: `internal/domain/order.go`
- Modify: `internal/usecase/checkout_uc.go`
- Create: `internal/usecase/held_order_uc.go`
- Modify: `internal/delivery/http/handlers.go`
- Modify: `internal/delivery/http/router.go`
- Test: `internal/usecase/split_checkout_test.go`

- [x] **Step 1: Write failing test for split payment checkout and held orders**
  Test order checkout with multiple payment methods (e.g., IDR 30,000 Cash + IDR 47,000 QRIS), parking an order, and resuming it.
- [x] **Step 2: Run test to ensure failure**
  Run `go test ./internal/usecase -run TestSplitCheckout`.
- [x] **Step 3: Update `CheckoutRequest` to support split tender**
  Allow `Payments []CheckoutPaymentRequest` while keeping backward compatibility with single `Payment`.
- [x] **Step 4: Implement Held Orders workflow**
  Add `OrderStatusHeld` (`HELD`) and endpoints `/v1/orders/hold` and `/v1/orders/held`.
- [x] **Step 5: Implement Receipt Formatter endpoint**
  Add `/v1/orders/:id/receipt` returning structured print data (ESC/POS text formatting with store header, line items, breakdown, and cashier info).
- [x] **Step 6: Run tests and ensure green**
  Run `go test -v ./internal/usecase -run TestSplitCheckout`.
- [x] **Step 7: Commit changes**
  Commit with message: `feat(checkout): support split tender payments, held orders, and receipt formatting`

---

### Task 4: Scaffold React POS Client Application (Option 3 Setup)

**Files:**
- Create: `web/package.json`
- Create: `web/vite.config.ts`
- Create: `web/tsconfig.json`
- Create: `web/tailwind.config.js`
- Create: `web/src/index.css`
- Create: `web/src/main.tsx`
- Create: `web/src/App.tsx`
- Test: `web/src/services/api.test.ts`

- [x] **Step 1: Initialize Vite React TypeScript in `web/`**
  Configure Vite with proxy to `http://localhost:8081` (Edge default) and install Tailwind CSS & Lucide React outside sandbox (`BypassSandbox: true`).
- [x] **Step 2: Setup Clean Client Architecture Directory Structure**
  Create:
  - `web/src/domain/` (Entities: Item, CartItem, Order, Payment, Shift)
  - `web/src/services/api/` (API client, idempotency key generator)
  - `web/src/store/` (CartStore, ShiftStore, CatalogStore)
  - `web/src/components/` (Layout, Register, Cart, Shift, Modals)
- [x] **Step 3: Write API Client unit tests**
  Write tests in `web/src/services/api.test.ts` for handling network errors, timeout, and automatic idempotency key generation.
- [x] **Step 4: Verify Vitest test passes**
  Run `pnpm test` in `web/`.
- [x] **Step 5: Commit changes**
  Commit with message: `feat(web): initialize clean architecture react pos application`

---

### Task 5: Cashier Register UI (Catalog Browsing, Barcode Search, Modifiers, Cart)

**Files:**
- Create: `web/src/components/register/CatalogGrid.tsx`
- Create: `web/src/components/register/BarcodeScannerInput.tsx`
- Create: `web/src/components/register/ModifierModal.tsx`
- Create: `web/src/components/cart/CartDrawer.tsx`
- Create: `web/src/store/useCartStore.ts`
- Test: `web/src/store/useCartStore.test.ts`

- [x] **Step 1: Write failing unit test for `useCartStore`**
  Test adding items across sectors (F&B modifiers, Retail barcode quantity, Tourism pax, Hospitality room), calculating subtotal, tax, and discount.
- [x] **Step 2: Implement `useCartStore`**
  Implement clean reactive state with atomic item addition, modifier adjustment, line deletion, and hold/park cart.
- [x] **Step 3: Build `CatalogGrid` & `BarcodeScannerInput`**
  Build sector-filtered item grid with responsive cards, instant category switching, and keyboard barcode listener (`Enter` to submit).
- [x] **Step 4: Build `ModifierModal`**
  Support F&B recipe customization (e.g., extra syrup, alternative milk) updating price delta dynamically.
- [x] **Step 5: Build `CartDrawer`**
  Display line items, quantity increment/decrement, tax breakdown, and clear/park buttons.
- [x] **Step 6: Run tests and verify in browser**
  Run `pnpm test` and build check.
- [x] **Step 7: Commit changes**
  Commit with message: `feat(ui): implement cashier catalog grid, cart drawer, and modifier selector`

---

### Task 6: Payment, Split-Tender & Receipt Modal

**Files:**
- Create: `web/src/components/payment/PaymentModal.tsx`
- Create: `web/src/components/payment/SplitPaymentTender.tsx`
- Create: `web/src/components/receipt/ReceiptModal.tsx`
- Create: `web/src/components/cart/HeldOrdersModal.tsx`
- Test: `web/src/components/payment/split_payment.test.ts`

- [x] **Step 1: Write test for split tender validation logic**
  Ensure balance calculation correctly tracks remaining amount and only permits checkout when remaining == 0.
- [x] **Step 2: Build `PaymentModal` & `SplitPaymentTender`**
  Support one-click full payment (Cash, QRIS, Card, Room Charge) or split payment breakdown (e.g. IDR 50k cash + IDR 25k card).
  Include quick cash buttons (`Exact`, `50,000`, `100,000`) and live change calculation.
- [x] **Step 3: Integrate Checkout API with Idempotency Key**
  Send checkout payload to `/v1/checkout` and handle success/error states gracefully.
- [x] **Step 4: Build `ReceiptModal`**
  Render simulated 58mm/80mm thermal receipt with print action (`window.print()` / ESC-POS preview) and order reference.
- [x] **Step 5: Build `HeldOrdersModal`**
  Allow cashier to view parked carts and restore them to the active register with one tap.
- [x] **Step 6: Run verification tests**
  Execute unit tests and test checkout flow against Edge backend.
- [x] **Step 7: Commit changes**
  Commit with message: `feat(ui): implement split-tender payment modal, receipt preview, and held order restore`

---

### Task 7: Shift Management & Cash Drawer UI

**Files:**
- Create: `web/src/components/shift/ShiftStatusBanner.tsx`
- Create: `web/src/components/shift/OpenShiftModal.tsx`
- Create: `web/src/components/shift/CashMovementModal.tsx`
- Create: `web/src/components/shift/CloseShiftModal.tsx`
- Create: `web/src/store/useShiftStore.ts`
- Test: `web/src/store/useShiftStore.test.ts`

- [x] **Step 1: Write unit tests for `useShiftStore`**
  Test open shift, record cash in/out, and calculate expected cash at close.
- [x] **Step 2: Implement `useShiftStore`**
  Fetch active shift on app mount from `/v1/shifts/current`; block checkout if shift is not opened.
- [x] **Step 3: Build `OpenShiftModal`**
  Prompt cashier for opening float amount before beginning sales.
- [x] **Step 4: Build `ShiftStatusBanner` & `CashMovementModal`**
  Show active cashier name, opening time, and provide quick actions for "Cash In" and "Cash Out / Drop".
- [x] **Step 5: Build `CloseShiftModal` (Z-Report)**
  Input actual counted cash, display system expected cash, highlight discrepancy, and print shift closing summary.
- [x] **Step 6: Verify full end-to-end flow**
  Run all backend tests (`go test ./...`) and frontend tests (`pnpm test` & `pnpm build`).
- [x] **Step 7: Commit changes**
  Commit with message: `feat(ui): implement cashier shift management and cash drawer reconciliation`

---

### Task 8: Staging Option 2 (Fulfillment / KDS / Turnstile Live Hub)

**Files:**
- Create: `web/src/components/kds/KdsQueueView.tsx`
- Create: `web/src/components/gate/TurnstileScanner.tsx`
- Modify: `web/src/App.tsx`
- Test: `web/src/services/ws.test.ts`

- [x] **Step 1: Connect WebSocket Hub in Frontend**
  Connect to `/ws` and handle incoming domain events (`checkout.completed`, `ticket.redeemed`).
- [x] **Step 2: Build KDS Queue View**
  Display incoming kitchen tickets with ticket age timer, items, and status toggles (`PENDING` -> `PREPARING` -> `READY`).
- [x] **Step 3: Build Gate Turnstile Scanner View**
  Camera / barcode reader for offline Ed25519 QR ticket redemption testing with instant visual green/red barrier trigger.
- [x] **Step 4: Commit changes**
  Commit with message: `feat(kds): scaffold live fulfillment queue and gate turnstile monitor`

---

## Verification Checklist
- [ ] `go test -v ./...` passes with 0 failures.
- [ ] `pnpm --prefix web test` executes and passes all test suites.
- [ ] `pnpm --prefix web build` compiles cleanly without TypeScript or CSS errors.
- [ ] Manual test: Open shift -> add items (F&B modifier, retail barcode) -> split payment checkout -> verify receipt -> close shift with cash balance check.
