ALTER TABLE telegram_group_rules
    DROP CONSTRAINT telegram_group_rules_rule_type_check;
ALTER TABLE telegram_group_rules
    ADD CONSTRAINT telegram_group_rules_rule_type_check
    CHECK (rule_type IN ('keyword_filter','sensitive_word','anti_spam','anti_ad','welcome','leave_notice','join_verification'));

CREATE TABLE telegram_join_verification_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bot_id UUID NOT NULL REFERENCES telegram_bots(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    telegram_user_id BIGINT NOT NULL,
    answer SMALLINT NOT NULL CHECK (answer BETWEEN 2 AND 18),
    status VARCHAR(16) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','verifying','verified','expired','failed')),
    failure_action VARCHAR(16) NOT NULL DEFAULT 'restrict'
        CHECK (failure_action IN ('restrict','kick')),
    challenge_message_id BIGINT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (group_id, telegram_user_id)
);
CREATE INDEX telegram_join_verification_pending_idx
    ON telegram_join_verification_challenges(expires_at)
    WHERE status='pending';
