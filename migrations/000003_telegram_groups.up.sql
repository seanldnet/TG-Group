ALTER TABLE telegram_bots
    ADD COLUMN next_update_offset BIGINT NOT NULL DEFAULT 0;

CREATE TABLE telegram_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bot_id UUID NOT NULL REFERENCES telegram_bots(id) ON DELETE CASCADE,
    telegram_chat_id BIGINT NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    username VARCHAR(64) NOT NULL DEFAULT '',
    chat_type VARCHAR(20) NOT NULL CHECK (chat_type IN ('group', 'supergroup', 'channel')),
    is_forum BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'left', 'kicked')),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(bot_id, telegram_chat_id)
);
CREATE INDEX telegram_groups_tenant_seen_idx ON telegram_groups(tenant_id, last_seen_at DESC);
CREATE INDEX telegram_groups_bot_idx ON telegram_groups(bot_id);
