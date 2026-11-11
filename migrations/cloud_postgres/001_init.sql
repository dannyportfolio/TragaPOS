-- Cloud Central PostgreSQL Schema

CREATE TABLE IF NOT EXISTS items (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    outlet_id VARCHAR(64) NOT NULL,
    sku VARCHAR(64) NOT NULL,
    barcode VARCHAR(64),
    name VARCHAR(255) NOT NULL,
    sector VARCHAR(32) NOT NULL,
    base_price BIGINT NOT NULL,
    currency VARCHAR(8) NOT NULL DEFAULT 'IDR',
    tax_rate NUMERIC(6, 4) NOT NULL DEFAULT 0.0000,
    sector_traits JSONB NOT NULL DEFAULT '{}'::jsonb,
    version BIGINT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_tenant_sku UNIQUE (tenant_id, sku)
);

CREATE INDEX IF NOT EXISTS idx_items_tenant_barcode ON items (tenant_id, barcode);
CREATE INDEX IF NOT EXISTS idx_items_version ON items (tenant_id, version);

CREATE TABLE IF NOT EXISTS inventory_balances (
    tenant_id VARCHAR(64) NOT NULL,
    outlet_id VARCHAR(64) NOT NULL,
    sku VARCHAR(64) NOT NULL,
    on_hand_qty NUMERIC(14, 4) NOT NULL DEFAULT 0.0000,
    allocated_qty NUMERIC(14, 4) NOT NULL DEFAULT 0.0000,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, outlet_id, sku)
);

CREATE TABLE IF NOT EXISTS stock_movements (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    outlet_id VARCHAR(64) NOT NULL,
    sku VARCHAR(64) NOT NULL,
    delta_qty NUMERIC(14, 4) NOT NULL,
    reason VARCHAR(64) NOT NULL,
    reference_id VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_stock_movements_sku ON stock_movements (tenant_id, outlet_id, sku);

CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    outlet_id VARCHAR(64) NOT NULL,
    order_number VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    subtotal BIGINT NOT NULL DEFAULT 0,
    tax_amount BIGINT NOT NULL DEFAULT 0,
    discount_amount BIGINT NOT NULL DEFAULT 0,
    total_amount BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(8) NOT NULL DEFAULT 'IDR',
    notes TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_orders_tenant_outlet ON orders (tenant_id, outlet_id, created_at DESC);

CREATE TABLE IF NOT EXISTS order_items (
    id VARCHAR(36) PRIMARY KEY,
    order_id VARCHAR(36) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    item_id VARCHAR(36) NOT NULL,
    sku VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    sector VARCHAR(32) NOT NULL,
    quantity NUMERIC(14, 4) NOT NULL DEFAULT 1.0000,
    uom VARCHAR(32) NOT NULL DEFAULT 'PCS',
    unit_price BIGINT NOT NULL,
    subtotal BIGINT NOT NULL,
    tax_amount BIGINT NOT NULL DEFAULT 0,
    total_amount BIGINT NOT NULL,
    selected_modifiers JSONB NOT NULL DEFAULT '[]'::jsonb,
    sector_payload JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items (order_id);

CREATE TABLE IF NOT EXISTS payments (
    id VARCHAR(36) PRIMARY KEY,
    order_id VARCHAR(36) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    method VARCHAR(32) NOT NULL,
    amount BIGINT NOT NULL,
    reference VARCHAR(128),
    paid_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_payments_order ON payments (order_id);

CREATE TABLE IF NOT EXISTS guest_folios (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    outlet_id VARCHAR(64) NOT NULL,
    room_number VARCHAR(32) NOT NULL,
    guest_name VARCHAR(128) NOT NULL,
    credit_limit BIGINT NOT NULL DEFAULT 0,
    current_total BIGINT NOT NULL DEFAULT 0,
    is_open BOOLEAN NOT NULL DEFAULT TRUE,
    check_in_at TIMESTAMP WITH TIME ZONE NOT NULL,
    check_out_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_folios_room ON guest_folios (tenant_id, room_number, is_open);

CREATE TABLE IF NOT EXISTS folio_charges (
    id VARCHAR(36) PRIMARY KEY,
    folio_id VARCHAR(36) NOT NULL REFERENCES guest_folios(id) ON DELETE CASCADE,
    order_id VARCHAR(36) NOT NULL,
    amount BIGINT NOT NULL,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS issued_tickets (
    ticket_id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    category VARCHAR(64) NOT NULL,
    pax INT NOT NULL DEFAULT 1,
    valid_until BIGINT NOT NULL,
    issued_at BIGINT NOT NULL,
    order_id VARCHAR(36),
    signature TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS redeemed_tickets (
    id VARCHAR(36) PRIMARY KEY,
    ticket_id VARCHAR(36) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    outlet_id VARCHAR(64) NOT NULL,
    gate_zone VARCHAR(64) NOT NULL,
    pax_redeemed INT NOT NULL DEFAULT 1,
    redeemed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    device_id VARCHAR(64),
    CONSTRAINT uq_ticket_redemption UNIQUE (tenant_id, ticket_id)
);
