CREATE TABLE agent_api_key (
  uuid varchar(36) PRIMARY KEY,
  user_uuid varchar(36) NOT NULL,
  key_hash char(64) NOT NULL UNIQUE,
  key_hint varchar(32) NOT NULL,
  create_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_agent_api_key_user
    FOREIGN KEY (user_uuid) REFERENCES manage_user(uuid) ON DELETE CASCADE
);

CREATE INDEX idx_agent_api_key_user ON agent_api_key(user_uuid);

