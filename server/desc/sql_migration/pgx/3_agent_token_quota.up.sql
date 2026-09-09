CREATE TABLE "agent_token_quota" (
    id bigserial NOT NULL,
    scope varchar(16) NOT NULL,
    user_uuid varchar(36),
    enabled boolean NOT NULL DEFAULT true,
    five_hour_disabled boolean NOT NULL DEFAULT false,
    five_hour_limit_tokens bigint,
    seven_day_limit_tokens bigint,
    quota_reset_at timestamptz,
    create_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT fk_agent_token_quota_user
        FOREIGN KEY (user_uuid) REFERENCES "manage_user" (uuid) ON DELETE CASCADE,
    CONSTRAINT chk_agent_token_quota_scope
        CHECK ((scope = 'global' AND user_uuid IS NULL) OR (scope = 'user' AND user_uuid IS NOT NULL)),
    CONSTRAINT chk_agent_token_quota_five_hour
        CHECK (five_hour_limit_tokens IS NULL OR five_hour_limit_tokens > 0),
    CONSTRAINT chk_agent_token_quota_seven_day
        CHECK (seven_day_limit_tokens IS NULL OR seven_day_limit_tokens > 0),
    CONSTRAINT chk_agent_token_quota_global
        CHECK (scope <> 'global' OR (
            enabled = true
            AND five_hour_disabled = false
            AND five_hour_limit_tokens IS NOT NULL
            AND seven_day_limit_tokens IS NOT NULL
        ))
);

CREATE UNIQUE INDEX uniq_agent_token_quota_global
    ON "agent_token_quota" (scope)
    WHERE scope = 'global';

CREATE UNIQUE INDEX uniq_agent_token_quota_user
    ON "agent_token_quota" (user_uuid)
    WHERE scope = 'user';

INSERT INTO "agent_token_quota" (
    scope,
    enabled,
    five_hour_disabled,
    five_hour_limit_tokens,
    seven_day_limit_tokens
) VALUES ('global', true, false, 5000000, 50000000);

CREATE INDEX idx_conversation_token_usage_event_user_time
    ON "conversation_token_usage_event" (user_uuid, create_time DESC);

INSERT INTO "manage_menu" (
    uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name,
    hide_in_menu, active_menu, "order", route_name, route_path, component,
    icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab,
    query, permissions, constant, button_code
) VALUES (
    'dc10f1d2-8d73-4ff9-9702-4f7395b7a001',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    '1',
    'cc10f1d2-8d73-4ff9-9702-4f7395b7a001',
    '3',
    'Token 额度管理',
    0,
    '',
    0,
    '',
    '',
    '',
    '',
    '1',
    '',
    0,
    '',
    0,
    0,
    '[]',
    '[
      {"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"},
      {"code":"v1:manage:agent:saveTokenQuotaGlobal","desc":"保存全局 Token 额度"},
      {"code":"v1:manage:agent:getUserTokenQuota","desc":"读取用户 Token 额度"},
      {"code":"v1:manage:agent:saveUserTokenQuota","desc":"保存用户 Token 额度"},
      {"code":"v1:manage:agent:deleteUserTokenQuota","desc":"恢复用户全局额度"},
      {"code":"v1:manage:agent:resetUserTokenQuota","desc":"重置用户 Token 额度"}
    ]',
    0,
    'v1:manage:agent:getUserTokenQuota'
);

INSERT INTO "manage_role_menu" (
    uuid, create_time, update_time, role_uuid, menu_uuid, is_home
) VALUES (
    'dc20f1d2-8d73-4ff9-9702-4f7395b7a001',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    '1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d',
    'dc10f1d2-8d73-4ff9-9702-4f7395b7a001',
    0
);
