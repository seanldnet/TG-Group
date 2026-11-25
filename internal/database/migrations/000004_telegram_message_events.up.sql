CREATE TABLE telegram_message_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bot_id UUID NOT NULL REFERENCES telegram_bots(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    telegram_update_id BIGINT NOT NULL,
    telegram_message_id BIGINT NOT NULL,
    telegram_user_id BIGINT,
    sender_username VARCHAR(64) NOT NULL DEFAULT '',
    sender_display_name VARCHAR(255) NOT NULL DEFAULT '',
    message_type VARCHAR(32) NOT NULL,
    message_text TEXT NOT NULL DEFAULT '',
    telegram_sent_at TIMESTAMPTZ,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(bot_id, telegram_update_id)
);
CREATE INDEX telegram_message_events_group_received_idx
    ON telegram_message_events(tenant_id, group_id, received_at DESC);
CREATE INDEX telegram_message_events_sender_idx
    ON telegram_message_events(tenant_id, telegram_user_id, received_at DESC);
