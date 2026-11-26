ALTER TABLE telegram_join_verification_challenges
    ADD COLUMN challenge_type VARCHAR(24) NOT NULL DEFAULT 'math'
        CHECK (challenge_type IN ('math','letters','alphanumeric')),
    ADD COLUMN answer_text VARCHAR(16) NOT NULL DEFAULT '',
    ADD COLUMN challenge_options JSONB NOT NULL DEFAULT '[]'::jsonb
        CHECK (jsonb_typeof(challenge_options)='array');

UPDATE telegram_join_verification_challenges
SET answer_text=answer::text,
    challenge_options=jsonb_build_array(answer::text,(answer+1)::text,(answer-1)::text)
WHERE answer_text='';

