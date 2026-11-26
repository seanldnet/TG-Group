CREATE TABLE telegram_group_warning_escalations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    group_id UUID NOT NULL,
    rule_id UUID NOT NULL,
    telegram_user_id BIGINT NOT NULL,
    warning_count INTEGER NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('mute','ban')),
    duration_minutes INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'processing' CHECK (status IN ('processing','succeeded','failed','skipped')),
    detail TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id,group_id,rule_id,telegram_user_id,warning_count),
    FOREIGN KEY (tenant_id,group_id) REFERENCES telegram_groups(tenant_id,id) ON DELETE CASCADE
);
CREATE INDEX telegram_group_warning_escalations_recent_idx
    ON telegram_group_warning_escalations(tenant_id,group_id,created_at DESC);