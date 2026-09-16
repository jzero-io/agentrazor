-- Login only authenticates the provider. Applying the selected provider/model still
-- goes through the unified settings save endpoint, so both login actions depend on it.
UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:startChatGPTLogin","desc":"ChatGPT 登录"},{"code":"v1:manage:agent:saveSettings","desc":"保存并启用模型配置"}]',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'e110f1d2-8d73-4ff9-9702-4f7395b7a002';

UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:loginApiKey","desc":"API Key 登录"},{"code":"v1:manage:agent:saveSettings","desc":"保存并启用模型配置"}]',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'e110f1d2-8d73-4ff9-9702-4f7395b7a003';

-- Keep the permission tree consistent for roles that already have either login action.
INSERT INTO "manage_role_menu" (uuid, create_time, update_time, role_uuid, menu_uuid, is_home)
SELECT md5(login.role_uuid || ':e110f1d2-8d73-4ff9-9702-4f7395b7a001'),
       CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, login.role_uuid,
       'e110f1d2-8d73-4ff9-9702-4f7395b7a001', 0
FROM (
    SELECT DISTINCT role_uuid
    FROM "manage_role_menu"
    WHERE menu_uuid IN (
        'e110f1d2-8d73-4ff9-9702-4f7395b7a002',
        'e110f1d2-8d73-4ff9-9702-4f7395b7a003'
    )
) login
WHERE NOT EXISTS (
    SELECT 1 FROM "manage_role_menu" existing
    WHERE existing.role_uuid = login.role_uuid
      AND existing.menu_uuid = 'e110f1d2-8d73-4ff9-9702-4f7395b7a001'
)
ON CONFLICT (uuid) DO NOTHING;

INSERT INTO "casbin_rule" (p_type, v0, v1, v2, v3, v4, v5)
SELECT 'p', login.role_uuid, 'v1:manage:agent:saveSettings', '', '', '', ''
FROM (
    SELECT DISTINCT role_uuid
    FROM "manage_role_menu"
    WHERE menu_uuid IN (
        'e110f1d2-8d73-4ff9-9702-4f7395b7a002',
        'e110f1d2-8d73-4ff9-9702-4f7395b7a003'
    )
) login
WHERE NOT EXISTS (
    SELECT 1 FROM "casbin_rule" existing
    WHERE existing.p_type = 'p'
      AND existing.v0 = login.role_uuid
      AND existing.v1 = 'v1:manage:agent:saveSettings'
);
