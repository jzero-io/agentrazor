CREATE TABLE `conversation_group` (
  `uuid` varchar(64) NOT NULL,
  `user_uuid` varchar(64) NOT NULL,
  `name` varchar(80) NOT NULL,
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`uuid`),
  KEY `idx_conversation_group_user` (`user_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `conversation` (
  `id` varchar(128) NOT NULL,
  `user_uuid` varchar(64) NOT NULL,
  `group_uuid` varchar(64) DEFAULT NULL,
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_conversation_user` (`user_uuid`),
  KEY `idx_conversation_group` (`group_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
