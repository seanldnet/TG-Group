ALTER TABLE telegram_group_warning_escalations
    ADD COLUMN attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    ADD COLUMN next_retry_at TIMESTAMPTZ,
    ADD COLUMN processing_started_at TIMESTAMPTZ;

UPDATE telegram_group_warning_escalations
SET next_retry_at = now()
WHERE status = 'failed';

UPDATE telegram_group_warning_escalations
SET processing_started_at = updated_at
WHERE status = 'processing';

CREATE INDEX telegram_group_warning_escalations_retry_idx
    ON telegram_group_warning_escalations(next_retry_at, id)
    WHERE status IN ('failed', 'processing');
