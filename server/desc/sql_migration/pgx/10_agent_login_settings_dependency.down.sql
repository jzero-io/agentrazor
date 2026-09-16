-- Restore independent permission metadata. Role assignments are intentionally
-- retained so rollback never revokes an explicitly or subsequently granted save right.
UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:startChatGPTLogin","desc":"ChatGPT 登录"}]',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'e110f1d2-8d73-4ff9-9702-4f7395b7a002';

UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:loginApiKey","desc":"API Key 登录"}]',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'e110f1d2-8d73-4ff9-9702-4f7395b7a003';
