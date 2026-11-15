package database

const EdgeSchemaSQL = `
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
`

const CloudSchemaSQL = `
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

CREATE TABLE IF NOT EXISTS payments (
	id VARCHAR(36) PRIMARY KEY,
	order_id VARCHAR(36) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	method VARCHAR(32) NOT NULL,
	amount BIGINT NOT NULL,
	reference VARCHAR(128),
	paid_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

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
`
