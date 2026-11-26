DROP TABLE IF EXISTS telegram_identity_link_challenges;
ALTER TABLE users DROP COLUMN IF EXISTS telegram_last_name;
ALTER TABLE users DROP COLUMN IF EXISTS telegram_first_name;
ALTER TABLE users DROP COLUMN IF EXISTS telegram_username;
