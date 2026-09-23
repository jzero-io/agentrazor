DROP INDEX IF EXISTS idx_conversation_character;
ALTER TABLE "conversation" DROP CONSTRAINT IF EXISTS fk_conversation_character;
ALTER TABLE "conversation" DROP COLUMN IF EXISTS character_uuid;
DROP TABLE IF EXISTS "agent_character";
