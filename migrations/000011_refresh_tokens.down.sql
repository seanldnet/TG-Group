DROP TABLE IF EXISTS user_refresh_tokens;
ALTER TABLE user_sessions DROP COLUMN IF EXISTS family_id;
