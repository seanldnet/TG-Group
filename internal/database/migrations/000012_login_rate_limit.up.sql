CREATE TABLE auth_login_failures (
    id BIGSERIAL PRIMARY KEY,
    identifier_type TEXT NOT NULL CHECK (identifier_type IN ('email', 'ip')),
    identifier_hash BYTEA NOT NULL,
    failed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX auth_login_failures_lookup_idx
    ON auth_login_failures(identifier_type, identifier_hash, failed_at DESC);
CREATE INDEX auth_login_failures_failed_at_idx
    ON auth_login_failures(failed_at);
