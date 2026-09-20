-- Host the Agent conversation application as a normal child page of Agent Management.
INSERT INTO "manage_menu" (
    uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name,
    hide_in_menu, active_menu, "order", route_name, route_path, component,
    icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab,
    query, permissions, constant, button_code
) VALUES (
    'aa10f1d2-8d73-4ff9-9702-4f7395b7a004', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
    '1', 'aa10f1d2-8d73-4ff9-9702-4f7395b7a001', '2', '会话管理', 0, '', 4,
    'agent_conversations', '/agent/conversations', 'view.agent_conversations',
    'carbon:chat', '1', 'route.agent_conversations', 0, '', 0, 0,
    '[]', '[]', 0, ''
);

-- Only the built-in super administrator receives the Admin-hosted page.
INSERT INTO "manage_role_menu" (
    uuid, create_time, update_time, role_uuid, menu_uuid, is_home
) VALUES (
    'aa10f1d2-8d73-4ff9-9702-4f7395b7d104', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
    '1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d',
    'aa10f1d2-8d73-4ff9-9702-4f7395b7a004', 0
);
