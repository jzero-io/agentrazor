UPDATE "manage_menu"
SET menu_name = '保存模型配置',
    permissions = '[{"code":"v1:manage:agent:saveSelection","desc":"切换 Provider 和模型"},{"code":"v1:manage:agent:saveProviderApiKey","desc":"保存 Provider API Key"}]',
    button_code = 'v1:manage:agent:saveSelection', update_time = CURRENT_TIMESTAMP
WHERE uuid = 'e110f1d2-8d73-4ff9-9702-4f7395b7a001';

INSERT INTO "manage_menu" (
    uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name,
    hide_in_menu, active_menu, "order", route_name, route_path, component,
    icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab,
    query, permissions, constant, button_code
) VALUES
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a006',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a003','3','保存默认系统提示词',0,'',6,'','','','','1','button.agent.config.saveDefaultSystemPrompt',0,'',0,0,'[]','[{"code":"v1:manage:agent:getDefaultSystemPrompt","desc":"读取默认系统提示词"},{"code":"v1:manage:agent:saveDefaultSystemPrompt","desc":"保存默认系统提示词"}]',0,'v1:manage:agent:saveDefaultSystemPrompt'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a007',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a003','3','读取默认系统提示词',0,'',5,'','','','','1','button.agent.config.readDefaultSystemPrompt',0,'',0,0,'[]','[{"code":"v1:manage:agent:getDefaultSystemPrompt","desc":"读取默认系统提示词"}]',0,'v1:manage:agent:getDefaultSystemPrompt')
ON CONFLICT (uuid) DO NOTHING;

INSERT INTO "manage_role_menu" (uuid, create_time, update_time, role_uuid, menu_uuid, is_home)
SELECT md5(source.role_uuid || ':' || target.menu_uuid), CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
       source.role_uuid, target.menu_uuid, 0
FROM "manage_role_menu" source
CROSS JOIN (VALUES
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a006'),
    ('e110f1d2-8d73-4ff9-9702-4f7395b7a007')
) target(menu_uuid)
WHERE source.menu_uuid = 'e110f1d2-8d73-4ff9-9702-4f7395b7a001'
ON CONFLICT (uuid) DO NOTHING;

INSERT INTO "casbin_rule" (p_type, v0, v1, v2, v3, v4, v5)
SELECT 'p', roles.v0, permissions.code, '', '', '', ''
FROM (
    SELECT DISTINCT v0 FROM "casbin_rule"
    WHERE p_type = 'p' AND v1 = 'v1:manage:agent:saveSettings'
) roles
CROSS JOIN (VALUES
    ('v1:manage:agent:saveSelection'),
    ('v1:manage:agent:saveProviderApiKey'),
    ('v1:manage:agent:getDefaultSystemPrompt'),
    ('v1:manage:agent:saveDefaultSystemPrompt')
) permissions(code)
WHERE NOT EXISTS (
    SELECT 1 FROM "casbin_rule" existing
    WHERE existing.p_type = 'p' AND existing.v0 = roles.v0
      AND existing.v1 = permissions.code
);

DELETE FROM "casbin_rule"
WHERE p_type = 'p' AND v1 = 'v1:manage:agent:saveSettings';
