INSERT INTO "manage_menu" (uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name, hide_in_menu, active_menu, "order", route_name, route_path, component, icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab, query, permissions, constant, button_code)
VALUES
    ('bb10f1d2-8d73-4ff9-9702-4f7395b7a001','2024-12-06 00:00:00','2024-12-06 00:00:00','1','a1b2c3d4-e5f6-4782-91a0-b9c8d7e6f5a4','2','邮箱配置',0,'',7,'manage_email','/manage/email','view.manage_email','carbon:email','1','route.manage_email',0,'',0,0,'[]','[{"code":"v1:manage:email:getConfig","desc":"读取邮箱配置"},{"code":"v1:manage:email:saveConfig","desc":"保存邮箱配置"},{"code":"v1:manage:email:testConfig","desc":"发送测试邮件"}]',0,'');

INSERT INTO "manage_role_menu" (uuid, create_time, update_time, role_uuid, menu_uuid, is_home)
VALUES
    ('bb20f1d2-8d73-4ff9-9702-4f7395b7a001','2024-12-06 00:00:00','2024-12-06 00:00:00','1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d','bb10f1d2-8d73-4ff9-9702-4f7395b7a001',0);
