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

-- Agent menus
INSERT INTO "manage_menu" (uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name, hide_in_menu, active_menu, "order", route_name, route_path, component, icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab, query, permissions, constant, button_code)
VALUES
    ('aa10f1d2-8d73-4ff9-9702-4f7395b7a001','2024-12-05 00:00:00','2024-12-05 00:00:00','1','','1','Agent 管理',0,'',6,'agent','/agent','layout.base','carbon:bot','1','route.agent',0,'',0,0,'[]','[]',0,''),
    ('aa10f1d2-8d73-4ff9-9702-4f7395b7a003','2024-12-05 00:00:00','2024-12-05 00:00:00','1','aa10f1d2-8d73-4ff9-9702-4f7395b7a001','2','配置管理',0,'',1,'agent_config','/agent/config','view.agent_config','carbon:settings-services','1','route.agent_config',0,'',0,0,'[]','[{"code":"v1:manage:agent:getSettings","desc":"读取模型配置"},{"code":"v1:manage:agent:saveSelection","desc":"切换 Provider 和模型"},{"code":"v1:manage:agent:saveProviderApiKey","desc":"保存 Provider API Key"},{"code":"v1:manage:agent:getAccountStatus","desc":"读取模型账号状态"},{"code":"v1:manage:agent:loginApiKey","desc":"API Key 登录"},{"code":"v1:manage:agent:startChatGPTLogin","desc":"ChatGPT 登录"},{"code":"v1:manage:agent:logout","desc":"退出模型账号"},{"code":"v1:manage:agent:restartRuntime","desc":"重启 Runtime"}]',0,''),
    ('aa10f1d2-8d73-4ff9-9702-4f7395b7a002','2024-12-05 00:00:00','2024-12-05 00:00:00','1','aa10f1d2-8d73-4ff9-9702-4f7395b7a001','2','Skills 管理',0,'',2,'agent_skills','/agent/skills','view.agent_skills','carbon:skill-level-basic','1','route.agent_skills',0,'',0,0,'[]','[{"code":"v1:manage:agent:listSkills","desc":"Skills 列表"},{"code":"v1:manage:agent:skillDetail","desc":"Skill 详情"},{"code":"v1:manage:agent:uploadSkill","desc":"上传 Skill"},{"code":"v1:manage:agent:deleteSkill","desc":"删除 Skill"},{"code":"v1:manage:agent:updateSkillFile","desc":"更新 Skill 文件"}]',0,''),
    ('cc10f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a001','2','Token 消耗',0,'',3,'agent_token-usage','/agent/token-usage','view.agent_token-usage','carbon:meter','1','route.agent_token-usage',0,'',0,0,'[]','[]',0,'');

INSERT INTO "manage_role_menu" (uuid, create_time, update_time, role_uuid, menu_uuid, is_home)
VALUES
    ('aa20f1d2-8d73-4ff9-9702-4f7395b7a001','2024-12-05 00:00:00','2024-12-05 00:00:00','1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d','aa10f1d2-8d73-4ff9-9702-4f7395b7a001',0),
    ('aa20f1d2-8d73-4ff9-9702-4f7395b7a002','2024-12-05 00:00:00','2024-12-05 00:00:00','1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d','aa10f1d2-8d73-4ff9-9702-4f7395b7a002',0),
    ('aa20f1d2-8d73-4ff9-9702-4f7395b7a003','2024-12-05 00:00:00','2024-12-05 00:00:00','1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d','aa10f1d2-8d73-4ff9-9702-4f7395b7a003',0),
    ('cc20f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d','cc10f1d2-8d73-4ff9-9702-4f7395b7a001',0);

-- Default Agent user role
INSERT INTO "manage_role" (uuid, create_time, update_time, name, status, code, "desc")
VALUES
    ('9c7f5e1a-2b34-4d68-8f90-a1b2c3d4e5f6',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'普通 Agent 用户','1','R_AGENT_USER','普通 Agent 用户');

INSERT INTO "manage_role_menu" (uuid, create_time, update_time, role_uuid, menu_uuid, is_home)
VALUES
    ('ad8e6f2b-3c45-4e79-8a01-b2c3d4e5f607',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'9c7f5e1a-2b34-4d68-8f90-a1b2c3d4e5f6','f7e8d9c6-b5a4-4382-8271-605f4e3d2c1b',1);

UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:homeOverview","desc":"读取首页概览"}]',
    update_time = CURRENT_TIMESTAMP
WHERE route_name = 'home';
