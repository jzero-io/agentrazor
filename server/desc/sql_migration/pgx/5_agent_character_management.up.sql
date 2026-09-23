INSERT INTO "manage_menu" (
    uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name,
    hide_in_menu, active_menu, "order", route_name, route_path, component,
    icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab,
    query, permissions, constant, button_code
) VALUES
    ('aa10f1d2-8d73-4ff9-9702-4f7395b7a005',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a001','2','角色管理',0,'',3,'agent_characters','/agent/characters','view.agent_characters','carbon:user-role','1','route.agent_characters',0,'',0,0,'[]','[{"code":"v1:manage:agent:character:list","desc":"Agent 角色列表"}]',0,''),
    ('e210f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a005','3','新增内置角色',0,'',0,'','','','','1','button.agent.characters.create',0,'',0,0,'[]','[{"code":"v1:manage:agent:character:create","desc":"新增内置 Agent 角色"}]',0,'v1:manage:agent:character:create'),
    ('e210f1d2-8d73-4ff9-9702-4f7395b7a002',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a005','3','编辑内置角色',0,'',1,'','','','','1','button.agent.characters.update',0,'',0,0,'[]','[{"code":"v1:manage:agent:character:update","desc":"编辑内置 Agent 角色"}]',0,'v1:manage:agent:character:update'),
    ('e210f1d2-8d73-4ff9-9702-4f7395b7a003',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a005','3','删除内置角色',0,'',2,'','','','','1','button.agent.characters.delete',0,'',0,0,'[]','[{"code":"v1:manage:agent:character:delete","desc":"删除内置 Agent 角色"}]',0,'v1:manage:agent:character:delete');

UPDATE "manage_menu"
SET "order" = CASE uuid WHEN 'aa10f1d2-8d73-4ff9-9702-4f7395b7a002' THEN 4 WHEN 'cc10f1d2-8d73-4ff9-9702-4f7395b7a001' THEN 5 END,
    update_time = CURRENT_TIMESTAMP
WHERE uuid IN ('aa10f1d2-8d73-4ff9-9702-4f7395b7a002','cc10f1d2-8d73-4ff9-9702-4f7395b7a001');

INSERT INTO "manage_role_menu" (uuid, create_time, update_time, role_uuid, menu_uuid, is_home)
SELECT md5('1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d:' || menu_uuid), CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
       '1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d', menu_uuid, 0
FROM (VALUES
    ('aa10f1d2-8d73-4ff9-9702-4f7395b7a005'),
    ('e210f1d2-8d73-4ff9-9702-4f7395b7a001'),
    ('e210f1d2-8d73-4ff9-9702-4f7395b7a002'),
    ('e210f1d2-8d73-4ff9-9702-4f7395b7a003')
) AS character_menus(menu_uuid);
