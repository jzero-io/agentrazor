-- Casbin is shared infrastructure and may be created only after migrations.
-- Remove Agent policies when it exists, but never drop the shared table.
DO $$
BEGIN
    IF to_regclass('casbin_rule') IS NOT NULL THEN
        EXECUTE 'DELETE FROM "casbin_rule"
                 WHERE (p_type = ''p'' AND (
                     v1 LIKE ''v1:agent:%''
                     OR v1 LIKE ''v1:manage:agent:%''
                 ))
                 OR v0 = ''9c7f5e1a-2b34-4d68-8f90-a1b2c3d4e5f6''';
    END IF;
END
$$;

-- Keep unrelated home permissions intact.
UPDATE "manage_menu"
SET permissions = (
        SELECT COALESCE(
            jsonb_agg(permission ORDER BY position),
            '[]'::jsonb
        )::text
        FROM jsonb_array_elements(COALESCE(NULLIF("manage_menu".permissions, ''), '[]')::jsonb)
            WITH ORDINALITY AS permission_entry(permission, position)
        WHERE permission ->> 'code' <> 'v1:manage:agent:homeOverview'
    ),
    update_time = CURRENT_TIMESTAMP
WHERE route_name = 'home';

WITH RECURSIVE agent_menus AS (
    SELECT uuid
    FROM "manage_menu"
    WHERE uuid = 'aa10f1d2-8d73-4ff9-9702-4f7395b7a001'

    UNION ALL

    SELECT child.uuid
    FROM "manage_menu" child
    JOIN agent_menus parent ON child.parent_uuid = parent.uuid
)
DELETE FROM "manage_role_menu"
WHERE role_uuid = '9c7f5e1a-2b34-4d68-8f90-a1b2c3d4e5f6'
   OR menu_uuid IN (SELECT uuid FROM agent_menus);

DELETE FROM "manage_user_role"
WHERE role_uuid = '9c7f5e1a-2b34-4d68-8f90-a1b2c3d4e5f6';

DELETE FROM "manage_role"
WHERE uuid = '9c7f5e1a-2b34-4d68-8f90-a1b2c3d4e5f6';

WITH RECURSIVE agent_menus AS (
    SELECT uuid
    FROM "manage_menu"
    WHERE uuid = 'aa10f1d2-8d73-4ff9-9702-4f7395b7a001'

    UNION ALL

    SELECT child.uuid
    FROM "manage_menu" child
    JOIN agent_menus parent ON child.parent_uuid = parent.uuid
)
DELETE FROM "manage_menu"
WHERE uuid IN (SELECT uuid FROM agent_menus);

DROP TABLE IF EXISTS "agent_token_quota";
DROP TABLE IF EXISTS "agent_api_key";
DROP TABLE IF EXISTS "conversation_token_usage_event";
DROP TABLE IF EXISTS "conversation";
DROP TABLE IF EXISTS "conversation_group";
