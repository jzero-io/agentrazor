INSERT INTO "manage_role_menu" (uuid, role_uuid, menu_uuid, is_home)
SELECT
    'ad8e6f2b-3c45-4e79-8a01-b2c3d4e5f607',
    r.uuid,
    m.uuid,
    1
FROM "manage_role" AS r
JOIN "manage_menu" AS m ON m.route_name = 'home'
WHERE r.code = 'R_AGENT_USER'
ON CONFLICT DO NOTHING;
