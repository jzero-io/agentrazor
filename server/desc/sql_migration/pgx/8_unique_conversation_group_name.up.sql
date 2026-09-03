CREATE UNIQUE INDEX uk_conversation_group_user_name
  ON conversation_group(user_uuid, name);
