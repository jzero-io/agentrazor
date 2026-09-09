DELETE FROM "manage_role_menu"
WHERE menu_uuid = 'dc10f1d2-8d73-4ff9-9702-4f7395b7a001';

DELETE FROM "manage_menu"
WHERE uuid = 'dc10f1d2-8d73-4ff9-9702-4f7395b7a001';

DROP INDEX IF EXISTS idx_conversation_token_usage_event_user_time;

DROP TABLE IF EXISTS "agent_token_quota";
