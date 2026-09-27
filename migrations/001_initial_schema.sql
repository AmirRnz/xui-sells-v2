-- +goose Up
-- SQL migration for xui-sells-v2 core schema

CREATE TABLE IF NOT EXISTS instances (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL, -- 'customer', 'reseller', 'child_customer'
    parent_instance_id VARCHAR(64) REFERENCES instances(id) ON DELETE CASCADE,
    bot_token TEXT NOT NULL UNIQUE,
    bot_username VARCHAR(128) NOT NULL DEFAULT '',
    admin_tg_id BIGINT NOT NULL,
    panel_url TEXT NOT NULL,
    panel_api_key TEXT NOT NULL,
    default_lang VARCHAR(8) NOT NULL DEFAULT 'fa',
    web_panel_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    group_name VARCHAR(128) NOT NULL DEFAULT 'default',
    card_number VARCHAR(64) NOT NULL DEFAULT '',
    cardholder_name VARCHAR(128) NOT NULL DEFAULT '',
    min_topup BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(16) NOT NULL DEFAULT 'TOMAN',
    support_tg_id VARCHAR(128) NOT NULL DEFAULT '',
    referral_percentage INT NOT NULL DEFAULT 0,
    refund_window_days INT NOT NULL DEFAULT 1,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reseller_profiles (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    user_tg_id BIGINT NOT NULL,
    service_name VARCHAR(128) NOT NULL,
    membership_tier VARCHAR(32) NOT NULL DEFAULT 'free',
    membership_expires_at TIMESTAMPTZ,
    web_panel_password_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(instance_id, user_tg_id)
);

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    tg_id BIGINT NOT NULL,
    username VARCHAR(128) NOT NULL DEFAULT '',
    first_name VARCHAR(128) NOT NULL DEFAULT '',
    last_name VARCHAR(128) NOT NULL DEFAULT '',
    language VARCHAR(8) NOT NULL DEFAULT 'fa',
    wallet_balance BIGINT NOT NULL DEFAULT 0,
    referrer_tg_id BIGINT,
    is_banned BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(instance_id, tg_id)
);

CREATE TABLE IF NOT EXISTS plans (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    usage_notes TEXT NOT NULL DEFAULT '',
    inbound_ids JSONB NOT NULL DEFAULT '[]',
    base_price BIGINT NOT NULL DEFAULT 0,
    extra_user_price BIGINT NOT NULL DEFAULT 0,
    base_users INT NOT NULL DEFAULT 1,
    max_users INT NOT NULL DEFAULT 10,
    durations JSONB NOT NULL DEFAULT '[1]',
    discounts JSONB NOT NULL DEFAULT '{}',
    traffic_bytes BIGINT NOT NULL DEFAULT 0,
    trial_traffic_bytes BIGINT NOT NULL DEFAULT 0,
    trial_validity_seconds INT NOT NULL DEFAULT 0,
    trial_cooldown_seconds INT NOT NULL DEFAULT 0,
    trial_max_per_window INT NOT NULL DEFAULT 0,
    trial_window_seconds INT NOT NULL DEFAULT 0,
    tier_trial_configs JSONB NOT NULL DEFAULT '{}',
    min_reseller_tier VARCHAR(32) NOT NULL DEFAULT 'free',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS services (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    user_tg_id BIGINT NOT NULL,
    plan_id VARCHAR(64) NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    client_email VARCHAR(128) NOT NULL,
    client_uuid VARCHAR(64) NOT NULL,
    sub_id VARCHAR(64) NOT NULL,
    group_name VARCHAR(128) NOT NULL DEFAULT '',
    inbound_ids JSONB NOT NULL DEFAULT '[]',
    user_count INT NOT NULL DEFAULT 1,
    total_bytes BIGINT NOT NULL DEFAULT 0,
    expiry_time_ms BIGINT NOT NULL DEFAULT 0,
    subscription_url TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    user_tg_id BIGINT NOT NULL,
    order_type VARCHAR(32) NOT NULL,
    plan_id VARCHAR(64) REFERENCES plans(id) ON DELETE SET NULL,
    service_id VARCHAR(64) REFERENCES services(id) ON DELETE SET NULL,
    user_count INT NOT NULL DEFAULT 1,
    duration_months INT NOT NULL DEFAULT 1,
    amount BIGINT NOT NULL,
    currency VARCHAR(16) NOT NULL DEFAULT 'TOMAN',
    payment_method VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_approval',
    receipt_text TEXT,
    receipt_media_path TEXT,
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reseller_funding_reservations (
    id VARCHAR(64) PRIMARY KEY,
    parent_instance_id VARCHAR(64) NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    reseller_user_tg_id BIGINT NOT NULL,
    child_instance_id VARCHAR(64) NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    order_id VARCHAR(64) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'reserved',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallet_transactions (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    user_tg_id BIGINT NOT NULL,
    order_id VARCHAR(64) REFERENCES orders(id) ON DELETE SET NULL,
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    tx_type VARCHAR(32) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS trial_records (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    user_tg_id BIGINT NOT NULL,
    plan_id VARCHAR(64) NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    tier_variant VARCHAR(32) NOT NULL DEFAULT 'free',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS refund_requests (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    user_tg_id BIGINT NOT NULL,
    service_id VARCHAR(64) NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    order_id VARCHAR(64) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    reason TEXT NOT NULL DEFAULT '',
    photo_path TEXT,
    amount BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    admin_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tickets (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    user_tg_id BIGINT NOT NULL,
    subject VARCHAR(256) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ticket_messages (
    id VARCHAR(64) PRIMARY KEY,
    ticket_id VARCHAR(64) NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    sender_type VARCHAR(32) NOT NULL,
    message_text TEXT NOT NULL,
    media_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(64) PRIMARY KEY,
    user_tg_id BIGINT NOT NULL,
    instance_id VARCHAR(64) NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    token_hash VARCHAR(128) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_instance_tg ON users(instance_id, tg_id);
CREATE INDEX IF NOT EXISTS idx_services_instance_tg ON services(instance_id, user_tg_id);
CREATE INDEX IF NOT EXISTS idx_services_client_email ON services(client_email);
CREATE INDEX IF NOT EXISTS idx_orders_instance_status ON orders(instance_id, status);
CREATE INDEX IF NOT EXISTS idx_trial_records_lookup ON trial_records(instance_id, user_tg_id, plan_id, created_at);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_user ON wallet_transactions(instance_id, user_tg_id, created_at);
CREATE INDEX IF NOT EXISTS idx_refund_instance_status ON refund_requests(instance_id, status);
CREATE INDEX IF NOT EXISTS idx_tickets_instance_user ON tickets(instance_id, user_tg_id);

-- +goose Down
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS ticket_messages;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS refund_requests;
DROP TABLE IF EXISTS trial_records;
DROP TABLE IF EXISTS wallet_transactions;
DROP TABLE IF EXISTS reseller_funding_reservations;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS plans;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS reseller_profiles;
DROP TABLE IF EXISTS instances;
