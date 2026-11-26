CREATE TABLE telegram_group_pending_inputs (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    telegram_user_id BIGINT NOT NULL,
    action VARCHAR(24) NOT NULL CHECK (action IN ('rule_text','rule_words','rule_patterns','announcement')),
    rule_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, group_id, telegram_user_id)
);
CREATE INDEX telegram_group_pending_inputs_expiry_idx
    ON telegram_group_pending_inputs(expires_at);
