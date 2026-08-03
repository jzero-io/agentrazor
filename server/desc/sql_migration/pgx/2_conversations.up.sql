CREATE TABLE conversation_group (
  uuid varchar(64) PRIMARY KEY,
  user_uuid varchar(64) NOT NULL,
  name varchar(80) NOT NULL,
  pinned_at timestamptz NULL,
  created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_conversation_group_user ON conversation_group(user_uuid);

CREATE TABLE conversation (
  id varchar(128) PRIMARY KEY,
  user_uuid varchar(64) NOT NULL,
  group_uuid varchar(64) NULL,
  created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_conversation_user ON conversation(user_uuid);
CREATE INDEX idx_conversation_group ON conversation(group_uuid);
