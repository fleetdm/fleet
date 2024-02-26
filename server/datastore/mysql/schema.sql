/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `activities` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `user_id` int(10) unsigned DEFAULT NULL,
  `user_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `activity_type` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `details` json DEFAULT NULL,
  `streamed` tinyint(1) NOT NULL DEFAULT '0',
  `user_email` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `fk_activities_user_id` (`user_id`),
  KEY `activities_streamed_idx` (`streamed`),
  KEY `activities_created_at_idx` (`created_at`),
  CONSTRAINT `activities_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `aggregated_stats` (
  `id` bigint(20) unsigned NOT NULL,
  `type` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `json_value` json NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `global_stats` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`,`type`,`global_stats`),
  KEY `idx_aggregated_stats_updated_at` (`updated_at`),
  KEY `aggregated_stats_type_idx` (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `app_config_json` VALUES (1,'{\"mdm\": {\"macos_setup\": {\"bootstrap_package\": null, \"macos_setup_assistant\": null, \"enable_end_user_authentication\": false}, \"macos_updates\": {\"deadline\": null, \"minimum_version\": null}, \"macos_settings\": {\"custom_settings\": null}, \"macos_migration\": {\"mode\": \"\", \"enable\": false, \"webhook_url\": \"\"}, \"windows_updates\": {\"deadline_days\": null, \"grace_period_days\": null}, \"windows_settings\": {\"custom_settings\": null}, \"apple_bm_default_team\": \"\", \"apple_bm_terms_expired\": false, \"enable_disk_encryption\": false, \"enabled_and_configured\": false, \"end_user_authentication\": {\"idp_name\": \"\", \"metadata\": \"\", \"entity_id\": \"\", \"issuer_uri\": \"\", \"metadata_url\": \"\"}, \"windows_enabled_and_configured\": false, \"apple_bm_enabled_and_configured\": false}, \"scripts\": null, \"features\": {\"enable_host_users\": true, \"enable_software_inventory\": false}, \"org_info\": {\"org_name\": \"\", \"contact_url\": \"\", \"org_logo_url\": \"\", \"org_logo_url_light_background\": \"\"}, \"integrations\": {\"jira\": null, \"zendesk\": null}, \"sso_settings\": {\"idp_name\": \"\", \"metadata\": \"\", \"entity_id\": \"\", \"enable_sso\": false, \"issuer_uri\": \"\", \"metadata_url\": \"\", \"idp_image_url\": \"\", \"enable_jit_role_sync\": false, \"enable_sso_idp_login\": false, \"enable_jit_provisioning\": false}, \"agent_options\": {\"config\": {\"options\": {\"logger_plugin\": \"tls\", \"pack_delimiter\": \"/\", \"logger_tls_period\": 10, \"distributed_plugin\": \"tls\", \"disable_distributed\": false, \"logger_tls_endpoint\": \"/api/osquery/log\", \"distributed_interval\": 10, \"distributed_tls_max_attempts\": 3}, \"decorators\": {\"load\": [\"SELECT uuid AS host_uuid FROM system_info;\", \"SELECT hostname AS hostname FROM system_info;\"]}}, \"overrides\": {}}, \"fleet_desktop\": {\"transparency_url\": \"\"}, \"smtp_settings\": {\"port\": 587, \"domain\": \"\", \"server\": \"\", \"password\": \"\", \"user_name\": \"\", \"configured\": false, \"enable_smtp\": false, \"enable_ssl_tls\": true, \"sender_address\": \"\", \"enable_start_tls\": true, \"verify_ssl_certs\": true, \"authentication_type\": \"0\", \"authentication_method\": \"0\"}, \"server_settings\": {\"server_url\": \"\", \"enable_analytics\": false, \"scripts_disabled\": false, \"deferred_save_host\": false, \"live_query_disabled\": false, \"query_reports_disabled\": false}, \"webhook_settings\": {\"interval\": \"0s\", \"host_status_webhook\": {\"days_count\": 0, \"destination_url\": \"\", \"host_percentage\": 0, \"enable_host_status_webhook\": false}, \"vulnerabilities_webhook\": {\"destination_url\": \"\", \"host_batch_size\": 0, \"enable_vulnerabilities_webhook\": false}, \"failing_policies_webhook\": {\"policy_ids\": null, \"destination_url\": \"\", \"host_batch_size\": 0, \"enable_failing_policies_webhook\": false}}, \"host_expiry_settings\": {\"host_expiry_window\": 0, \"host_expiry_enabled\": false}, \"vulnerability_settings\": {\"databases_path\": \"\"}}','2020-01-01 01:01:01','2020-01-01 01:01:01');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `carve_blocks` (
  `metadata_id` int(10) unsigned NOT NULL,
  `block_id` int(11) NOT NULL,
  `data` longblob,
  PRIMARY KEY (`metadata_id`,`block_id`),
  CONSTRAINT `carve_blocks_ibfk_1` FOREIGN KEY (`metadata_id`) REFERENCES `carve_metadata` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `carve_metadata` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int(10) unsigned NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `block_count` int(10) unsigned NOT NULL,
  `block_size` int(10) unsigned NOT NULL,
  `carve_size` bigint(20) unsigned NOT NULL,
  `carve_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `request_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `session_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `expired` tinyint(4) DEFAULT '0',
  `max_block` int(11) DEFAULT '-1',
  `error` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_session_id` (`session_id`),
  UNIQUE KEY `idx_name` (`name`),
  KEY `host_id` (`host_id`),
  CONSTRAINT `carve_metadata_ibfk_1` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `cron_stats` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `instance` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `stats_type` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_cron_stats_name_created_at` (`name`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `cve_meta` (
  `cve` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `cvss_score` double DEFAULT NULL,
  `epss_probability` double DEFAULT NULL,
  `cisa_known_exploit` tinyint(1) DEFAULT NULL,
  `published` timestamp NULL DEFAULT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`cve`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `distributed_query_campaign_targets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `type` int(11) DEFAULT NULL,
  `distributed_query_campaign_id` int(10) unsigned DEFAULT NULL,
  `target_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `email_changes` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int(10) unsigned NOT NULL,
  `token` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL,
  `new_email` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_email_changes_token` (`token`) USING BTREE,
  KEY `fk_email_changes_users` (`user_id`),
  CONSTRAINT `fk_email_changes_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `eulas` (
  `id` int(10) unsigned NOT NULL,
  `token` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `bytes` longblob,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_activities` (
  `host_id` int(10) unsigned NOT NULL,
  `activity_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`host_id`,`activity_id`),
  KEY `fk_host_activities_activity_id` (`activity_id`),
  CONSTRAINT `host_activities_ibfk_1` FOREIGN KEY (`activity_id`) REFERENCES `activities` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_additional` (
  `host_id` int(10) unsigned NOT NULL,
  `additional` json DEFAULT NULL,
  PRIMARY KEY (`host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_batteries` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int(10) unsigned NOT NULL,
  `serial_number` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `cycle_count` int(10) NOT NULL,
  `health` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_host_batteries_host_id_serial_number` (`host_id`,`serial_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_dep_assignments` (
  `host_id` int(10) unsigned NOT NULL,
  `added_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_device_auth` (
  `host_id` int(10) unsigned NOT NULL,
  `token` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`host_id`),
  UNIQUE KEY `idx_host_device_auth_token` (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_disk_encryption_keys` (
  `host_id` int(10) unsigned NOT NULL,
  `base64_encrypted` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `decryptable` tinyint(1) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `reset_requested` tinyint(1) NOT NULL DEFAULT '0',
  `client_error` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`host_id`),
  KEY `idx_host_disk_encryption_keys_decryptable` (`decryptable`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_disks` (
  `host_id` int(10) unsigned NOT NULL,
  `gigs_disk_space_available` decimal(10,2) NOT NULL DEFAULT '0.00',
  `percent_disk_space_available` decimal(10,2) NOT NULL DEFAULT '0.00',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `encrypted` tinyint(1) DEFAULT NULL,
  `gigs_total_disk_space` decimal(10,2) NOT NULL DEFAULT '0.00',
  PRIMARY KEY (`host_id`),
  KEY `idx_host_disks_gigs_disk_space_available` (`gigs_disk_space_available`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_display_names` (
  `host_id` int(10) unsigned NOT NULL,
  `display_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`host_id`),
  KEY `display_name` (`display_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_emails` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int(10) unsigned NOT NULL,
  `email` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_host_emails_host_id_email` (`host_id`,`email`),
  KEY `idx_host_emails_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_mdm` (
  `host_id` int(10) unsigned NOT NULL,
  `enrolled` tinyint(1) NOT NULL DEFAULT '0',
  `server_url` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `installed_from_dep` tinyint(1) NOT NULL DEFAULT '0',
  `mdm_id` int(10) unsigned DEFAULT NULL,
  `is_server` tinyint(1) DEFAULT NULL,
  `fleet_enroll_ref` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`host_id`),
  KEY `host_mdm_mdm_id_idx` (`mdm_id`),
  KEY `host_mdm_enrolled_installed_from_dep_idx` (`enrolled`,`installed_from_dep`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_mdm_actions` (
  `host_id` int(10) unsigned NOT NULL,
  `lock_ref` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `wipe_ref` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `unlock_pin` varchar(6) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `unlock_ref` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_mdm_apple_bootstrap_packages` (
  `host_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`host_uuid`),
  KEY `command_uuid` (`command_uuid`),
  CONSTRAINT `host_mdm_apple_bootstrap_packages_ibfk_1` FOREIGN KEY (`command_uuid`) REFERENCES `nano_commands` (`command_uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_mdm_apple_profiles` (
  `profile_identifier` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `host_uuid` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `operation_type` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `detail` text COLLATE utf8mb4_unicode_ci,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `profile_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `checksum` binary(16) NOT NULL,
  `retries` tinyint(3) unsigned NOT NULL DEFAULT '0',
  `profile_uuid` varchar(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`host_uuid`,`profile_uuid`),
  KEY `status` (`status`),
  KEY `operation_type` (`operation_type`),
  CONSTRAINT `host_mdm_apple_profiles_ibfk_1` FOREIGN KEY (`status`) REFERENCES `mdm_delivery_status` (`status`) ON UPDATE CASCADE,
  CONSTRAINT `host_mdm_apple_profiles_ibfk_2` FOREIGN KEY (`operation_type`) REFERENCES `mdm_operation_types` (`operation_type`) ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_mdm_windows_profiles` (
  `host_uuid` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `operation_type` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `detail` text COLLATE utf8mb4_unicode_ci,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `profile_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `retries` tinyint(3) unsigned NOT NULL DEFAULT '0',
  `profile_uuid` varchar(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`host_uuid`,`profile_uuid`),
  KEY `status` (`status`),
  KEY `operation_type` (`operation_type`),
  CONSTRAINT `host_mdm_windows_profiles_ibfk_1` FOREIGN KEY (`status`) REFERENCES `mdm_delivery_status` (`status`) ON UPDATE CASCADE,
  CONSTRAINT `host_mdm_windows_profiles_ibfk_2` FOREIGN KEY (`operation_type`) REFERENCES `mdm_operation_types` (`operation_type`) ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_munki_info` (
  `host_id` int(10) unsigned NOT NULL,
  `version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_munki_issues` (
  `host_id` int(10) unsigned NOT NULL,
  `munki_issue_id` int(10) unsigned NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`host_id`,`munki_issue_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_operating_system` (
  `host_id` int(10) unsigned NOT NULL,
  `os_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`host_id`),
  KEY `idx_host_operating_system_id` (`os_id`),
  CONSTRAINT `host_operating_system_ibfk_1` FOREIGN KEY (`os_id`) REFERENCES `operating_systems` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_orbit_info` (
  `host_id` int(10) unsigned NOT NULL,
  `version` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`host_id`),
  KEY `idx_host_orbit_info_version` (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_script_results` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int(10) unsigned NOT NULL,
  `execution_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `script_contents` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `output` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `runtime` int(10) unsigned NOT NULL DEFAULT '0',
  `exit_code` int(10) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `script_id` int(10) unsigned DEFAULT NULL,
  `user_id` int(10) unsigned DEFAULT NULL,
  `sync_request` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_host_script_results_execution_id` (`execution_id`),
  KEY `idx_host_script_results_host_exit_created` (`host_id`,`exit_code`,`created_at`),
  KEY `fk_host_script_results_script_id` (`script_id`),
  KEY `idx_host_script_created_at` (`host_id`,`script_id`,`created_at`),
  KEY `fk_host_script_results_user_id` (`user_id`),
  CONSTRAINT `fk_host_script_results_script_id` FOREIGN KEY (`script_id`) REFERENCES `scripts` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_host_script_results_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_seen_times` (
  `host_id` int(10) unsigned NOT NULL,
  `seen_time` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`host_id`),
  KEY `idx_host_seen_times_seen_time` (`seen_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_software` (
  `host_id` int(10) unsigned NOT NULL,
  `software_id` bigint(20) unsigned NOT NULL,
  `last_opened_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`host_id`,`software_id`),
  KEY `host_software_software_fk` (`software_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_software_installed_paths` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int(10) unsigned NOT NULL,
  `software_id` bigint(20) unsigned NOT NULL,
  `installed_path` text COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `host_id_software_id_idx` (`host_id`,`software_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_updates` (
  `host_id` int(10) unsigned NOT NULL,
  `software_updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `host_users` (
  `host_id` int(10) unsigned NOT NULL,
  `uid` int(10) unsigned NOT NULL,
  `username` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `groupname` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `removed_at` timestamp NULL DEFAULT NULL,
  `user_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `shell` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '',
  PRIMARY KEY (`host_id`,`uid`,`username`),
  UNIQUE KEY `idx_uid_username` (`host_id`,`uid`,`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `hosts` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `osquery_host_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `detail_updated_at` timestamp NULL DEFAULT NULL,
  `node_key` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `hostname` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `uuid` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `platform` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `osquery_version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `os_version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `build` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `platform_like` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `code_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `uptime` bigint(20) NOT NULL DEFAULT '0',
  `memory` bigint(20) NOT NULL DEFAULT '0',
  `cpu_type` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cpu_subtype` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cpu_brand` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cpu_physical_cores` int(11) NOT NULL DEFAULT '0',
  `cpu_logical_cores` int(11) NOT NULL DEFAULT '0',
  `hardware_vendor` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `hardware_model` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `hardware_version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `hardware_serial` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `computer_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `primary_ip_id` int(10) unsigned DEFAULT NULL,
  `distributed_interval` int(11) DEFAULT '0',
  `logger_tls_period` int(11) DEFAULT '0',
  `config_tls_refresh` int(11) DEFAULT '0',
  `primary_ip` varchar(45) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `primary_mac` varchar(17) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `label_updated_at` timestamp NOT NULL DEFAULT '2000-01-01 00:00:00',
  `last_enrolled_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `refetch_requested` tinyint(1) NOT NULL DEFAULT '0',
  `team_id` int(10) unsigned DEFAULT NULL,
  `policy_updated_at` timestamp NOT NULL DEFAULT '2000-01-01 00:00:00',
  `public_ip` varchar(45) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `orbit_node_key` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `refetch_critical_queries_until` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_osquery_host_id` (`osquery_host_id`),
  UNIQUE KEY `idx_host_unique_nodekey` (`node_key`),
  UNIQUE KEY `idx_host_unique_orbitnodekey` (`orbit_node_key`),
  KEY `fk_hosts_team_id` (`team_id`),
  KEY `hosts_platform_idx` (`platform`),
  KEY `idx_hosts_hardware_serial` (`hardware_serial`),
  FULLTEXT KEY `host_ip_mac_search` (`primary_ip`,`primary_mac`),
  FULLTEXT KEY `hosts_search` (`hostname`,`uuid`,`computer_name`),
  CONSTRAINT `hosts_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `invite_teams` (
  `invite_id` int(10) unsigned NOT NULL,
  `team_id` int(10) unsigned NOT NULL,
  `role` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`invite_id`,`team_id`),
  KEY `fk_team_id` (`team_id`),
  CONSTRAINT `invite_teams_ibfk_1` FOREIGN KEY (`invite_id`) REFERENCES `invites` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `invite_teams_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `invites` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `invited_by` int(10) unsigned NOT NULL,
  `email` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `position` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `token` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sso_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `global_role` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_invite_unique_email` (`email`),
  UNIQUE KEY `idx_invite_unique_key` (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `jobs` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `args` json DEFAULT NULL,
  `state` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `retries` int(11) NOT NULL DEFAULT '0',
  `error` text COLLATE utf8mb4_unicode_ci,
  `not_before` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `labels` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `query` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `platform` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `label_type` int(10) unsigned NOT NULL DEFAULT '1',
  `label_membership_type` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_label_unique_name` (`name`),
  FULLTEXT KEY `labels_search` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `locks` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `owner` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `expires_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mdm_apple_bootstrap_packages` (
  `team_id` int(10) unsigned NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sha256` binary(32) NOT NULL,
  `bytes` longblob,
  `token` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`team_id`),
  UNIQUE KEY `idx_token` (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mdm_apple_configuration_profiles` (
  `profile_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `team_id` int(10) unsigned NOT NULL DEFAULT '0',
  `identifier` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `mobileconfig` blob NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `uploaded_at` timestamp NULL DEFAULT NULL,
  `checksum` binary(16) NOT NULL,
  `profile_uuid` varchar(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`profile_uuid`),
  UNIQUE KEY `idx_mdm_apple_config_prof_team_identifier` (`team_id`,`identifier`),
  UNIQUE KEY `idx_mdm_apple_config_prof_team_name` (`team_id`,`name`),
  UNIQUE KEY `idx_mdm_apple_config_prof_id` (`profile_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mdm_apple_default_setup_assistants` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `team_id` int(10) unsigned DEFAULT NULL,
  `global_or_team_id` int(10) unsigned NOT NULL DEFAULT '0',
  `profile_uuid` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mdm_default_setup_assistant_global_or_team_id` (`global_or_team_id`),
  KEY `fk_mdm_default_setup_assistant_team_id` (`team_id`),
  CONSTRAINT `mdm_apple_default_setup_assistants_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mdm_apple_enrollment_profiles` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `token` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `type` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'automatic',
  `dep_profile` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_type` (`type`),
  UNIQUE KEY `idx_token` (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mdm_apple_installers` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `size` bigint(20) NOT NULL,
  `manifest` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `installer` longblob,
  `url_token` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mdm_apple_setup_assistants` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `team_id` int(10) unsigned DEFAULT NULL,
  `global_or_team_id` int(10) unsigned NOT NULL DEFAULT '0',
  `name` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `profile` json NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `profile_uuid` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mdm_setup_assistant_global_or_team_id` (`global_or_team_id`),
  KEY `fk_mdm_setup_assistant_team_id` (`team_id`),
  CONSTRAINT `mdm_apple_setup_assistants_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mdm_configuration_profile_labels` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `apple_profile_uuid` varchar(37) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `windows_profile_uuid` varchar(37) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `label_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `label_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mdm_configuration_profile_labels_apple_label_name` (`apple_profile_uuid`,`label_name`),
  UNIQUE KEY `idx_mdm_configuration_profile_labels_windows_label_name` (`windows_profile_uuid`,`label_name`),
  KEY `label_id` (`label_id`),
  CONSTRAINT `mdm_configuration_profile_labels_ibfk_1` FOREIGN KEY (`apple_profile_uuid`) REFERENCES `mdm_apple_configuration_profiles` (`profile_uuid`) ON DELETE CASCADE,
  CONSTRAINT `mdm_configuration_profile_labels_ibfk_2` FOREIGN KEY (`windows_profile_uuid`) REFERENCES `mdm_windows_configuration_profiles` (`profile_uuid`) ON DELETE CASCADE,
  CONSTRAINT `mdm_configuration_profile_labels_ibfk_3` FOREIGN KEY (`label_id`) REFERENCES `labels` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mdm_delivery_status` (
  `status` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `mdm_delivery_status` VALUES ('failed'),('pending'),('verified'),('verifying');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mdm_idp_accounts` (
  `uuid` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `username` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `fullname` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `email` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `unique_idp_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mdm_operation_types` (
  `operation_type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`operation_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `mdm_operation_types` VALUES ('install'),('remove');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mdm_windows_configuration_profiles` (
  `team_id` int(10) unsigned NOT NULL DEFAULT '0',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `syncml` mediumblob NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `uploaded_at` timestamp NULL DEFAULT NULL,
  `profile_uuid` varchar(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`profile_uuid`),
  UNIQUE KEY `idx_mdm_windows_configuration_profiles_team_id_name` (`team_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mdm_windows_enrollments` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `mdm_device_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `mdm_hardware_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `device_state` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `device_type` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `device_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `enroll_type` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `enroll_user_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `enroll_proto_version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `enroll_client_version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `not_in_oobe` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `host_uuid` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_type` (`mdm_hardware_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
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
) ENGINE=InnoDB AUTO_INCREMENT=251 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `migration_status_tables` VALUES (1,0,1,'2020-01-01 01:01:01'),(2,20161118193812,1,'2020-01-01 01:01:01'),(3,20161118211713,1,'2020-01-01 01:01:01'),(4,20161118212436,1,'2020-01-01 01:01:01'),(5,20161118212515,1,'2020-01-01 01:01:01'),(6,20161118212528,1,'2020-01-01 01:01:01'),(7,20161118212538,1,'2020-01-01 01:01:01'),(8,20161118212549,1,'2020-01-01 01:01:01'),(9,20161118212557,1,'2020-01-01 01:01:01'),(10,20161118212604,1,'2020-01-01 01:01:01'),(11,20161118212613,1,'2020-01-01 01:01:01'),(12,20161118212621,1,'2020-01-01 01:01:01'),(13,20161118212630,1,'2020-01-01 01:01:01'),(14,20161118212641,1,'2020-01-01 01:01:01'),(15,20161118212649,1,'2020-01-01 01:01:01'),(16,20161118212656,1,'2020-01-01 01:01:01'),(17,20161118212758,1,'2020-01-01 01:01:01'),(18,20161128234849,1,'2020-01-01 01:01:01'),(19,20161230162221,1,'2020-01-01 01:01:01'),(20,20170104113816,1,'2020-01-01 01:01:01'),(21,20170105151732,1,'2020-01-01 01:01:01'),(22,20170108191242,1,'2020-01-01 01:01:01'),(23,20170109094020,1,'2020-01-01 01:01:01'),(24,20170109130438,1,'2020-01-01 01:01:01'),(25,20170110202752,1,'2020-01-01 01:01:01'),(26,20170111133013,1,'2020-01-01 01:01:01'),(27,20170117025759,1,'2020-01-01 01:01:01'),(28,20170118191001,1,'2020-01-01 01:01:01'),(29,20170119234632,1,'2020-01-01 01:01:01'),(30,20170124230432,1,'2020-01-01 01:01:01'),(31,20170127014618,1,'2020-01-01 01:01:01'),(32,20170131232841,1,'2020-01-01 01:01:01'),(33,20170223094154,1,'2020-01-01 01:01:01'),(34,20170306075207,1,'2020-01-01 01:01:01'),(35,20170309100733,1,'2020-01-01 01:01:01'),(36,20170331111922,1,'2020-01-01 01:01:01'),(37,20170502143928,1,'2020-01-01 01:01:01'),(38,20170504130602,1,'2020-01-01 01:01:01'),(39,20170509132100,1,'2020-01-01 01:01:01'),(40,20170519105647,1,'2020-01-01 01:01:01'),(41,20170519105648,1,'2020-01-01 01:01:01'),(42,20170831234300,1,'2020-01-01 01:01:01'),(43,20170831234301,1,'2020-01-01 01:01:01'),(44,20170831234303,1,'2020-01-01 01:01:01'),(45,20171116163618,1,'2020-01-01 01:01:01'),(46,20171219164727,1,'2020-01-01 01:01:01'),(47,20180620164811,1,'2020-01-01 01:01:01'),(48,20180620175054,1,'2020-01-01 01:01:01'),(49,20180620175055,1,'2020-01-01 01:01:01'),(50,20191010101639,1,'2020-01-01 01:01:01'),(51,20191010155147,1,'2020-01-01 01:01:01'),(52,20191220130734,1,'2020-01-01 01:01:01'),(53,20200311140000,1,'2020-01-01 01:01:01'),(54,20200405120000,1,'2020-01-01 01:01:01'),(55,20200407120000,1,'2020-01-01 01:01:01'),(56,20200420120000,1,'2020-01-01 01:01:01'),(57,20200504120000,1,'2020-01-01 01:01:01'),(58,20200512120000,1,'2020-01-01 01:01:01'),(59,20200707120000,1,'2020-01-01 01:01:01'),(60,20201011162341,1,'2020-01-01 01:01:01'),(61,20201021104586,1,'2020-01-01 01:01:01'),(62,20201102112520,1,'2020-01-01 01:01:01'),(63,20201208121729,1,'2020-01-01 01:01:01'),(64,20201215091637,1,'2020-01-01 01:01:01'),(65,20210119174155,1,'2020-01-01 01:01:01'),(66,20210326182902,1,'2020-01-01 01:01:01'),(67,20210421112652,1,'2020-01-01 01:01:01'),(68,20210506095025,1,'2020-01-01 01:01:01'),(69,20210513115729,1,'2020-01-01 01:01:01'),(70,20210526113559,1,'2020-01-01 01:01:01'),(71,20210601000001,1,'2020-01-01 01:01:01'),(72,20210601000002,1,'2020-01-01 01:01:01'),(73,20210601000003,1,'2020-01-01 01:01:01'),(74,20210601000004,1,'2020-01-01 01:01:01'),(75,20210601000005,1,'2020-01-01 01:01:01'),(76,20210601000006,1,'2020-01-01 01:01:01'),(77,20210601000007,1,'2020-01-01 01:01:01'),(78,20210601000008,1,'2020-01-01 01:01:01'),(79,20210606151329,1,'2020-01-01 01:01:01'),(80,20210616163757,1,'2020-01-01 01:01:01'),(81,20210617174723,1,'2020-01-01 01:01:01'),(82,20210622160235,1,'2020-01-01 01:01:01'),(83,20210623100031,1,'2020-01-01 01:01:01'),(84,20210623133615,1,'2020-01-01 01:01:01'),(85,20210708143152,1,'2020-01-01 01:01:01'),(86,20210709124443,1,'2020-01-01 01:01:01'),(87,20210712155608,1,'2020-01-01 01:01:01'),(88,20210714102108,1,'2020-01-01 01:01:01'),(89,20210719153709,1,'2020-01-01 01:01:01'),(90,20210721171531,1,'2020-01-01 01:01:01'),(91,20210723135713,1,'2020-01-01 01:01:01'),(92,20210802135933,1,'2020-01-01 01:01:01'),(93,20210806112844,1,'2020-01-01 01:01:01'),(94,20210810095603,1,'2020-01-01 01:01:01'),(95,20210811150223,1,'2020-01-01 01:01:01'),(96,20210818151827,1,'2020-01-01 01:01:01'),(97,20210818151828,1,'2020-01-01 01:01:01'),(98,20210818182258,1,'2020-01-01 01:01:01'),(99,20210819131107,1,'2020-01-01 01:01:01'),(100,20210819143446,1,'2020-01-01 01:01:01'),(101,20210903132338,1,'2020-01-01 01:01:01'),(102,20210915144307,1,'2020-01-01 01:01:01'),(103,20210920155130,1,'2020-01-01 01:01:01'),(104,20210927143115,1,'2020-01-01 01:01:01'),(105,20210927143116,1,'2020-01-01 01:01:01'),(106,20211013133706,1,'2020-01-01 01:01:01'),(107,20211013133707,1,'2020-01-01 01:01:01'),(108,20211102135149,1,'2020-01-01 01:01:01'),(109,20211109121546,1,'2020-01-01 01:01:01'),(110,20211110163320,1,'2020-01-01 01:01:01'),(111,20211116184029,1,'2020-01-01 01:01:01'),(112,20211116184030,1,'2020-01-01 01:01:01'),(113,20211202092042,1,'2020-01-01 01:01:01'),(114,20211202181033,1,'2020-01-01 01:01:01'),(115,20211207161856,1,'2020-01-01 01:01:01'),(116,20211216131203,1,'2020-01-01 01:01:01'),(117,20211221110132,1,'2020-01-01 01:01:01'),(118,20220107155700,1,'2020-01-01 01:01:01'),(119,20220125105650,1,'2020-01-01 01:01:01'),(120,20220201084510,1,'2020-01-01 01:01:01'),(121,20220208144830,1,'2020-01-01 01:01:01'),(122,20220208144831,1,'2020-01-01 01:01:01'),(123,20220215152203,1,'2020-01-01 01:01:01'),(124,20220223113157,1,'2020-01-01 01:01:01'),(125,20220307104655,1,'2020-01-01 01:01:01'),(126,20220309133956,1,'2020-01-01 01:01:01'),(127,20220316155700,1,'2020-01-01 01:01:01'),(128,20220323152301,1,'2020-01-01 01:01:01'),(129,20220330100659,1,'2020-01-01 01:01:01'),(130,20220404091216,1,'2020-01-01 01:01:01'),(131,20220419140750,1,'2020-01-01 01:01:01'),(132,20220428140039,1,'2020-01-01 01:01:01'),(133,20220503134048,1,'2020-01-01 01:01:01'),(134,20220524102918,1,'2020-01-01 01:01:01'),(135,20220526123327,1,'2020-01-01 01:01:01'),(136,20220526123328,1,'2020-01-01 01:01:01'),(137,20220526123329,1,'2020-01-01 01:01:01'),(138,20220608113128,1,'2020-01-01 01:01:01'),(139,20220627104817,1,'2020-01-01 01:01:01'),(140,20220704101843,1,'2020-01-01 01:01:01'),(141,20220708095046,1,'2020-01-01 01:01:01'),(142,20220713091130,1,'2020-01-01 01:01:01'),(143,20220802135510,1,'2020-01-01 01:01:01'),(144,20220818101352,1,'2020-01-01 01:01:01'),(145,20220822161445,1,'2020-01-01 01:01:01'),(146,20220831100036,1,'2020-01-01 01:01:01'),(147,20220831100151,1,'2020-01-01 01:01:01'),(148,20220908181826,1,'2020-01-01 01:01:01'),(149,20220914154915,1,'2020-01-01 01:01:01'),(150,20220915165115,1,'2020-01-01 01:01:01'),(151,20220915165116,1,'2020-01-01 01:01:01'),(152,20220928100158,1,'2020-01-01 01:01:01'),(153,20221014084130,1,'2020-01-01 01:01:01'),(154,20221027085019,1,'2020-01-01 01:01:01'),(155,20221101103952,1,'2020-01-01 01:01:01'),(156,20221104144401,1,'2020-01-01 01:01:01'),(157,20221109100749,1,'2020-01-01 01:01:01'),(158,20221115104546,1,'2020-01-01 01:01:01'),(159,20221130114928,1,'2020-01-01 01:01:01'),(160,20221205112142,1,'2020-01-01 01:01:01'),(161,20221216115820,1,'2020-01-01 01:01:01'),(162,20221220195934,1,'2020-01-01 01:01:01'),(163,20221220195935,1,'2020-01-01 01:01:01'),(164,20221223174807,1,'2020-01-01 01:01:01'),(165,20221227163855,1,'2020-01-01 01:01:01'),(166,20221227163856,1,'2020-01-01 01:01:01'),(167,20230202224725,1,'2020-01-01 01:01:01'),(168,20230206163608,1,'2020-01-01 01:01:01'),(169,20230214131519,1,'2020-01-01 01:01:01'),(170,20230303135738,1,'2020-01-01 01:01:01'),(171,20230313135301,1,'2020-01-01 01:01:01'),(172,20230313141819,1,'2020-01-01 01:01:01'),(173,20230315104937,1,'2020-01-01 01:01:01'),(174,20230317173844,1,'2020-01-01 01:01:01'),(175,20230320133602,1,'2020-01-01 01:01:01'),(176,20230330100011,1,'2020-01-01 01:01:01'),(177,20230330134823,1,'2020-01-01 01:01:01'),(178,20230405232025,1,'2020-01-01 01:01:01'),(179,20230408084104,1,'2020-01-01 01:01:01'),(180,20230411102858,1,'2020-01-01 01:01:01'),(181,20230421155932,1,'2020-01-01 01:01:01'),(182,20230425082126,1,'2020-01-01 01:01:01'),(183,20230425105727,1,'2020-01-01 01:01:01'),(184,20230501154913,1,'2020-01-01 01:01:01'),(185,20230503101418,1,'2020-01-01 01:01:01'),(186,20230515144206,1,'2020-01-01 01:01:01'),(187,20230517140952,1,'2020-01-01 01:01:01'),(188,20230517152807,1,'2020-01-01 01:01:01'),(189,20230518114155,1,'2020-01-01 01:01:01'),(190,20230520153236,1,'2020-01-01 01:01:01'),(191,20230525151159,1,'2020-01-01 01:01:01'),(192,20230530122103,1,'2020-01-01 01:01:01'),(193,20230602111827,1,'2020-01-01 01:01:01'),(194,20230608103123,1,'2020-01-01 01:01:01'),(195,20230629140529,1,'2020-01-01 01:01:01'),(196,20230629140530,1,'2020-01-01 01:01:01'),(197,20230711144622,1,'2020-01-01 01:01:01'),(198,20230721135421,1,'2020-01-01 01:01:01'),(199,20230721161508,1,'2020-01-01 01:01:01'),(200,20230726115701,1,'2020-01-01 01:01:01'),(201,20230807100822,1,'2020-01-01 01:01:01'),(202,20230814150442,1,'2020-01-01 01:01:01'),(203,20230823122728,1,'2020-01-01 01:01:01'),(204,20230906152143,1,'2020-01-01 01:01:01'),(205,20230911163618,1,'2020-01-01 01:01:01'),(206,20230912101759,1,'2020-01-01 01:01:01'),(207,20230915101341,1,'2020-01-01 01:01:01'),(208,20230918132351,1,'2020-01-01 01:01:01'),(209,20231004144339,1,'2020-01-01 01:01:01'),(210,20231009094541,1,'2020-01-01 01:01:01'),(211,20231009094542,1,'2020-01-01 01:01:01'),(212,20231009094543,1,'2020-01-01 01:01:01'),(213,20231009094544,1,'2020-01-01 01:01:01'),(214,20231016091915,1,'2020-01-01 01:01:01'),(215,20231024174135,1,'2020-01-01 01:01:01'),(216,20231025120016,1,'2020-01-01 01:01:01'),(217,20231025160156,1,'2020-01-01 01:01:01'),(218,20231031165350,1,'2020-01-01 01:01:01'),(219,20231106144110,1,'2020-01-01 01:01:01'),(220,20231107130934,1,'2020-01-01 01:01:01'),(221,20231109115838,1,'2020-01-01 01:01:01'),(222,20231121054530,1,'2020-01-01 01:01:01'),(223,20231122101320,1,'2020-01-01 01:01:01'),(224,20231130132828,1,'2020-01-01 01:01:01'),(225,20231130132931,1,'2020-01-01 01:01:01'),(226,20231204155427,1,'2020-01-01 01:01:01'),(227,20231206142340,1,'2020-01-01 01:01:01'),(228,20231207102320,1,'2020-01-01 01:01:01'),(229,20231207102321,1,'2020-01-01 01:01:01'),(230,20231207133731,1,'2020-01-01 01:01:01'),(231,20231212094238,1,'2020-01-01 01:01:01'),(232,20231212095734,1,'2020-01-01 01:01:01'),(233,20231212161121,1,'2020-01-01 01:01:01'),(234,20231215122713,1,'2020-01-01 01:01:01'),(235,20231219143041,1,'2020-01-01 01:01:01'),(236,20231224070653,1,'2020-01-01 01:01:01'),(237,20240110134315,1,'2020-01-01 01:01:01'),(238,20240119091637,1,'2020-01-01 01:01:01'),(239,20240126020642,1,'2020-01-01 01:01:01'),(240,20240126020643,1,'2020-01-01 01:01:01'),(241,20240129162819,1,'2020-01-01 01:01:01'),(242,20240130115133,1,'2020-01-01 01:01:01'),(243,20240131083822,1,'2020-01-01 01:01:01'),(244,20240205095928,1,'2020-01-01 01:01:01'),(245,20240205121956,1,'2020-01-01 01:01:01'),(246,20240209110212,1,'2020-01-01 01:01:01'),(247,20240212111533,1,'2020-01-01 01:01:01'),(248,20240221112844,1,'2020-01-01 01:01:01'),(249,20240222073518,1,'2020-01-01 01:01:01'),(250,20240222135115,1,'2020-01-01 01:01:01');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mobile_device_management_solutions` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `server_url` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mobile_device_management_solutions_name` (`name`,`server_url`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `munki_issues` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `issue_type` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_munki_issues_name` (`name`,`issue_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `nano_cert_auth_associations` (
  `id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sha256` char(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `cert_not_valid_after` timestamp NULL DEFAULT NULL,
  `renew_command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`,`sha256`),
  KEY `renew_command_uuid_fk` (`renew_command_uuid`),
  CONSTRAINT `renew_command_uuid_fk` FOREIGN KEY (`renew_command_uuid`) REFERENCES `nano_commands` (`command_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `nano_command_results` (
  `id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(31) COLLATE utf8mb4_unicode_ci NOT NULL,
  `result` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `not_now_at` timestamp NULL DEFAULT NULL,
  `not_now_tally` int(11) NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`,`command_uuid`),
  KEY `command_uuid` (`command_uuid`),
  KEY `status` (`status`),
  CONSTRAINT `nano_command_results_ibfk_1` FOREIGN KEY (`id`) REFERENCES `nano_enrollments` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `nano_command_results_ibfk_2` FOREIGN KEY (`command_uuid`) REFERENCES `nano_commands` (`command_uuid`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `nano_commands` (
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `request_type` varchar(63) COLLATE utf8mb4_unicode_ci NOT NULL,
  `command` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`command_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `nano_dep_names` (
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `consumer_key` text COLLATE utf8mb4_unicode_ci,
  `consumer_secret` text COLLATE utf8mb4_unicode_ci,
  `access_token` text COLLATE utf8mb4_unicode_ci,
  `access_secret` text COLLATE utf8mb4_unicode_ci,
  `access_token_expiry` timestamp NULL DEFAULT NULL,
  `config_base_url` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tokenpki_cert_pem` text COLLATE utf8mb4_unicode_ci,
  `tokenpki_key_pem` text COLLATE utf8mb4_unicode_ci,
  `syncer_cursor` varchar(1024) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `syncer_cursor_at` timestamp NULL DEFAULT NULL,
  `assigner_profile_uuid` text COLLATE utf8mb4_unicode_ci,
  `assigner_profile_uuid_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `nano_devices` (
  `id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `identity_cert` text COLLATE utf8mb4_unicode_ci,
  `serial_number` varchar(127) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `unlock_token` mediumblob,
  `unlock_token_at` timestamp NULL DEFAULT NULL,
  `authenticate` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `authenticate_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `token_update` text COLLATE utf8mb4_unicode_ci,
  `token_update_at` timestamp NULL DEFAULT NULL,
  `bootstrap_token_b64` text COLLATE utf8mb4_unicode_ci,
  `bootstrap_token_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `serial_number` (`serial_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `nano_enrollment_queue` (
  `id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `active` tinyint(1) NOT NULL DEFAULT '1',
  `priority` tinyint(4) NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`,`command_uuid`),
  KEY `command_uuid` (`command_uuid`),
  KEY `priority` (`priority`,`created_at`),
  CONSTRAINT `nano_enrollment_queue_ibfk_1` FOREIGN KEY (`id`) REFERENCES `nano_enrollments` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `nano_enrollment_queue_ibfk_2` FOREIGN KEY (`command_uuid`) REFERENCES `nano_commands` (`command_uuid`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `nano_enrollments` (
  `id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `device_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `type` varchar(31) COLLATE utf8mb4_unicode_ci NOT NULL,
  `topic` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `push_magic` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `token_hex` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `token_update_tally` int(11) NOT NULL DEFAULT '1',
  `last_seen_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_id` (`user_id`),
  KEY `device_id` (`device_id`),
  KEY `type` (`type`),
  CONSTRAINT `nano_enrollments_ibfk_1` FOREIGN KEY (`device_id`) REFERENCES `nano_devices` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `nano_enrollments_ibfk_2` FOREIGN KEY (`user_id`) REFERENCES `nano_users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `nano_push_certs` (
  `topic` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `cert_pem` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `key_pem` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `stale_token` int(11) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`topic`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `nano_users` (
  `id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `device_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_short_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_long_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `token_update` text COLLATE utf8mb4_unicode_ci,
  `token_update_at` timestamp NULL DEFAULT NULL,
  `user_authenticate` text COLLATE utf8mb4_unicode_ci,
  `user_authenticate_at` timestamp NULL DEFAULT NULL,
  `user_authenticate_digest` text COLLATE utf8mb4_unicode_ci,
  `user_authenticate_digest_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`,`device_id`),
  KEY `device_id` (`device_id`),
  CONSTRAINT `nano_users_ibfk_1` FOREIGN KEY (`device_id`) REFERENCES `nano_devices` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
SET @saved_cs_client     = @@character_set_client;
SET character_set_client = utf8;
/*!50001 CREATE VIEW `nano_view_queue` AS SELECT 
 1 AS `id`,
 1 AS `created_at`,
 1 AS `active`,
 1 AS `priority`,
 1 AS `command_uuid`,
 1 AS `request_type`,
 1 AS `command`,
 1 AS `result_updated_at`,
 1 AS `status`,
 1 AS `result`*/;
SET character_set_client = @saved_cs_client;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `network_interfaces` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int(10) unsigned NOT NULL,
  `mac` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `ip_address` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `broadcast` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `ibytes` bigint(20) NOT NULL DEFAULT '0',
  `interface` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `ipackets` bigint(20) NOT NULL DEFAULT '0',
  `last_change` bigint(20) NOT NULL DEFAULT '0',
  `mask` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `metric` int(11) NOT NULL DEFAULT '0',
  `mtu` int(11) NOT NULL DEFAULT '0',
  `obytes` bigint(20) NOT NULL DEFAULT '0',
  `ierrors` bigint(20) NOT NULL DEFAULT '0',
  `oerrors` bigint(20) NOT NULL DEFAULT '0',
  `opackets` bigint(20) NOT NULL DEFAULT '0',
  `point_to_point` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `type` int(11) NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_network_interfaces_unique_ip_host_intf` (`ip_address`,`host_id`,`interface`),
  KEY `idx_network_interfaces_hosts_fk` (`host_id`),
  FULLTEXT KEY `ip_address_search` (`ip_address`),
  CONSTRAINT `network_interfaces_ibfk_1` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `operating_system_vulnerabilities` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `operating_system_id` int(10) unsigned NOT NULL,
  `cve` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source` smallint(6) DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `resolved_in_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_os_vulnerabilities_unq_os_id_cve` (`operating_system_id`,`cve`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `operating_systems` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `arch` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `kernel_version` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `platform` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `display_version` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `os_version_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_os` (`name`,`version`,`arch`,`kernel_version`,`platform`,`display_version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `osquery_options` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `override_type` int(1) NOT NULL,
  `override_identifier` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `options` json NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `packs` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `disabled` tinyint(1) NOT NULL DEFAULT '0',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `platform` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `pack_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_pack_unique_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `password_reset_requests` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `expires_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `user_id` int(10) unsigned NOT NULL,
  `token` varchar(1024) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `policies` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `team_id` int(10) unsigned DEFAULT NULL,
  `resolution` text COLLATE utf8mb4_unicode_ci,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `query` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `author_id` int(10) unsigned DEFAULT NULL,
  `platforms` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `critical` tinyint(1) NOT NULL DEFAULT '0',
  `checksum` binary(16) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_policies_checksum` (`checksum`),
  KEY `idx_policies_author_id` (`author_id`),
  KEY `idx_policies_team_id` (`team_id`),
  CONSTRAINT `policies_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `policies_queries_ibfk_1` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `policy_automation_iterations` (
  `policy_id` int(10) unsigned NOT NULL,
  `iteration` int(11) NOT NULL,
  PRIMARY KEY (`policy_id`),
  CONSTRAINT `policy_automation_iterations_ibfk_1` FOREIGN KEY (`policy_id`) REFERENCES `policies` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `policy_membership` (
  `policy_id` int(10) unsigned NOT NULL,
  `host_id` int(10) unsigned NOT NULL,
  `passes` tinyint(1) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `automation_iteration` int(11) DEFAULT NULL,
  PRIMARY KEY (`policy_id`,`host_id`),
  KEY `idx_policy_membership_passes` (`passes`),
  KEY `idx_policy_membership_policy_id` (`policy_id`),
  KEY `idx_policy_membership_host_id_passes` (`host_id`,`passes`),
  CONSTRAINT `policy_membership_ibfk_1` FOREIGN KEY (`policy_id`) REFERENCES `policies` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `policy_stats` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `policy_id` int(10) unsigned NOT NULL,
  `inherited_team_id` int(10) unsigned NOT NULL DEFAULT '0',
  `passing_host_count` mediumint(8) unsigned NOT NULL DEFAULT '0',
  `failing_host_count` mediumint(8) unsigned NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `policy_team_unique` (`policy_id`,`inherited_team_id`),
  CONSTRAINT `policy_stats_ibfk_1` FOREIGN KEY (`policy_id`) REFERENCES `policies` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `queries` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `saved` tinyint(1) NOT NULL DEFAULT '0',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `query` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `author_id` int(10) unsigned DEFAULT NULL,
  `observer_can_run` tinyint(1) NOT NULL DEFAULT '0',
  `team_id` int(10) unsigned DEFAULT NULL,
  `team_id_char` char(10) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `platform` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `min_osquery_version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `schedule_interval` int(10) unsigned NOT NULL DEFAULT '0',
  `automations_enabled` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `logging_type` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'snapshot',
  `discard_data` tinyint(1) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_team_id_name_unq` (`team_id_char`,`name`),
  UNIQUE KEY `idx_name_team_id_unq` (`name`,`team_id_char`),
  KEY `author_id` (`author_id`),
  KEY `idx_team_id_saved_auto_interval` (`team_id`,`saved`,`automations_enabled`,`schedule_interval`),
  CONSTRAINT `queries_ibfk_1` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `queries_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `query_results` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `query_id` int(10) unsigned NOT NULL,
  `host_id` int(10) unsigned NOT NULL,
  `osquery_version` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `error` text COLLATE utf8mb4_unicode_ci,
  `last_fetched` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `data` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `query_id` (`query_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `scep_certificates` (
  `serial` bigint(20) NOT NULL,
  `name` varchar(1024) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `not_valid_before` datetime NOT NULL,
  `not_valid_after` datetime NOT NULL,
  `certificate_pem` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `revoked` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`serial`),
  CONSTRAINT `scep_certificates_ibfk_1` FOREIGN KEY (`serial`) REFERENCES `scep_serials` (`serial`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `scep_serials` (
  `serial` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`serial`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
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
  `platform` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `shard` int(10) unsigned DEFAULT NULL,
  `query_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` varchar(1023) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `denylist` tinyint(1) DEFAULT NULL,
  `team_id_char` char(10) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_names_in_packs` (`name`,`pack_id`),
  KEY `scheduled_queries_pack_id` (`pack_id`),
  KEY `scheduled_queries_query_name` (`query_name`),
  KEY `fk_scheduled_queries_queries` (`team_id_char`,`query_name`),
  CONSTRAINT `scheduled_queries_ibfk_1` FOREIGN KEY (`team_id_char`, `query_name`) REFERENCES `queries` (`team_id_char`, `name`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `scheduled_queries_pack_id` FOREIGN KEY (`pack_id`) REFERENCES `packs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `scheduled_query_stats` (
  `host_id` int(10) unsigned NOT NULL,
  `scheduled_query_id` int(10) unsigned NOT NULL,
  `average_memory` bigint(20) unsigned NOT NULL,
  `denylisted` tinyint(1) DEFAULT NULL,
  `executions` bigint(20) unsigned NOT NULL,
  `schedule_interval` int(11) DEFAULT NULL,
  `last_executed` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `output_size` bigint(20) unsigned NOT NULL,
  `system_time` bigint(20) unsigned NOT NULL,
  `user_time` bigint(20) unsigned NOT NULL,
  `wall_time` bigint(20) unsigned NOT NULL,
  `query_type` tinyint(4) NOT NULL DEFAULT '0',
  PRIMARY KEY (`host_id`,`scheduled_query_id`,`query_type`),
  KEY `scheduled_query_id` (`scheduled_query_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `scripts` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `team_id` int(10) unsigned DEFAULT NULL,
  `global_or_team_id` int(10) unsigned NOT NULL DEFAULT '0',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `script_contents` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_scripts_global_or_team_id_name` (`global_or_team_id`,`name`),
  UNIQUE KEY `idx_scripts_team_name` (`team_id`,`name`),
  CONSTRAINT `scripts_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `sessions` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `accessed_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `user_id` int(10) unsigned NOT NULL,
  `key` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_session_unique_key` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `software` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `source` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `bundle_identifier` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `release` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `vendor_old` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `arch` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `vendor` varchar(114) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `browser` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `extension_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `title_id` int(10) unsigned DEFAULT NULL,
  `checksum` binary(16) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_software_checksum` (`checksum`),
  KEY `software_listing_idx` (`name`,`id`),
  KEY `software_source_vendor_idx` (`source`,`vendor_old`),
  KEY `title_id` (`title_id`),
  KEY `idx_sw_name_source_browser` (`name`,`source`,`browser`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `software_cpe` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `software_id` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `cpe` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unq_software_id` (`software_id`),
  KEY `software_cpe_cpe_idx` (`cpe`),
  CONSTRAINT `software_cpe_ibfk_1` FOREIGN KEY (`software_id`) REFERENCES `software` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `software_cve` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `cve` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `source` int(11) DEFAULT '0',
  `software_id` bigint(20) unsigned DEFAULT NULL,
  `resolved_in_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unq_software_id_cve` (`software_id`,`cve`),
  KEY `software_cve_software_id` (`software_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `software_host_counts` (
  `software_id` bigint(20) unsigned NOT NULL,
  `hosts_count` int(10) unsigned NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `team_id` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`software_id`,`team_id`),
  KEY `idx_software_host_counts_updated_at_software_id` (`updated_at`,`software_id`),
  KEY `idx_software_host_counts_team_id_hosts_count_software_id` (`team_id`,`hosts_count`,`software_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `software_titles` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `browser` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sw_titles` (`name`,`source`,`browser`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `software_titles_host_counts` (
  `software_title_id` int(10) unsigned NOT NULL,
  `hosts_count` int(10) unsigned NOT NULL,
  `team_id` int(10) unsigned NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`software_title_id`,`team_id`),
  KEY `idx_software_titles_host_counts_team_counts_title` (`team_id`,`hosts_count`,`software_title_id`),
  KEY `idx_software_titles_host_counts_updated_at_software_title_id` (`updated_at`,`software_title_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `statistics` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `anonymous_identifier` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `teams` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` varchar(1023) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `config` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `user_teams` (
  `user_id` int(10) unsigned NOT NULL,
  `team_id` int(10) unsigned NOT NULL,
  `role` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`user_id`,`team_id`),
  KEY `fk_user_teams_team_id` (`team_id`),
  CONSTRAINT `user_teams_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `user_teams_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `password` varbinary(255) NOT NULL,
  `salt` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `email` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `admin_forced_password_reset` tinyint(1) NOT NULL DEFAULT '0',
  `gravatar_url` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `position` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `sso_enabled` tinyint(4) NOT NULL DEFAULT '0',
  `global_role` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `api_only` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_unique_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `vulnerability_host_counts` (
  `cve` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `team_id` int(10) unsigned NOT NULL DEFAULT '0',
  `host_count` int(10) unsigned NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `cve_team_id` (`cve`,`team_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `windows_mdm_command_queue` (
  `enrollment_id` int(10) unsigned NOT NULL,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`enrollment_id`,`command_uuid`),
  KEY `command_uuid` (`command_uuid`),
  CONSTRAINT `windows_mdm_command_queue_ibfk_1` FOREIGN KEY (`enrollment_id`) REFERENCES `mdm_windows_enrollments` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `windows_mdm_command_queue_ibfk_2` FOREIGN KEY (`command_uuid`) REFERENCES `windows_mdm_commands` (`command_uuid`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `windows_mdm_command_results` (
  `enrollment_id` int(10) unsigned NOT NULL,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `raw_result` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `response_id` int(10) unsigned NOT NULL,
  `status_code` varchar(31) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`enrollment_id`,`command_uuid`),
  KEY `command_uuid` (`command_uuid`),
  KEY `response_id` (`response_id`),
  CONSTRAINT `windows_mdm_command_results_ibfk_1` FOREIGN KEY (`enrollment_id`) REFERENCES `mdm_windows_enrollments` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `windows_mdm_command_results_ibfk_2` FOREIGN KEY (`command_uuid`) REFERENCES `windows_mdm_commands` (`command_uuid`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `windows_mdm_command_results_ibfk_3` FOREIGN KEY (`response_id`) REFERENCES `windows_mdm_responses` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `windows_mdm_commands` (
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `raw_command` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_loc_uri` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`command_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `windows_mdm_responses` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `enrollment_id` int(10) unsigned NOT NULL,
  `raw_response` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `enrollment_id` (`enrollment_id`),
  CONSTRAINT `windows_mdm_responses_ibfk_1` FOREIGN KEY (`enrollment_id`) REFERENCES `mdm_windows_enrollments` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `windows_updates` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int(10) unsigned NOT NULL,
  `date_epoch` int(10) unsigned NOT NULL,
  `kb_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_windows_updates` (`host_id`,`kb_id`),
  KEY `idx_update_date` (`host_id`,`date_epoch`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `wstep_cert_auth_associations` (
  `id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sha256` char(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`,`sha256`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `wstep_certificates` (
  `serial` bigint(20) unsigned NOT NULL,
  `name` varchar(1024) COLLATE utf8mb4_unicode_ci NOT NULL,
  `not_valid_before` datetime NOT NULL,
  `not_valid_after` datetime NOT NULL,
  `certificate_pem` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `revoked` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`serial`),
  CONSTRAINT `wstep_certificates_ibfk_1` FOREIGN KEY (`serial`) REFERENCES `wstep_serials` (`serial`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `wstep_serials` (
  `serial` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`serial`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!50001 DROP VIEW IF EXISTS `nano_view_queue`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb4 */;
/*!50001 SET character_set_results     = utf8mb4 */;
/*!50001 SET collation_connection      = utf8mb4_unicode_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`root`@`%` SQL SECURITY DEFINER */
/*!50001 VIEW `nano_view_queue` AS select `q`.`id` AS `id`,`q`.`created_at` AS `created_at`,`q`.`active` AS `active`,`q`.`priority` AS `priority`,`c`.`command_uuid` AS `command_uuid`,`c`.`request_type` AS `request_type`,`c`.`command` AS `command`,`r`.`updated_at` AS `result_updated_at`,`r`.`status` AS `status`,`r`.`result` AS `result` from ((`nano_enrollment_queue` `q` join `nano_commands` `c` on((`q`.`command_uuid` = `c`.`command_uuid`))) left join `nano_command_results` `r` on(((`r`.`command_uuid` = `q`.`command_uuid`) and (`r`.`id` = `q`.`id`)))) order by `q`.`priority` desc,`q`.`created_at` */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;
