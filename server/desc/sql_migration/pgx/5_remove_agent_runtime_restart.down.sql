INSERT INTO "manage_menu" (
    uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name,
    hide_in_menu, active_menu, "order", route_name, route_path, component,
    icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab,
    query, permissions, constant, button_code
) VALUES (
    'e110f1d2-8d73-4ff9-9702-4f7395b7a005', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
    '1', 'aa10f1d2-8d73-4ff9-9702-4f7395b7a003', '3', '重启 Runtime',
    0, '', 4, '', '', '', '', '1', 'button.agent.config.restartRuntime',
    0, '', 0, 0, '[]',
    '[{"code":"v1:manage:agent:restartRuntime","desc":"重启 Runtime"}]',
    0, 'v1:manage:agent:restartRuntime'
)
ON CONFLICT (uuid) DO NOTHING;

INSERT INTO "manage_role_menu" (uuid, create_time, update_time, role_uuid, menu_uuid, is_home)
VALUES (
    md5('1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d:e110f1d2-8d73-4ff9-9702-4f7395b7a005'),
    CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
    '1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d',
    'e110f1d2-8d73-4ff9-9702-4f7395b7a005', 0
)
ON CONFLICT (uuid) DO NOTHING;

INSERT INTO "casbin_rule" (p_type, v0, v1, v2, v3, v4, v5)
SELECT 'p', '1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d',
       'v1:manage:agent:restartRuntime', '', '', '', ''
WHERE NOT EXISTS (
    SELECT 1 FROM "casbin_rule"
    WHERE p_type = 'p'
      AND v0 = '1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d'
      AND v1 = 'v1:manage:agent:restartRuntime'
);
