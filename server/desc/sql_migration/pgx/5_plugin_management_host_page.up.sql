-- Plugin Management is a normal Admin menu group. Plugin migrations add their
-- own second-level menus under it, just like the entries in System Management.
UPDATE "manage_menu"
SET menu_type = '1',
    route_name = 'plugin_management',
    component = 'layout.base',
    i18n_key = '',
    href = ''
WHERE uuid = '7d4c5bf1-4e92-4b50-95f1-0e166b90d001';
