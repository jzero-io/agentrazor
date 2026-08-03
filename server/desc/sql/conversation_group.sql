CREATE TABLE `conversation_group` (
  `uuid` varchar(64) NOT NULL,
  `user_uuid` varchar(64) NOT NULL,
  `name` varchar(80) NOT NULL,
  `pinned_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`uuid`),
  KEY `idx_conversation_group_user` (`user_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
