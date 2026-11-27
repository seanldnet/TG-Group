CREATE TABLE agent_upgrade_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_no VARCHAR(40) NOT NULL UNIQUE DEFAULT ('AG' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 24))),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    from_level_id UUID REFERENCES agent_levels(id) ON DELETE RESTRICT,
    target_level_id UUID NOT NULL REFERENCES agent_levels(id) ON DELETE RESTRICT,
    amount_usdt BIGINT NOT NULL CHECK (amount_usdt >= 0),
    payment_method VARCHAR(16) NOT NULL CHECK (payment_method IN ('wallet', 'usdt', 'epusdt', 'gmpay')),
    status VARCHAR(16) NOT NULL CHECK (status IN ('pending', 'paid', 'failed', 'cancelled')),
    idempotency_key VARCHAR(120) NOT NULL,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, idempotency_key)
);
CREATE INDEX agent_upgrade_orders_user_created_idx ON agent_upgrade_orders(user_id, created_at DESC);
CREATE INDEX agent_upgrade_orders_status_idx ON agent_upgrade_orders(status, created_at);
