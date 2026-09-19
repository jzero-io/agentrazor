-- Existing installations previously used Plugin Management as a leaf page.
-- Convert it back to a normal menu group so plugins are its second-level
-- Admin menus, in the same shape as System Management -> User Management.
UPDATE "manage_menu"
SET menu_type = '1',
    route_name = 'plugin_management',
    route_path = '/plugins',
    component = 'layout.base',
    i18n_key = '',
    href = '',
    permissions = COALESCE((
        SELECT jsonb_agg(permission)
        FROM jsonb_array_elements(COALESCE(NULLIF(permissions, ''), '[]')::jsonb) AS permission
        WHERE permission->>'code' <> 'v1:manage:menu:getPluginPages'
    ), '[]'::jsonb)::text
WHERE uuid = '7d4c5bf1-4e92-4b50-95f1-0e166b90d001';

DELETE FROM "casbin_rule"
WHERE v1 = 'v1:manage:menu:getPluginPages';
