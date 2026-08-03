CREATE TABLE conversation_group (
  uuid TEXT PRIMARY KEY,
  user_uuid TEXT NOT NULL,
  name TEXT NOT NULL,
  pinned_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_conversation_group_user ON conversation_group(user_uuid);

CREATE TABLE conversation (
  id TEXT PRIMARY KEY,
  user_uuid TEXT NOT NULL,
  group_uuid TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_conversation_user ON conversation(user_uuid);
CREATE INDEX idx_conversation_group ON conversation(group_uuid);
