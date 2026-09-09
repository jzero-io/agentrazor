DELETE FROM "manage_role_menu"
WHERE menu_uuid IN (
    'e110f1d2-8d73-4ff9-9702-4f7395b7a001','e110f1d2-8d73-4ff9-9702-4f7395b7a002',
    'e110f1d2-8d73-4ff9-9702-4f7395b7a003','e110f1d2-8d73-4ff9-9702-4f7395b7a004',
    'e110f1d2-8d73-4ff9-9702-4f7395b7a005','f110f1d2-8d73-4ff9-9702-4f7395b7a001',
    'f110f1d2-8d73-4ff9-9702-4f7395b7a002','f110f1d2-8d73-4ff9-9702-4f7395b7a003',
    'd110f1d2-8d73-4ff9-9702-4f7395b7a001','d110f1d2-8d73-4ff9-9702-4f7395b7a002',
    'd110f1d2-8d73-4ff9-9702-4f7395b7a003','d110f1d2-8d73-4ff9-9702-4f7395b7a004',
    'd110f1d2-8d73-4ff9-9702-4f7395b7a005'
);

DELETE FROM "manage_menu"
WHERE uuid IN (
    'e110f1d2-8d73-4ff9-9702-4f7395b7a001','e110f1d2-8d73-4ff9-9702-4f7395b7a002',
    'e110f1d2-8d73-4ff9-9702-4f7395b7a003','e110f1d2-8d73-4ff9-9702-4f7395b7a004',
    'e110f1d2-8d73-4ff9-9702-4f7395b7a005','f110f1d2-8d73-4ff9-9702-4f7395b7a001',
    'f110f1d2-8d73-4ff9-9702-4f7395b7a002','f110f1d2-8d73-4ff9-9702-4f7395b7a003',
    'd110f1d2-8d73-4ff9-9702-4f7395b7a001','d110f1d2-8d73-4ff9-9702-4f7395b7a002',
    'd110f1d2-8d73-4ff9-9702-4f7395b7a003','d110f1d2-8d73-4ff9-9702-4f7395b7a004',
    'd110f1d2-8d73-4ff9-9702-4f7395b7a005'
);

UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:getSettings","desc":"读取模型配置"},{"code":"v1:manage:agent:saveSelection","desc":"切换 Provider 和模型"},{"code":"v1:manage:agent:saveProviderApiKey","desc":"保存 Provider API Key"},{"code":"v1:manage:agent:getAccountStatus","desc":"读取模型账号状态"},{"code":"v1:manage:agent:loginApiKey","desc":"API Key 登录"},{"code":"v1:manage:agent:startChatGPTLogin","desc":"ChatGPT 登录"},{"code":"v1:manage:agent:logout","desc":"退出模型账号"},{"code":"v1:manage:agent:restartRuntime","desc":"重启 Runtime"}]', update_time = CURRENT_TIMESTAMP
WHERE uuid = 'aa10f1d2-8d73-4ff9-9702-4f7395b7a003';

UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:listSkills","desc":"Skills 列表"},{"code":"v1:manage:agent:skillDetail","desc":"Skill 详情"},{"code":"v1:manage:agent:uploadSkill","desc":"上传 Skill"},{"code":"v1:manage:agent:deleteSkill","desc":"删除 Skill"},{"code":"v1:manage:agent:updateSkillFile","desc":"更新 Skill 文件"}]', update_time = CURRENT_TIMESTAMP
WHERE uuid = 'aa10f1d2-8d73-4ff9-9702-4f7395b7a002';

UPDATE "manage_menu" SET permissions = '[]', update_time = CURRENT_TIMESTAMP
WHERE uuid = 'cc10f1d2-8d73-4ff9-9702-4f7395b7a001';

UPDATE "manage_menu"
SET menu_name = 'Token 额度管理', "order" = 0,
    permissions = '[{"code":"v1:manage:agent:getTokenQuotaGlobal","desc":"读取全局 Token 额度"},{"code":"v1:manage:agent:saveTokenQuotaGlobal","desc":"保存全局 Token 额度"},{"code":"v1:manage:agent:getUserTokenQuota","desc":"读取用户 Token 额度"},{"code":"v1:manage:agent:saveUserTokenQuota","desc":"保存用户 Token 额度"},{"code":"v1:manage:agent:deleteUserTokenQuota","desc":"恢复用户全局额度"},{"code":"v1:manage:agent:resetUserTokenQuota","desc":"重置用户 Token 额度"}]',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'dc10f1d2-8d73-4ff9-9702-4f7395b7a001';


-- Restore the coarse policies represented by the rolled-back page/button model.
WITH permission_sources(menu_uuid, permission_code) AS (
    VALUES
        ('aa10f1d2-8d73-4ff9-9702-4f7395b7a003', 'v1:manage:agent:saveSelection'),
        ('aa10f1d2-8d73-4ff9-9702-4f7395b7a003', 'v1:manage:agent:saveProviderApiKey'),
        ('aa10f1d2-8d73-4ff9-9702-4f7395b7a003', 'v1:manage:agent:loginApiKey'),
        ('aa10f1d2-8d73-4ff9-9702-4f7395b7a003', 'v1:manage:agent:startChatGPTLogin'),
        ('aa10f1d2-8d73-4ff9-9702-4f7395b7a003', 'v1:manage:agent:logout'),
        ('aa10f1d2-8d73-4ff9-9702-4f7395b7a003', 'v1:manage:agent:restartRuntime'),
        ('aa10f1d2-8d73-4ff9-9702-4f7395b7a002', 'v1:manage:agent:uploadSkill'),
        ('aa10f1d2-8d73-4ff9-9702-4f7395b7a002', 'v1:manage:agent:updateSkillFile'),
        ('aa10f1d2-8d73-4ff9-9702-4f7395b7a002', 'v1:manage:agent:deleteSkill'),
        ('dc10f1d2-8d73-4ff9-9702-4f7395b7a001', 'v1:manage:agent:saveTokenQuotaGlobal'),
        ('dc10f1d2-8d73-4ff9-9702-4f7395b7a001', 'v1:manage:agent:saveUserTokenQuota'),
        ('dc10f1d2-8d73-4ff9-9702-4f7395b7a001', 'v1:manage:agent:deleteUserTokenQuota'),
        ('dc10f1d2-8d73-4ff9-9702-4f7395b7a001', 'v1:manage:agent:resetUserTokenQuota')
), role_permissions AS (
    SELECT DISTINCT rm.role_uuid, source.permission_code
    FROM "manage_role_menu" rm
    JOIN permission_sources source ON source.menu_uuid = rm.menu_uuid
)
INSERT INTO "casbin_rule" (p_type, v0, v1, v2, v3, v4, v5)
SELECT 'p', role_permissions.role_uuid, role_permissions.permission_code, '', '', '', ''
FROM role_permissions
WHERE NOT EXISTS (
    SELECT 1 FROM "casbin_rule" existing
    WHERE existing.p_type = 'p'
      AND existing.v0 = role_permissions.role_uuid
      AND existing.v1 = role_permissions.permission_code
);
