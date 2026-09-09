-- Page nodes only grant read access. Mutating operations are represented by
-- independent button nodes, consistent with the system-management menus.
UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:getSettings","desc":"读取模型配置"},{"code":"v1:manage:agent:getAccountStatus","desc":"读取模型账号状态"}]',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'aa10f1d2-8d73-4ff9-9702-4f7395b7a003';

UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:listSkills","desc":"Skills 列表"},{"code":"v1:manage:agent:skillDetail","desc":"Skill 详情"}]',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'aa10f1d2-8d73-4ff9-9702-4f7395b7a002';

UPDATE "manage_menu"
SET permissions = '[{"code":"v1:conversation:tokenUsageTrend","desc":"Token 消耗趋势"},{"code":"v1:conversation:tokenUsageDetails","desc":"Token 消耗账号明细"},{"code":"v1:conversation:tokenUsageConversations","desc":"Token 消耗对话明细"}]',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'cc10f1d2-8d73-4ff9-9702-4f7395b7a001';

INSERT INTO "manage_menu" (
    uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name,
    hide_in_menu, active_menu, "order", route_name, route_path, component,
    icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab,
    query, permissions, constant, button_code
) VALUES
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a003','3','保存模型配置',0,'',0,'','','','','1','button.agent.config.save',0,'',0,0,'[]','[{"code":"v1:manage:agent:saveSelection","desc":"切换 Provider 和模型"},{"code":"v1:manage:agent:saveProviderApiKey","desc":"保存 Provider API Key"}]',0,'v1:manage:agent:saveSelection'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a002',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a003','3','ChatGPT 登录',0,'',1,'','','','','1','button.agent.config.chatgptLogin',0,'',0,0,'[]','[{"code":"v1:manage:agent:startChatGPTLogin","desc":"ChatGPT 登录"}]',0,'v1:manage:agent:startChatGPTLogin'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a003',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a003','3','API Key 登录',0,'',2,'','','','','1','button.agent.config.apiKeyLogin',0,'',0,0,'[]','[{"code":"v1:manage:agent:loginApiKey","desc":"API Key 登录"}]',0,'v1:manage:agent:loginApiKey'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a004',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a003','3','退出模型账号',0,'',3,'','','','','1','button.agent.config.logout',0,'',0,0,'[]','[{"code":"v1:manage:agent:logout","desc":"退出模型账号"}]',0,'v1:manage:agent:logout'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a005',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a003','3','重启 Runtime',0,'',4,'','','','','1','button.agent.config.restartRuntime',0,'',0,0,'[]','[{"code":"v1:manage:agent:restartRuntime","desc":"重启 Runtime"}]',0,'v1:manage:agent:restartRuntime'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a002','3','上传 Skill',0,'',0,'','','','','1','button.agent.skills.upload',0,'',0,0,'[]','[{"code":"v1:manage:agent:uploadSkill","desc":"上传 Skill"}]',0,'v1:manage:agent:uploadSkill'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a002',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a002','3','编辑 Skill',0,'',1,'','','','','1','button.agent.skills.edit',0,'',0,0,'[]','[{"code":"v1:manage:agent:updateSkillFile","desc":"更新 Skill 文件"}]',0,'v1:manage:agent:updateSkillFile'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a003',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a002','3','删除 Skill',0,'',2,'','','','','1','button.agent.skills.delete',0,'',0,0,'[]','[{"code":"v1:manage:agent:deleteSkill","desc":"删除 Skill"}]',0,'v1:manage:agent:deleteSkill'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','cc10f1d2-8d73-4ff9-9702-4f7395b7a001','3','读取全局 Token 额度',0,'',0,'','','','','1','button.agent.tokenQuota.viewGlobal',0,'',0,0,'[]','[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"}]',0,'v1:manage:agent:getTokenQuotaGlobal'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a002',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','cc10f1d2-8d73-4ff9-9702-4f7395b7a001','3','保存全局 Token 额度',0,'',1,'','','','','1','button.agent.tokenQuota.saveGlobal',0,'',0,0,'[]','[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"},{"code":"v1:manage:agent:saveTokenQuotaGlobal","desc":"保存全局 Token 额度"}]',0,'v1:manage:agent:saveTokenQuotaGlobal'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a003',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','cc10f1d2-8d73-4ff9-9702-4f7395b7a001','3','保存用户 Token 额度',0,'',3,'','','','','1','button.agent.tokenQuota.saveUser',0,'',0,0,'[]','[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"},{"code":"v1:manage:agent:getUserTokenQuota","desc":"读取用户 Token 额度"},{"code":"v1:manage:agent:saveUserTokenQuota","desc":"保存用户 Token 额度"}]',0,'v1:manage:agent:saveUserTokenQuota'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a004',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','cc10f1d2-8d73-4ff9-9702-4f7395b7a001','3','恢复用户全局额度',0,'',4,'','','','','1','button.agent.tokenQuota.restoreUser',0,'',0,0,'[]','[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"},{"code":"v1:manage:agent:getUserTokenQuota","desc":"读取用户 Token 额度"},{"code":"v1:manage:agent:deleteUserTokenQuota","desc":"恢复用户全局额度"}]',0,'v1:manage:agent:deleteUserTokenQuota'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a005',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','cc10f1d2-8d73-4ff9-9702-4f7395b7a001','3','重置用户 Token 用量',0,'',5,'','','','','1','button.agent.tokenQuota.resetUser',0,'',0,0,'[]','[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"},{"code":"v1:manage:agent:getUserTokenQuota","desc":"读取用户 Token 额度"},{"code":"v1:manage:agent:resetUserTokenQuota","desc":"重置用户 Token 用量"}]',0,'v1:manage:agent:resetUserTokenQuota');

-- Narrow the former all-in-one quota node to read-user access.
UPDATE "manage_menu"
SET menu_name = '读取用户 Token 额度', "order" = 2,
    permissions = '[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"},{"code":"v1:manage:agent:getUserTokenQuota","desc":"读取用户 Token 额度"}]',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'dc10f1d2-8d73-4ff9-9702-4f7395b7a001';

-- New destructive/write permissions are default-deny for existing custom
-- roles. The built-in super administrator retains complete access.
INSERT INTO "manage_role_menu" (uuid, create_time, update_time, role_uuid, menu_uuid, is_home)
SELECT md5('1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d:' || button_uuid),
       CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
       '1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d', button_uuid, 0
FROM (VALUES
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a001'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a002'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a003'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a004'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a005'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a001'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a002'),
    ('f110f1d2-8d73-4ff9-9702-4f7395b7a003'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a001'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a002'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a003'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a004'),
    ('d110f1d2-8d73-4ff9-9702-4f7395b7a005')
) AS buttons(button_uuid)
WHERE NOT EXISTS (
    SELECT 1 FROM "manage_role_menu" existing
    WHERE existing.role_uuid = '1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d'
      AND existing.menu_uuid = buttons.button_uuid
);

-- The adapter normally creates this table on first startup. Creating it here
-- also makes the migration safe for a fresh database before ServiceContext is built.
CREATE TABLE IF NOT EXISTS "casbin_rule" (
    p_type varchar(32) NOT NULL DEFAULT '',
    v0 varchar(255) NOT NULL DEFAULT '',
    v1 varchar(255) NOT NULL DEFAULT '',
    v2 varchar(255) NOT NULL DEFAULT '',
    v3 varchar(255) NOT NULL DEFAULT '',
    v4 varchar(255) NOT NULL DEFAULT '',
    v5 varchar(255) NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_casbin_rule ON "casbin_rule" (p_type, v0, v1);

-- Remove only the write policies that used to be inherited from the coarse
-- Agent pages. Other modules and all super-administrator policies are untouched.
DELETE FROM "casbin_rule"
WHERE p_type = 'p'
  AND v0 <> '1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d'
  AND v1 IN (
      'v1:manage:agent:saveSelection',
      'v1:manage:agent:saveProviderApiKey',
      'v1:manage:agent:loginApiKey',
      'v1:manage:agent:startChatGPTLogin',
      'v1:manage:agent:logout',
      'v1:manage:agent:restartRuntime',
      'v1:manage:agent:uploadSkill',
      'v1:manage:agent:updateSkillFile',
      'v1:manage:agent:deleteSkill',
      'v1:manage:agent:saveTokenQuotaGlobal',
      'v1:manage:agent:saveUserTokenQuota',
      'v1:manage:agent:deleteUserTokenQuota',
      'v1:manage:agent:resetUserTokenQuota'
  );
