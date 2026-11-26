CREATE TABLE telegram_identity_link_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX telegram_identity_link_challenges_user_created_idx
    ON telegram_identity_link_challenges(user_id, created_at DESC);
CREATE INDEX telegram_identity_link_challenges_expiry_idx
    ON telegram_identity_link_challenges(expires_at)
    WHERE consumed_at IS NULL;

ALTER TABLE users ADD COLUMN telegram_username VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN telegram_first_name VARCHAR(120) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN telegram_last_name VARCHAR(120) NOT NULL DEFAULT '';
