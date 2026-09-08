UPDATE "manage_menu"
SET permissions = '[]',
    update_time = CURRENT_TIMESTAMP
WHERE route_name = 'home';

DELETE FROM "manage_user_role"
WHERE role_uuid = '9c7f5e1a-2b34-4d68-8f90-a1b2c3d4e5f6';

DELETE FROM "manage_role_menu"
WHERE role_uuid = '9c7f5e1a-2b34-4d68-8f90-a1b2c3d4e5f6'
   OR menu_uuid IN (
       'aa10f1d2-8d73-4ff9-9702-4f7395b7a001',
       'aa10f1d2-8d73-4ff9-9702-4f7395b7a002',
       'aa10f1d2-8d73-4ff9-9702-4f7395b7a003',
       'cc10f1d2-8d73-4ff9-9702-4f7395b7a001'
   );

DELETE FROM "manage_role"
WHERE uuid = '9c7f5e1a-2b34-4d68-8f90-a1b2c3d4e5f6';

DELETE FROM "manage_menu"
WHERE uuid IN (
    'aa10f1d2-8d73-4ff9-9702-4f7395b7a002',
    'aa10f1d2-8d73-4ff9-9702-4f7395b7a003',
    'cc10f1d2-8d73-4ff9-9702-4f7395b7a001',
    'aa10f1d2-8d73-4ff9-9702-4f7395b7a001'
);

DROP TABLE IF EXISTS "conversation_token_usage_event";
DROP TABLE IF EXISTS "agent_api_key";
DROP TABLE IF EXISTS "conversation";
DROP TABLE IF EXISTS "conversation_group";
