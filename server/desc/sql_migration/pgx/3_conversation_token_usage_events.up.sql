CREATE TABLE conversation_token_usage_event (
  id bigserial PRIMARY KEY,
  conversation_id varchar(128) NOT NULL,
  user_uuid varchar(64) NOT NULL,
  turn_id varchar(128) NOT NULL,
  last_input_tokens bigint NOT NULL DEFAULT 0,
  last_cached_input_tokens bigint NOT NULL DEFAULT 0,
  last_cache_write_input_tokens bigint NOT NULL DEFAULT 0,
  last_output_tokens bigint NOT NULL DEFAULT 0,
  last_reasoning_output_tokens bigint NOT NULL DEFAULT 0,
  last_total_tokens bigint NOT NULL DEFAULT 0,
  total_input_tokens bigint NOT NULL DEFAULT 0,
  total_cached_input_tokens bigint NOT NULL DEFAULT 0,
  total_cache_write_input_tokens bigint NOT NULL DEFAULT 0,
  total_output_tokens bigint NOT NULL DEFAULT 0,
  total_reasoning_output_tokens bigint NOT NULL DEFAULT 0,
  total_tokens bigint NOT NULL DEFAULT 0,
  model_context_window bigint NULL,
  create_time timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_conversation_token_usage_event_conversation
  ON conversation_token_usage_event(conversation_id, id DESC);

CREATE INDEX idx_conversation_token_usage_event_user
  ON conversation_token_usage_event(user_uuid, id DESC);

CREATE INDEX idx_conversation_token_usage_event_turn
  ON conversation_token_usage_event(conversation_id, turn_id, id DESC);
