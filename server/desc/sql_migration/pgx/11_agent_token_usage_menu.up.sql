INSERT INTO "manage_menu" (uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name, hide_in_menu, active_menu, "order", route_name, route_path, component, icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab, query, permissions, constant, button_code)
VALUES
    ('cc10f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1','aa10f1d2-8d73-4ff9-9702-4f7395b7a001','2','Token 消耗',0,'',3,'agent_token-usage','/agent/token-usage','view.agent_token-usage','carbon:meter','1','route.agent_token-usage',0,'',0,0,'[]','[]',0,'');

INSERT INTO "manage_role_menu" (uuid, create_time, update_time, role_uuid, menu_uuid, is_home)
VALUES
    ('cc20f1d2-8d73-4ff9-9702-4f7395b7a001',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d','cc10f1d2-8d73-4ff9-9702-4f7395b7a001',0);
