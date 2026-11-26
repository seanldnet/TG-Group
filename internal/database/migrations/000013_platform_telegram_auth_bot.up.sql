CREATE TABLE platform_telegram_auth_bot_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    token_ciphertext BYTEA NOT NULL,
    token_nonce BYTEA NOT NULL,
    username VARCHAR(32) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
