/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `abm_tokens` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `organization_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `apple_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `terms_expired` tinyint(1) NOT NULL DEFAULT '0',
  `renew_at` timestamp NOT NULL,
  `token` blob NOT NULL,
  `macos_default_team_id` int unsigned DEFAULT NULL,
  `ios_default_team_id` int unsigned DEFAULT NULL,
  `ipados_default_team_id` int unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_abm_tokens_organization_name` (`organization_name`),
  KEY `fk_abm_tokens_macos_default_team_id` (`macos_default_team_id`),
  KEY `fk_abm_tokens_ios_default_team_id` (`ios_default_team_id`),
  KEY `fk_abm_tokens_ipados_default_team_id` (`ipados_default_team_id`),
  CONSTRAINT `fk_abm_tokens_ios_default_team_id` FOREIGN KEY (`ios_default_team_id`) REFERENCES `teams` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_abm_tokens_ipados_default_team_id` FOREIGN KEY (`ipados_default_team_id`) REFERENCES `teams` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_abm_tokens_macos_default_team_id` FOREIGN KEY (`macos_default_team_id`) REFERENCES `teams` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `activities` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `user_id` int unsigned DEFAULT NULL,
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
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `aggregated_stats` (
  `id` bigint unsigned NOT NULL,
  `type` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `json_value` json NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `global_stats` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`,`type`,`global_stats`),
  KEY `idx_aggregated_stats_updated_at` (`updated_at`),
  KEY `aggregated_stats_type_idx` (`type`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_config_json` (
  `id` int unsigned NOT NULL DEFAULT '1',
  `json_value` json NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `app_config_json` VALUES (1,'{\"mdm\": {\"ios_updates\": {\"deadline\": null, \"minimum_version\": null}, \"macos_setup\": {\"script\": null, \"software\": null, \"bootstrap_package\": null, \"macos_setup_assistant\": null, \"enable_end_user_authentication\": false, \"enable_release_device_manually\": false}, \"macos_updates\": {\"deadline\": null, \"minimum_version\": null}, \"ipados_updates\": {\"deadline\": null, \"minimum_version\": null}, \"macos_settings\": {\"custom_settings\": null}, \"macos_migration\": {\"mode\": \"\", \"enable\": false, \"webhook_url\": \"\"}, \"windows_updates\": {\"deadline_days\": null, \"grace_period_days\": null}, \"apple_server_url\": \"\", \"windows_settings\": {\"custom_settings\": null}, \"apple_bm_terms_expired\": false, \"apple_business_manager\": null, \"enable_disk_encryption\": false, \"enabled_and_configured\": false, \"end_user_authentication\": {\"idp_name\": \"\", \"metadata\": \"\", \"entity_id\": \"\", \"issuer_uri\": \"\", \"metadata_url\": \"\"}, \"volume_purchasing_program\": null, \"windows_migration_enabled\": false, \"windows_enabled_and_configured\": false, \"apple_bm_enabled_and_configured\": false}, \"scripts\": null, \"features\": {\"enable_host_users\": true, \"enable_software_inventory\": false}, \"org_info\": {\"org_name\": \"\", \"contact_url\": \"\", \"org_logo_url\": \"\", \"org_logo_url_light_background\": \"\"}, \"integrations\": {\"jira\": null, \"zendesk\": null, \"google_calendar\": null, \"ndes_scep_proxy\": null}, \"sso_settings\": {\"idp_name\": \"\", \"metadata\": \"\", \"entity_id\": \"\", \"enable_sso\": false, \"issuer_uri\": \"\", \"metadata_url\": \"\", \"idp_image_url\": \"\", \"enable_jit_role_sync\": false, \"enable_sso_idp_login\": false, \"enable_jit_provisioning\": false}, \"agent_options\": {\"config\": {\"options\": {\"logger_plugin\": \"tls\", \"pack_delimiter\": \"/\", \"logger_tls_period\": 10, \"distributed_plugin\": \"tls\", \"disable_distributed\": false, \"logger_tls_endpoint\": \"/api/osquery/log\", \"distributed_interval\": 10, \"distributed_tls_max_attempts\": 3}, \"decorators\": {\"load\": [\"SELECT uuid AS host_uuid FROM system_info;\", \"SELECT hostname AS hostname FROM system_info;\"]}}, \"overrides\": {}}, \"fleet_desktop\": {\"transparency_url\": \"\"}, \"smtp_settings\": {\"port\": 587, \"domain\": \"\", \"server\": \"\", \"password\": \"\", \"user_name\": \"\", \"configured\": false, \"enable_smtp\": false, \"enable_ssl_tls\": true, \"sender_address\": \"\", \"enable_start_tls\": true, \"verify_ssl_certs\": true, \"authentication_type\": \"0\", \"authentication_method\": \"0\"}, \"server_settings\": {\"server_url\": \"\", \"enable_analytics\": false, \"query_report_cap\": 0, \"scripts_disabled\": false, \"deferred_save_host\": false, \"live_query_disabled\": false, \"ai_features_disabled\": false, \"query_reports_disabled\": false}, \"webhook_settings\": {\"interval\": \"0s\", \"activities_webhook\": {\"destination_url\": \"\", \"enable_activities_webhook\": false}, \"host_status_webhook\": {\"days_count\": 0, \"destination_url\": \"\", \"host_percentage\": 0, \"enable_host_status_webhook\": false}, \"vulnerabilities_webhook\": {\"destination_url\": \"\", \"host_batch_size\": 0, \"enable_vulnerabilities_webhook\": false}, \"failing_policies_webhook\": {\"policy_ids\": null, \"destination_url\": \"\", \"host_batch_size\": 0, \"enable_failing_policies_webhook\": false}}, \"host_expiry_settings\": {\"host_expiry_window\": 0, \"host_expiry_enabled\": false}, \"vulnerability_settings\": {\"databases_path\": \"\"}, \"activity_expiry_settings\": {\"activity_expiry_window\": 0, \"activity_expiry_enabled\": false}}','2020-01-01 01:01:01','2020-01-01 01:01:01');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `calendar_events` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `email` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `start_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `end_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `event` json NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `timezone` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `uuid_bin` binary(16) NOT NULL,
  `uuid` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci GENERATED ALWAYS AS (insert(insert(insert(insert(hex(`uuid_bin`),9,0,_utf8mb4'-'),14,0,_utf8mb4'-'),19,0,_utf8mb4'-'),24,0,_utf8mb4'-')) VIRTUAL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_one_calendar_event_per_email` (`email`),
  UNIQUE KEY `idx_calendar_events_uuid_bin_unique` (`uuid_bin`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `carve_blocks` (
  `metadata_id` int unsigned NOT NULL,
  `block_id` int NOT NULL,
  `data` longblob,
  PRIMARY KEY (`metadata_id`,`block_id`),
  CONSTRAINT `carve_blocks_ibfk_1` FOREIGN KEY (`metadata_id`) REFERENCES `carve_metadata` (`id`) ON DELETE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `carve_metadata` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `block_count` int unsigned NOT NULL,
  `block_size` int unsigned NOT NULL,
  `carve_size` bigint unsigned NOT NULL,
  `carve_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `request_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `session_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `expired` tinyint DEFAULT '0',
  `max_block` int DEFAULT '-1',
  `error` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_session_id` (`session_id`),
  UNIQUE KEY `idx_name` (`name`),
  KEY `host_id` (`host_id`),
  CONSTRAINT `carve_metadata_ibfk_1` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cron_stats` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `instance` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `stats_type` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `errors` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_cron_stats_name_created_at` (`name`,`created_at`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cve_meta` (
  `cve` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `cvss_score` double DEFAULT NULL,
  `epss_probability` double DEFAULT NULL,
  `cisa_known_exploit` tinyint(1) DEFAULT NULL,
  `published` timestamp NULL DEFAULT NULL,
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`cve`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `distributed_query_campaign_targets` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `type` int DEFAULT NULL,
  `distributed_query_campaign_id` int unsigned DEFAULT NULL,
  `target_id` int unsigned DEFAULT NULL,
  PRIMARY KEY (`id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `distributed_query_campaigns` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `query_id` int unsigned DEFAULT NULL,
  `status` int DEFAULT NULL,
  `user_id` int unsigned DEFAULT NULL,
  PRIMARY KEY (`id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `email_changes` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int unsigned NOT NULL,
  `token` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL,
  `new_email` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_email_changes_token` (`token`) USING BTREE,
  KEY `fk_email_changes_users` (`user_id`),
  CONSTRAINT `fk_email_changes_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `enroll_secrets` (
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `secret` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `team_id` int unsigned DEFAULT NULL,
  PRIMARY KEY (`secret`),
  KEY `fk_enroll_secrets_team_id` (`team_id`),
  CONSTRAINT `enroll_secrets_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `eulas` (
  `id` int unsigned NOT NULL,
  `token` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `bytes` longblob,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `fleet_library_apps` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `token` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `platform` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `installer_url` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sha256` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `bundle_identifier` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `install_script_content_id` int unsigned NOT NULL,
  `uninstall_script_content_id` int unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_fleet_library_apps_token` (`token`),
  KEY `fk_fleet_library_apps_install_script_content` (`install_script_content_id`),
  KEY `fk_fleet_library_apps_uninstall_script_content` (`uninstall_script_content_id`),
  CONSTRAINT `fk_fleet_library_apps_install_script_content` FOREIGN KEY (`install_script_content_id`) REFERENCES `script_contents` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_fleet_library_apps_uninstall_script_content` FOREIGN KEY (`uninstall_script_content_id`) REFERENCES `script_contents` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_activities` (
  `host_id` int unsigned NOT NULL,
  `activity_id` int unsigned NOT NULL,
  PRIMARY KEY (`host_id`,`activity_id`),
  KEY `fk_host_activities_activity_id` (`activity_id`),
  CONSTRAINT `host_activities_ibfk_1` FOREIGN KEY (`activity_id`) REFERENCES `activities` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_additional` (
  `host_id` int unsigned NOT NULL,
  `additional` json DEFAULT NULL,
  PRIMARY KEY (`host_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_batteries` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int unsigned NOT NULL,
  `serial_number` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `cycle_count` int NOT NULL,
  `health` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_host_batteries_host_id_serial_number` (`host_id`,`serial_number`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_calendar_events` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int unsigned NOT NULL,
  `calendar_event_id` int unsigned NOT NULL,
  `webhook_status` tinyint NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_one_calendar_event_per_host` (`host_id`),
  KEY `calendar_event_id` (`calendar_event_id`),
  CONSTRAINT `host_calendar_events_ibfk_1` FOREIGN KEY (`calendar_event_id`) REFERENCES `calendar_events` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_dep_assignments` (
  `host_id` int unsigned NOT NULL,
  `added_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `profile_uuid` varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `assign_profile_response` varchar(15) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `response_updated_at` timestamp NULL DEFAULT NULL,
  `retry_job_id` int unsigned NOT NULL DEFAULT '0',
  `abm_token_id` int unsigned DEFAULT NULL,
  PRIMARY KEY (`host_id`),
  KEY `idx_hdep_response` (`assign_profile_response`,`response_updated_at`),
  KEY `fk_host_dep_assignments_abm_token_id` (`abm_token_id`),
  CONSTRAINT `fk_host_dep_assignments_abm_token_id` FOREIGN KEY (`abm_token_id`) REFERENCES `abm_tokens` (`id`) ON DELETE SET NULL
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_device_auth` (
  `host_id` int unsigned NOT NULL,
  `token` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`host_id`),
  UNIQUE KEY `idx_host_device_auth_token` (`token`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_disk_encryption_keys` (
  `host_id` int unsigned NOT NULL,
  `base64_encrypted` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `base64_encrypted_salt` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `key_slot` tinyint unsigned DEFAULT NULL,
  `decryptable` tinyint(1) DEFAULT NULL,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `reset_requested` tinyint(1) NOT NULL DEFAULT '0',
  `client_error` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`host_id`),
  KEY `idx_host_disk_encryption_keys_decryptable` (`decryptable`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_disk_encryption_keys_archive` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int unsigned NOT NULL,
  `hardware_serial` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `base64_encrypted` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `base64_encrypted_salt` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `key_slot` tinyint unsigned DEFAULT NULL,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `idx_host_disk_encryption_keys_archive_host_created_at` (`host_id`,`created_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_disks` (
  `host_id` int unsigned NOT NULL,
  `gigs_disk_space_available` decimal(10,2) NOT NULL DEFAULT '0.00',
  `percent_disk_space_available` decimal(10,2) NOT NULL DEFAULT '0.00',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `encrypted` tinyint(1) DEFAULT NULL,
  `gigs_total_disk_space` decimal(10,2) NOT NULL DEFAULT '0.00',
  PRIMARY KEY (`host_id`),
  KEY `idx_host_disks_gigs_disk_space_available` (`gigs_disk_space_available`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_display_names` (
  `host_id` int unsigned NOT NULL,
  `display_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`host_id`),
  KEY `display_name` (`display_name`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_emails` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int unsigned NOT NULL,
  `email` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_host_emails_host_id_email` (`host_id`,`email`),
  KEY `idx_host_emails_email` (`email`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_issues` (
  `host_id` int unsigned NOT NULL,
  `failing_policies_count` int unsigned NOT NULL DEFAULT '0',
  `critical_vulnerabilities_count` int unsigned NOT NULL DEFAULT '0',
  `total_issues_count` int unsigned NOT NULL DEFAULT '0',
  `created_at` timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`host_id`),
  KEY `total_issues_count` (`total_issues_count`)
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
CREATE TABLE `host_mdm_actions` (
  `host_id` int unsigned NOT NULL,
  `lock_ref` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `wipe_ref` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `unlock_pin` varchar(6) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `unlock_ref` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `fleet_platform` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`host_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_mdm_apple_awaiting_configuration` (
  `host_uuid` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `awaiting_configuration` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`host_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_mdm_apple_bootstrap_packages` (
  `host_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`host_uuid`),
  KEY `command_uuid` (`command_uuid`),
  CONSTRAINT `host_mdm_apple_bootstrap_packages_ibfk_1` FOREIGN KEY (`command_uuid`) REFERENCES `nano_commands` (`command_uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_mdm_apple_declarations` (
  `host_uuid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `operation_type` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `detail` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `token` binary(16) NOT NULL,
  `declaration_uuid` varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `declaration_identifier` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `declaration_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `secrets_updated_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`host_uuid`,`declaration_uuid`),
  KEY `status` (`status`),
  KEY `operation_type` (`operation_type`),
  CONSTRAINT `host_mdm_apple_declarations_ibfk_1` FOREIGN KEY (`status`) REFERENCES `mdm_delivery_status` (`status`) ON UPDATE CASCADE,
  CONSTRAINT `host_mdm_apple_declarations_ibfk_2` FOREIGN KEY (`operation_type`) REFERENCES `mdm_operation_types` (`operation_type`) ON UPDATE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_mdm_apple_profiles` (
  `profile_identifier` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `host_uuid` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `operation_type` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `detail` text COLLATE utf8mb4_unicode_ci,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `profile_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `checksum` binary(16) NOT NULL,
  `retries` tinyint unsigned NOT NULL DEFAULT '0',
  `profile_uuid` varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `secrets_updated_at` datetime(6) DEFAULT NULL,
  `ignore_error` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`host_uuid`,`profile_uuid`),
  KEY `status` (`status`),
  KEY `operation_type` (`operation_type`),
  CONSTRAINT `host_mdm_apple_profiles_ibfk_1` FOREIGN KEY (`status`) REFERENCES `mdm_delivery_status` (`status`) ON UPDATE CASCADE,
  CONSTRAINT `host_mdm_apple_profiles_ibfk_2` FOREIGN KEY (`operation_type`) REFERENCES `mdm_operation_types` (`operation_type`) ON UPDATE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_mdm_commands` (
  `host_id` int unsigned NOT NULL,
  `command_type` varchar(31) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`host_id`,`command_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_mdm_managed_certificates` (
  `host_uuid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `profile_uuid` varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `challenge_retrieved_at` timestamp(6) NULL DEFAULT NULL,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`host_uuid`,`profile_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_mdm_windows_profiles` (
  `host_uuid` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `operation_type` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `detail` text COLLATE utf8mb4_unicode_ci,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `profile_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `retries` tinyint unsigned NOT NULL DEFAULT '0',
  `profile_uuid` varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`host_uuid`,`profile_uuid`),
  KEY `status` (`status`),
  KEY `operation_type` (`operation_type`),
  CONSTRAINT `host_mdm_windows_profiles_ibfk_1` FOREIGN KEY (`status`) REFERENCES `mdm_delivery_status` (`status`) ON UPDATE CASCADE,
  CONSTRAINT `host_mdm_windows_profiles_ibfk_2` FOREIGN KEY (`operation_type`) REFERENCES `mdm_operation_types` (`operation_type`) ON UPDATE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_munki_info` (
  `host_id` int unsigned NOT NULL,
  `version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`host_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_munki_issues` (
  `host_id` int unsigned NOT NULL,
  `munki_issue_id` int unsigned NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`host_id`,`munki_issue_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_operating_system` (
  `host_id` int unsigned NOT NULL,
  `os_id` int unsigned NOT NULL,
  PRIMARY KEY (`host_id`),
  KEY `idx_host_operating_system_id` (`os_id`),
  CONSTRAINT `host_operating_system_ibfk_1` FOREIGN KEY (`os_id`) REFERENCES `operating_systems` (`id`) ON DELETE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_orbit_info` (
  `host_id` int unsigned NOT NULL,
  `version` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `desktop_version` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `scripts_enabled` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`host_id`),
  KEY `idx_host_orbit_info_version` (`version`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_script_results` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int unsigned NOT NULL,
  `execution_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `output` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `runtime` int unsigned NOT NULL DEFAULT '0',
  `exit_code` int DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `script_id` int unsigned DEFAULT NULL,
  `user_id` int unsigned DEFAULT NULL,
  `sync_request` tinyint(1) NOT NULL DEFAULT '0',
  `script_content_id` int unsigned DEFAULT NULL,
  `host_deleted_at` timestamp NULL DEFAULT NULL,
  `timeout` int DEFAULT NULL,
  `policy_id` int unsigned DEFAULT NULL,
  `setup_experience_script_id` int unsigned DEFAULT NULL,
  `is_internal` tinyint(1) DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_host_script_results_execution_id` (`execution_id`),
  KEY `idx_host_script_results_host_exit_created` (`host_id`,`exit_code`,`created_at`),
  KEY `fk_host_script_results_script_id` (`script_id`),
  KEY `idx_host_script_created_at` (`host_id`,`script_id`,`created_at`),
  KEY `fk_host_script_results_user_id` (`user_id`),
  KEY `script_content_id` (`script_content_id`),
  KEY `fk_script_result_policy_id` (`policy_id`),
  KEY `fk_host_script_results_setup_experience_id` (`setup_experience_script_id`),
  CONSTRAINT `fk_host_script_results_script_id` FOREIGN KEY (`script_id`) REFERENCES `scripts` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_host_script_results_setup_experience_id` FOREIGN KEY (`setup_experience_script_id`) REFERENCES `setup_experience_scripts` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_host_script_results_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `host_script_results_ibfk_1` FOREIGN KEY (`script_content_id`) REFERENCES `script_contents` (`id`) ON DELETE CASCADE,
  CONSTRAINT `host_script_results_ibfk_2` FOREIGN KEY (`policy_id`) REFERENCES `policies` (`id`) ON DELETE SET NULL
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_seen_times` (
  `host_id` int unsigned NOT NULL,
  `seen_time` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`host_id`),
  KEY `idx_host_seen_times_seen_time` (`seen_time`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_software` (
  `host_id` int unsigned NOT NULL,
  `software_id` bigint unsigned NOT NULL,
  `last_opened_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`host_id`,`software_id`),
  KEY `host_software_software_fk` (`software_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_software_installed_paths` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int unsigned NOT NULL,
  `software_id` bigint unsigned NOT NULL,
  `installed_path` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `team_identifier` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `host_id_software_id_idx` (`host_id`,`software_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_software_installs` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `execution_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `host_id` int unsigned NOT NULL,
  `software_installer_id` int unsigned DEFAULT NULL,
  `pre_install_query_output` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `install_script_output` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `install_script_exit_code` int DEFAULT NULL,
  `post_install_script_output` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `post_install_script_exit_code` int DEFAULT NULL,
  `user_id` int unsigned DEFAULT NULL,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `self_service` tinyint(1) NOT NULL DEFAULT '0',
  `host_deleted_at` timestamp(6) NULL DEFAULT NULL,
  `removed` tinyint NOT NULL DEFAULT '0',
  `uninstall_script_output` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `uninstall_script_exit_code` int DEFAULT NULL,
  `uninstall` tinyint unsigned NOT NULL DEFAULT '0',
  `status` enum('pending_install','failed_install','installed','pending_uninstall','failed_uninstall') COLLATE utf8mb4_unicode_ci GENERATED ALWAYS AS ((case when (`removed` = 1) then NULL when ((`post_install_script_exit_code` is not null) and (`post_install_script_exit_code` = 0)) then _utf8mb4'installed' when ((`post_install_script_exit_code` is not null) and (`post_install_script_exit_code` <> 0)) then _utf8mb4'failed_install' when ((`install_script_exit_code` is not null) and (`install_script_exit_code` = 0)) then _utf8mb4'installed' when ((`install_script_exit_code` is not null) and (`install_script_exit_code` <> 0)) then _utf8mb4'failed_install' when ((`pre_install_query_output` is not null) and (`pre_install_query_output` = _utf8mb4'')) then _utf8mb4'failed_install' when ((`host_id` is not null) and (`uninstall` = 0)) then _utf8mb4'pending_install' when ((`uninstall_script_exit_code` is not null) and (`uninstall_script_exit_code` <> 0)) then _utf8mb4'failed_uninstall' when ((`uninstall_script_exit_code` is not null) and (`uninstall_script_exit_code` = 0)) then NULL when ((`host_id` is not null) and (`uninstall` = 1)) then _utf8mb4'pending_uninstall' else NULL end)) STORED,
  `policy_id` int unsigned DEFAULT NULL,
  `installer_filename` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '[deleted installer]',
  `version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'unknown',
  `software_title_id` int unsigned DEFAULT NULL,
  `software_title_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '[deleted title]',
  `execution_status` enum('pending_install','failed_install','installed','pending_uninstall','failed_uninstall') COLLATE utf8mb4_unicode_ci GENERATED ALWAYS AS ((case when ((`post_install_script_exit_code` is not null) and (`post_install_script_exit_code` = 0)) then _utf8mb4'installed' when ((`post_install_script_exit_code` is not null) and (`post_install_script_exit_code` <> 0)) then _utf8mb4'failed_install' when ((`install_script_exit_code` is not null) and (`install_script_exit_code` = 0)) then _utf8mb4'installed' when ((`install_script_exit_code` is not null) and (`install_script_exit_code` <> 0)) then _utf8mb4'failed_install' when ((`pre_install_query_output` is not null) and (`pre_install_query_output` = _utf8mb4'')) then _utf8mb4'failed_install' when ((`host_id` is not null) and (`uninstall` = 0)) then _utf8mb4'pending_install' when ((`uninstall_script_exit_code` is not null) and (`uninstall_script_exit_code` <> 0)) then _utf8mb4'failed_uninstall' when ((`uninstall_script_exit_code` is not null) and (`uninstall_script_exit_code` = 0)) then NULL when ((`host_id` is not null) and (`uninstall` = 1)) then _utf8mb4'pending_uninstall' else NULL end)) VIRTUAL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_host_software_installs_execution_id` (`execution_id`),
  KEY `fk_host_software_installs_user_id` (`user_id`),
  KEY `idx_host_software_installs_host_installer` (`host_id`,`software_installer_id`),
  KEY `fk_software_install_policy_id` (`policy_id`),
  KEY `fk_host_software_installs_installer_id` (`software_installer_id`),
  KEY `fk_host_software_installs_software_title_id` (`software_title_id`),
  CONSTRAINT `fk_host_software_installs_installer_id` FOREIGN KEY (`software_installer_id`) REFERENCES `software_installers` (`id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_host_software_installs_software_title_id` FOREIGN KEY (`software_title_id`) REFERENCES `software_titles` (`id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_host_software_installs_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `host_software_installs_ibfk_1` FOREIGN KEY (`policy_id`) REFERENCES `policies` (`id`) ON DELETE SET NULL
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_updates` (
  `host_id` int unsigned NOT NULL,
  `software_updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`host_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_users` (
  `host_id` int unsigned NOT NULL,
  `uid` int unsigned NOT NULL,
  `username` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `groupname` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `removed_at` timestamp NULL DEFAULT NULL,
  `user_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `shell` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '',
  PRIMARY KEY (`host_id`,`uid`,`username`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `host_vpp_software_installs` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int unsigned NOT NULL,
  `adam_id` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` int unsigned DEFAULT NULL,
  `self_service` tinyint(1) NOT NULL DEFAULT '0',
  `associated_event_id` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `platform` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `removed` tinyint NOT NULL DEFAULT '0',
  `vpp_token_id` int unsigned DEFAULT NULL,
  `policy_id` int unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_host_vpp_software_installs_command_uuid` (`command_uuid`),
  KEY `user_id` (`user_id`),
  KEY `adam_id` (`adam_id`,`platform`),
  KEY `fk_host_vpp_software_installs_vpp_token_id` (`vpp_token_id`),
  KEY `fk_host_vpp_software_installs_policy_id` (`policy_id`),
  CONSTRAINT `fk_host_vpp_software_installs_vpp_token_id` FOREIGN KEY (`vpp_token_id`) REFERENCES `vpp_tokens` (`id`) ON DELETE SET NULL,
  CONSTRAINT `host_vpp_software_installs_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `host_vpp_software_installs_ibfk_3` FOREIGN KEY (`adam_id`, `platform`) REFERENCES `vpp_apps` (`adam_id`, `platform`) ON DELETE CASCADE,
  CONSTRAINT `host_vpp_software_installs_ibfk_4` FOREIGN KEY (`policy_id`) REFERENCES `policies` (`id`) ON DELETE SET NULL
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
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
CREATE TABLE `invite_teams` (
  `invite_id` int unsigned NOT NULL,
  `team_id` int unsigned NOT NULL,
  `role` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`invite_id`,`team_id`),
  KEY `fk_team_id` (`team_id`),
  CONSTRAINT `invite_teams_ibfk_1` FOREIGN KEY (`invite_id`) REFERENCES `invites` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `invite_teams_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `invites` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `invited_by` int unsigned NOT NULL,
  `email` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `position` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `token` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sso_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `global_role` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `mfa_enabled` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_invite_unique_email` (`email`),
  UNIQUE KEY `idx_invite_unique_key` (`token`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `jobs` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `args` json DEFAULT NULL,
  `state` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `retries` int NOT NULL DEFAULT '0',
  `error` text COLLATE utf8mb4_unicode_ci,
  `not_before` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_jobs_state_not_before_updated_at` (`state`,`not_before`,`updated_at`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `jobs` VALUES (1,'2024-03-20 00:00:00','2024-03-20 00:00:00','macos_setup_assistant','{\"task\": \"update_all_profiles\"}','queued',0,'','2024-03-20 00:00:00');
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
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_label_unique_name` (`name`),
  FULLTEXT KEY `labels_search` (`name`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `labels` VALUES (1,'2024-04-03 00:00:00','2024-04-03 00:00:00','macOS 14+ (Sonoma+)','macOS hosts with version 14 and above','select 1 from os_version where platform = \'darwin\' and major >= 14;','darwin',1,0),(2,'2024-06-28 00:00:00','2024-06-28 00:00:00','iOS','All iOS hosts','','ios',1,1),(3,'2024-06-28 00:00:00','2024-06-28 00:00:00','iPadOS','All iPadOS hosts','','ipados',1,1),(4,'2024-09-27 00:00:00','2024-09-27 00:00:00','Fedora Linux','All Fedora hosts','select 1 from os_version where name = \'Fedora Linux\';','rhel',1,0);
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `locks` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `owner` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `expires_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_apple_bootstrap_packages` (
  `team_id` int unsigned NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sha256` binary(32) NOT NULL,
  `bytes` longblob,
  `token` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`team_id`),
  UNIQUE KEY `idx_token` (`token`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_apple_configuration_profiles` (
  `profile_id` int unsigned NOT NULL AUTO_INCREMENT,
  `team_id` int unsigned NOT NULL DEFAULT '0',
  `identifier` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `mobileconfig` mediumblob NOT NULL,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `uploaded_at` timestamp(6) NULL DEFAULT NULL,
  `checksum` binary(16) NOT NULL,
  `profile_uuid` varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `secrets_updated_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`profile_uuid`),
  UNIQUE KEY `idx_mdm_apple_config_prof_team_identifier` (`team_id`,`identifier`),
  UNIQUE KEY `idx_mdm_apple_config_prof_team_name` (`team_id`,`name`),
  UNIQUE KEY `idx_mdm_apple_config_prof_id` (`profile_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_apple_declaration_activation_references` (
  `declaration_uuid` varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `reference` varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`declaration_uuid`,`reference`),
  KEY `reference` (`reference`),
  CONSTRAINT `mdm_apple_declaration_activation_references_ibfk_1` FOREIGN KEY (`declaration_uuid`) REFERENCES `mdm_apple_declarations` (`declaration_uuid`) ON UPDATE CASCADE,
  CONSTRAINT `mdm_apple_declaration_activation_references_ibfk_2` FOREIGN KEY (`reference`) REFERENCES `mdm_apple_declarations` (`declaration_uuid`) ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_apple_declarations` (
  `declaration_uuid` varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `team_id` int unsigned NOT NULL DEFAULT '0',
  `identifier` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `raw_json` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `uploaded_at` timestamp(6) NULL DEFAULT NULL,
  `auto_increment` bigint NOT NULL AUTO_INCREMENT,
  `secrets_updated_at` datetime(6) DEFAULT NULL,
  `token` binary(16) GENERATED ALWAYS AS (unhex(md5(concat(`raw_json`,ifnull(`secrets_updated_at`,_utf8mb4''))))) STORED,
  PRIMARY KEY (`declaration_uuid`),
  UNIQUE KEY `idx_mdm_apple_declaration_team_identifier` (`team_id`,`identifier`),
  UNIQUE KEY `idx_mdm_apple_declaration_team_name` (`team_id`,`name`),
  UNIQUE KEY `auto_increment` (`auto_increment`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_apple_declarative_requests` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `enrollment_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `message_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `raw_json` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  KEY `mdm_apple_declarative_requests_enrollment_id` (`enrollment_id`),
  CONSTRAINT `mdm_apple_declarative_requests_enrollment_id` FOREIGN KEY (`enrollment_id`) REFERENCES `nano_enrollments` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_apple_default_setup_assistants` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `team_id` int unsigned DEFAULT NULL,
  `global_or_team_id` int unsigned NOT NULL DEFAULT '0',
  `profile_uuid` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `abm_token_id` int unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mdm_default_setup_assistant_global_or_team_id_abm_token_id` (`global_or_team_id`,`abm_token_id`),
  KEY `fk_mdm_default_setup_assistant_team_id` (`team_id`),
  KEY `fk_mdm_default_setup_assistant_abm_token_id` (`abm_token_id`),
  CONSTRAINT `fk_mdm_default_setup_assistant_abm_token_id` FOREIGN KEY (`abm_token_id`) REFERENCES `abm_tokens` (`id`) ON DELETE CASCADE,
  CONSTRAINT `mdm_apple_default_setup_assistants_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_apple_enrollment_profiles` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `token` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `type` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'automatic',
  `dep_profile` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_type` (`type`),
  UNIQUE KEY `idx_token` (`token`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_apple_installers` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `size` bigint NOT NULL,
  `manifest` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `installer` longblob,
  `url_token` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_apple_setup_assistant_profiles` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `setup_assistant_id` int unsigned NOT NULL,
  `abm_token_id` int unsigned NOT NULL,
  `profile_uuid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mdm_apple_setup_assistant_profiles_asst_id_tok_id` (`setup_assistant_id`,`abm_token_id`),
  KEY `fk_mdm_apple_setup_assistant_profiles_abm_token_id` (`abm_token_id`),
  CONSTRAINT `fk_mdm_apple_setup_assistant_profiles_abm_token_id` FOREIGN KEY (`abm_token_id`) REFERENCES `abm_tokens` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_mdm_apple_setup_assistant_profiles_setup_assistant_id` FOREIGN KEY (`setup_assistant_id`) REFERENCES `mdm_apple_setup_assistants` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_apple_setup_assistants` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `team_id` int unsigned DEFAULT NULL,
  `global_or_team_id` int unsigned NOT NULL DEFAULT '0',
  `name` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `profile` json NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mdm_setup_assistant_global_or_team_id` (`global_or_team_id`),
  KEY `fk_mdm_setup_assistant_team_id` (`team_id`),
  CONSTRAINT `mdm_apple_setup_assistants_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_config_assets` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `value` longblob NOT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `deletion_uuid` varchar(127) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `md5_checksum` binary(16) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mdm_config_assets_name_deletion_uuid` (`name`,`deletion_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_configuration_profile_labels` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `apple_profile_uuid` varchar(37) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `windows_profile_uuid` varchar(37) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `label_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `label_id` int unsigned DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `exclude` tinyint(1) NOT NULL DEFAULT '0',
  `require_all` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mdm_configuration_profile_labels_apple_label_name` (`apple_profile_uuid`,`label_name`),
  UNIQUE KEY `idx_mdm_configuration_profile_labels_windows_label_name` (`windows_profile_uuid`,`label_name`),
  KEY `label_id` (`label_id`),
  CONSTRAINT `mdm_configuration_profile_labels_ibfk_1` FOREIGN KEY (`apple_profile_uuid`) REFERENCES `mdm_apple_configuration_profiles` (`profile_uuid`) ON DELETE CASCADE,
  CONSTRAINT `mdm_configuration_profile_labels_ibfk_2` FOREIGN KEY (`windows_profile_uuid`) REFERENCES `mdm_windows_configuration_profiles` (`profile_uuid`) ON DELETE CASCADE,
  CONSTRAINT `mdm_configuration_profile_labels_ibfk_3` FOREIGN KEY (`label_id`) REFERENCES `labels` (`id`) ON DELETE SET NULL,
  CONSTRAINT `ck_mdm_configuration_profile_labels_apple_or_windows` CHECK (((`apple_profile_uuid` is null) <> (`windows_profile_uuid` is null)))
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_declaration_labels` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `apple_declaration_uuid` varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `label_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `label_id` int unsigned DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `uploaded_at` timestamp NULL DEFAULT NULL,
  `exclude` tinyint(1) NOT NULL DEFAULT '0',
  `require_all` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mdm_declaration_labels_label_name` (`apple_declaration_uuid`,`label_name`),
  KEY `label_id` (`label_id`),
  CONSTRAINT `mdm_declaration_labels_ibfk_1` FOREIGN KEY (`apple_declaration_uuid`) REFERENCES `mdm_apple_declarations` (`declaration_uuid`) ON DELETE CASCADE,
  CONSTRAINT `mdm_declaration_labels_ibfk_3` FOREIGN KEY (`label_id`) REFERENCES `labels` (`id`) ON DELETE SET NULL
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_delivery_status` (
  `status` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`status`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `mdm_delivery_status` VALUES ('failed'),('pending'),('verified'),('verifying');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_idp_accounts` (
  `uuid` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `username` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `fullname` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `unique_idp_email` (`email`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_operation_types` (
  `operation_type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`operation_type`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `mdm_operation_types` VALUES ('install'),('remove');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_windows_configuration_profiles` (
  `team_id` int unsigned NOT NULL DEFAULT '0',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `syncml` mediumblob NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `uploaded_at` timestamp NULL DEFAULT NULL,
  `profile_uuid` varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `auto_increment` bigint NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`profile_uuid`),
  UNIQUE KEY `idx_mdm_windows_configuration_profiles_team_id_name` (`team_id`,`name`),
  UNIQUE KEY `auto_increment` (`auto_increment`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mdm_windows_enrollments` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
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
  `host_uuid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_type` (`mdm_hardware_id`),
  KEY `idx_mdm_windows_enrollments_mdm_device_id` (`mdm_device_id`),
  KEY `idx_mdm_windows_enrollments_host_uuid` (`host_uuid`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `migration_status_tables` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `version_id` bigint NOT NULL,
  `is_applied` tinyint(1) NOT NULL,
  `tstamp` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB AUTO_INCREMENT=350 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `migration_status_tables` VALUES (1,0,1,'2020-01-01 01:01:01'),(2,20161118193812,1,'2020-01-01 01:01:01'),(3,20161118211713,1,'2020-01-01 01:01:01'),(4,20161118212436,1,'2020-01-01 01:01:01'),(5,20161118212515,1,'2020-01-01 01:01:01'),(6,20161118212528,1,'2020-01-01 01:01:01'),(7,20161118212538,1,'2020-01-01 01:01:01'),(8,20161118212549,1,'2020-01-01 01:01:01'),(9,20161118212557,1,'2020-01-01 01:01:01'),(10,20161118212604,1,'2020-01-01 01:01:01'),(11,20161118212613,1,'2020-01-01 01:01:01'),(12,20161118212621,1,'2020-01-01 01:01:01'),(13,20161118212630,1,'2020-01-01 01:01:01'),(14,20161118212641,1,'2020-01-01 01:01:01'),(15,20161118212649,1,'2020-01-01 01:01:01'),(16,20161118212656,1,'2020-01-01 01:01:01'),(17,20161118212758,1,'2020-01-01 01:01:01'),(18,20161128234849,1,'2020-01-01 01:01:01'),(19,20161230162221,1,'2020-01-01 01:01:01'),(20,20170104113816,1,'2020-01-01 01:01:01'),(21,20170105151732,1,'2020-01-01 01:01:01'),(22,20170108191242,1,'2020-01-01 01:01:01'),(23,20170109094020,1,'2020-01-01 01:01:01'),(24,20170109130438,1,'2020-01-01 01:01:01'),(25,20170110202752,1,'2020-01-01 01:01:01'),(26,20170111133013,1,'2020-01-01 01:01:01'),(27,20170117025759,1,'2020-01-01 01:01:01'),(28,20170118191001,1,'2020-01-01 01:01:01'),(29,20170119234632,1,'2020-01-01 01:01:01'),(30,20170124230432,1,'2020-01-01 01:01:01'),(31,20170127014618,1,'2020-01-01 01:01:01'),(32,20170131232841,1,'2020-01-01 01:01:01'),(33,20170223094154,1,'2020-01-01 01:01:01'),(34,20170306075207,1,'2020-01-01 01:01:01'),(35,20170309100733,1,'2020-01-01 01:01:01'),(36,20170331111922,1,'2020-01-01 01:01:01'),(37,20170502143928,1,'2020-01-01 01:01:01'),(38,20170504130602,1,'2020-01-01 01:01:01'),(39,20170509132100,1,'2020-01-01 01:01:01'),(40,20170519105647,1,'2020-01-01 01:01:01'),(41,20170519105648,1,'2020-01-01 01:01:01'),(42,20170831234300,1,'2020-01-01 01:01:01'),(43,20170831234301,1,'2020-01-01 01:01:01'),(44,20170831234303,1,'2020-01-01 01:01:01'),(45,20171116163618,1,'2020-01-01 01:01:01'),(46,20171219164727,1,'2020-01-01 01:01:01'),(47,20180620164811,1,'2020-01-01 01:01:01'),(48,20180620175054,1,'2020-01-01 01:01:01'),(49,20180620175055,1,'2020-01-01 01:01:01'),(50,20191010101639,1,'2020-01-01 01:01:01'),(51,20191010155147,1,'2020-01-01 01:01:01'),(52,20191220130734,1,'2020-01-01 01:01:01'),(53,20200311140000,1,'2020-01-01 01:01:01'),(54,20200405120000,1,'2020-01-01 01:01:01'),(55,20200407120000,1,'2020-01-01 01:01:01'),(56,20200420120000,1,'2020-01-01 01:01:01'),(57,20200504120000,1,'2020-01-01 01:01:01'),(58,20200512120000,1,'2020-01-01 01:01:01'),(59,20200707120000,1,'2020-01-01 01:01:01'),(60,20201011162341,1,'2020-01-01 01:01:01'),(61,20201021104586,1,'2020-01-01 01:01:01'),(62,20201102112520,1,'2020-01-01 01:01:01'),(63,20201208121729,1,'2020-01-01 01:01:01'),(64,20201215091637,1,'2020-01-01 01:01:01'),(65,20210119174155,1,'2020-01-01 01:01:01'),(66,20210326182902,1,'2020-01-01 01:01:01'),(67,20210421112652,1,'2020-01-01 01:01:01'),(68,20210506095025,1,'2020-01-01 01:01:01'),(69,20210513115729,1,'2020-01-01 01:01:01'),(70,20210526113559,1,'2020-01-01 01:01:01'),(71,20210601000001,1,'2020-01-01 01:01:01'),(72,20210601000002,1,'2020-01-01 01:01:01'),(73,20210601000003,1,'2020-01-01 01:01:01'),(74,20210601000004,1,'2020-01-01 01:01:01'),(75,20210601000005,1,'2020-01-01 01:01:01'),(76,20210601000006,1,'2020-01-01 01:01:01'),(77,20210601000007,1,'2020-01-01 01:01:01'),(78,20210601000008,1,'2020-01-01 01:01:01'),(79,20210606151329,1,'2020-01-01 01:01:01'),(80,20210616163757,1,'2020-01-01 01:01:01'),(81,20210617174723,1,'2020-01-01 01:01:01'),(82,20210622160235,1,'2020-01-01 01:01:01'),(83,20210623100031,1,'2020-01-01 01:01:01'),(84,20210623133615,1,'2020-01-01 01:01:01'),(85,20210708143152,1,'2020-01-01 01:01:01'),(86,20210709124443,1,'2020-01-01 01:01:01'),(87,20210712155608,1,'2020-01-01 01:01:01'),(88,20210714102108,1,'2020-01-01 01:01:01'),(89,20210719153709,1,'2020-01-01 01:01:01'),(90,20210721171531,1,'2020-01-01 01:01:01'),(91,20210723135713,1,'2020-01-01 01:01:01'),(92,20210802135933,1,'2020-01-01 01:01:01'),(93,20210806112844,1,'2020-01-01 01:01:01'),(94,20210810095603,1,'2020-01-01 01:01:01'),(95,20210811150223,1,'2020-01-01 01:01:01'),(96,20210818151827,1,'2020-01-01 01:01:01'),(97,20210818151828,1,'2020-01-01 01:01:01'),(98,20210818182258,1,'2020-01-01 01:01:01'),(99,20210819131107,1,'2020-01-01 01:01:01'),(100,20210819143446,1,'2020-01-01 01:01:01'),(101,20210903132338,1,'2020-01-01 01:01:01'),(102,20210915144307,1,'2020-01-01 01:01:01'),(103,20210920155130,1,'2020-01-01 01:01:01'),(104,20210927143115,1,'2020-01-01 01:01:01'),(105,20210927143116,1,'2020-01-01 01:01:01'),(106,20211013133706,1,'2020-01-01 01:01:01'),(107,20211013133707,1,'2020-01-01 01:01:01'),(108,20211102135149,1,'2020-01-01 01:01:01'),(109,20211109121546,1,'2020-01-01 01:01:01'),(110,20211110163320,1,'2020-01-01 01:01:01'),(111,20211116184029,1,'2020-01-01 01:01:01'),(112,20211116184030,1,'2020-01-01 01:01:01'),(113,20211202092042,1,'2020-01-01 01:01:01'),(114,20211202181033,1,'2020-01-01 01:01:01'),(115,20211207161856,1,'2020-01-01 01:01:01'),(116,20211216131203,1,'2020-01-01 01:01:01'),(117,20211221110132,1,'2020-01-01 01:01:01'),(118,20220107155700,1,'2020-01-01 01:01:01'),(119,20220125105650,1,'2020-01-01 01:01:01'),(120,20220201084510,1,'2020-01-01 01:01:01'),(121,20220208144830,1,'2020-01-01 01:01:01'),(122,20220208144831,1,'2020-01-01 01:01:01'),(123,20220215152203,1,'2020-01-01 01:01:01'),(124,20220223113157,1,'2020-01-01 01:01:01'),(125,20220307104655,1,'2020-01-01 01:01:01'),(126,20220309133956,1,'2020-01-01 01:01:01'),(127,20220316155700,1,'2020-01-01 01:01:01'),(128,20220323152301,1,'2020-01-01 01:01:01'),(129,20220330100659,1,'2020-01-01 01:01:01'),(130,20220404091216,1,'2020-01-01 01:01:01'),(131,20220419140750,1,'2020-01-01 01:01:01'),(132,20220428140039,1,'2020-01-01 01:01:01'),(133,20220503134048,1,'2020-01-01 01:01:01'),(134,20220524102918,1,'2020-01-01 01:01:01'),(135,20220526123327,1,'2020-01-01 01:01:01'),(136,20220526123328,1,'2020-01-01 01:01:01'),(137,20220526123329,1,'2020-01-01 01:01:01'),(138,20220608113128,1,'2020-01-01 01:01:01'),(139,20220627104817,1,'2020-01-01 01:01:01'),(140,20220704101843,1,'2020-01-01 01:01:01'),(141,20220708095046,1,'2020-01-01 01:01:01'),(142,20220713091130,1,'2020-01-01 01:01:01'),(143,20220802135510,1,'2020-01-01 01:01:01'),(144,20220818101352,1,'2020-01-01 01:01:01'),(145,20220822161445,1,'2020-01-01 01:01:01'),(146,20220831100036,1,'2020-01-01 01:01:01'),(147,20220831100151,1,'2020-01-01 01:01:01'),(148,20220908181826,1,'2020-01-01 01:01:01'),(149,20220914154915,1,'2020-01-01 01:01:01'),(150,20220915165115,1,'2020-01-01 01:01:01'),(151,20220915165116,1,'2020-01-01 01:01:01'),(152,20220928100158,1,'2020-01-01 01:01:01'),(153,20221014084130,1,'2020-01-01 01:01:01'),(154,20221027085019,1,'2020-01-01 01:01:01'),(155,20221101103952,1,'2020-01-01 01:01:01'),(156,20221104144401,1,'2020-01-01 01:01:01'),(157,20221109100749,1,'2020-01-01 01:01:01'),(158,20221115104546,1,'2020-01-01 01:01:01'),(159,20221130114928,1,'2020-01-01 01:01:01'),(160,20221205112142,1,'2020-01-01 01:01:01'),(161,20221216115820,1,'2020-01-01 01:01:01'),(162,20221220195934,1,'2020-01-01 01:01:01'),(163,20221220195935,1,'2020-01-01 01:01:01'),(164,20221223174807,1,'2020-01-01 01:01:01'),(165,20221227163855,1,'2020-01-01 01:01:01'),(166,20221227163856,1,'2020-01-01 01:01:01'),(167,20230202224725,1,'2020-01-01 01:01:01'),(168,20230206163608,1,'2020-01-01 01:01:01'),(169,20230214131519,1,'2020-01-01 01:01:01'),(170,20230303135738,1,'2020-01-01 01:01:01'),(171,20230313135301,1,'2020-01-01 01:01:01'),(172,20230313141819,1,'2020-01-01 01:01:01'),(173,20230315104937,1,'2020-01-01 01:01:01'),(174,20230317173844,1,'2020-01-01 01:01:01'),(175,20230320133602,1,'2020-01-01 01:01:01'),(176,20230330100011,1,'2020-01-01 01:01:01'),(177,20230330134823,1,'2020-01-01 01:01:01'),(178,20230405232025,1,'2020-01-01 01:01:01'),(179,20230408084104,1,'2020-01-01 01:01:01'),(180,20230411102858,1,'2020-01-01 01:01:01'),(181,20230421155932,1,'2020-01-01 01:01:01'),(182,20230425082126,1,'2020-01-01 01:01:01'),(183,20230425105727,1,'2020-01-01 01:01:01'),(184,20230501154913,1,'2020-01-01 01:01:01'),(185,20230503101418,1,'2020-01-01 01:01:01'),(186,20230515144206,1,'2020-01-01 01:01:01'),(187,20230517140952,1,'2020-01-01 01:01:01'),(188,20230517152807,1,'2020-01-01 01:01:01'),(189,20230518114155,1,'2020-01-01 01:01:01'),(190,20230520153236,1,'2020-01-01 01:01:01'),(191,20230525151159,1,'2020-01-01 01:01:01'),(192,20230530122103,1,'2020-01-01 01:01:01'),(193,20230602111827,1,'2020-01-01 01:01:01'),(194,20230608103123,1,'2020-01-01 01:01:01'),(195,20230629140529,1,'2020-01-01 01:01:01'),(196,20230629140530,1,'2020-01-01 01:01:01'),(197,20230711144622,1,'2020-01-01 01:01:01'),(198,20230721135421,1,'2020-01-01 01:01:01'),(199,20230721161508,1,'2020-01-01 01:01:01'),(200,20230726115701,1,'2020-01-01 01:01:01'),(201,20230807100822,1,'2020-01-01 01:01:01'),(202,20230814150442,1,'2020-01-01 01:01:01'),(203,20230823122728,1,'2020-01-01 01:01:01'),(204,20230906152143,1,'2020-01-01 01:01:01'),(205,20230911163618,1,'2020-01-01 01:01:01'),(206,20230912101759,1,'2020-01-01 01:01:01'),(207,20230915101341,1,'2020-01-01 01:01:01'),(208,20230918132351,1,'2020-01-01 01:01:01'),(209,20231004144339,1,'2020-01-01 01:01:01'),(210,20231009094541,1,'2020-01-01 01:01:01'),(211,20231009094542,1,'2020-01-01 01:01:01'),(212,20231009094543,1,'2020-01-01 01:01:01'),(213,20231009094544,1,'2020-01-01 01:01:01'),(214,20231016091915,1,'2020-01-01 01:01:01'),(215,20231024174135,1,'2020-01-01 01:01:01'),(216,20231025120016,1,'2020-01-01 01:01:01'),(217,20231025160156,1,'2020-01-01 01:01:01'),(218,20231031165350,1,'2020-01-01 01:01:01'),(219,20231106144110,1,'2020-01-01 01:01:01'),(220,20231107130934,1,'2020-01-01 01:01:01'),(221,20231109115838,1,'2020-01-01 01:01:01'),(222,20231121054530,1,'2020-01-01 01:01:01'),(223,20231122101320,1,'2020-01-01 01:01:01'),(224,20231130132828,1,'2020-01-01 01:01:01'),(225,20231130132931,1,'2020-01-01 01:01:01'),(226,20231204155427,1,'2020-01-01 01:01:01'),(227,20231206142340,1,'2020-01-01 01:01:01'),(228,20231207102320,1,'2020-01-01 01:01:01'),(229,20231207102321,1,'2020-01-01 01:01:01'),(230,20231207133731,1,'2020-01-01 01:01:01'),(231,20231212094238,1,'2020-01-01 01:01:01'),(232,20231212095734,1,'2020-01-01 01:01:01'),(233,20231212161121,1,'2020-01-01 01:01:01'),(234,20231215122713,1,'2020-01-01 01:01:01'),(235,20231219143041,1,'2020-01-01 01:01:01'),(236,20231224070653,1,'2020-01-01 01:01:01'),(237,20240110134315,1,'2020-01-01 01:01:01'),(238,20240119091637,1,'2020-01-01 01:01:01'),(239,20240126020642,1,'2020-01-01 01:01:01'),(240,20240126020643,1,'2020-01-01 01:01:01'),(241,20240129162819,1,'2020-01-01 01:01:01'),(242,20240130115133,1,'2020-01-01 01:01:01'),(243,20240131083822,1,'2020-01-01 01:01:01'),(244,20240205095928,1,'2020-01-01 01:01:01'),(245,20240205121956,1,'2020-01-01 01:01:01'),(246,20240209110212,1,'2020-01-01 01:01:01'),(247,20240212111533,1,'2020-01-01 01:01:01'),(248,20240221112844,1,'2020-01-01 01:01:01'),(249,20240222073518,1,'2020-01-01 01:01:01'),(250,20240222135115,1,'2020-01-01 01:01:01'),(251,20240226082255,1,'2020-01-01 01:01:01'),(252,20240228082706,1,'2020-01-01 01:01:01'),(253,20240301173035,1,'2020-01-01 01:01:01'),(254,20240302111134,1,'2020-01-01 01:01:01'),(255,20240312103753,1,'2020-01-01 01:01:01'),(256,20240313143416,1,'2020-01-01 01:01:01'),(257,20240314085226,1,'2020-01-01 01:01:01'),(258,20240314151747,1,'2020-01-01 01:01:01'),(259,20240320145650,1,'2020-01-01 01:01:01'),(260,20240327115530,1,'2020-01-01 01:01:01'),(261,20240327115617,1,'2020-01-01 01:01:01'),(262,20240408085837,1,'2020-01-01 01:01:01'),(263,20240415104633,1,'2020-01-01 01:01:01'),(264,20240430111727,1,'2020-01-01 01:01:01'),(265,20240515200020,1,'2020-01-01 01:01:01'),(266,20240521143023,1,'2020-01-01 01:01:01'),(267,20240521143024,1,'2020-01-01 01:01:01'),(268,20240601174138,1,'2020-01-01 01:01:01'),(269,20240607133721,1,'2020-01-01 01:01:01'),(270,20240612150059,1,'2020-01-01 01:01:01'),(271,20240613162201,1,'2020-01-01 01:01:01'),(272,20240613172616,1,'2020-01-01 01:01:01'),(273,20240618142419,1,'2020-01-01 01:01:01'),(274,20240625093543,1,'2020-01-01 01:01:01'),(275,20240626195531,1,'2020-01-01 01:01:01'),(276,20240702123921,1,'2020-01-01 01:01:01'),(277,20240703154849,1,'2020-01-01 01:01:01'),(278,20240707134035,1,'2020-01-01 01:01:01'),(279,20240707134036,1,'2020-01-01 01:01:01'),(280,20240709124958,1,'2020-01-01 01:01:01'),(281,20240709132642,1,'2020-01-01 01:01:01'),(282,20240709183940,1,'2020-01-01 01:01:01'),(283,20240710155623,1,'2020-01-01 01:01:01'),(284,20240723102712,1,'2020-01-01 01:01:01'),(285,20240725152735,1,'2020-01-01 01:01:01'),(286,20240725182118,1,'2020-01-01 01:01:01'),(287,20240726100517,1,'2020-01-01 01:01:01'),(288,20240730171504,1,'2020-01-01 01:01:01'),(289,20240730174056,1,'2020-01-01 01:01:01'),(290,20240730215453,1,'2020-01-01 01:01:01'),(291,20240730374423,1,'2020-01-01 01:01:01'),(292,20240801115359,1,'2020-01-01 01:01:01'),(293,20240802101043,1,'2020-01-01 01:01:01'),(294,20240802113716,1,'2020-01-01 01:01:01'),(295,20240814135330,1,'2020-01-01 01:01:01'),(296,20240815000000,1,'2020-01-01 01:01:01'),(297,20240815000001,1,'2020-01-01 01:01:01'),(298,20240816103247,1,'2020-01-01 01:01:01'),(299,20240820091218,1,'2020-01-01 01:01:01'),(300,20240826111228,1,'2020-01-01 01:01:01'),(301,20240826160025,1,'2020-01-01 01:01:01'),(302,20240829165448,1,'2020-01-01 01:01:01'),(303,20240829165605,1,'2020-01-01 01:01:01'),(304,20240829165715,1,'2020-01-01 01:01:01'),(305,20240829165930,1,'2020-01-01 01:01:01'),(306,20240829170023,1,'2020-01-01 01:01:01'),(307,20240829170033,1,'2020-01-01 01:01:01'),(308,20240829170044,1,'2020-01-01 01:01:01'),(309,20240905105135,1,'2020-01-01 01:01:01'),(310,20240905140514,1,'2020-01-01 01:01:01'),(311,20240905200000,1,'2020-01-01 01:01:01'),(312,20240905200001,1,'2020-01-01 01:01:01'),(313,20241002104104,1,'2020-01-01 01:01:01'),(314,20241002104105,1,'2020-01-01 01:01:01'),(315,20241002104106,1,'2020-01-01 01:01:01'),(316,20241002210000,1,'2020-01-01 01:01:01'),(317,20241003145349,1,'2020-01-01 01:01:01'),(318,20241004005000,1,'2020-01-01 01:01:01'),(319,20241008083925,1,'2020-01-01 01:01:01'),(320,20241009090010,1,'2020-01-01 01:01:01'),(321,20241017163402,1,'2020-01-01 01:01:01'),(322,20241021224359,1,'2020-01-01 01:01:01'),(323,20241022140321,1,'2020-01-01 01:01:01'),(324,20241025111236,1,'2020-01-01 01:01:01'),(325,20241025112748,1,'2020-01-01 01:01:01'),(326,20241025141855,1,'2020-01-01 01:01:01'),(327,20241110152839,1,'2020-01-01 01:01:01'),(328,20241110152840,1,'2020-01-01 01:01:01'),(329,20241110152841,1,'2020-01-01 01:01:01'),(330,20241116233322,1,'2020-01-01 01:01:01'),(331,20241122171434,1,'2020-01-01 01:01:01'),(332,20241125150614,1,'2020-01-01 01:01:01'),(333,20241203125346,1,'2020-01-01 01:01:01'),(334,20241203130032,1,'2020-01-01 01:01:01'),(335,20241205122800,1,'2020-01-01 01:01:01'),(336,20241209164540,1,'2020-01-01 01:01:01'),(337,20241210140021,1,'2020-01-01 01:01:01'),(338,20241219180042,1,'2020-01-01 01:01:01'),(339,20241220100000,1,'2020-01-01 01:01:01'),(340,20241220114903,1,'2020-01-01 01:01:01'),(341,20241220114904,1,'2020-01-01 01:01:01'),(342,20241224000000,1,'2020-01-01 01:01:01'),(343,20241230000000,1,'2020-01-01 01:01:01'),(344,20241231112624,1,'2020-01-01 01:01:01'),(345,20250102121439,1,'2020-01-01 01:01:01'),(346,20250107165731,1,'2020-01-01 01:01:01'),(347,20250109150150,1,'2020-01-01 01:01:01'),(348,20250110205257,1,'2020-01-01 01:01:01'),(349,20250121094045,1,'2020-01-01 01:01:01');
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
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `munki_issues` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `issue_type` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_munki_issues_name` (`name`,`issue_type`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `nano_cert_auth_associations` (
  `id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sha256` char(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `cert_not_valid_after` timestamp NULL DEFAULT NULL,
  `renew_command_uuid` varchar(127) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`,`sha256`),
  KEY `renew_command_uuid_fk` (`renew_command_uuid`),
  CONSTRAINT `renew_command_uuid_fk` FOREIGN KEY (`renew_command_uuid`) REFERENCES `nano_commands` (`command_uuid`),
  CONSTRAINT `nano_cert_auth_associations_chk_1` CHECK ((`id` <> _utf8mb4'')),
  CONSTRAINT `nano_cert_auth_associations_chk_2` CHECK ((`sha256` <> _utf8mb4''))
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `nano_command_results` (
  `id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(31) COLLATE utf8mb4_unicode_ci NOT NULL,
  `result` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `not_now_at` timestamp NULL DEFAULT NULL,
  `not_now_tally` int NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`,`command_uuid`),
  KEY `command_uuid` (`command_uuid`),
  KEY `status` (`status`),
  CONSTRAINT `nano_command_results_ibfk_1` FOREIGN KEY (`id`) REFERENCES `nano_enrollments` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `nano_command_results_ibfk_2` FOREIGN KEY (`command_uuid`) REFERENCES `nano_commands` (`command_uuid`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `nano_command_results_chk_1` CHECK ((`status` <> _utf8mb4'')),
  CONSTRAINT `nano_command_results_chk_2` CHECK ((substr(`result`,1,5) = _utf8mb4'<?xml'))
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `nano_commands` (
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `request_type` varchar(63) COLLATE utf8mb4_unicode_ci NOT NULL,
  `command` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `subtype` enum('None','ProfileWithSecrets') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'None',
  PRIMARY KEY (`command_uuid`),
  CONSTRAINT `nano_commands_chk_1` CHECK ((`command_uuid` <> _utf8mb4'')),
  CONSTRAINT `nano_commands_chk_2` CHECK ((`request_type` <> _utf8mb4''))
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`name`),
  CONSTRAINT `nano_dep_names_chk_1` CHECK (((`tokenpki_cert_pem` is null) or (substr(`tokenpki_cert_pem`,1,27) = _utf8mb4'-----BEGIN CERTIFICATE-----'))),
  CONSTRAINT `nano_dep_names_chk_2` CHECK (((`tokenpki_key_pem` is null) or (substr(`tokenpki_key_pem`,1,5) = _utf8mb4'-----')))
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `serial_number` (`serial_number`),
  CONSTRAINT `nano_devices_chk_1` CHECK (((`identity_cert` is null) or (substr(`identity_cert`,1,27) = _utf8mb4'-----BEGIN CERTIFICATE-----'))),
  CONSTRAINT `nano_devices_chk_2` CHECK (((`serial_number` is null) or (`serial_number` <> _utf8mb4''))),
  CONSTRAINT `nano_devices_chk_3` CHECK (((`unlock_token` is null) or (length(`unlock_token`) > 0))),
  CONSTRAINT `nano_devices_chk_4` CHECK ((`authenticate` <> _utf8mb4'')),
  CONSTRAINT `nano_devices_chk_5` CHECK (((`token_update` is null) or (`token_update` <> _utf8mb4''))),
  CONSTRAINT `nano_devices_chk_6` CHECK (((`bootstrap_token_b64` is null) or (`bootstrap_token_b64` <> _utf8mb4'')))
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `nano_enrollment_queue` (
  `id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `active` tinyint(1) NOT NULL DEFAULT '1',
  `priority` tinyint NOT NULL DEFAULT '0',
  `created_at` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`,`command_uuid`),
  KEY `command_uuid` (`command_uuid`),
  KEY `priority` (`priority` DESC,`created_at`),
  CONSTRAINT `nano_enrollment_queue_ibfk_1` FOREIGN KEY (`id`) REFERENCES `nano_enrollments` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `nano_enrollment_queue_ibfk_2` FOREIGN KEY (`command_uuid`) REFERENCES `nano_commands` (`command_uuid`) ON DELETE CASCADE ON UPDATE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `nano_enrollments` (
  `id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `device_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `type` varchar(31) COLLATE utf8mb4_unicode_ci NOT NULL,
  `topic` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `push_magic` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `token_hex` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `token_update_tally` int NOT NULL DEFAULT '1',
  `last_seen_at` timestamp NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `enrolled_from_migration` tinyint unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_id` (`user_id`),
  KEY `device_id` (`device_id`),
  KEY `type` (`type`),
  CONSTRAINT `nano_enrollments_ibfk_1` FOREIGN KEY (`device_id`) REFERENCES `nano_devices` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `nano_enrollments_ibfk_2` FOREIGN KEY (`user_id`) REFERENCES `nano_users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `nano_enrollments_chk_1` CHECK ((`id` <> _utf8mb4'')),
  CONSTRAINT `nano_enrollments_chk_2` CHECK ((`type` <> _utf8mb4'')),
  CONSTRAINT `nano_enrollments_chk_3` CHECK ((`topic` <> _utf8mb4'')),
  CONSTRAINT `nano_enrollments_chk_4` CHECK ((`push_magic` <> _utf8mb4'')),
  CONSTRAINT `nano_enrollments_chk_5` CHECK ((`token_hex` <> _utf8mb4''))
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `nano_push_certs` (
  `topic` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `cert_pem` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `key_pem` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `stale_token` int NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`topic`),
  CONSTRAINT `nano_push_certs_chk_1` CHECK ((`topic` <> _utf8mb4'')),
  CONSTRAINT `nano_push_certs_chk_2` CHECK ((substr(`cert_pem`,1,27) = _utf8mb4'-----BEGIN CERTIFICATE-----')),
  CONSTRAINT `nano_push_certs_chk_3` CHECK ((substr(`key_pem`,1,5) = _utf8mb4'-----'))
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`,`device_id`),
  UNIQUE KEY `idx_unique_id` (`id`),
  KEY `device_id` (`device_id`),
  CONSTRAINT `nano_users_ibfk_1` FOREIGN KEY (`device_id`) REFERENCES `nano_devices` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `nano_users_chk_1` CHECK (((`user_short_name` is null) or (`user_short_name` <> _utf8mb4''))),
  CONSTRAINT `nano_users_chk_2` CHECK (((`user_long_name` is null) or (`user_long_name` <> _utf8mb4''))),
  CONSTRAINT `nano_users_chk_3` CHECK (((`token_update` is null) or (`token_update` <> _utf8mb4''))),
  CONSTRAINT `nano_users_chk_4` CHECK (((`user_authenticate` is null) or (`user_authenticate` <> _utf8mb4''))),
  CONSTRAINT `nano_users_chk_5` CHECK (((`user_authenticate_digest` is null) or (`user_authenticate_digest` <> _utf8mb4'')))
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
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
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `network_interfaces` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int unsigned NOT NULL,
  `mac` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `ip_address` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `broadcast` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `ibytes` bigint NOT NULL DEFAULT '0',
  `interface` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `ipackets` bigint NOT NULL DEFAULT '0',
  `last_change` bigint NOT NULL DEFAULT '0',
  `mask` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `metric` int NOT NULL DEFAULT '0',
  `mtu` int NOT NULL DEFAULT '0',
  `obytes` bigint NOT NULL DEFAULT '0',
  `ierrors` bigint NOT NULL DEFAULT '0',
  `oerrors` bigint NOT NULL DEFAULT '0',
  `opackets` bigint NOT NULL DEFAULT '0',
  `point_to_point` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `type` int NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_network_interfaces_unique_ip_host_intf` (`ip_address`,`host_id`,`interface`),
  KEY `idx_network_interfaces_hosts_fk` (`host_id`),
  FULLTEXT KEY `ip_address_search` (`ip_address`),
  CONSTRAINT `network_interfaces_ibfk_1` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `operating_system_vulnerabilities` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `operating_system_id` int unsigned NOT NULL,
  `cve` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source` smallint DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `resolved_in_version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_os_vulnerabilities_unq_os_id_cve` (`operating_system_id`,`cve`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `operating_systems` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `arch` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `kernel_version` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `platform` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `display_version` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `os_version_id` int unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_os` (`name`,`version`,`arch`,`kernel_version`,`platform`,`display_version`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `osquery_options` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `override_type` int NOT NULL,
  `override_identifier` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `options` json NOT NULL,
  PRIMARY KEY (`id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `osquery_options` VALUES (1,0,'','{\"options\": {\"logger_plugin\": \"tls\", \"pack_delimiter\": \"/\", \"logger_tls_period\": 10, \"distributed_plugin\": \"tls\", \"disable_distributed\": false, \"logger_tls_endpoint\": \"/api/v1/osquery/log\", \"distributed_interval\": 10, \"distributed_tls_max_attempts\": 3}, \"decorators\": {\"load\": [\"SELECT uuid AS host_uuid FROM system_info;\", \"SELECT hostname AS hostname FROM system_info;\"]}}');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pack_targets` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `pack_id` int unsigned DEFAULT NULL,
  `type` int DEFAULT NULL,
  `target_id` int unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `constraint_pack_target_unique` (`pack_id`,`target_id`,`type`),
  CONSTRAINT `pack_targets_ibfk_1` FOREIGN KEY (`pack_id`) REFERENCES `packs` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `packs` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `disabled` tinyint(1) NOT NULL DEFAULT '0',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `platform` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `pack_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_pack_unique_name` (`name`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `password_reset_requests` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `expires_at` timestamp NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `user_id` int unsigned NOT NULL,
  `token` varchar(1024) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `policies` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `team_id` int unsigned DEFAULT NULL,
  `resolution` text COLLATE utf8mb4_unicode_ci,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `query` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `author_id` int unsigned DEFAULT NULL,
  `platforms` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `critical` tinyint(1) NOT NULL DEFAULT '0',
  `checksum` binary(16) NOT NULL,
  `calendar_events_enabled` tinyint unsigned NOT NULL DEFAULT '0',
  `software_installer_id` int unsigned DEFAULT NULL,
  `script_id` int unsigned DEFAULT NULL,
  `vpp_apps_teams_id` int unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_policies_checksum` (`checksum`),
  KEY `idx_policies_author_id` (`author_id`),
  KEY `idx_policies_team_id` (`team_id`),
  KEY `fk_policies_software_installer_id` (`software_installer_id`),
  KEY `fk_policies_script_id` (`script_id`),
  KEY `fk_policies_vpp_apps_team_id` (`vpp_apps_teams_id`),
  CONSTRAINT `policies_ibfk_3` FOREIGN KEY (`software_installer_id`) REFERENCES `software_installers` (`id`),
  CONSTRAINT `policies_ibfk_4` FOREIGN KEY (`script_id`) REFERENCES `scripts` (`id`),
  CONSTRAINT `policies_ibfk_5` FOREIGN KEY (`vpp_apps_teams_id`) REFERENCES `vpp_apps_teams` (`id`),
  CONSTRAINT `policies_queries_ibfk_1` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `policy_automation_iterations` (
  `policy_id` int unsigned NOT NULL,
  `iteration` int NOT NULL,
  PRIMARY KEY (`policy_id`),
  CONSTRAINT `policy_automation_iterations_ibfk_1` FOREIGN KEY (`policy_id`) REFERENCES `policies` (`id`) ON DELETE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `policy_membership` (
  `policy_id` int unsigned NOT NULL,
  `host_id` int unsigned NOT NULL,
  `passes` tinyint(1) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `automation_iteration` int DEFAULT NULL,
  PRIMARY KEY (`policy_id`,`host_id`),
  KEY `idx_policy_membership_passes` (`passes`),
  KEY `idx_policy_membership_host_id_passes` (`host_id`,`passes`),
  CONSTRAINT `policy_membership_ibfk_1` FOREIGN KEY (`policy_id`) REFERENCES `policies` (`id`) ON DELETE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `policy_stats` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `policy_id` int unsigned NOT NULL,
  `inherited_team_id` int unsigned DEFAULT NULL,
  `passing_host_count` mediumint unsigned NOT NULL DEFAULT '0',
  `failing_host_count` mediumint unsigned NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `inherited_team_id_char` char(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci GENERATED ALWAYS AS (if((`inherited_team_id` is null),_utf8mb4'global',cast(`inherited_team_id` as char charset utf8mb4))) VIRTUAL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `policy_id` (`policy_id`,`inherited_team_id_char`),
  CONSTRAINT `policy_stats_ibfk_1` FOREIGN KEY (`policy_id`) REFERENCES `policies` (`id`) ON DELETE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `queries` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `saved` tinyint(1) NOT NULL DEFAULT '0',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `query` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `author_id` int unsigned DEFAULT NULL,
  `observer_can_run` tinyint(1) NOT NULL DEFAULT '0',
  `team_id` int unsigned DEFAULT NULL,
  `team_id_char` char(10) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `platform` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `min_osquery_version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `schedule_interval` int unsigned NOT NULL DEFAULT '0',
  `automations_enabled` tinyint unsigned NOT NULL DEFAULT '0',
  `logging_type` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'snapshot',
  `discard_data` tinyint(1) NOT NULL DEFAULT '1',
  `is_scheduled` tinyint(1) GENERATED ALWAYS AS ((`schedule_interval` > 0)) STORED NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_team_id_name_unq` (`team_id_char`,`name`),
  UNIQUE KEY `idx_name_team_id_unq` (`name`,`team_id_char`),
  KEY `author_id` (`author_id`),
  KEY `idx_team_id_saved_auto_interval` (`team_id`,`saved`,`automations_enabled`,`schedule_interval`),
  KEY `idx_queries_schedule_automations` (`is_scheduled`,`automations_enabled`),
  CONSTRAINT `queries_ibfk_1` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `queries_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `query_results` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `query_id` int unsigned NOT NULL,
  `host_id` int unsigned NOT NULL,
  `osquery_version` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `error` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `last_fetched` timestamp NOT NULL,
  `data` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_query_id_host_id_last_fetched` (`query_id`,`host_id`,`last_fetched`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `scep_certificates` (
  `serial` bigint NOT NULL,
  `name` varchar(1024) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `not_valid_before` datetime NOT NULL,
  `not_valid_after` datetime NOT NULL,
  `certificate_pem` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `revoked` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`serial`),
  CONSTRAINT `scep_certificates_ibfk_1` FOREIGN KEY (`serial`) REFERENCES `scep_serials` (`serial`),
  CONSTRAINT `scep_certificates_chk_1` CHECK ((substr(`certificate_pem`,1,27) = _utf8mb4'-----BEGIN CERTIFICATE-----')),
  CONSTRAINT `scep_certificates_chk_2` CHECK (((`name` is null) or (`name` <> _utf8mb4'')))
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `scep_serials` (
  `serial` bigint NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`serial`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `scheduled_queries` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `pack_id` int unsigned DEFAULT NULL,
  `query_id` int unsigned DEFAULT NULL,
  `interval` int unsigned DEFAULT NULL,
  `snapshot` tinyint(1) DEFAULT NULL,
  `removed` tinyint(1) DEFAULT NULL,
  `platform` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `shard` int unsigned DEFAULT NULL,
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
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `scheduled_query_stats` (
  `host_id` int unsigned NOT NULL,
  `scheduled_query_id` int unsigned NOT NULL,
  `average_memory` bigint unsigned NOT NULL,
  `denylisted` tinyint(1) DEFAULT NULL,
  `executions` bigint unsigned NOT NULL,
  `schedule_interval` int DEFAULT NULL,
  `last_executed` timestamp NULL DEFAULT NULL,
  `output_size` bigint unsigned NOT NULL,
  `system_time` bigint unsigned NOT NULL,
  `user_time` bigint unsigned NOT NULL,
  `wall_time` bigint unsigned NOT NULL,
  `query_type` tinyint NOT NULL DEFAULT '0',
  PRIMARY KEY (`host_id`,`scheduled_query_id`,`query_type`),
  KEY `scheduled_query_id` (`scheduled_query_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `script_contents` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `md5_checksum` binary(16) NOT NULL,
  `contents` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_script_contents_md5_checksum` (`md5_checksum`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `scripts` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `team_id` int unsigned DEFAULT NULL,
  `global_or_team_id` int unsigned NOT NULL DEFAULT '0',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `script_content_id` int unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_scripts_global_or_team_id_name` (`global_or_team_id`,`name`),
  UNIQUE KEY `idx_scripts_team_name` (`team_id`,`name`),
  KEY `script_content_id` (`script_content_id`),
  CONSTRAINT `scripts_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `scripts_ibfk_2` FOREIGN KEY (`script_content_id`) REFERENCES `script_contents` (`id`) ON DELETE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `secret_variables` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `value` blob NOT NULL,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_secret_variables_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sessions` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `accessed_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `user_id` int unsigned NOT NULL,
  `key` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_session_unique_key` (`key`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `setup_experience_scripts` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `team_id` int unsigned DEFAULT NULL,
  `global_or_team_id` int unsigned NOT NULL DEFAULT '0',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `script_content_id` int unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_setup_experience_scripts_global_or_team_id` (`global_or_team_id`),
  KEY `idx_script_content_id` (`script_content_id`),
  KEY `fk_setup_experience_scripts_ibfk_1` (`team_id`),
  CONSTRAINT `fk_setup_experience_scripts_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_setup_experience_scripts_ibfk_2` FOREIGN KEY (`script_content_id`) REFERENCES `script_contents` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `setup_experience_status_results` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `host_uuid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` enum('pending','running','success','failure') COLLATE utf8mb4_unicode_ci NOT NULL,
  `software_installer_id` int unsigned DEFAULT NULL,
  `host_software_installs_execution_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `vpp_app_team_id` int unsigned DEFAULT NULL,
  `nano_command_uuid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `setup_experience_script_id` int unsigned DEFAULT NULL,
  `script_execution_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `error` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_setup_experience_scripts_host_uuid` (`host_uuid`),
  KEY `idx_setup_experience_scripts_hsi_id` (`host_software_installs_execution_id`),
  KEY `idx_setup_experience_scripts_nano_command_uuid` (`nano_command_uuid`),
  KEY `idx_setup_experience_scripts_script_execution_id` (`script_execution_id`),
  KEY `fk_setup_experience_status_results_si_id` (`software_installer_id`),
  KEY `fk_setup_experience_status_results_va_id` (`vpp_app_team_id`),
  KEY `fk_setup_experience_status_results_ses_id` (`setup_experience_script_id`),
  CONSTRAINT `fk_setup_experience_status_results_ses_id` FOREIGN KEY (`setup_experience_script_id`) REFERENCES `setup_experience_scripts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_setup_experience_status_results_si_id` FOREIGN KEY (`software_installer_id`) REFERENCES `software_installers` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_setup_experience_status_results_va_id` FOREIGN KEY (`vpp_app_team_id`) REFERENCES `vpp_apps_teams` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `software` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
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
  `title_id` int unsigned DEFAULT NULL,
  `checksum` binary(16) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_software_checksum` (`checksum`),
  KEY `software_source_vendor_idx` (`source`,`vendor_old`),
  KEY `title_id` (`title_id`),
  KEY `idx_sw_name_source_browser` (`name`,`source`,`browser`),
  KEY `software_listing_idx` (`name`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `software_cpe` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `software_id` bigint unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `cpe` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unq_software_id` (`software_id`),
  KEY `software_cpe_cpe_idx` (`cpe`),
  CONSTRAINT `software_cpe_ibfk_1` FOREIGN KEY (`software_id`) REFERENCES `software` (`id`) ON DELETE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `software_cve` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `cve` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `source` int DEFAULT '0',
  `software_id` bigint unsigned DEFAULT NULL,
  `resolved_in_version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unq_software_id_cve` (`software_id`,`cve`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `software_host_counts` (
  `software_id` bigint unsigned NOT NULL,
  `hosts_count` int unsigned NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `team_id` int unsigned NOT NULL DEFAULT '0',
  `global_stats` tinyint unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`software_id`,`team_id`,`global_stats`),
  KEY `idx_software_host_counts_updated_at_software_id` (`updated_at`,`software_id`),
  KEY `idx_software_host_counts_team_id_hosts_count_software_id` (`team_id`,`hosts_count`,`software_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `software_installer_labels` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `software_installer_id` int unsigned NOT NULL,
  `label_id` int unsigned NOT NULL,
  `exclude` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_software_installer_labels_software_installer_id_label_id` (`software_installer_id`,`label_id`),
  KEY `label_id` (`label_id`),
  CONSTRAINT `software_installer_labels_ibfk_1` FOREIGN KEY (`software_installer_id`) REFERENCES `software_installers` (`id`) ON DELETE CASCADE,
  CONSTRAINT `software_installer_labels_ibfk_2` FOREIGN KEY (`label_id`) REFERENCES `labels` (`id`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `software_installers` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `team_id` int unsigned DEFAULT NULL,
  `global_or_team_id` int unsigned NOT NULL DEFAULT '0',
  `title_id` int unsigned DEFAULT NULL,
  `filename` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `platform` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `pre_install_query` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `install_script_content_id` int unsigned NOT NULL,
  `post_install_script_content_id` int unsigned DEFAULT NULL,
  `storage_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `uploaded_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `self_service` tinyint(1) NOT NULL DEFAULT '0',
  `user_id` int unsigned DEFAULT NULL,
  `user_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `user_email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `url` varchar(4095) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `package_ids` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `extension` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `uninstall_script_content_id` int unsigned NOT NULL,
  `updated_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `fleet_library_app_id` int unsigned DEFAULT NULL,
  `install_during_setup` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_software_installers_team_id_title_id` (`global_or_team_id`,`title_id`),
  KEY `fk_software_installers_title` (`title_id`),
  KEY `fk_software_installers_install_script_content_id` (`install_script_content_id`),
  KEY `fk_software_installers_post_install_script_content_id` (`post_install_script_content_id`),
  KEY `fk_software_installers_team_id` (`team_id`),
  KEY `idx_software_installers_platform_title_id` (`platform`,`title_id`),
  KEY `fk_software_installers_user_id` (`user_id`),
  KEY `fk_uninstall_script_content_id` (`uninstall_script_content_id`),
  KEY `fk_software_installers_fleet_library_app_id` (`fleet_library_app_id`),
  CONSTRAINT `fk_software_installers_install_script_content_id` FOREIGN KEY (`install_script_content_id`) REFERENCES `script_contents` (`id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_software_installers_post_install_script_content_id` FOREIGN KEY (`post_install_script_content_id`) REFERENCES `script_contents` (`id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_software_installers_team_id` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_software_installers_title` FOREIGN KEY (`title_id`) REFERENCES `software_titles` (`id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_software_installers_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_uninstall_script_content_id` FOREIGN KEY (`uninstall_script_content_id`) REFERENCES `script_contents` (`id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `software_installers_ibfk_1` FOREIGN KEY (`fleet_library_app_id`) REFERENCES `fleet_library_apps` (`id`) ON DELETE SET NULL
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `software_titles` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `browser` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `bundle_identifier` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `additional_identifier` tinyint unsigned GENERATED ALWAYS AS ((case when (`source` = _utf8mb4'ios_apps') then 1 when (`source` = _utf8mb4'ipados_apps') then 2 when (`bundle_identifier` is not null) then 0 else NULL end)) VIRTUAL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sw_titles` (`name`,`source`,`browser`),
  UNIQUE KEY `idx_software_titles_bundle_identifier` (`bundle_identifier`,`additional_identifier`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `software_titles_host_counts` (
  `software_title_id` int unsigned NOT NULL,
  `hosts_count` int unsigned NOT NULL,
  `team_id` int unsigned NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `global_stats` tinyint unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`software_title_id`,`team_id`,`global_stats`),
  KEY `idx_software_titles_host_counts_team_counts_title` (`team_id`,`hosts_count`,`software_title_id`),
  KEY `idx_software_titles_host_counts_updated_at_software_title_id` (`updated_at`,`software_title_id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `statistics` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `anonymous_identifier` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `teams` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` varchar(1023) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `config` json DEFAULT NULL,
  `name_bin` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin GENERATED ALWAYS AS (`name`) VIRTUAL,
  `filename` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_teams_filename` (`filename`),
  UNIQUE KEY `idx_name_bin` (`name_bin`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_teams` (
  `user_id` int unsigned NOT NULL,
  `team_id` int unsigned NOT NULL,
  `role` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`user_id`,`team_id`),
  KEY `fk_user_teams_team_id` (`team_id`),
  CONSTRAINT `user_teams_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `user_teams_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
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
  `global_role` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `api_only` tinyint(1) NOT NULL DEFAULT '0',
  `mfa_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `settings` json NOT NULL DEFAULT (json_object()),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_unique_email` (`email`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `verification_tokens` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int unsigned NOT NULL,
  `token` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `token` (`token`),
  KEY `verification_tokens_users` (`user_id`),
  CONSTRAINT `verification_tokens_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `vpp_app_team_labels` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `vpp_app_team_id` int unsigned NOT NULL,
  `label_id` int unsigned NOT NULL,
  `exclude` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_vpp_app_team_labels_vpp_app_team_id_label_id` (`vpp_app_team_id`,`label_id`),
  KEY `label_id` (`label_id`),
  CONSTRAINT `vpp_app_team_labels_ibfk_1` FOREIGN KEY (`vpp_app_team_id`) REFERENCES `vpp_apps_teams` (`id`) ON DELETE CASCADE,
  CONSTRAINT `vpp_app_team_labels_ibfk_2` FOREIGN KEY (`label_id`) REFERENCES `labels` (`id`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `vpp_apps` (
  `adam_id` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `title_id` int unsigned DEFAULT NULL,
  `bundle_identifier` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `icon_url` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `latest_version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `platform` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`adam_id`,`platform`),
  KEY `fk_vpp_apps_title` (`title_id`),
  CONSTRAINT `fk_vpp_apps_title` FOREIGN KEY (`title_id`) REFERENCES `software_titles` (`id`) ON DELETE SET NULL ON UPDATE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `vpp_apps_teams` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `adam_id` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `team_id` int unsigned DEFAULT NULL,
  `global_or_team_id` int NOT NULL DEFAULT '0',
  `platform` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `self_service` tinyint(1) NOT NULL DEFAULT '0',
  `vpp_token_id` int unsigned NOT NULL,
  `install_during_setup` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_global_or_team_id_adam_id` (`global_or_team_id`,`adam_id`,`platform`),
  KEY `team_id` (`team_id`),
  KEY `adam_id` (`adam_id`,`platform`),
  KEY `fk_vpp_apps_teams_vpp_token_id` (`vpp_token_id`),
  CONSTRAINT `fk_vpp_apps_teams_vpp_token_id` FOREIGN KEY (`vpp_token_id`) REFERENCES `vpp_tokens` (`id`) ON DELETE CASCADE,
  CONSTRAINT `vpp_apps_teams_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE,
  CONSTRAINT `vpp_apps_teams_ibfk_3` FOREIGN KEY (`adam_id`, `platform`) REFERENCES `vpp_apps` (`adam_id`, `platform`) ON DELETE CASCADE
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `vpp_token_teams` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `vpp_token_id` int unsigned NOT NULL,
  `team_id` int unsigned DEFAULT NULL,
  `null_team_type` enum('none','allteams','noteam') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT 'none',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_vpp_token_teams_team_id` (`team_id`),
  KEY `fk_vpp_token_teams_vpp_token_id` (`vpp_token_id`),
  CONSTRAINT `fk_vpp_token_teams_team_id` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_vpp_token_teams_vpp_token_id` FOREIGN KEY (`vpp_token_id`) REFERENCES `vpp_tokens` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `vpp_tokens` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `organization_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `location` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `renew_at` timestamp NOT NULL,
  `token` blob NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_vpp_tokens_location` (`location`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `vulnerability_host_counts` (
  `cve` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `team_id` int unsigned NOT NULL DEFAULT '0',
  `host_count` int unsigned NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `global_stats` tinyint(1) NOT NULL DEFAULT '0',
  UNIQUE KEY `cve_team_id_global_stats` (`cve`,`team_id`,`global_stats`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `windows_mdm_command_queue` (
  `enrollment_id` int unsigned NOT NULL,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`enrollment_id`,`command_uuid`),
  KEY `command_uuid` (`command_uuid`),
  CONSTRAINT `windows_mdm_command_queue_ibfk_1` FOREIGN KEY (`enrollment_id`) REFERENCES `mdm_windows_enrollments` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `windows_mdm_command_queue_ibfk_2` FOREIGN KEY (`command_uuid`) REFERENCES `windows_mdm_commands` (`command_uuid`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `windows_mdm_command_results` (
  `enrollment_id` int unsigned NOT NULL,
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `raw_result` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `response_id` int unsigned NOT NULL,
  `status_code` varchar(31) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`enrollment_id`,`command_uuid`),
  KEY `command_uuid` (`command_uuid`),
  KEY `response_id` (`response_id`),
  CONSTRAINT `windows_mdm_command_results_ibfk_1` FOREIGN KEY (`enrollment_id`) REFERENCES `mdm_windows_enrollments` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `windows_mdm_command_results_ibfk_2` FOREIGN KEY (`command_uuid`) REFERENCES `windows_mdm_commands` (`command_uuid`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `windows_mdm_command_results_ibfk_3` FOREIGN KEY (`response_id`) REFERENCES `windows_mdm_responses` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `windows_mdm_commands` (
  `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  `raw_command` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_loc_uri` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`command_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `windows_mdm_responses` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `enrollment_id` int unsigned NOT NULL,
  `raw_response` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `enrollment_id` (`enrollment_id`),
  CONSTRAINT `windows_mdm_responses_ibfk_1` FOREIGN KEY (`enrollment_id`) REFERENCES `mdm_windows_enrollments` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `windows_updates` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int unsigned NOT NULL,
  `date_epoch` int unsigned NOT NULL,
  `kb_id` int unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_windows_updates` (`host_id`,`kb_id`),
  KEY `idx_update_date` (`host_id`,`date_epoch`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `wstep_cert_auth_associations` (
  `id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sha256` char(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`,`sha256`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `wstep_certificates` (
  `serial` bigint unsigned NOT NULL,
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
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `wstep_serials` (
  `serial` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`serial`)
) /*!50100 TABLESPACE `innodb_system` */ ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `yara_rules` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `contents` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_yara_rules_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
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
