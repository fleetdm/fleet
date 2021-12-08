/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `activities` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `user_id` int(10) unsigned DEFAULT NULL,
  `user_name` varchar(255) DEFAULT NULL,
  `activity_type` varchar(255) NOT NULL,
  `details` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_activities_user_id` (`user_id`),
  CONSTRAINT `activities_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `aggregated_stats` (
  `id` int(10) unsigned NOT NULL,
  `type` varchar(255) NOT NULL,
  `json_value` json NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`,`type`),
  KEY `idx_aggregated_stats_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `app_config_json` (
  `id` int(10) unsigned NOT NULL DEFAULT '1',
  `json_value` json NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `app_config_json` VALUES (1,'{\"org_info\": {\"org_name\": \"\", \"org_logo_url\": \"\"}, \"sso_settings\": {\"idp_name\": \"\", \"metadata\": \"\", \"entity_id\": \"\", \"enable_sso\": false, \"issuer_uri\": \"\", \"metadata_url\": \"\", \"idp_image_url\": \"\", \"enable_sso_idp_login\": false}, \"agent_options\": {\"config\": {\"options\": {\"logger_plugin\": \"tls\", \"pack_delimiter\": \"/\", \"logger_tls_period\": 10, \"distributed_plugin\": \"tls\", \"disable_distributed\": false, \"logger_tls_endpoint\": \"/api/v1/osquery/log\", \"distributed_interval\": 10, \"distributed_tls_max_attempts\": 3}, \"decorators\": {\"load\": [\"SELECT uuid AS host_uuid FROM system_info;\", \"SELECT hostname AS hostname FROM system_info;\"]}}, \"overrides\": {}}, \"host_settings\": {\"enable_host_users\": true, \"enable_software_inventory\": false}, \"smtp_settings\": {\"port\": 587, \"domain\": \"\", \"server\": \"\", \"password\": \"\", \"user_name\": \"\", \"configured\": false, \"enable_smtp\": false, \"enable_ssl_tls\": true, \"sender_address\": \"\", \"enable_start_tls\": true, \"verify_ssl_certs\": true, \"authentication_type\": \"0\", \"authentication_method\": \"0\"}, \"server_settings\": {\"server_url\": \"\", \"enable_analytics\": false, \"deferred_save_host\": false, \"live_query_disabled\": false}, \"webhook_settings\": {\"interval\": \"24h0m0s\", \"host_status_webhook\": {\"days_count\": 0, \"destination_url\": \"\", \"host_percentage\": 0, \"enable_host_status_webhook\": false}}, \"host_expiry_settings\": {\"host_expiry_window\": 0, \"host_expiry_enabled\": false}, \"vulnerability_settings\": {\"databases_path\": \"\"}}','2020-01-01 01:01:01','2020-01-01 01:01:01');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `carve_blocks` (
  `metadata_id` int(10) unsigned NOT NULL,
  `block_id` int(11) NOT NULL,
  `data` longblob,
  PRIMARY KEY (`metadata_id`,`block_id`),
  CONSTRAINT `carve_blocks_ibfk_1` FOREIGN KEY (`metadata_id`) REFERENCES `carve_metadata` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `carve_metadata` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int(10) unsigned NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `name` varchar(255) DEFAULT NULL,
  `block_count` int(10) unsigned NOT NULL,
  `block_size` int(10) unsigned NOT NULL,
  `carve_size` bigint(20) unsigned NOT NULL,
  `carve_id` varchar(64) NOT NULL,
  `request_id` varchar(64) NOT NULL,
  `session_id` varchar(255) NOT NULL,
  `expired` tinyint(4) DEFAULT '0',
  `max_block` int(11) DEFAULT '-1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_session_id` (`session_id`),
  UNIQUE KEY `idx_name` (`name`),
  KEY `host_id` (`host_id`),
  CONSTRAINT `carve_metadata_ibfk_1` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `distributed_query_campaign_targets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `type` int(11) DEFAULT NULL,
  `distributed_query_campaign_id` int(10) unsigned DEFAULT NULL,
  `target_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `distributed_query_campaigns` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `query_id` int(10) unsigned DEFAULT NULL,
  `status` int(11) DEFAULT NULL,
  `user_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `email_changes` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int(10) unsigned NOT NULL,
  `token` varchar(128) NOT NULL,
  `new_email` varchar(255) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_email_changes_token` (`token`) USING BTREE,
  KEY `fk_email_changes_users` (`user_id`),
  CONSTRAINT `fk_email_changes_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `enroll_secrets` (
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `secret` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `team_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`secret`),
  KEY `fk_enroll_secrets_team_id` (`team_id`),
  CONSTRAINT `enroll_secrets_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_additional` (
  `host_id` int(10) unsigned NOT NULL,
  `additional` json DEFAULT NULL,
  PRIMARY KEY (`host_id`),
  CONSTRAINT `host_additional_ibfk_1` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_seen_times` (
  `host_id` int(10) unsigned NOT NULL,
  `seen_time` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`host_id`),
  KEY `idx_host_seen_times_seen_time` (`seen_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_software` (
  `host_id` int(10) unsigned NOT NULL,
  `software_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`host_id`,`software_id`),
  KEY `host_software_software_fk` (`software_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_users` (
  `host_id` int(10) unsigned NOT NULL,
  `uid` int(10) unsigned NOT NULL,
  `username` varchar(255) NOT NULL,
  `groupname` varchar(255) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `removed_at` timestamp NULL DEFAULT NULL,
  `user_type` varchar(255) DEFAULT NULL,
  `shell` varchar(255) DEFAULT '',
  PRIMARY KEY (`host_id`,`uid`,`username`),
  UNIQUE KEY `idx_uid_username` (`host_id`,`uid`,`username`),
  CONSTRAINT `host_users_ibfk_1` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `hosts` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `osquery_host_id` varchar(255) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `detail_updated_at` timestamp NULL DEFAULT NULL,
  `node_key` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `hostname` varchar(255) NOT NULL DEFAULT '',
  `uuid` varchar(255) NOT NULL DEFAULT '',
  `platform` varchar(255) NOT NULL DEFAULT '',
  `osquery_version` varchar(255) NOT NULL DEFAULT '',
  `os_version` varchar(255) NOT NULL DEFAULT '',
  `build` varchar(255) NOT NULL DEFAULT '',
  `platform_like` varchar(255) NOT NULL DEFAULT '',
  `code_name` varchar(255) NOT NULL DEFAULT '',
  `uptime` bigint(20) NOT NULL DEFAULT '0',
  `memory` bigint(20) NOT NULL DEFAULT '0',
  `cpu_type` varchar(255) NOT NULL DEFAULT '',
  `cpu_subtype` varchar(255) NOT NULL DEFAULT '',
  `cpu_brand` varchar(255) NOT NULL DEFAULT '',
  `cpu_physical_cores` int(11) NOT NULL DEFAULT '0',
  `cpu_logical_cores` int(11) NOT NULL DEFAULT '0',
  `hardware_vendor` varchar(255) NOT NULL DEFAULT '',
  `hardware_model` varchar(255) NOT NULL DEFAULT '',
  `hardware_version` varchar(255) NOT NULL DEFAULT '',
  `hardware_serial` varchar(255) NOT NULL DEFAULT '',
  `computer_name` varchar(255) NOT NULL DEFAULT '',
  `primary_ip_id` int(10) unsigned DEFAULT NULL,
  `distributed_interval` int(11) DEFAULT '0',
  `logger_tls_period` int(11) DEFAULT '0',
  `config_tls_refresh` int(11) DEFAULT '0',
  `primary_ip` varchar(45) NOT NULL DEFAULT '',
  `primary_mac` varchar(17) NOT NULL DEFAULT '',
  `label_updated_at` timestamp NOT NULL DEFAULT '2000-01-01 00:00:00',
  `last_enrolled_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `refetch_requested` tinyint(1) NOT NULL DEFAULT '0',
  `team_id` int(10) unsigned DEFAULT NULL,
  `gigs_disk_space_available` float NOT NULL DEFAULT '0',
  `percent_disk_space_available` float NOT NULL DEFAULT '0',
  `policy_updated_at` timestamp NOT NULL DEFAULT '2000-01-01 00:00:00',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_osquery_host_id` (`osquery_host_id`),
  UNIQUE KEY `idx_host_unique_nodekey` (`node_key`),
  KEY `fk_hosts_team_id` (`team_id`),
  FULLTEXT KEY `hosts_search` (`hostname`,`uuid`),
  FULLTEXT KEY `host_ip_mac_search` (`primary_ip`,`primary_mac`),
  CONSTRAINT `hosts_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `invite_teams` (
  `invite_id` int(10) unsigned NOT NULL,
  `team_id` int(10) unsigned NOT NULL,
  `role` varchar(64) NOT NULL,
  PRIMARY KEY (`invite_id`,`team_id`),
  KEY `fk_team_id` (`team_id`),
  CONSTRAINT `invite_teams_ibfk_1` FOREIGN KEY (`invite_id`) REFERENCES `invites` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `invite_teams_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `invites` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `invited_by` int(10) unsigned NOT NULL,
  `email` varchar(255) NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `position` varchar(255) DEFAULT NULL,
  `token` varchar(255) NOT NULL,
  `sso_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `global_role` varchar(64) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_invite_unique_email` (`email`),
  UNIQUE KEY `idx_invite_unique_key` (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `label_membership` (
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `label_id` int(10) unsigned NOT NULL,
  `host_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`host_id`,`label_id`),
  KEY `idx_lm_label_id` (`label_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `labels` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `name` varchar(255) NOT NULL,
  `description` varchar(255) DEFAULT NULL,
  `query` mediumtext NOT NULL,
  `platform` varchar(255) DEFAULT NULL,
  `label_type` int(10) unsigned NOT NULL DEFAULT '1',
  `label_membership_type` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_label_unique_name` (`name`),
  FULLTEXT KEY `labels_search` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `locks` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) DEFAULT NULL,
  `owner` varchar(255) DEFAULT NULL,
  `expires_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `migration_status_tables` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `version_id` bigint(20) NOT NULL,
  `is_applied` tinyint(1) NOT NULL,
  `tstamp` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=116 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `migration_status_tables` VALUES (1,0,1,'2020-01-01 01:01:01'),(2,20161118193812,1,'2020-01-01 01:01:01'),(3,20161118211713,1,'2020-01-01 01:01:01'),(4,20161118212436,1,'2020-01-01 01:01:01'),(5,20161118212515,1,'2020-01-01 01:01:01'),(6,20161118212528,1,'2020-01-01 01:01:01'),(7,20161118212538,1,'2020-01-01 01:01:01'),(8,20161118212549,1,'2020-01-01 01:01:01'),(9,20161118212557,1,'2020-01-01 01:01:01'),(10,20161118212604,1,'2020-01-01 01:01:01'),(11,20161118212613,1,'2020-01-01 01:01:01'),(12,20161118212621,1,'2020-01-01 01:01:01'),(13,20161118212630,1,'2020-01-01 01:01:01'),(14,20161118212641,1,'2020-01-01 01:01:01'),(15,20161118212649,1,'2020-01-01 01:01:01'),(16,20161118212656,1,'2020-01-01 01:01:01'),(17,20161118212758,1,'2020-01-01 01:01:01'),(18,20161128234849,1,'2020-01-01 01:01:01'),(19,20161230162221,1,'2020-01-01 01:01:01'),(20,20170104113816,1,'2020-01-01 01:01:01'),(21,20170105151732,1,'2020-01-01 01:01:01'),(22,20170108191242,1,'2020-01-01 01:01:01'),(23,20170109094020,1,'2020-01-01 01:01:01'),(24,20170109130438,1,'2020-01-01 01:01:01'),(25,20170110202752,1,'2020-01-01 01:01:01'),(26,20170111133013,1,'2020-01-01 01:01:01'),(27,20170117025759,1,'2020-01-01 01:01:01'),(28,20170118191001,1,'2020-01-01 01:01:01'),(29,20170119234632,1,'2020-01-01 01:01:01'),(30,20170124230432,1,'2020-01-01 01:01:01'),(31,20170127014618,1,'2020-01-01 01:01:01'),(32,20170131232841,1,'2020-01-01 01:01:01'),(33,20170223094154,1,'2020-01-01 01:01:01'),(34,20170306075207,1,'2020-01-01 01:01:01'),(35,20170309100733,1,'2020-01-01 01:01:01'),(36,20170331111922,1,'2020-01-01 01:01:01'),(37,20170502143928,1,'2020-01-01 01:01:01'),(38,20170504130602,1,'2020-01-01 01:01:01'),(39,20170509132100,1,'2020-01-01 01:01:01'),(40,20170519105647,1,'2020-01-01 01:01:01'),(41,20170519105648,1,'2020-01-01 01:01:01'),(42,20170831234300,1,'2020-01-01 01:01:01'),(43,20170831234301,1,'2020-01-01 01:01:01'),(44,20170831234303,1,'2020-01-01 01:01:01'),(45,20171116163618,1,'2020-01-01 01:01:01'),(46,20171219164727,1,'2020-01-01 01:01:01'),(47,20180620164811,1,'2020-01-01 01:01:01'),(48,20180620175054,1,'2020-01-01 01:01:01'),(49,20180620175055,1,'2020-01-01 01:01:01'),(50,20191010101639,1,'2020-01-01 01:01:01'),(51,20191010155147,1,'2020-01-01 01:01:01'),(52,20191220130734,1,'2020-01-01 01:01:01'),(53,20200311140000,1,'2020-01-01 01:01:01'),(54,20200405120000,1,'2020-01-01 01:01:01'),(55,20200407120000,1,'2020-01-01 01:01:01'),(56,20200420120000,1,'2020-01-01 01:01:01'),(57,20200504120000,1,'2020-01-01 01:01:01'),(58,20200512120000,1,'2020-01-01 01:01:01'),(59,20200707120000,1,'2020-01-01 01:01:01'),(60,20201011162341,1,'2020-01-01 01:01:01'),(61,20201021104586,1,'2020-01-01 01:01:01'),(62,20201102112520,1,'2020-01-01 01:01:01'),(63,20201208121729,1,'2020-01-01 01:01:01'),(64,20201215091637,1,'2020-01-01 01:01:01'),(65,20210119174155,1,'2020-01-01 01:01:01'),(66,20210326182902,1,'2020-01-01 01:01:01'),(67,20210421112652,1,'2020-01-01 01:01:01'),(68,20210506095025,1,'2020-01-01 01:01:01'),(69,20210513115729,1,'2020-01-01 01:01:01'),(70,20210526113559,1,'2020-01-01 01:01:01'),(71,20210601000001,1,'2020-01-01 01:01:01'),(72,20210601000002,1,'2020-01-01 01:01:01'),(73,20210601000003,1,'2020-01-01 01:01:01'),(74,20210601000004,1,'2020-01-01 01:01:01'),(75,20210601000005,1,'2020-01-01 01:01:01'),(76,20210601000006,1,'2020-01-01 01:01:01'),(77,20210601000007,1,'2020-01-01 01:01:01'),(78,20210601000008,1,'2020-01-01 01:01:01'),(79,20210606151329,1,'2020-01-01 01:01:01'),(80,20210616163757,1,'2020-01-01 01:01:01'),(81,20210617174723,1,'2020-01-01 01:01:01'),(82,20210622160235,1,'2020-01-01 01:01:01'),(83,20210623100031,1,'2020-01-01 01:01:01'),(84,20210623133615,1,'2020-01-01 01:01:01'),(85,20210708143152,1,'2020-01-01 01:01:01'),(86,20210709124443,1,'2020-01-01 01:01:01'),(87,20210712155608,1,'2020-01-01 01:01:01'),(88,20210714102108,1,'2020-01-01 01:01:01'),(89,20210719153709,1,'2020-01-01 01:01:01'),(90,20210721171531,1,'2020-01-01 01:01:01'),(91,20210723135713,1,'2020-01-01 01:01:01'),(92,20210802135933,1,'2020-01-01 01:01:01'),(93,20210806112844,1,'2020-01-01 01:01:01'),(94,20210810095603,1,'2020-01-01 01:01:01'),(95,20210811150223,1,'2020-01-01 01:01:01'),(96,20210818151827,1,'2020-01-01 01:01:01'),(97,20210818151828,1,'2020-01-01 01:01:01'),(98,20210818182258,1,'2020-01-01 01:01:01'),(99,20210819131107,1,'2020-01-01 01:01:01'),(100,20210819143446,1,'2020-01-01 01:01:01'),(101,20210903132338,1,'2020-01-01 01:01:01'),(102,20210915144307,1,'2020-01-01 01:01:01'),(103,20210920155130,1,'2020-01-01 01:01:01'),(104,20210927143115,1,'2020-01-01 01:01:01'),(105,20210927143116,1,'2020-01-01 01:01:01'),(106,20211013133706,1,'2020-01-01 01:01:01'),(107,20211013133707,1,'2020-01-01 01:01:01'),(108,20211102135149,1,'2020-01-01 01:01:01'),(109,20211109121546,1,'2020-01-01 01:01:01'),(110,20211110163320,1,'2020-01-01 01:01:01'),(111,20211116184029,1,'2020-01-01 01:01:01'),(112,20211116184030,1,'2020-01-01 01:01:01'),(113,20211202092042,1,'2020-01-01 01:01:01'),(114,20211202181033,1,'2020-01-01 01:01:01'),(115,20211207161856,1,'2020-01-01 01:01:01');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `network_interfaces` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int(10) unsigned NOT NULL,
  `mac` varchar(255) NOT NULL DEFAULT '',
  `ip_address` varchar(255) NOT NULL DEFAULT '',
  `broadcast` varchar(255) NOT NULL DEFAULT '',
  `ibytes` bigint(20) NOT NULL DEFAULT '0',
  `interface` varchar(255) NOT NULL DEFAULT '',
  `ipackets` bigint(20) NOT NULL DEFAULT '0',
  `last_change` bigint(20) NOT NULL DEFAULT '0',
  `mask` varchar(255) NOT NULL DEFAULT '',
  `metric` int(11) NOT NULL DEFAULT '0',
  `mtu` int(11) NOT NULL DEFAULT '0',
  `obytes` bigint(20) NOT NULL DEFAULT '0',
  `ierrors` bigint(20) NOT NULL DEFAULT '0',
  `oerrors` bigint(20) NOT NULL DEFAULT '0',
  `opackets` bigint(20) NOT NULL DEFAULT '0',
  `point_to_point` varchar(255) NOT NULL DEFAULT '',
  `type` int(11) NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_network_interfaces_unique_ip_host_intf` (`ip_address`,`host_id`,`interface`),
  KEY `idx_network_interfaces_hosts_fk` (`host_id`),
  FULLTEXT KEY `ip_address_search` (`ip_address`),
  CONSTRAINT `network_interfaces_ibfk_1` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `osquery_options` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `override_type` int(1) NOT NULL,
  `override_identifier` varchar(255) NOT NULL DEFAULT '',
  `options` json NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `osquery_options` VALUES (1,0,'','{\"options\": {\"logger_plugin\": \"tls\", \"pack_delimiter\": \"/\", \"logger_tls_period\": 10, \"distributed_plugin\": \"tls\", \"disable_distributed\": false, \"logger_tls_endpoint\": \"/api/v1/osquery/log\", \"distributed_interval\": 10, \"distributed_tls_max_attempts\": 3}, \"decorators\": {\"load\": [\"SELECT uuid AS host_uuid FROM system_info;\", \"SELECT hostname AS hostname FROM system_info;\"]}}');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `pack_targets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `pack_id` int(10) unsigned DEFAULT NULL,
  `type` int(11) DEFAULT NULL,
  `target_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `constraint_pack_target_unique` (`pack_id`,`target_id`,`type`),
  CONSTRAINT `pack_targets_ibfk_1` FOREIGN KEY (`pack_id`) REFERENCES `packs` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `packs` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `disabled` tinyint(1) NOT NULL DEFAULT '0',
  `name` varchar(255) NOT NULL,
  `description` varchar(255) DEFAULT NULL,
  `platform` varchar(255) DEFAULT NULL,
  `pack_type` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_pack_unique_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `password_reset_requests` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `expires_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `user_id` int(10) unsigned NOT NULL,
  `token` varchar(1024) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `policies` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `team_id` int(10) unsigned DEFAULT NULL,
  `resolution` text,
  `name` varchar(255) NOT NULL,
  `query` mediumtext NOT NULL,
  `description` mediumtext NOT NULL,
  `author_id` int(10) unsigned DEFAULT NULL,
  `platforms` varchar(255) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_policies_unique_name` (`name`),
  KEY `idx_policies_author_id` (`author_id`),
  KEY `idx_policies_team_id` (`team_id`),
  CONSTRAINT `policies_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `policies_queries_ibfk_1` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `policy_membership` (
  `policy_id` int(10) unsigned NOT NULL,
  `host_id` int(10) unsigned NOT NULL,
  `passes` tinyint(1) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`policy_id`,`host_id`),
  KEY `idx_policy_membership_passes` (`passes`),
  KEY `idx_policy_membership_policy_id` (`policy_id`),
  KEY `idx_policy_membership_host_id_passes` (`host_id`,`passes`),
  CONSTRAINT `policy_membership_ibfk_1` FOREIGN KEY (`policy_id`) REFERENCES `policies` (`id`) ON DELETE CASCADE,
  CONSTRAINT `policy_membership_ibfk_2` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `queries` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `saved` tinyint(1) NOT NULL DEFAULT '0',
  `name` varchar(255) NOT NULL,
  `description` mediumtext NOT NULL,
  `query` mediumtext NOT NULL,
  `author_id` int(10) unsigned DEFAULT NULL,
  `observer_can_run` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_query_unique_name` (`name`),
  UNIQUE KEY `constraint_query_name_unique` (`name`),
  KEY `author_id` (`author_id`),
  CONSTRAINT `queries_ibfk_1` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `scheduled_queries` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `pack_id` int(10) unsigned DEFAULT NULL,
  `query_id` int(10) unsigned DEFAULT NULL,
  `interval` int(10) unsigned DEFAULT NULL,
  `snapshot` tinyint(1) DEFAULT NULL,
  `removed` tinyint(1) DEFAULT NULL,
  `platform` varchar(255) DEFAULT '',
  `version` varchar(255) DEFAULT '',
  `shard` int(10) unsigned DEFAULT NULL,
  `query_name` varchar(255) NOT NULL,
  `name` varchar(255) NOT NULL,
  `description` varchar(1023) DEFAULT '',
  `denylist` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_names_in_packs` (`name`,`pack_id`),
  KEY `scheduled_queries_pack_id` (`pack_id`),
  KEY `scheduled_queries_query_name` (`query_name`),
  CONSTRAINT `scheduled_queries_pack_id` FOREIGN KEY (`pack_id`) REFERENCES `packs` (`id`) ON DELETE CASCADE,
  CONSTRAINT `scheduled_queries_query_name` FOREIGN KEY (`query_name`) REFERENCES `queries` (`name`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `scheduled_query_stats` (
  `host_id` int(10) unsigned NOT NULL,
  `scheduled_query_id` int(10) unsigned NOT NULL,
  `average_memory` int(11) DEFAULT NULL,
  `denylisted` tinyint(1) DEFAULT NULL,
  `executions` int(11) DEFAULT NULL,
  `schedule_interval` int(11) DEFAULT NULL,
  `last_executed` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `output_size` int(11) DEFAULT NULL,
  `system_time` int(11) DEFAULT NULL,
  `user_time` int(11) DEFAULT NULL,
  `wall_time` int(11) DEFAULT NULL,
  PRIMARY KEY (`host_id`,`scheduled_query_id`),
  KEY `scheduled_query_id` (`scheduled_query_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `sessions` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `accessed_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `user_id` int(10) unsigned NOT NULL,
  `key` varchar(255) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_session_unique_key` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `software` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `version` varchar(255) NOT NULL DEFAULT '',
  `source` varchar(64) NOT NULL,
  `bundle_identifier` varchar(255) DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_version` (`name`,`version`,`source`),
  KEY `software_listing_idx` (`name`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `software_cpe` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `software_id` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `cpe` varchar(255) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_software_cpe_software_id` (`software_id`),
  CONSTRAINT `software_cpe_ibfk_1` FOREIGN KEY (`software_id`) REFERENCES `software` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `software_cve` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `cpe_id` int(10) unsigned DEFAULT NULL,
  `cve` varchar(255) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_cpe_cve` (`cpe_id`,`cve`),
  CONSTRAINT `software_cve_ibfk_1` FOREIGN KEY (`cpe_id`) REFERENCES `software_cpe` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `statistics` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `anonymous_identifier` varchar(255) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `teams` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `name` varchar(255) NOT NULL,
  `description` varchar(1023) NOT NULL DEFAULT '',
  `agent_options` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `user_teams` (
  `user_id` int(10) unsigned NOT NULL,
  `team_id` int(10) unsigned NOT NULL,
  `role` varchar(64) NOT NULL,
  PRIMARY KEY (`user_id`,`team_id`),
  KEY `fk_user_teams_team_id` (`team_id`),
  CONSTRAINT `user_teams_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `user_teams_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `password` varbinary(255) NOT NULL,
  `salt` varchar(255) NOT NULL,
  `name` varchar(255) NOT NULL DEFAULT '',
  `email` varchar(255) NOT NULL,
  `admin_forced_password_reset` tinyint(1) NOT NULL DEFAULT '0',
  `gravatar_url` varchar(255) NOT NULL DEFAULT '',
  `position` varchar(255) NOT NULL DEFAULT '',
  `sso_enabled` tinyint(4) NOT NULL DEFAULT '0',
  `global_role` varchar(64) DEFAULT NULL,
  `api_only` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_unique_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
