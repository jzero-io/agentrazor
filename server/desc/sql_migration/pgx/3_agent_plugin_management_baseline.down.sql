-- Revert plugin management and Agent Management menu configuration.


UPDATE "manage_menu"
SET "order" = CASE uuid
        WHEN 'aa10f1d2-8d73-4ff9-9702-4f7395b7a002' THEN 2
        WHEN 'cc10f1d2-8d73-4ff9-9702-4f7395b7a001' THEN 3
      END,
    update_time = CURRENT_TIMESTAMP
WHERE uuid IN (
    'aa10f1d2-8d73-4ff9-9702-4f7395b7a002',
    'cc10f1d2-8d73-4ff9-9702-4f7395b7a001'
);

UPDATE "manage_menu"
SET menu_name = 'Token 消耗',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'cc10f1d2-8d73-4ff9-9702-4f7395b7a001';

DELETE FROM "manage_role_menu"
WHERE menu_uuid = 'aa10f1d2-8d73-4ff9-9702-4f7395b7a004';

DELETE FROM "manage_menu"
WHERE uuid = 'aa10f1d2-8d73-4ff9-9702-4f7395b7a004';

DELETE FROM "manage_role_menu"
WHERE menu_uuid = '7d4c5bf1-4e92-4b50-95f1-0e166b90d001';

DELETE FROM "manage_menu"
WHERE uuid = '7d4c5bf1-4e92-4b50-95f1-0e166b90d001';
