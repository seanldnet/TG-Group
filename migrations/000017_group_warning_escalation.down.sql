ALTER TABLE telegram_group_rules
    DROP CONSTRAINT telegram_group_rules_rule_type_check;
ALTER TABLE telegram_group_rules
    ADD CONSTRAINT telegram_group_rules_rule_type_check
    CHECK (rule_type IN ('keyword_filter','sensitive_word','anti_spam','anti_ad','welcome','leave_notice','join_verification','qa'));
