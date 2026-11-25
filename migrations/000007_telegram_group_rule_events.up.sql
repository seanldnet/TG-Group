CREATE TABLE telegram_group_rule_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    rule_id UUID REFERENCES telegram_group_rules(id) ON DELETE SET NULL,
    telegram_update_id BIGINT NOT NULL,
    telegram_message_id BIGINT,
    telegram_user_id BIGINT,
    action VARCHAR(32) NOT NULL,
    status VARCHAR(24) NOT NULL CHECK (status IN ('succeeded','skipped','failed')),
    detail VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (group_id, telegram_update_id, rule_id, action, telegram_user_id)
);
CREATE INDEX telegram_group_rule_events_group_created_idx
    ON telegram_group_rule_events(tenant_id, group_id, created_at DESC);
