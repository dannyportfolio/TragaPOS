# API Reference & Contracts

## Base URLs & Headers
- Cloud Default: `http://localhost:8080/v1`
- Edge Default: `http://localhost:8081/v1`
- Standard Headers:
  - `Content-Type: application/json`
  - `X-Idempotency-Key`: Optional UUID string for safe request retries on `POST` and `PUT` endpoints.

---

## Health & Probes

### GET `/healthz`
Liveness probe.
- Response `200 OK`:
  ```json
  {
    "status": "HEALTHY",
    "mode": "EDGE",
    "timestamp": "2026-11-12T00:00:00Z"
  }
  ```

### GET `/readyz`
Readiness probe testing database ping.
- Response `200 OK` when database is healthy.
- Response `503 Service Unavailable` if database connection fails.

---

## Catalog Endpoints

### POST `/v1/catalog/items`
Upserts a catalog item.
- Request Body:
  ```json
  {
    "sku": "COF-LATTE-01",
    "barcode": "899123456001",
    "name": "Caffe Latte",
    "sector": "FB",
    "base_price": 30000,
    "currency": "IDR",
    "tax_rate": 0.1,
    "sector_traits": {
      "bom": [
        {"raw_sku": "RAW-BEANS-G", "qty": 18, "uom": "G"},
        {"raw_sku": "RAW-MILK-ML", "qty": 200, "uom": "ML"}
      ]
    }
  }
  ```

### GET `/v1/catalog/items`
Lists active catalog items.
- Query Parameters: `limit` (default 50), `offset` (default 0).

### GET `/v1/catalog/items/:sku`
Fetches a single item by SKU.

### GET `/v1/catalog/barcode/:barcode`
Fetches a single item by barcode.

### GET `/v1/catalog/stock/:sku`
Fetches inventory balance for a SKU.

---

## Checkout & Orders

### POST `/v1/checkout`
Executes an atomic checkout.
- Request Body:
  ```json
  {
    "tenant_id": "tenant_1",
    "outlet_id": "outlet_1",
    "items": [
      {
        "sku": "COF-LATTE-01",
        "quantity": 2,
        "selected_modifiers": [
          {"name": "Extra Shot", "price_delta": 5000}
        ]
      }
    ],
    "payment": {
      "method": "CASH",
      "amount": 77000
    }
  }
  ```

### GET `/v1/orders/:id`
Retrieves an order by its ID, including ordered line items and payment records.

---

## Hospitality Folios

### POST `/v1/folios`
Opens a new guest room folio.
- Request Body:
  ```json
  {
    "tenant_id": "tenant_1",
    "outlet_id": "outlet_1",
    "room_number": "101",
    "guest_name": "Alice Smith",
    "credit_limit": 5000000,
    "check_in_at": "2026-11-12T14:00:00Z",
    "check_out_at": "2026-11-14T12:00:00Z"
  }
  ```

### GET `/v1/folios/room/:room`
Fetches active folio and current accumulated charges for a given room number.

---

## Cryptographic Ticketing

### POST `/v1/tickets/issue` (CLOUD Mode)
Signs a ticket payload using the central Ed25519 private key.
- Request Body:
  ```json
  {
    "tenant_id": "tenant_1",
    "category": "VIP_PASS",
    "pax": 2,
    "valid_hours": 24
  }
  ```
- Response `200 OK`:
  ```json
  {
    "ticket_id": "018f...uuidv7",
    "qr_string": "ey...payload.<signature>",
    "valid_until": 1794500000
  }
  ```

### POST `/v1/tickets/redeem` (EDGE Mode)
Verifies the cryptographic signature offline and checks for double redemption.
- Request Body:
  ```json
  {
    "qr_string": "ey...payload.<signature>",
    "tenant_id": "tenant_1",
    "outlet_id": "gate_1",
    "gate_zone": "MAIN_GATE",
    "device_id": "TURNSTILE_01"
  }
  ```

---

## Sync Engine

### POST `/v1/sync/push` (CLOUD Mode)
Receives outbox event batches pushed by Edge nodes.

### GET `/v1/sync/pull` (CLOUD Mode)
Returns catalog items updated since query parameter `since_version`.

### POST `/v1/sync/trigger` (EDGE Mode)
Triggers an immediate background synchronization cycle on an Edge node.

---

## WebSocket Event Hub

### GET `/ws`
Connects to the server event broker for real-time notifications:
- Kitchen Display System: topic `kds`, event `ORDER_PLACED`
- Turnstile Gateway: topic `turnstile`, event `GATE_OPEN`
