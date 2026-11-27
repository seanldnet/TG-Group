CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_no VARCHAR(40) NOT NULL UNIQUE DEFAULT ('OR' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 24))),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE RESTRICT,
    plan_name VARCHAR(80) NOT NULL,
    amount_usdt BIGINT NOT NULL CHECK (amount_usdt >= 0),
    payment_method VARCHAR(16) NOT NULL CHECK (payment_method IN ('wallet')),
    status VARCHAR(16) NOT NULL CHECK (status IN ('pending', 'paid', 'failed', 'cancelled')),
    idempotency_key VARCHAR(120) NOT NULL,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, idempotency_key)
);
CREATE INDEX orders_user_created_idx ON orders(user_id,created_at DESC,id DESC);
CREATE INDEX orders_status_created_idx ON orders(status,created_at DESC);

CREATE TABLE licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE RESTRICT,
    last_order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    plan_name VARCHAR(80) NOT NULL,
    status VARCHAR(16) NOT NULL CHECK (status IN ('active','expired','suspended')),
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    bot_limit INTEGER NOT NULL CHECK (bot_limit >= 0),
    group_limit INTEGER NOT NULL CHECK (group_limit >= 0),
    features JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id,plan_id),
    CHECK (expires_at > starts_at)
);
CREATE INDEX licenses_user_status_idx ON licenses(user_id,status,expires_at DESC);
CREATE INDEX licenses_expiry_idx ON licenses(status,expires_at);
