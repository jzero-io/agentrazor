CREATE TABLE IF NOT EXISTS telegram_plugin_configuration (
  id smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  encrypted_value text NOT NULL,
  updated_by varchar(64) NOT NULL DEFAULT '',
  create_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS telegram_chat_conversation (
  chat_id bigint PRIMARY KEY,
  conversation_id varchar(64) NOT NULL,
  title varchar(255) NOT NULL DEFAULT '',
  create_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS telegram_processed_update (
  update_id bigint PRIMARY KEY,
  processed_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_telegram_processed_update_processed_at
  ON telegram_processed_update (processed_at);
