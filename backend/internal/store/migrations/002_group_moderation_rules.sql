CREATE TABLE IF NOT EXISTS group_moderation_rules (
    group_id BIGINT PRIMARY KEY REFERENCES telegram_groups(id) ON DELETE CASCADE,
    block_links BOOLEAN NOT NULL DEFAULT FALSE,
    blocked_keywords JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

