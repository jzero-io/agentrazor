-- Agent conversations
DROP TABLE IF EXISTS "conversation_group";

CREATE TABLE "conversation_group" (
    uuid varchar(64) NOT NULL,
    user_uuid varchar(64) NOT NULL,
    name varchar(80) NOT NULL,
    create_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (uuid)
);

CREATE INDEX idx_conversation_group_user ON "conversation_group" (user_uuid);
CREATE UNIQUE INDEX uk_conversation_group_user_name ON "conversation_group" (user_uuid, name);

DROP TABLE IF EXISTS "conversation";

CREATE TABLE "conversation" (
    id varchar(128) NOT NULL,
    user_uuid varchar(64) NOT NULL,
    group_uuid varchar(64),
    create_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE INDEX idx_conversation_user ON "conversation" (user_uuid);
CREATE INDEX idx_conversation_group ON "conversation" (group_uuid);

-- Agent token usage
DROP TABLE IF EXISTS "conversation_token_usage_event";

CREATE TABLE "conversation_token_usage_event" (
    id bigserial NOT NULL,
    conversation_id varchar(128) NOT NULL,
    user_uuid varchar(64) NOT NULL,
    turn_id varchar(128) NOT NULL,
    last_input_tokens bigint NOT NULL DEFAULT 0,
    last_cached_input_tokens bigint NOT NULL DEFAULT 0,
    last_cache_write_input_tokens bigint NOT NULL DEFAULT 0,
    last_output_tokens bigint NOT NULL DEFAULT 0,
    last_reasoning_output_tokens bigint NOT NULL DEFAULT 0,
    last_total_tokens bigint NOT NULL DEFAULT 0,
    total_input_tokens bigint NOT NULL DEFAULT 0,
    total_cached_input_tokens bigint NOT NULL DEFAULT 0,
    total_cache_write_input_tokens bigint NOT NULL DEFAULT 0,
    total_output_tokens bigint NOT NULL DEFAULT 0,
    total_reasoning_output_tokens bigint NOT NULL DEFAULT 0,
    total_tokens bigint NOT NULL DEFAULT 0,
    model_context_window bigint,
    create_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE INDEX idx_conversation_token_usage_event_conversation
    ON "conversation_token_usage_event" (conversation_id, id DESC);
CREATE INDEX idx_conversation_token_usage_event_user
    ON "conversation_token_usage_event" (user_uuid, id DESC);
CREATE INDEX idx_conversation_token_usage_event_turn
    ON "conversation_token_usage_event" (conversation_id, turn_id, id DESC);
CREATE INDEX idx_conversation_token_usage_event_user_time
    ON "conversation_token_usage_event" (user_uuid, create_time DESC);

-- Agent token quota
DROP TABLE IF EXISTS "agent_token_quota";

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

-- Agent API keys
DROP TABLE IF EXISTS "agent_api_key";

CREATE TABLE "agent_api_key" (
    uuid varchar(36) NOT NULL,
    user_uuid varchar(36) NOT NULL,
    key_hash char(64) NOT NULL UNIQUE,
    key_hint varchar(32) NOT NULL,
    create_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (uuid),
    CONSTRAINT fk_agent_api_key_user
        FOREIGN KEY (user_uuid) REFERENCES "manage_user" (uuid) ON DELETE CASCADE
);

CREATE INDEX idx_agent_api_key_user ON "agent_api_key" (user_uuid);

-- Agent menus are seeded directly in their final form. Button permissions stay
-- independent so custom roles can grant each mutating operation explicitly.
INSERT INTO "manage_menu" (
    uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name,
    hide_in_menu, active_menu, "order", route_name, route_path, component,
    icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab,
    query, permissions, constant, button_code
) VALUES
    ('aa10f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','','1','Agent 管理',0,'',6,'agent','/agent','layout.base','carbon:bot','1','route.agent',0,'',0,0,'[]','[]',0,''),
    ('aa10f1d2-8d73-4ff9-9702-4f7395b7a003',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a001','2','配置管理',0,'',1,'agent_config','/agent/config','view.agent_config','carbon:settings-services','1','route.agent_config',0,'',0,0,'[]','[{"code":"v1:manage:agent:getSettings","desc":"读取模型配置"},{"code":"v1:manage:agent:getAccountStatus","desc":"读取模型账号状态"}]',0,''),
    ('aa10f1d2-8d73-4ff9-9702-4f7395b7a002',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a001','2','Skills 管理',0,'',2,'agent_skills','/agent/skills','view.agent_skills','carbon:skill-level-basic','1','route.agent_skills',0,'',0,0,'[]','[{"code":"v1:manage:agent:listSkills","desc":"Skills 列表"},{"code":"v1:manage:agent:skillDetail","desc":"Skill 详情"}]',0,''),
    ('cc10f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a001','2','Token 消耗',0,'',3,'agent_token-usage','/agent/token-usage','view.agent_token-usage','carbon:meter','1','route.agent_token-usage',0,'',0,0,'[]','[{"code":"v1:agent:token:usageTrend","desc":"Token 消耗趋势"},{"code":"v1:agent:token:usageDetails","desc":"Token 消耗账号明细"},{"code":"v1:agent:token:usageConversations","desc":"Token 消耗对话明细"}]',0,''),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a003','3','保存模型配置',0,'',0,'','','','','1','button.agent.config.save',0,'',0,0,'[]','[{"code":"v1:manage:agent:saveSettings","desc":"保存模型、Provider Key 和默认系统提示词"}]',0,'v1:manage:agent:saveSettings'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a002',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a003','3','ChatGPT 登录',0,'',1,'','','','','1','button.agent.config.chatgptLogin',0,'',0,0,'[]','[{"code":"v1:manage:agent:startChatGPTLogin","desc":"ChatGPT 登录"},{"code":"v1:manage:agent:saveSettings","desc":"保存并启用模型配置"}]',0,'v1:manage:agent:startChatGPTLogin'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a003',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a003','3','API Key 登录',0,'',2,'','','','','1','button.agent.config.apiKeyLogin',0,'',0,0,'[]','[{"code":"v1:manage:agent:loginApiKey","desc":"API Key 登录"},{"code":"v1:manage:agent:saveSettings","desc":"保存并启用模型配置"}]',0,'v1:manage:agent:loginApiKey'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a004',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a003','3','退出模型账号',0,'',3,'','','','','1','button.agent.config.logout',0,'',0,0,'[]','[{"code":"v1:manage:agent:logout","desc":"退出模型账号"}]',0,'v1:manage:agent:logout'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a002','3','上传 Skill',0,'',0,'','','','','1','button.agent.skills.upload',0,'',0,0,'[]','[{"code":"v1:manage:agent:uploadSkill","desc":"上传 Skill"}]',0,'v1:manage:agent:uploadSkill'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a002',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a002','3','编辑 Skill',0,'',1,'','','','','1','button.agent.skills.edit',0,'',0,0,'[]','[{"code":"v1:manage:agent:updateSkillFile","desc":"更新 Skill 文件"}]',0,'v1:manage:agent:updateSkillFile'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a003',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a002','3','删除 Skill',0,'',2,'','','','','1','button.agent.skills.delete',0,'',0,0,'[]','[{"code":"v1:manage:agent:deleteSkill","desc":"删除 Skill"}]',0,'v1:manage:agent:deleteSkill'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a004',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a002','3','启用/禁用 Skill',0,'',3,'','','','','1','button.agent.skills.status',0,'',0,0,'[]','[{"code":"v1:manage:agent:setSkillStatus","desc":"启用或禁用 Skill"}]',0,'v1:manage:agent:setSkillStatus'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','cc10f1d2-8d73-4ff9-9702-4f7395b7a001','3','读取全局 Token 额度',0,'',0,'','','','','1','button.agent.tokenQuota.viewGlobal',0,'',0,0,'[]','[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"}]',0,'v1:manage:agent:getTokenQuotaGlobal'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a002',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','cc10f1d2-8d73-4ff9-9702-4f7395b7a001','3','保存全局 Token 额度',0,'',1,'','','','','1','button.agent.tokenQuota.saveGlobal',0,'',0,0,'[]','[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"},{"code":"v1:manage:agent:saveTokenQuotaGlobal","desc":"保存全局 Token 额度"}]',0,'v1:manage:agent:saveTokenQuotaGlobal'),
    ('dc10f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','cc10f1d2-8d73-4ff9-9702-4f7395b7a001','3','读取用户 Token 额度',0,'',2,'','','','','1','',0,'',0,0,'[]','[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"},{"code":"v1:manage:agent:getUserTokenQuota","desc":"读取用户 Token 额度"}]',0,'v1:manage:agent:getUserTokenQuota'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a003',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','cc10f1d2-8d73-4ff9-9702-4f7395b7a001','3','保存用户 Token 额度',0,'',3,'','','','','1','button.agent.tokenQuota.saveUser',0,'',0,0,'[]','[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"},{"code":"v1:manage:agent:getUserTokenQuota","desc":"读取用户 Token 额度"},{"code":"v1:manage:agent:saveUserTokenQuota","desc":"保存用户 Token 额度"}]',0,'v1:manage:agent:saveUserTokenQuota'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a004',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','cc10f1d2-8d73-4ff9-9702-4f7395b7a001','3','恢复用户全局额度',0,'',4,'','','','','1','button.agent.tokenQuota.restoreUser',0,'',0,0,'[]','[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"},{"code":"v1:manage:agent:getUserTokenQuota","desc":"读取用户 Token 额度"},{"code":"v1:manage:agent:deleteUserTokenQuota","desc":"恢复用户全局额度"}]',0,'v1:manage:agent:deleteUserTokenQuota'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a005',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','cc10f1d2-8d73-4ff9-9702-4f7395b7a001','3','重置用户 Token 用量',0,'',5,'','','','','1','button.agent.tokenQuota.resetUser',0,'',0,0,'[]','[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"},{"code":"v1:manage:agent:getUserTokenQuota","desc":"读取用户 Token 额度"},{"code":"v1:manage:agent:resetUserTokenQuota","desc":"重置用户 Token 用量"}]',0,'v1:manage:agent:resetUserTokenQuota');

-- The built-in super administrator receives the complete Agent menu tree.
INSERT INTO "manage_role_menu" (
    uuid, create_time, update_time, role_uuid, menu_uuid, is_home
)
SELECT md5('1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d:' || menu_uuid),
       CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
       '1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d', menu_uuid, 0
FROM (VALUES
    ('aa10f1d2-8d73-4ff9-9702-4f7395b7a001'),
    ('aa10f1d2-8d73-4ff9-9702-4f7395b7a003'),
    ('aa10f1d2-8d73-4ff9-9702-4f7395b7a002'),
    ('cc10f1d2-8d73-4ff9-9702-4f7395b7a001'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a001'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a002'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a003'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a004'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a001'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a002'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a003'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a004'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a001'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a002'),
    ('dc10f1d2-8d73-4ff9-9702-4f7395b7a001'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a003'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a004'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a005')
) AS agent_menus(menu_uuid);

-- Default Agent user role starts on the shared home page.
INSERT INTO "manage_role" (uuid, create_time, update_time, name, status, code, "desc")
VALUES (
    '9c7f5e1a-2b34-4d68-8f90-a1b2c3d4e5f6',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    '普通 Agent 用户',
    '1',
    'R_AGENT_USER',
    '普通 Agent 用户'
);

INSERT INTO "manage_role_menu" (
    uuid, create_time, update_time, role_uuid, menu_uuid, is_home
) VALUES (
    'ad8e6f2b-3c45-4e79-8a01-b2c3d4e5f607',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    '9c7f5e1a-2b34-4d68-8f90-a1b2c3d4e5f6',
    'f7e8d9c6-b5a4-4382-8271-605f4e3d2c1b',
    1
);

UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:homeOverview","desc":"读取首页概览"}]',
    update_time = CURRENT_TIMESTAMP
WHERE route_name = 'home';
