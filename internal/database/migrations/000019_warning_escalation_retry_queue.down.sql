DROP INDEX IF EXISTS telegram_group_warning_escalations_retry_idx;
ALTER TABLE telegram_group_warning_escalations
    DROP COLUMN IF EXISTS processing_started_at,
    DROP COLUMN IF EXISTS next_retry_at,
    DROP COLUMN IF EXISTS attempt_count;
