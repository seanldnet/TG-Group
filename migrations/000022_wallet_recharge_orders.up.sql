-- Recharge orders and encrypted EPUSDT merchant channel configuration.
-- Wallet credit remains integer USDT; fractional settled amounts are rejected.
CREATE TABLE payment_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(24) NOT NULL UNIQUE CHECK (code IN ('epusdt_gmpay','epusdt_epay')),
    name VARCHAR(80) NOT NULL,
    base_url TEXT NOT NULL,
    merchant_pid VARCHAR(64) NOT NULL,
    secret_ciphertext BYTEA NOT NULL,
    secret_nonce BYTEA NOT NULL,
    currency VARCHAR(8) NOT NULL DEFAULT 'usd',
    token VARCHAR(24) NOT NULL DEFAULT 'usdt',
    network VARCHAR(32) NOT NULL DEFAULT 'tron',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (base_url LIKE 'https://%'),
    CHECK (currency = lower(currency)),
    CHECK (token = lower(token)),
    CHECK (network = lower(network))
);

CREATE TABLE wallet_recharge_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_no VARCHAR(40) NOT NULL UNIQUE DEFAULT ('RC' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 24))),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    channel_id UUID NOT NULL REFERENCES payment_channels(id) ON DELETE RESTRICT,
    channel_code VARCHAR(24) NOT NULL,
    merchant_order_id VARCHAR(32) NOT NULL UNIQUE,
    provider_trade_id VARCHAR(64),
    amount_usdt BIGINT NOT NULL CHECK (amount_usdt > 0),
    actual_amount NUMERIC(36,12),
    actual_token VARCHAR(24),
    actual_network VARCHAR(32),
    receive_address TEXT,
    payment_url TEXT,
    provider_status SMALLINT,
    block_transaction_id VARCHAR(160),
    status VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','paid','expired','failed','review')),
    expires_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (channel_code, provider_trade_id),
    UNIQUE (channel_code, block_transaction_id)
);
CREATE INDEX wallet_recharge_orders_user_created_idx ON wallet_recharge_orders(user_id, created_at DESC, id DESC);
CREATE INDEX wallet_recharge_orders_status_created_idx ON wallet_recharge_orders(status, created_at);
CREATE INDEX wallet_recharge_orders_provider_trade_idx ON wallet_recharge_orders(channel_code, provider_trade_id);

CREATE TABLE payment_callback_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    channel_code VARCHAR(24) NOT NULL,
    provider_trade_id VARCHAR(64) NOT NULL,
    merchant_order_id VARCHAR(32) NOT NULL,
    payload_hash BYTEA NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (channel_code, provider_trade_id, payload_hash)
);
