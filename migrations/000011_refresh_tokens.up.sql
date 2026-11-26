ALTER TABLE user_sessions ADD COLUMN family_id UUID;

CREATE TABLE user_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id UUID NOT NULL DEFAULT gen_random_uuid(),
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    replaced_by_hash BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_agent TEXT NOT NULL DEFAULT '',
    ip_address INET
);
CREATE INDEX user_refresh_tokens_user_id_idx ON user_refresh_tokens(user_id);
CREATE INDEX user_refresh_tokens_family_id_idx ON user_refresh_tokens(family_id);
CREATE INDEX user_refresh_tokens_expires_at_idx ON user_refresh_tokens(expires_at);
