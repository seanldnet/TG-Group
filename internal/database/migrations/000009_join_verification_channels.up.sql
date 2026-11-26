ALTER TABLE telegram_join_verification_challenges
    ADD COLUMN required_channels JSONB NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(required_channels)='array');
