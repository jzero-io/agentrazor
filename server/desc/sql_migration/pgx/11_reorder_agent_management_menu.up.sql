-- Keep Conversation Management immediately after Config Management.
UPDATE "manage_menu"
SET "order" = CASE uuid
        WHEN 'aa10f1d2-8d73-4ff9-9702-4f7395b7a003' THEN 1
        WHEN 'aa10f1d2-8d73-4ff9-9702-4f7395b7a004' THEN 2
        WHEN 'aa10f1d2-8d73-4ff9-9702-4f7395b7a002' THEN 3
        WHEN 'cc10f1d2-8d73-4ff9-9702-4f7395b7a001' THEN 4
      END,
    update_time = CURRENT_TIMESTAMP
WHERE uuid IN (
    'aa10f1d2-8d73-4ff9-9702-4f7395b7a003',
    'aa10f1d2-8d73-4ff9-9702-4f7395b7a004',
    'aa10f1d2-8d73-4ff9-9702-4f7395b7a002',
    'cc10f1d2-8d73-4ff9-9702-4f7395b7a001'
);
