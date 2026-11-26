CREATE TABLE telegram_group_member_warnings (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    telegram_user_id BIGINT NOT NULL CHECK (telegram_user_id > 0),
    actor_telegram_user_id BIGINT CHECK (actor_telegram_user_id IS NULL OR actor_telegram_user_id > 0),
    reason VARCHAR(500) NOT NULL,
    source VARCHAR(24) NOT NULL DEFAULT 'manual' CHECK (source IN ('manual','automatic')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX telegram_group_member_warnings_lookup_idx
    ON telegram_group_member_warnings(tenant_id, group_id, telegram_user_id, created_at DESC);

CREATE TABLE telegram_group_member_lists (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    telegram_user_id BIGINT NOT NULL CHECK (telegram_user_id > 0),
    list_type VARCHAR(16) NOT NULL CHECK (list_type IN ('blacklist','whitelist','exempt')),
    reason VARCHAR(500) NOT NULL DEFAULT '',
    actor_telegram_user_id BIGINT NOT NULL CHECK (actor_telegram_user_id > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, group_id, telegram_user_id, list_type)
);
CREATE INDEX telegram_group_member_lists_lookup_idx
    ON telegram_group_member_lists(tenant_id, group_id, list_type, telegram_user_id);
