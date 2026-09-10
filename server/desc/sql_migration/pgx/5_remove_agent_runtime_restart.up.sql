DELETE FROM "casbin_rule"
WHERE p_type = 'p'
  AND v1 = 'v1:manage:agent:restartRuntime';

DELETE FROM "manage_role_menu"
WHERE menu_uuid = 'e110f1d2-8d73-4ff9-9702-4f7395b7a005';

DELETE FROM "manage_menu"
WHERE uuid = 'e110f1d2-8d73-4ff9-9702-4f7395b7a005';

UPDATE "manage_menu"
SET permissions = COALESCE((
        SELECT jsonb_agg(permission)
        FROM jsonb_array_elements(COALESCE(NULLIF(permissions, ''), '[]')::jsonb) AS permission
        WHERE permission ->> 'code' <> 'v1:manage:agent:restartRuntime'
    ), '[]'::jsonb)::text,
    update_time = CURRENT_TIMESTAMP
WHERE permissions LIKE '%v1:manage:agent:restartRuntime%';
