CREATE TABLE plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(40) NOT NULL UNIQUE,
    name VARCHAR(80) NOT NULL,
    description VARCHAR(2000) NOT NULL DEFAULT '',
    price_usdt BIGINT NOT NULL CHECK (price_usdt >= 0),
    duration_days INTEGER NOT NULL CHECK (duration_days BETWEEN 1 AND 36500),
    bot_limit INTEGER NOT NULL DEFAULT 0 CHECK (bot_limit >= 0),
    group_limit INTEGER NOT NULL DEFAULT 0 CHECK (group_limit >= 0),
    features JSONB NOT NULL DEFAULT '{}'::jsonb,
    sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX plans_catalog_idx ON plans(enabled,sort_order,price_usdt,id);
