/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `android_enterprises` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `signup_name` varchar(63) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `enterprise_id` varchar(63) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` datetime(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `signup_token` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `pubsub_topic_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `user_id` int unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `android_devices` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int unsigned NOT NULL,
  `device_id` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `enterprise_specific_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `android_policy_id` int unsigned DEFAULT NULL,
  `last_policy_sync_time` datetime(3) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_android_devices_host_id` (`host_id`),
  UNIQUE KEY `idx_android_devices_device_id` (`device_id`),
  UNIQUE KEY `idx_android_devices_enterprise_specific_id` (`enterprise_specific_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `hosts` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `osquery_host_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `detail_updated_at` timestamp NULL DEFAULT NULL,
  `node_key` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `hostname` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `uuid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `platform` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `osquery_version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `os_version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `build` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `platform_like` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `code_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `uptime` bigint NOT NULL DEFAULT '0',
  `memory` bigint NOT NULL DEFAULT '0',
  `cpu_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cpu_subtype` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cpu_brand` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cpu_physical_cores` int NOT NULL DEFAULT '0',
  `cpu_logical_cores` int NOT NULL DEFAULT '0',
  `hardware_vendor` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `hardware_model` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `hardware_version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `hardware_serial` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `computer_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `primary_ip_id` int unsigned DEFAULT NULL,
  `distributed_interval` int DEFAULT '0',
  `logger_tls_period` int DEFAULT '0',
  `config_tls_refresh` int DEFAULT '0',
  `primary_ip` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `primary_mac` varchar(17) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `label_updated_at` timestamp NOT NULL DEFAULT '2000-01-01 00:00:00',
  `last_enrolled_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `refetch_requested` tinyint(1) NOT NULL DEFAULT '0',
  `team_id` int unsigned DEFAULT NULL,
  `policy_updated_at` timestamp NOT NULL DEFAULT '2000-01-01 00:00:00',
  `public_ip` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `orbit_node_key` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `refetch_critical_queries_until` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_osquery_host_id` (`osquery_host_id`),
  UNIQUE KEY `idx_host_unique_nodekey` (`node_key`),
  UNIQUE KEY `idx_host_unique_orbitnodekey` (`orbit_node_key`),
  KEY `fk_hosts_team_id` (`team_id`),
  KEY `hosts_platform_idx` (`platform`),
  KEY `idx_hosts_hardware_serial` (`hardware_serial`),
  KEY `idx_hosts_uuid` (`uuid`),
  CONSTRAINT `hosts_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE SET NULL
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_display_names` (
  `host_id` int unsigned NOT NULL,
  `display_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`host_id`),
  KEY `display_name` (`display_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_mdm` (
  `host_id` int unsigned NOT NULL,
  `enrolled` tinyint(1) NOT NULL DEFAULT '0',
  `server_url` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `installed_from_dep` tinyint(1) NOT NULL DEFAULT '0',
  `mdm_id` int unsigned DEFAULT NULL,
  `is_server` tinyint(1) DEFAULT NULL,
  `fleet_enroll_ref` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `enrollment_status` enum('On (manual)','On (automatic)','Pending','Off') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci GENERATED ALWAYS AS ((case when (`is_server` = 1) then NULL when ((`enrolled` = 1) and (`installed_from_dep` = 0)) then _utf8mb4'On (manual)' when ((`enrolled` = 1) and (`installed_from_dep` = 1)) then _utf8mb4'On (automatic)' when ((`enrolled` = 0) and (`installed_from_dep` = 1)) then _utf8mb4'Pending' when ((`enrolled` = 0) and (`installed_from_dep` = 0)) then _utf8mb4'Off' else NULL end)) VIRTUAL,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`host_id`),
  KEY `host_mdm_mdm_id_idx` (`mdm_id`),
  KEY `host_mdm_enrolled_installed_from_dep_idx` (`enrolled`,`installed_from_dep`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `password` varbinary(255) NOT NULL,
  `salt` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `email` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `admin_forced_password_reset` tinyint(1) NOT NULL DEFAULT '0',
  `gravatar_url` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `position` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `sso_enabled` tinyint NOT NULL DEFAULT '0',
  `global_role` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `api_only` tinyint(1) NOT NULL DEFAULT '0',
  `mfa_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `settings` json NOT NULL DEFAULT (json_object()),
  `invite_id` int unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_unique_email` (`email`),
  UNIQUE KEY `invite_id` (`invite_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `labels` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `query` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `platform` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `label_type` int unsigned NOT NULL DEFAULT '1',
  `label_membership_type` int unsigned NOT NULL DEFAULT '0',
  `author_id` int unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_label_unique_name` (`name`),
  KEY `author_id` (`author_id`),
  FULLTEXT KEY `labels_search` (`name`),
  CONSTRAINT `labels_ibfk_1` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `labels` VALUES (1,'2024-04-03 00:00:00','2024-04-03 00:00:00','macOS 14+ (Sonoma+)','macOS hosts with version 14 and above','select 1 from os_version where platform = \'darwin\' and major >= 14;','darwin',1,0,NULL),(2,'2024-06-28 00:00:00','2024-06-28 00:00:00','iOS','All iOS hosts','','ios',1,1,NULL),(3,'2024-06-28 00:00:00','2024-06-28 00:00:00','iPadOS','All iPadOS hosts','','ipados',1,1,NULL),(4,'2024-09-27 00:00:00','2024-09-27 00:00:00','Fedora Linux','All Fedora hosts','select 1 from os_version where name = \'Fedora Linux\';','rhel',1,0,NULL),(5,'2025-02-25 00:00:00','2025-02-25 00:00:00','Android','All Android hosts','','android',1,1,NULL);
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `label_membership` (
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `label_id` int unsigned NOT NULL,
  `host_id` int unsigned NOT NULL,
  PRIMARY KEY (`host_id`,`label_id`),
  KEY `idx_lm_label_id` (`label_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mobile_device_management_solutions` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `server_url` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mobile_device_management_solutions_name` (`name`,`server_url`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
