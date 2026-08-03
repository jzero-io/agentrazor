CREATE TABLE `conversation` (
  `id` varchar(128) NOT NULL,
  `user_uuid` varchar(64) NOT NULL,
  `group_uuid` varchar(64) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_conversation_user` (`user_uuid`),
  KEY `idx_conversation_group` (`group_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
