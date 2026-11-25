CREATE TABLE telegram_group_members (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bot_id UUID NOT NULL REFERENCES telegram_bots(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    telegram_user_id BIGINT NOT NULL,
    username VARCHAR(64) NOT NULL DEFAULT '',
    first_name VARCHAR(255) NOT NULL DEFAULT '',
    last_name VARCHAR(255) NOT NULL DEFAULT '',
    is_bot BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(20) NOT NULL CHECK (status IN ('member','administrator','creator','restricted','left','kicked')),
    is_anonymous BOOLEAN NOT NULL DEFAULT false,
    can_manage_chat BOOLEAN NOT NULL DEFAULT false,
    can_delete_messages BOOLEAN NOT NULL DEFAULT false,
    can_restrict_members BOOLEAN NOT NULL DEFAULT false,
    can_invite_users BOOLEAN NOT NULL DEFAULT false,
    can_pin_messages BOOLEAN NOT NULL DEFAULT false,
    can_manage_topics BOOLEAN NOT NULL DEFAULT false,
    joined_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (group_id, telegram_user_id)
);
CREATE INDEX telegram_group_members_tenant_group_idx
    ON telegram_group_members(tenant_id, group_id, status, last_seen_at DESC);
CREATE INDEX telegram_group_members_user_idx
    ON telegram_group_members(tenant_id, telegram_user_id);
