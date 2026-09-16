-- Consolidate model selection, provider key and default system prompt under one save permission.
INSERT INTO "manage_role_menu" (uuid, create_time, update_time, role_uuid, menu_uuid, is_home)
SELECT md5(existing.role_uuid || ':e110f1d2-8d73-4ff9-9702-4f7395b7a001'), CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
       existing.role_uuid, 'e110f1d2-8d73-4ff9-9702-4f7395b7a001', 0
FROM "manage_role_menu" existing
WHERE existing.menu_uuid IN ('e110f1d2-8d73-4ff9-9702-4f7395b7a006', 'e110f1d2-8d73-4ff9-9702-4f7395b7a007')
ON CONFLICT (uuid) DO NOTHING;

UPDATE "manage_menu"
SET menu_name = '保存模型配置',
    permissions = '[{"code":"v1:manage:agent:saveSettings","desc":"保存模型、Provider Key 和默认系统提示词"}]',
    button_code = 'v1:manage:agent:saveSettings', "order" = 0, update_time = CURRENT_TIMESTAMP
WHERE uuid = 'e110f1d2-8d73-4ff9-9702-4f7395b7a001';

DELETE FROM "manage_role_menu"
WHERE menu_uuid IN ('e110f1d2-8d73-4ff9-9702-4f7395b7a006', 'e110f1d2-8d73-4ff9-9702-4f7395b7a007');

DELETE FROM "manage_menu"
WHERE uuid IN ('e110f1d2-8d73-4ff9-9702-4f7395b7a006', 'e110f1d2-8d73-4ff9-9702-4f7395b7a007');

INSERT INTO "casbin_rule" (p_type, v0, v1, v2, v3, v4, v5)
SELECT 'p', roles.v0, 'v1:manage:agent:saveSettings', '', '', '', ''
FROM (
    SELECT DISTINCT v0 FROM "casbin_rule"
    WHERE p_type = 'p' AND v1 IN (
        'v1:manage:agent:saveSelection',
        'v1:manage:agent:saveProviderApiKey',
        'v1:manage:agent:saveDefaultSystemPrompt'
    )
) roles
WHERE NOT EXISTS (
    SELECT 1 FROM "casbin_rule" existing
    WHERE existing.p_type = 'p' AND existing.v0 = roles.v0
      AND existing.v1 = 'v1:manage:agent:saveSettings'
);

DELETE FROM "casbin_rule"
WHERE p_type = 'p' AND v1 IN (
    'v1:manage:agent:saveSelection',
    'v1:manage:agent:saveProviderApiKey',
    'v1:manage:agent:getDefaultSystemPrompt',
    'v1:manage:agent:saveDefaultSystemPrompt'
);
