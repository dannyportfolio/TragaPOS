-- Edge SQLite Schema (Runs locally 100% offline on Mini PCs)
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS items (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    outlet_id TEXT NOT NULL,
    sku TEXT NOT NULL,
    barcode TEXT,
    name TEXT NOT NULL,
    sector TEXT NOT NULL,
    base_price INTEGER NOT NULL,
    currency TEXT NOT NULL DEFAULT 'IDR',
    tax_rate REAL NOT NULL DEFAULT 0.0,
    sector_traits TEXT NOT NULL DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (tenant_id, sku)
);

CREATE INDEX IF NOT EXISTS idx_items_barcode ON items (tenant_id, barcode);
CREATE INDEX IF NOT EXISTS idx_items_version ON items (tenant_id, version);

CREATE TABLE IF NOT EXISTS inventory_balances (
    tenant_id TEXT NOT NULL,
    outlet_id TEXT NOT NULL,
    sku TEXT NOT NULL,
    on_hand_qty REAL NOT NULL DEFAULT 0.0,
    allocated_qty REAL NOT NULL DEFAULT 0.0,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (tenant_id, outlet_id, sku)
);

CREATE TABLE IF NOT EXISTS stock_movements (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    outlet_id TEXT NOT NULL,
    sku TEXT NOT NULL,
    delta_qty REAL NOT NULL,
    reason TEXT NOT NULL,
    reference_id TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    outlet_id TEXT NOT NULL,
    order_number TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    subtotal INTEGER NOT NULL DEFAULT 0,
    tax_amount INTEGER NOT NULL DEFAULT 0,
    discount_amount INTEGER NOT NULL DEFAULT 0,
    total_amount INTEGER NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'IDR',
    notes TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_outlet ON orders (tenant_id, outlet_id, created_at DESC);

CREATE TABLE IF NOT EXISTS order_items (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    item_id TEXT NOT NULL,
    sku TEXT NOT NULL,
    name TEXT NOT NULL,
    sector TEXT NOT NULL,
    quantity REAL NOT NULL DEFAULT 1.0,
    uom TEXT NOT NULL DEFAULT 'PCS',
    unit_price INTEGER NOT NULL,
    subtotal INTEGER NOT NULL,
    tax_amount INTEGER NOT NULL DEFAULT 0,
    total_amount INTEGER NOT NULL,
    selected_modifiers TEXT NOT NULL DEFAULT '[]',
    sector_payload TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    method TEXT NOT NULL,
    amount INTEGER NOT NULL,
    reference TEXT,
    paid_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS guest_folios (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    outlet_id TEXT NOT NULL,
    room_number TEXT NOT NULL,
    guest_name TEXT NOT NULL,
    credit_limit INTEGER NOT NULL DEFAULT 0,
    current_total INTEGER NOT NULL DEFAULT 0,
    is_open INTEGER NOT NULL DEFAULT 1,
    check_in_at TEXT NOT NULL,
    check_out_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_folios_room ON guest_folios (tenant_id, room_number, is_open);

CREATE TABLE IF NOT EXISTS folio_charges (
    id TEXT PRIMARY KEY,
    folio_id TEXT NOT NULL REFERENCES guest_folios(id) ON DELETE CASCADE,
    order_id TEXT NOT NULL,
    amount INTEGER NOT NULL,
    notes TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS redeemed_tickets (
    id TEXT PRIMARY KEY,
    ticket_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    outlet_id TEXT NOT NULL,
    gate_zone TEXT NOT NULL,
    pax_redeemed INTEGER NOT NULL DEFAULT 1,
    redeemed_at TEXT NOT NULL,
    device_id TEXT,
    UNIQUE (tenant_id, ticket_id)
);

CREATE TABLE IF NOT EXISTS issued_tickets (
    ticket_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    category TEXT NOT NULL,
    pax INTEGER NOT NULL DEFAULT 1,
    valid_until INTEGER NOT NULL,
    issued_at INTEGER NOT NULL,
    order_id TEXT,
    signature TEXT NOT NULL
);

-- Transactional Outbox for Offline Upstream Sync
CREATE TABLE IF NOT EXISTS sync_outbox (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    outlet_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    payload TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_attempt_at TEXT,
    error_message TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON sync_outbox (status, retry_count, created_at ASC);

-- Shift & Cash Drawer Sessions
CREATE TABLE IF NOT EXISTS shift_sessions (
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

CREATE INDEX IF NOT EXISTS idx_shift_active ON shift_sessions (tenant_id, outlet_id, cashier_id, status);

CREATE TABLE IF NOT EXISTS cash_movements (
    id TEXT PRIMARY KEY,
    shift_id TEXT NOT NULL REFERENCES shift_sessions(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    amount INTEGER NOT NULL,
    reason TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cash_movements_shift ON cash_movements (shift_id, created_at ASC);

