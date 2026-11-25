CREATE TABLE telegram_group_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    rule_type VARCHAR(32) NOT NULL CHECK (rule_type IN ('keyword_filter','sensitive_word','anti_spam','anti_ad','welcome','leave_notice')),
    name VARCHAR(80) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    priority INTEGER NOT NULL DEFAULT 100 CHECK (priority BETWEEN 0 AND 1000),
    config JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(config) = 'object'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, group_id, rule_type, name)
);
CREATE INDEX telegram_group_rules_execution_idx
    ON telegram_group_rules(tenant_id, group_id, enabled, priority, created_at);
