-- Mirror of internal/database/migrations/000020_agent_wallet_foundation.up.sql.
CREATE TABLE agent_levels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(40) NOT NULL UNIQUE,
    name VARCHAR(80) NOT NULL,
    price_usdt BIGINT NOT NULL DEFAULT 0 CHECK (price_usdt >= 0),
    sort_order INTEGER NOT NULL DEFAULT 0,
    benefits JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(benefits) = 'object'),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE agent_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE RESTRICT,
    level_id UUID NOT NULL REFERENCES agent_levels(id) ON DELETE RESTRICT,
    parent_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
    activated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (parent_user_id IS NULL OR parent_user_id <> user_id)
);
CREATE INDEX agent_profiles_parent_idx ON agent_profiles(parent_user_id);
CREATE INDEX agent_profiles_level_idx ON agent_profiles(level_id);
CREATE TABLE wallet_accounts (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE RESTRICT,
    currency VARCHAR(8) NOT NULL DEFAULT 'USDT' CHECK (currency = 'USDT'),
    balance_usdt BIGINT NOT NULL DEFAULT 0 CHECK (balance_usdt >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE wallet_ledger (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    entry_type VARCHAR(32) NOT NULL CHECK (entry_type IN ('deposit', 'agent_upgrade', 'order_payment', 'refund', 'commission', 'adjustment')),
    amount_usdt BIGINT NOT NULL CHECK (amount_usdt <> 0),
    balance_after_usdt BIGINT NOT NULL CHECK (balance_after_usdt >= 0),
    reference_type VARCHAR(40) NOT NULL DEFAULT '',
    reference_id VARCHAR(120) NOT NULL DEFAULT '',
    idempotency_key VARCHAR(120) NOT NULL UNIQUE,
    description VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX wallet_ledger_user_created_idx ON wallet_ledger(user_id, created_at DESC, id DESC);
INSERT INTO agent_levels(code,name,price_usdt,sort_order,benefits)
VALUES ('base','基础代理',0,0,'{"description":"基础代理等级"}'::jsonb) ON CONFLICT(code) DO NOTHING;
INSERT INTO agent_profiles(user_id,level_id)
SELECT u.id,l.id FROM users u CROSS JOIN agent_levels l
WHERE u.role='agent' AND l.code='base' ON CONFLICT(user_id) DO NOTHING;
INSERT INTO wallet_accounts(user_id)
SELECT id FROM users ON CONFLICT(user_id) DO NOTHING;
