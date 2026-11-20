CREATE UNIQUE INDEX IF NOT EXISTS telegram_bots_username_unique
ON telegram_bots (LOWER(username))
WHERE username IS NOT NULL;
