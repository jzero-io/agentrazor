CREATE TABLE "agent_character" (
    uuid varchar(36) NOT NULL,
    user_uuid varchar(36),
    name varchar(80) NOT NULL,
    description varchar(240) NOT NULL DEFAULT '',
    prompt text NOT NULL,
    is_builtin boolean NOT NULL DEFAULT false,
    sort integer NOT NULL DEFAULT 0,
    create_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (uuid),
    CONSTRAINT fk_agent_character_user
        FOREIGN KEY (user_uuid) REFERENCES "manage_user" (uuid) ON DELETE CASCADE,
    CONSTRAINT chk_agent_character_owner
        CHECK ((is_builtin = true AND user_uuid IS NULL) OR (is_builtin = false AND user_uuid IS NOT NULL)),
    CONSTRAINT chk_agent_character_prompt
        CHECK (length(btrim(prompt)) > 0 AND length(prompt) <= 16000)
);

CREATE UNIQUE INDEX uk_agent_character_builtin_name
    ON "agent_character" (name) WHERE is_builtin = true;
CREATE UNIQUE INDEX uk_agent_character_user_name
    ON "agent_character" (user_uuid, name) WHERE is_builtin = false;
CREATE INDEX idx_agent_character_user ON "agent_character" (user_uuid, sort, create_time);

ALTER TABLE "conversation"
    ADD COLUMN character_uuid varchar(36),
    ADD CONSTRAINT fk_conversation_character
        FOREIGN KEY (character_uuid) REFERENCES "agent_character" (uuid) ON DELETE SET NULL;

CREATE INDEX idx_conversation_character ON "conversation" (character_uuid);
