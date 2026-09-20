-- Keep the database-backed menu label aligned with the Admin route translation.
UPDATE "manage_menu"
SET menu_name = 'Token 管理',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'cc10f1d2-8d73-4ff9-9702-4f7395b7a001';
