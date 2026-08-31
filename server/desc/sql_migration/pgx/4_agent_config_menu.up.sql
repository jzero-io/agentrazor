INSERT INTO "manage_menu" (uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name, hide_in_menu, active_menu, "order", route_name, route_path, component, icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab, query, permissions, constant, button_code)
VALUES
    ('aa10f1d2-8d73-4ff9-9702-4f7395b7a003','2024-12-05 00:00:00','2024-12-05 00:00:00','1','aa10f1d2-8d73-4ff9-9702-4f7395b7a001','2','配置管理',0,'',1,'agent_config','/agent/config','view.agent_config','carbon:settings-services','1','route.agent_config',0,'',0,0,'[]','[{"code":"v1:manage:agent:listConfigFiles","desc":"配置文件列表"},{"code":"v1:manage:agent:configFile","desc":"读取配置文件"},{"code":"v1:manage:agent:updateConfigFile","desc":"更新配置文件"},{"code":"v1:manage:agent:runtimeStatus","desc":"Runtime 状态"},{"code":"v1:manage:agent:restartRuntime","desc":"重启 Runtime"}]',0,'');

INSERT INTO "manage_role_menu" (uuid, create_time, update_time, role_uuid, menu_uuid, is_home)
VALUES
    ('aa20f1d2-8d73-4ff9-9702-4f7395b7a003','2024-12-05 00:00:00','2024-12-05 00:00:00','1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d','aa10f1d2-8d73-4ff9-9702-4f7395b7a003',0);
