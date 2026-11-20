CREATE TABLE IF NOT EXISTS auth_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS auth_sessions_user_idx ON auth_sessions (user_id);
CREATE INDEX IF NOT EXISTS auth_sessions_expiry_idx ON auth_sessions (expires_at);
