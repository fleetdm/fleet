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
CREATE TABLE `app_config_json` (
  `id` int(10) unsigned NOT NULL DEFAULT '1',
  `json_value` json NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `app_config_json` VALUES (1,'{\"org_info\": {\"org_name\": \"\", \"org_logo_url\": \"\"}, \"sso_settings\": {\"idp_name\": \"\", \"metadata\": \"\", \"entity_id\": \"\", \"enable_sso\": false, \"issuer_uri\": \"\", \"metadata_url\": \"\", \"idp_image_url\": \"\", \"enable_sso_idp_login\": false}, \"agent_options\": {\"config\": {\"options\": {\"logger_plugin\": \"tls\", \"pack_delimiter\": \"/\", \"logger_tls_period\": 10, \"distributed_plugin\": \"tls\", \"disable_distributed\": false, \"logger_tls_endpoint\": \"/api/v1/osquery/log\", \"distributed_interval\": 10, \"distributed_tls_max_attempts\": 3}, \"decorators\": {\"load\": [\"SELECT uuid AS host_uuid FROM system_info;\", \"SELECT hostname AS hostname FROM system_info;\"]}}, \"overrides\": {}}, \"host_settings\": {\"enable_host_users\": true, \"enable_software_inventory\": false}, \"smtp_settings\": {\"port\": 587, \"domain\": \"\", \"server\": \"\", \"password\": \"\", \"user_name\": \"\", \"configured\": false, \"enable_smtp\": false, \"enable_ssl_tls\": true, \"sender_address\": \"\", \"enable_start_tls\": true, \"verify_ssl_certs\": true, \"authentication_type\": \"0\", \"authentication_method\": \"0\"}, \"server_settings\": {\"server_url\": \"\", \"enable_analytics\": false, \"live_query_disabled\": false}, \"host_expiry_settings\": {\"host_expiry_window\": 0, \"host_expiry_enabled\": false}, \"vulnerability_settings\": {\"databases_path\": \"\"}}','2021-08-24 18:21:29','2021-08-24 18:21:29');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `app_configs` (
  `id` int(10) unsigned NOT NULL DEFAULT '1',
  `org_name` varchar(255) NOT NULL DEFAULT '',
  `org_logo_url` varchar(255) NOT NULL DEFAULT '',
  `server_url` varchar(255) NOT NULL DEFAULT '',
  `smtp_configured` tinyint(1) NOT NULL DEFAULT '0',
  `smtp_sender_address` varchar(255) NOT NULL DEFAULT '',
  `smtp_server` varchar(255) NOT NULL DEFAULT '',
  `smtp_port` int(10) unsigned NOT NULL DEFAULT '587',
  `smtp_authentication_type` int(11) NOT NULL DEFAULT '0',
  `smtp_enable_ssl_tls` tinyint(1) NOT NULL DEFAULT '1',
  `smtp_authentication_method` int(11) NOT NULL DEFAULT '0',
  `smtp_domain` varchar(255) NOT NULL DEFAULT '',
  `smtp_user_name` varchar(255) NOT NULL DEFAULT '',
  `smtp_password` varchar(255) NOT NULL DEFAULT '',
  `smtp_verify_ssl_certs` tinyint(1) NOT NULL DEFAULT '1',
  `smtp_enable_start_tls` tinyint(1) NOT NULL DEFAULT '1',
  `entity_id` varchar(255) NOT NULL DEFAULT '',
  `issuer_uri` varchar(255) NOT NULL DEFAULT '',
  `idp_image_url` varchar(512) NOT NULL DEFAULT '',
  `metadata` text NOT NULL,
  `metadata_url` varchar(512) NOT NULL DEFAULT '',
  `idp_name` varchar(255) NOT NULL DEFAULT '',
  `enable_sso` tinyint(1) NOT NULL DEFAULT '0',
  `fim_interval` int(11) NOT NULL DEFAULT '300',
  `fim_file_accesses` varchar(255) NOT NULL DEFAULT '',
  `host_expiry_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `host_expiry_window` int(11) DEFAULT '0',
  `live_query_disabled` tinyint(1) NOT NULL DEFAULT '0',
  `additional_queries` json DEFAULT NULL,
  `enable_sso_idp_login` tinyint(1) NOT NULL DEFAULT '0',
  `agent_options` json DEFAULT NULL,
  `enable_analytics` tinyint(1) NOT NULL DEFAULT '0',
  `vulnerability_databases_path` text,
  `enable_host_users` tinyint(1) DEFAULT '1',
  `enable_software_inventory` tinyint(1) DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `app_configs` VALUES (1,'','','',0,'','',587,0,1,0,'','','',1,1,'','','','','','',0,300,'',0,0,0,NULL,0,'{\"config\": {\"options\": {\"logger_plugin\": \"tls\", \"pack_delimiter\": \"/\", \"logger_tls_period\": 10, \"distributed_plugin\": \"tls\", \"disable_distributed\": false, \"logger_tls_endpoint\": \"/api/v1/osquery/log\", \"distributed_interval\": 10, \"distributed_tls_max_attempts\": 3}, \"decorators\": {\"load\": [\"SELECT uuid AS host_uuid FROM system_info;\", \"SELECT hostname AS hostname FROM system_info;\"]}}, \"overrides\": {}}',0,NULL,1,0);
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
  `secret` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `team_id` int(10) unsigned DEFAULT NULL,
  UNIQUE KEY `secret` (`secret`),
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
CREATE TABLE `host_software` (
  `host_id` int(10) unsigned NOT NULL,
  `software_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`host_id`,`software_id`),
  KEY `host_software_software_fk` (`software_id`),
  CONSTRAINT `host_software_ibfk_1` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `host_software_ibfk_2` FOREIGN KEY (`software_id`) REFERENCES `software` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_users` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int(10) unsigned NOT NULL,
  `uid` int(10) unsigned NOT NULL,
  `username` varchar(255) DEFAULT NULL,
  `groupname` varchar(255) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `removed_at` timestamp NULL DEFAULT NULL,
  `user_type` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`),
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
  `seen_time` timestamp NULL DEFAULT NULL,
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
  KEY `idx_lm_label_id` (`label_id`),
  CONSTRAINT `fk_lm_host_id` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_lm_label_id` FOREIGN KEY (`label_id`) REFERENCES `labels` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
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
) ENGINE=InnoDB AUTO_INCREMENT=100 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `migration_status_tables` VALUES (1,0,1,'2021-08-24 18:21:23'),(2,20161118193812,1,'2021-08-24 18:21:23'),(3,20161118211713,1,'2021-08-24 18:21:23'),(4,20161118212436,1,'2021-08-24 18:21:23'),(5,20161118212515,1,'2021-08-24 18:21:24'),(6,20161118212528,1,'2021-08-24 18:21:24'),(7,20161118212538,1,'2021-08-24 18:21:24'),(8,20161118212549,1,'2021-08-24 18:21:24'),(9,20161118212557,1,'2021-08-24 18:21:24'),(10,20161118212604,1,'2021-08-24 18:21:24'),(11,20161118212613,1,'2021-08-24 18:21:24'),(12,20161118212621,1,'2021-08-24 18:21:24'),(13,20161118212630,1,'2021-08-24 18:21:24'),(14,20161118212641,1,'2021-08-24 18:21:24'),(15,20161118212649,1,'2021-08-24 18:21:24'),(16,20161118212656,1,'2021-08-24 18:21:24'),(17,20161118212758,1,'2021-08-24 18:21:24'),(18,20161128234849,1,'2021-08-24 18:21:24'),(19,20161230162221,1,'2021-08-24 18:21:24'),(20,20170104113816,1,'2021-08-24 18:21:24'),(21,20170105151732,1,'2021-08-24 18:21:25'),(22,20170108191242,1,'2021-08-24 18:21:25'),(23,20170109094020,1,'2021-08-24 18:21:25'),(24,20170109130438,1,'2021-08-24 18:21:25'),(25,20170110202752,1,'2021-08-24 18:21:25'),(26,20170111133013,1,'2021-08-24 18:21:25'),(27,20170117025759,1,'2021-08-24 18:21:25'),(28,20170118191001,1,'2021-08-24 18:21:25'),(29,20170119234632,1,'2021-08-24 18:21:25'),(30,20170124230432,1,'2021-08-24 18:21:25'),(31,20170127014618,1,'2021-08-24 18:21:25'),(32,20170131232841,1,'2021-08-24 18:21:25'),(33,20170223094154,1,'2021-08-24 18:21:25'),(34,20170306075207,1,'2021-08-24 18:21:26'),(35,20170309100733,1,'2021-08-24 18:21:26'),(36,20170331111922,1,'2021-08-24 18:21:26'),(37,20170502143928,1,'2021-08-24 18:21:26'),(38,20170504130602,1,'2021-08-24 18:21:26'),(39,20170509132100,1,'2021-08-24 18:21:26'),(40,20170519105647,1,'2021-08-24 18:21:26'),(41,20170519105648,1,'2021-08-24 18:21:26'),(42,20170831234300,1,'2021-08-24 18:21:26'),(43,20170831234301,1,'2021-08-24 18:21:26'),(44,20170831234303,1,'2021-08-24 18:21:26'),(45,20171116163618,1,'2021-08-24 18:21:26'),(46,20171219164727,1,'2021-08-24 18:21:26'),(47,20180620164811,1,'2021-08-24 18:21:26'),(48,20180620175054,1,'2021-08-24 18:21:26'),(49,20180620175055,1,'2021-08-24 18:21:26'),(50,20191010101639,1,'2021-08-24 18:21:26'),(51,20191010155147,1,'2021-08-24 18:21:26'),(52,20191220130734,1,'2021-08-24 18:21:26'),(53,20200311140000,1,'2021-08-24 18:21:27'),(54,20200405120000,1,'2021-08-24 18:21:27'),(55,20200407120000,1,'2021-08-24 18:21:27'),(56,20200420120000,1,'2021-08-24 18:21:27'),(57,20200504120000,1,'2021-08-24 18:21:27'),(58,20200512120000,1,'2021-08-24 18:21:27'),(59,20200707120000,1,'2021-08-24 18:21:27'),(60,20201011162341,1,'2021-08-24 18:21:27'),(61,20201021104586,1,'2021-08-24 18:21:27'),(62,20201102112520,1,'2021-08-24 18:21:28'),(63,20201208121729,1,'2021-08-24 18:21:28'),(64,20201215091637,1,'2021-08-24 18:21:28'),(65,20210119174155,1,'2021-08-24 18:21:28'),(66,20210326182902,1,'2021-08-24 18:21:28'),(67,20210421112652,1,'2021-08-24 18:21:28'),(68,20210506095025,1,'2021-08-24 18:21:28'),(69,20210513115729,1,'2021-08-24 18:21:28'),(70,20210526113559,1,'2021-08-24 18:21:28'),(71,20210601000001,1,'2021-08-24 18:21:28'),(72,20210601000002,1,'2021-08-24 18:21:28'),(73,20210601000003,1,'2021-08-24 18:21:28'),(74,20210601000004,1,'2021-08-24 18:21:28'),(75,20210601000005,1,'2021-08-24 18:21:28'),(76,20210601000006,1,'2021-08-24 18:21:28'),(77,20210601000007,1,'2021-08-24 18:21:28'),(78,20210601000008,1,'2021-08-24 18:21:28'),(79,20210606151329,1,'2021-08-24 18:21:28'),(80,20210616163757,1,'2021-08-24 18:21:28'),(81,20210617174723,1,'2021-08-24 18:21:28'),(82,20210622160235,1,'2021-08-24 18:21:28'),(83,20210623100031,1,'2021-08-24 18:21:28'),(84,20210623133615,1,'2021-08-24 18:21:28'),(85,20210708143152,1,'2021-08-24 18:21:28'),(86,20210709124443,1,'2021-08-24 18:21:28'),(87,20210712155608,1,'2021-08-24 18:21:29'),(88,20210714102108,1,'2021-08-24 18:21:29'),(89,20210719153709,1,'2021-08-24 18:21:29'),(90,20210721171531,1,'2021-08-24 18:21:29'),(91,20210723135713,1,'2021-08-24 18:21:29'),(92,20210802135933,1,'2021-08-24 18:21:29'),(93,20210806112844,1,'2021-08-24 18:21:29'),(94,20210810095603,1,'2021-08-24 18:21:29'),(95,20210811150223,1,'2021-08-24 18:21:29'),(96,20210816141251,1,'2021-08-24 18:21:29'),(97,20210818151827,1,'2021-08-24 18:21:29'),(98,20210818182258,1,'2021-08-24 18:21:29'),(99,20210819131107,1,'2021-08-24 18:21:29');
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
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_version` (`name`,`version`,`source`)
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
