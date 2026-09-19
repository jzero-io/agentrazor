-- The platform owns the plugins menu group. Individual serverless plugins add
-- their own iframe pages below it in their own migration sets.
INSERT INTO "manage_menu" (
    uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name,
    hide_in_menu, active_menu, "order", route_name, route_path, component,
    icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab,
    query, permissions, constant, button_code
) VALUES (
    '7d4c5bf1-4e92-4b50-95f1-0e166b90d001', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
    '1', '', '1', '插件管理', 0, '', 7, 'plugin_management', '/plugins',
    'layout.base', 'carbon:plug', '1', '', 0, '', 0, 0,
    '[]', '[]', 0, ''
);

INSERT INTO "manage_role_menu" (
    uuid, create_time, update_time, role_uuid, menu_uuid, is_home
) VALUES (
    '7d4c5bf1-4e92-4b50-95f1-0e166b90d101', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
    '1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d', '7d4c5bf1-4e92-4b50-95f1-0e166b90d001', 0
);
