-- Baseline migration for plugin management and Agent Management menu configuration.
-- Existing databases must be manually rebased to version 3 after their schema has been verified.

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
    'layout.base', 'carbon:plug', '1', 'route.plugin_management', 0, '', 0, 0,
    '[]', '[]', 0, ''
);

INSERT INTO "manage_role_menu" (
    uuid, create_time, update_time, role_uuid, menu_uuid, is_home
) VALUES (
    '7d4c5bf1-4e92-4b50-95f1-0e166b90d101', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
    '1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d', '7d4c5bf1-4e92-4b50-95f1-0e166b90d001', 0
);

-- Host the Agent conversation application as a normal child page of Agent Management.
INSERT INTO "manage_menu" (
    uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name,
    hide_in_menu, active_menu, "order", route_name, route_path, component,
    icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab,
    query, permissions, constant, button_code
) VALUES (
    'aa10f1d2-8d73-4ff9-9702-4f7395b7a004', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
    '1', 'aa10f1d2-8d73-4ff9-9702-4f7395b7a001', '2', '会话管理', 0, '', 2,
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

-- Keep the database-backed menu label aligned with the Admin route translation.
UPDATE "manage_menu"
SET menu_name = 'Token 管理',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'cc10f1d2-8d73-4ff9-9702-4f7395b7a001';

-- Keep Conversation Management immediately after Config Management.
UPDATE "manage_menu"
SET "order" = CASE uuid
        WHEN 'aa10f1d2-8d73-4ff9-9702-4f7395b7a002' THEN 3
        WHEN 'cc10f1d2-8d73-4ff9-9702-4f7395b7a001' THEN 4
      END,
    update_time = CURRENT_TIMESTAMP
WHERE uuid IN (
    'aa10f1d2-8d73-4ff9-9702-4f7395b7a002',
    'cc10f1d2-8d73-4ff9-9702-4f7395b7a001'
);
