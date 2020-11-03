-- MySQL dump 10.13  Distrib 5.7.16, for Win64 (x86_64)
--
-- Host: 192.168.99.100    Database: kolide
-- ------------------------------------------------------
-- Server version	5.7.16

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `app_configs`
--

DROP TABLE IF EXISTS `app_configs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `app_configs` (
  `id` int(10) unsigned NOT NULL DEFAULT '1',
  `org_name` varchar(255) NOT NULL DEFAULT '',
  `org_logo_url` varchar(255) NOT NULL DEFAULT '',
  `kolide_server_url` varchar(255) NOT NULL DEFAULT '',
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
  `osquery_enroll_secret` varchar(255) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_configs`
--

LOCK TABLES `app_configs` WRITE;
/*!40000 ALTER TABLE `app_configs` DISABLE KEYS */;
INSERT INTO `app_configs` VALUES (1,'Kolide','https://www.kolide.co/assets/kolide-nav-logo.svg','https://demo.kolide.kolide.net',0,'','',587,0,1,0,'','','',1,1,'');
/*!40000 ALTER TABLE `app_configs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `decorators`
--

DROP TABLE IF EXISTS `decorators`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `decorators` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `query` text NOT NULL,
  `type` int(10) unsigned NOT NULL,
  `interval` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `decorators`
--

LOCK TABLES `decorators` WRITE;
/*!40000 ALTER TABLE `decorators` DISABLE KEYS */;
/*!40000 ALTER TABLE `decorators` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `distributed_query_campaign_targets`
--

DROP TABLE IF EXISTS `distributed_query_campaign_targets`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `distributed_query_campaign_targets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `type` int(11) DEFAULT NULL,
  `distributed_query_campaign_id` int(10) unsigned DEFAULT NULL,
  `target_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `distributed_query_campaign_targets`
--

LOCK TABLES `distributed_query_campaign_targets` WRITE;
/*!40000 ALTER TABLE `distributed_query_campaign_targets` DISABLE KEYS */;
/*!40000 ALTER TABLE `distributed_query_campaign_targets` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `distributed_query_campaigns`
--

DROP TABLE IF EXISTS `distributed_query_campaigns`;
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `distributed_query_campaigns`
--

LOCK TABLES `distributed_query_campaigns` WRITE;
/*!40000 ALTER TABLE `distributed_query_campaigns` DISABLE KEYS */;
/*!40000 ALTER TABLE `distributed_query_campaigns` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `distributed_query_executions`
--

DROP TABLE IF EXISTS `distributed_query_executions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `distributed_query_executions` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `host_id` int(10) unsigned DEFAULT NULL,
  `distributed_query_campaign_id` int(10) unsigned DEFAULT NULL,
  `status` int(11) DEFAULT NULL,
  `error` varchar(1024) DEFAULT NULL,
  `execution_duration` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_dqe_unique_host_dqc_id` (`host_id`,`distributed_query_campaign_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `distributed_query_executions`
--

LOCK TABLES `distributed_query_executions` WRITE;
/*!40000 ALTER TABLE `distributed_query_executions` DISABLE KEYS */;
/*!40000 ALTER TABLE `distributed_query_executions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `file_integrity_monitoring_files`
--

DROP TABLE IF EXISTS `file_integrity_monitoring_files`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `file_integrity_monitoring_files` (
  `id` int(10) NOT NULL AUTO_INCREMENT,
  `file` varchar(255) NOT NULL DEFAULT '',
  `file_integrity_monitoring_id` int(10) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_fim_unique_file_name` (`file`) USING BTREE,
  KEY `fk_file_integrity_monitoring` (`file_integrity_monitoring_id`),
  CONSTRAINT `fk_file_integrity_monitoring` FOREIGN KEY (`file_integrity_monitoring_id`) REFERENCES `file_integrity_monitorings` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `file_integrity_monitoring_files`
--

LOCK TABLES `file_integrity_monitoring_files` WRITE;
/*!40000 ALTER TABLE `file_integrity_monitoring_files` DISABLE KEYS */;
/*!40000 ALTER TABLE `file_integrity_monitoring_files` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `file_integrity_monitorings`
--

DROP TABLE IF EXISTS `file_integrity_monitorings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `file_integrity_monitorings` (
  `id` int(10) NOT NULL AUTO_INCREMENT,
  `section_name` varchar(255) NOT NULL DEFAULT '',
  `description` varchar(255) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_section_name` (`section_name`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `file_integrity_monitorings`
--

LOCK TABLES `file_integrity_monitorings` WRITE;
/*!40000 ALTER TABLE `file_integrity_monitorings` DISABLE KEYS */;
/*!40000 ALTER TABLE `file_integrity_monitorings` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `hosts`
--

DROP TABLE IF EXISTS `hosts`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `hosts` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `osquery_host_id` varchar(255) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `detail_update_time` timestamp NULL DEFAULT NULL,
  `node_key` varchar(255) DEFAULT NULL,
  `host_name` varchar(255) NOT NULL DEFAULT '',
  `uuid` varchar(255) NOT NULL DEFAULT '',
  `platform` varchar(255) NOT NULL DEFAULT '',
  `osquery_version` varchar(255) NOT NULL DEFAULT '',
  `os_version` varchar(255) NOT NULL DEFAULT '',
  `build` varchar(255) NOT NULL DEFAULT '',
  `platform_like` varchar(255) NOT NULL DEFAULT '',
  `code_name` varchar(255) NOT NULL DEFAULT '',
  `uptime` bigint(20) NOT NULL DEFAULT '0',
  `physical_memory` bigint(20) NOT NULL DEFAULT '0',
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
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_osquery_host_id` (`osquery_host_id`),
  UNIQUE KEY `idx_host_unique_nodekey` (`node_key`),
  FULLTEXT KEY `hosts_search` (`host_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `hosts`
--

LOCK TABLES `hosts` WRITE;
/*!40000 ALTER TABLE `hosts` DISABLE KEYS */;
/*!40000 ALTER TABLE `hosts` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `invites`
--

DROP TABLE IF EXISTS `invites`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `invites` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `invited_by` int(10) unsigned NOT NULL,
  `email` varchar(255) NOT NULL,
  `admin` tinyint(1) DEFAULT NULL,
  `name` varchar(255) DEFAULT NULL,
  `position` varchar(255) DEFAULT NULL,
  `token` varchar(255) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_invite_unique_email` (`email`),
  UNIQUE KEY `idx_invite_unique_key` (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `invites`
--

LOCK TABLES `invites` WRITE;
/*!40000 ALTER TABLE `invites` DISABLE KEYS */;
/*!40000 ALTER TABLE `invites` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `label_query_executions`
--

DROP TABLE IF EXISTS `label_query_executions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `label_query_executions` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `matches` tinyint(1) NOT NULL DEFAULT '0',
  `label_id` int(10) unsigned DEFAULT NULL,
  `host_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_lqe_label_host` (`label_id`,`host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `label_query_executions`
--

LOCK TABLES `label_query_executions` WRITE;
/*!40000 ALTER TABLE `label_query_executions` DISABLE KEYS */;
/*!40000 ALTER TABLE `label_query_executions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `labels`
--

DROP TABLE IF EXISTS `labels`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `labels` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `name` varchar(255) NOT NULL,
  `description` varchar(255) DEFAULT NULL,
  `query` text NOT NULL,
  `platform` varchar(255) DEFAULT NULL,
  `label_type` int(10) unsigned NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_label_unique_name` (`name`),
  FULLTEXT KEY `labels_search` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `labels`
--

LOCK TABLES `labels` WRITE;
/*!40000 ALTER TABLE `labels` DISABLE KEYS */;
INSERT INTO `labels` VALUES (1,'2017-01-18 21:41:16','2017-01-18 21:41:16',NULL,0,'All Hosts','','select 1;','',1);
INSERT INTO `labels` VALUES (2,'2017-01-18 21:41:16','2017-01-18 21:41:16',NULL,0,'Mac OS X','','select 1 from osquery_info where build_platform = \'darwin\';','darwin',1);
INSERT INTO `labels` VALUES (3,'2017-01-18 21:41:16','2017-01-18 21:41:16',NULL,0,'Ubuntu Linux','','select 1 from osquery_info where build_platform = \'ubuntu\';','ubuntu',1);
INSERT INTO `labels` VALUES (4,'2017-01-18 21:41:16','2017-01-18 21:41:16',NULL,0,'CentOS Linux','','select 1 from osquery_info where build_platform = \'centos\';','centos',1);
INSERT INTO `labels` VALUES (5,'2017-01-18 21:41:16','2017-01-18 21:41:16',NULL,0,'MS Windows','','select 1 from osquery_info where build_platform = \'windows\';','windows',1);
INSERT INTO `labels` VALUES (6,'2017-01-19 01:22:08','2017-01-19 01:23:40',NULL,0,'macOS - update needed','The macOS hosts which have not yet updated to macOS Sierra.','select * from os_version where version != \'10.12\';','',0);
INSERT INTO `labels` VALUES (7,'2017-01-19 01:25:13','2017-01-19 01:25:13',NULL,0,'Windows- update needed','Windows hosts which have not installed Windows 10.','select * from os_version where major != \'10\';','',0);
/*!40000 ALTER TABLE `labels` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `migration_status_data`
--

DROP TABLE IF EXISTS `migration_status_data`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `migration_status_data` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `version_id` bigint(20) NOT NULL,
  `is_applied` tinyint(1) NOT NULL,
  `tstamp` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `migration_status_data`
--

LOCK TABLES `migration_status_data` WRITE;
/*!40000 ALTER TABLE `migration_status_data` DISABLE KEYS */;
INSERT INTO `migration_status_data` VALUES (1,0,1,'2017-01-18 21:41:16');
INSERT INTO `migration_status_data` VALUES (2,20161223115449,1,'2017-01-18 21:41:16');
INSERT INTO `migration_status_data` VALUES (3,20161229171615,1,'2017-01-18 21:41:16');
/*!40000 ALTER TABLE `migration_status_data` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `migration_status_tables`
--

DROP TABLE IF EXISTS `migration_status_tables`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `migration_status_tables` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `version_id` bigint(20) NOT NULL,
  `is_applied` tinyint(1) NOT NULL,
  `tstamp` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=30 DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `migration_status_tables`
--

LOCK TABLES `migration_status_tables` WRITE;
/*!40000 ALTER TABLE `migration_status_tables` DISABLE KEYS */;
INSERT INTO `migration_status_tables` VALUES (1,0,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (2,20161118193812,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (3,20161118211713,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (4,20161118212436,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (5,20161118212515,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (6,20161118212528,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (7,20161118212538,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (8,20161118212549,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (9,20161118212557,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (10,20161118212604,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (11,20161118212613,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (12,20161118212621,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (13,20161118212630,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (14,20161118212641,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (15,20161118212649,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (16,20161118212656,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (17,20161118212758,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (18,20161128234849,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (19,20161230162221,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (20,20170104113816,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (21,20170105151732,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (22,20170108191242,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (23,20170109094020,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (24,20170109130438,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (25,20170110202752,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (26,20170111133013,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (27,20170117025759,1,'2017-01-20 08:04:28');
INSERT INTO `migration_status_tables` VALUES (28,20170118191001,1,'2017-01-23 17:11:38');
INSERT INTO `migration_status_tables` VALUES (29,20170119234632,1,'2017-01-23 17:11:38');
/*!40000 ALTER TABLE `migration_status_tables` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `network_interfaces`
--

DROP TABLE IF EXISTS `network_interfaces`;
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
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_network_interfaces_unique_ip_host_intf` (`ip_address`,`host_id`,`interface`),
  KEY `idx_network_interfaces_hosts_fk` (`host_id`),
  FULLTEXT KEY `ip_address_search` (`ip_address`),
  CONSTRAINT `network_interfaces_ibfk_1` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `network_interfaces`
--

LOCK TABLES `network_interfaces` WRITE;
/*!40000 ALTER TABLE `network_interfaces` DISABLE KEYS */;
/*!40000 ALTER TABLE `network_interfaces` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `options`
--

DROP TABLE IF EXISTS `options`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `options` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `type` int(10) unsigned NOT NULL,
  `value` varchar(255) NOT NULL,
  `read_only` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_option_unique_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=57 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `options`
--

LOCK TABLES `options` WRITE;
/*!40000 ALTER TABLE `options` DISABLE KEYS */;
INSERT INTO `options` VALUES (1,'disable_distributed',2,'false',1);
INSERT INTO `options` VALUES (2,'distributed_plugin',0,'\"tls\"',1);
INSERT INTO `options` VALUES (3,'distributed_tls_read_endpoint',0,'\"/api/v1/osquery/distributed/read\"',1);
INSERT INTO `options` VALUES (4,'distributed_tls_write_endpoint',0,'\"/api/v1/osquery/distributed/write\"',1);
INSERT INTO `options` VALUES (5,'pack_delimiter',0,'\"/\"',1);
INSERT INTO `options` VALUES (6,'aws_access_key_id',0,'null',0);
INSERT INTO `options` VALUES (7,'aws_firehose_period',1,'null',0);
INSERT INTO `options` VALUES (8,'aws_firehose_stream',0,'null',0);
INSERT INTO `options` VALUES (9,'aws_kinesis_period',1,'null',0);
INSERT INTO `options` VALUES (10,'aws_kinesis_random_partition_key',2,'null',0);
INSERT INTO `options` VALUES (11,'aws_kinesis_stream',0,'null',0);
INSERT INTO `options` VALUES (12,'aws_profile_name',0,'null',0);
INSERT INTO `options` VALUES (13,'aws_region',0,'null',0);
INSERT INTO `options` VALUES (14,'aws_secret_access_key',0,'null',0);
INSERT INTO `options` VALUES (15,'aws_sts_arn_role',0,'null',0);
INSERT INTO `options` VALUES (16,'aws_sts_region',0,'null',0);
INSERT INTO `options` VALUES (17,'aws_sts_session_name',0,'null',0);
INSERT INTO `options` VALUES (18,'aws_sts_timeout',1,'null',0);
INSERT INTO `options` VALUES (19,'buffered_log_max',1,'null',0);
INSERT INTO `options` VALUES (20,'decorations_top_level',2,'null',0);
INSERT INTO `options` VALUES (21,'disable_caching',2,'null',0);
INSERT INTO `options` VALUES (22,'disable_database',2,'null',0);
INSERT INTO `options` VALUES (23,'disable_decorators',2,'null',0);
INSERT INTO `options` VALUES (24,'disable_events',2,'null',0);
INSERT INTO `options` VALUES (25,'disable_kernel',2,'null',0);
INSERT INTO `options` VALUES (26,'disable_logging',2,'null',0);
INSERT INTO `options` VALUES (27,'disable_tables',0,'null',0);
INSERT INTO `options` VALUES (28,'distributed_interval',1,'10',0);
INSERT INTO `options` VALUES (29,'distributed_tls_max_attempts',1,'3',0);
INSERT INTO `options` VALUES (30,'enable_foreign',2,'null',0);
INSERT INTO `options` VALUES (31,'enable_monitor',2,'null',0);
INSERT INTO `options` VALUES (32,'ephemeral',2,'null',0);
INSERT INTO `options` VALUES (33,'events_expiry',1,'null',0);
INSERT INTO `options` VALUES (34,'events_max',1,'null',0);
INSERT INTO `options` VALUES (35,'events_optimize',2,'null',0);
INSERT INTO `options` VALUES (36,'host_identifier',0,'null',0);
INSERT INTO `options` VALUES (37,'logger_event_type',2,'null',0);
INSERT INTO `options` VALUES (38,'logger_mode',0,'null',0);
INSERT INTO `options` VALUES (39,'logger_path',0,'null',0);
INSERT INTO `options` VALUES (40,'logger_plugin',0,'\"tls\"',0);
INSERT INTO `options` VALUES (41,'logger_secondary_status_only',2,'null',0);
INSERT INTO `options` VALUES (42,'logger_syslog_facility',1,'null',0);
INSERT INTO `options` VALUES (43,'logger_tls_compress',2,'null',0);
INSERT INTO `options` VALUES (44,'logger_tls_endpoint',0,'\"/api/v1/osquery/log\"',0);
INSERT INTO `options` VALUES (45,'logger_tls_max',1,'null',0);
INSERT INTO `options` VALUES (46,'logger_tls_period',1,'10',0);
INSERT INTO `options` VALUES (47,'pack_refresh_interval',1,'null',0);
INSERT INTO `options` VALUES (48,'read_max',1,'null',0);
INSERT INTO `options` VALUES (49,'read_user_max',1,'null',0);
INSERT INTO `options` VALUES (50,'schedule_default_interval',1,'null',0);
INSERT INTO `options` VALUES (51,'schedule_splay_percent',1,'null',0);
INSERT INTO `options` VALUES (52,'schedule_timeout',1,'null',0);
INSERT INTO `options` VALUES (53,'utc',2,'null',0);
INSERT INTO `options` VALUES (54,'value_max',1,'null',0);
INSERT INTO `options` VALUES (55,'verbose',2,'null',0);
INSERT INTO `options` VALUES (56,'worker_threads',1,'null',0);
/*!40000 ALTER TABLE `options` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `pack_targets`
--

DROP TABLE IF EXISTS `pack_targets`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `pack_targets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `pack_id` int(10) unsigned DEFAULT NULL,
  `type` int(11) DEFAULT NULL,
  `target_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `constraint_pack_target_unique` (`pack_id`,`target_id`,`type`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pack_targets`
--

LOCK TABLES `pack_targets` WRITE;
/*!40000 ALTER TABLE `pack_targets` DISABLE KEYS */;
INSERT INTO `pack_targets` VALUES (1,1,0,1);
INSERT INTO `pack_targets` VALUES (2,2,0,1);
INSERT INTO `pack_targets` VALUES (3,3,0,1);
INSERT INTO `pack_targets` VALUES (4,4,0,1);
INSERT INTO `pack_targets` VALUES (5,5,0,1);
INSERT INTO `pack_targets` VALUES (6,6,0,1);
INSERT INTO `pack_targets` VALUES (7,7,0,1);
INSERT INTO `pack_targets` VALUES (8,8,0,3);
INSERT INTO `pack_targets` VALUES (9,8,0,4);
/*!40000 ALTER TABLE `pack_targets` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `packs`
--

DROP TABLE IF EXISTS `packs`;
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
  `created_by` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_pack_unique_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `packs`
--

LOCK TABLES `packs` WRITE;
/*!40000 ALTER TABLE `packs` DISABLE KEYS */;
INSERT INTO `packs` VALUES (1,'2017-01-19 01:07:01','2017-01-19 01:09:03',NULL,0,0,'Intrusion Detection','A collection of queries that detect indicators of initial compromise via various tactics, techniques, and procedures.','',1);
INSERT INTO `packs` VALUES (2,'2017-01-19 01:08:08','2017-01-19 01:08:08',NULL,0,0,'Osquery Monitoring','Osquery exposes several tables which allow you to query the internal operations of the osqueryd process itself. This pack contains queries that allow us to maintain insight into the health and performance of the osquery fleet.','',1);
INSERT INTO `packs` VALUES (3,'2017-01-19 01:10:38','2017-01-19 01:10:38',NULL,0,0,'Asset Management','A collection of queries that tracks the company\'s assets, installed applications, etc.','',1);
INSERT INTO `packs` VALUES (4,'2017-01-19 01:12:28','2017-01-19 01:12:28',NULL,0,0,'Hardware Monitoring','A collection of queries which monitor the changes that occur in the lower-level, hardware configurations of assets. ','',1);
INSERT INTO `packs` VALUES (5,'2017-01-19 01:13:51','2017-01-19 01:13:51',NULL,0,0,'Incident Response','While responding to an incident, it\'s often useful to have a collection of certain historical data to be able to piece together the incident timeline. This pack is a collection of queries which are useful to have during the incident response process.','',1);
INSERT INTO `packs` VALUES (6,'2017-01-19 01:14:56','2017-01-19 01:14:56',NULL,0,0,'Compliance','In order to maintain compliance, we have to ensure that we are tracking certain events and operations that occur throughout our fleet.','',1);
INSERT INTO `packs` VALUES (7,'2017-01-19 01:16:51','2017-01-19 01:16:51',NULL,0,0,'Vulnerability Management','In order to ensure that our assets are not running vulnerable versions of key software, we deploy queries within this pack to track important application values and versions.','',1);
INSERT INTO `packs` VALUES (8,'2017-01-20 08:16:52','2017-01-20 08:16:52',NULL,0,0,'Systems Monitoring','Queries which track the health, stability, and performance of a system from an operations perspective.','',1);
/*!40000 ALTER TABLE `packs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `password_reset_requests`
--

DROP TABLE IF EXISTS `password_reset_requests`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `password_reset_requests` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `expires_at` timestamp NOT NULL DEFAULT '1970-01-01 00:00:01',
  `user_id` int(10) unsigned NOT NULL,
  `token` varchar(1024) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `password_reset_requests`
--

LOCK TABLES `password_reset_requests` WRITE;
/*!40000 ALTER TABLE `password_reset_requests` DISABLE KEYS */;
/*!40000 ALTER TABLE `password_reset_requests` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `queries`
--

DROP TABLE IF EXISTS `queries`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `queries` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `saved` tinyint(1) NOT NULL DEFAULT '0',
  `name` varchar(255) NOT NULL,
  `description` text NOT NULL,
  `query` text NOT NULL,
  `author_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_query_unique_name` (`name`),
  UNIQUE KEY `constraint_query_name_unique` (`name`),
  KEY `author_id` (`author_id`),
  CONSTRAINT `queries_ibfk_1` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB AUTO_INCREMENT=35 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `queries`
--

LOCK TABLES `queries` WRITE;
/*!40000 ALTER TABLE `queries` DISABLE KEYS */;
INSERT INTO `queries` VALUES (1,'2017-01-20 00:53:10','2017-01-20 00:53:10',NULL,0,1,'Osquery Events','Information about osquery\'s event publishers and subscribers, which are the implementation components of event-based tables.','select * from osquery_events;',1);
INSERT INTO `queries` VALUES (2,'2017-01-20 00:53:46','2017-01-20 00:53:46',NULL,0,1,'Osquery Extensions','A list of active osquery extensions.','select * from osquery_extensions;',1);
INSERT INTO `queries` VALUES (3,'2017-01-20 00:54:27','2017-01-20 00:54:27',NULL,0,1,'Osquery Flags','The values of configurable flags which modify osquery\'s behavior. ','select * from osquery_flags;',1);
INSERT INTO `queries` VALUES (4,'2017-01-20 00:55:04','2017-01-20 00:55:04',NULL,0,1,'Osquery General Info','Top-level information about the running osquery instance.','select * from osquery_info;',1);
INSERT INTO `queries` VALUES (5,'2017-01-20 00:55:43','2017-01-20 00:55:43',NULL,0,1,'Osquery Packs','Information about the current query packs that are loaded in osquery.','select * from osquery_packs;',1);
INSERT INTO `queries` VALUES (6,'2017-01-20 00:56:18','2017-01-20 00:56:18',NULL,0,1,'Osquery Registry','Information about the active items/plugins in the osquery application registry.','select * from osquery_registry;',1);
INSERT INTO `queries` VALUES (7,'2017-01-20 00:56:42','2017-01-20 00:56:42',NULL,0,1,'Osquery Schedule','Information about the current queries that are scheduled in osquery.','select * from osquery_schedule;',1);
INSERT INTO `queries` VALUES (8,'2017-01-20 00:59:50','2017-01-20 00:59:50',NULL,0,1,'Hosts File','A line-parsed readout of the /etc/hosts file.','select * from etc_hosts;',1);
INSERT INTO `queries` VALUES (9,'2017-01-20 01:00:12','2017-01-20 01:00:12',NULL,0,1,'Protocols File','A line-parsed readout of the /etc/protocols file.','select * from etc_protocols;',1);
INSERT INTO `queries` VALUES (10,'2017-01-20 01:00:30','2017-01-20 01:00:30',NULL,0,1,'Services File','A line-parsed readout of the /etc/services file.','select * from etc_services;',1);
INSERT INTO `queries` VALUES (11,'2017-01-20 01:01:14','2017-01-20 01:01:14',NULL,0,1,'OS Version Info','Information about the operating system name and version.','select * from os_version;',1);
INSERT INTO `queries` VALUES (12,'2017-01-20 01:01:50','2017-01-20 01:01:50',NULL,0,1,'System Info','Interesting system information about a host.','select * from system_info;',1);
INSERT INTO `queries` VALUES (13,'2017-01-20 01:03:42','2017-01-20 01:03:42',NULL,0,1,'Users','Information about the users on a system and their groups.','SELECT * FROM users u JOIN groups g WHERE u.gid = g.gid;',1);
INSERT INTO `queries` VALUES (14,'2017-01-20 01:04:23','2017-01-20 01:04:23',NULL,0,1,'Windows Services','All installed Windows services and relevant data.','select * from services;',1);
INSERT INTO `queries` VALUES (15,'2017-01-20 01:04:59','2017-01-20 01:04:59',NULL,0,1,'Windows Registry','All of the Windows registry hives.','select * from registry;',1);
INSERT INTO `queries` VALUES (16,'2017-01-20 01:05:32','2017-01-20 01:05:32',NULL,0,1,'Windows Drivers','Lists all installed and loaded Windows Drivers and their relevant data.','select * from drivers;',1);
INSERT INTO `queries` VALUES (17,'2017-01-20 01:05:56','2017-01-20 01:05:56',NULL,0,1,'Windows Patches','Lists all the patches applied. Note: This does not include patches applied via MSI or downloaded from Windows Update (e.g. Service Packs).','select * from patches;',1);
INSERT INTO `queries` VALUES (18,'2017-01-20 01:12:11','2017-01-20 01:12:11',NULL,0,1,'Windows Application Compatibility Shims','Application Compatibility shims are a way to persist malware. This table presents information about the Application Compatibility Shims from the registry in a nice format.','select * from appcompat_shims;',1);
INSERT INTO `queries` VALUES (19,'2017-01-20 01:13:34','2017-01-22 20:23:15',NULL,0,1,'Kernel Info','Basic information about the active kernel.','select * from kernel_info join hash using (path);',1);
INSERT INTO `queries` VALUES (20,'2017-01-22 20:12:09','2017-01-22 20:12:09',NULL,0,1,'Mac Applications','The applications that are installed on a user\'s Apple laptop.','select * from apps;',1);
INSERT INTO `queries` VALUES (21,'2017-01-22 20:14:04','2017-01-22 20:14:04',NULL,0,1,'Chrome Extensions','The Google Chrome Extensions that a user has installed in their browser.','select * from chrome_extensions;',1);
INSERT INTO `queries` VALUES (22,'2017-01-22 20:16:16','2017-01-22 20:16:16',NULL,0,1,'ACPI Tables','Firmware ACPI functional table common metadata and content.','select * from acpi_tables;',1);
INSERT INTO `queries` VALUES (23,'2017-01-22 20:17:18','2017-01-22 20:17:18',NULL,0,1,'CPU features','Useful CPU features from the cpuid ASM call.','select feature, value, output_register, output_bit, input_eax from cpuid;',1);
INSERT INTO `queries` VALUES (24,'2017-01-22 20:20:34','2017-01-22 20:20:34',NULL,0,1,'SMBIOS Tables','BIOS (DMI) structure common details and content.','select * from smbios_tables;',1);
INSERT INTO `queries` VALUES (25,'2017-01-22 20:22:18','2017-01-22 20:22:18',NULL,0,1,'NVRAM','NVRAM content.','select * from nvram where name not in (\'backlight-level\', \'SystemAudioVolumeDB\', \'SystemAudioVolume\');',1);
INSERT INTO `queries` VALUES (26,'2017-01-22 20:24:38','2017-01-22 20:24:38',NULL,0,1,'PCI Devices','An inventory of PCI devices.','select * from pci_devices;',1);
INSERT INTO `queries` VALUES (27,'2017-01-22 20:25:00','2017-01-22 20:25:00',NULL,0,1,'USB Devices','An inventory of USB devices.','select * from usb_devices;',1);
INSERT INTO `queries` VALUES (28,'2017-01-22 20:25:46','2017-01-22 20:25:46',NULL,0,1,'Hardware Events','Attaches and detaches of hardware inputs on a host.','select * from hardware_events;',1);
INSERT INTO `queries` VALUES (29,'2017-01-22 20:26:59','2017-01-22 20:26:59',NULL,0,1,'Kernel System Controls','Kernel system controls on macOS.','select * from system_controls where subsystem = \'kern\' and (name like \'%boot%\' or name like \'%secure%\' or name like \'%single%\');',1);
INSERT INTO `queries` VALUES (30,'2017-01-22 20:30:00','2017-01-22 20:30:00',NULL,0,1,'IOKit Device Tree','General inventory of IOKit\'s devices on macOS.','select * from iokit_devicetree;',1);
INSERT INTO `queries` VALUES (31,'2017-01-22 20:31:52','2017-01-22 20:31:52',NULL,0,1,'Kernel Extensions','A list of the kernel extensions on a macOS host.','select * from kernel_extensions;',1);
INSERT INTO `queries` VALUES (32,'2017-01-22 20:32:11','2017-01-22 20:32:11',NULL,0,1,'Kernel Modules','A list of the kernel modules on a Linux host.','select * from kernel_modules;',1);
INSERT INTO `queries` VALUES (33,'2017-01-22 20:34:29','2017-01-22 20:34:29',NULL,0,1,'Device Nodes','All devices nodes in /dev.','select file.path, uid, gid, mode, 0 as atime, mtime, ctime, block_size, mode, type from file where directory = \'/dev/\';',1);
INSERT INTO `queries` VALUES (34,'2017-01-22 20:42:19','2017-01-22 20:42:43',NULL,0,1,'Active Directory Configuration','Retrieves the Active Directory configuration for the target machine.','select * from ad_config;',1);
/*!40000 ALTER TABLE `queries` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `scheduled_queries`
--

DROP TABLE IF EXISTS `scheduled_queries`;
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
  `platform` varchar(255) DEFAULT NULL,
  `version` varchar(255) DEFAULT NULL,
  `shard` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=69 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `scheduled_queries`
--

LOCK TABLES `scheduled_queries` WRITE;
/*!40000 ALTER TABLE `scheduled_queries` DISABLE KEYS */;
INSERT INTO `scheduled_queries` VALUES (1,'2017-01-20 00:57:12','2017-01-20 00:57:12',NULL,0,2,1,3600,1,0,'','',NULL);
INSERT INTO `scheduled_queries` VALUES (2,'2017-01-20 00:57:27','2017-01-20 00:57:27',NULL,0,2,2,3600,1,0,'','',NULL);
INSERT INTO `scheduled_queries` VALUES (3,'2017-01-20 00:57:40','2017-01-20 00:57:40',NULL,0,2,3,3600,1,0,NULL,NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (4,'2017-01-20 00:57:48','2017-01-20 00:57:48',NULL,0,2,4,3600,1,0,NULL,NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (5,'2017-01-20 00:57:56','2017-01-20 00:57:56',NULL,0,2,5,3600,1,0,NULL,NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (6,'2017-01-20 00:58:05','2017-01-20 00:58:05',NULL,0,2,6,3600,1,0,NULL,NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (7,'2017-01-20 00:58:12','2017-01-20 00:58:12',NULL,0,2,7,3600,1,0,NULL,NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (8,'2017-01-22 20:03:09','2017-01-22 20:03:09',NULL,0,1,17,3600,0,1,'windows','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (9,'2017-01-22 20:04:14','2017-01-22 20:04:14',NULL,0,1,18,3600,0,1,'windows','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (10,'2017-01-22 20:04:55','2017-01-22 20:04:55',NULL,0,1,19,3600,0,1,'','',NULL);
INSERT INTO `scheduled_queries` VALUES (11,'2017-01-22 20:05:55','2017-01-22 20:05:55',NULL,0,1,16,1800,0,1,'windows','2.1.1',NULL);
INSERT INTO `scheduled_queries` VALUES (12,'2017-01-22 20:06:20','2017-01-22 20:06:20',NULL,0,1,15,1800,0,1,'windows','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (13,'2017-01-22 20:06:40','2017-01-22 20:06:40',NULL,0,1,14,3600,0,1,'windows','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (14,'2017-01-22 20:07:07','2017-01-22 20:07:07',NULL,0,1,13,1800,0,1,'','',NULL);
INSERT INTO `scheduled_queries` VALUES (15,'2017-01-22 20:07:32','2017-01-22 20:07:32',NULL,0,1,8,3600,0,1,'linux','',NULL);
INSERT INTO `scheduled_queries` VALUES (16,'2017-01-22 20:08:01','2017-01-22 20:08:01',NULL,0,1,9,3600,0,1,'linux','',NULL);
INSERT INTO `scheduled_queries` VALUES (17,'2017-01-22 20:08:19','2017-01-22 20:08:19',NULL,0,1,10,3600,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (18,'2017-01-22 20:10:33','2017-01-22 20:10:33',NULL,0,3,12,1800,1,0,'','',NULL);
INSERT INTO `scheduled_queries` VALUES (19,'2017-01-22 20:10:58','2017-01-22 20:10:58',NULL,0,3,13,1800,1,0,'','',NULL);
INSERT INTO `scheduled_queries` VALUES (20,'2017-01-22 20:12:31','2017-01-22 20:12:31',NULL,0,3,20,3600,0,1,'darwin','',NULL);
INSERT INTO `scheduled_queries` VALUES (21,'2017-01-22 20:14:30','2017-01-22 20:14:30',NULL,0,1,21,3600,0,1,'','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (22,'2017-01-22 20:15:03','2017-01-22 20:15:03',NULL,0,3,21,3600,0,1,'','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (23,'2017-01-22 20:19:18','2017-01-22 20:19:18',NULL,0,4,23,28800,0,1,'','',NULL);
INSERT INTO `scheduled_queries` VALUES (24,'2017-01-22 20:19:42','2017-01-22 20:19:42',NULL,0,4,22,28800,0,1,'linux','',NULL);
INSERT INTO `scheduled_queries` VALUES (25,'2017-01-22 20:21:09','2017-01-22 20:21:09',NULL,0,4,24,28800,0,1,'linux','',NULL);
INSERT INTO `scheduled_queries` VALUES (26,'2017-01-22 20:22:36','2017-01-22 20:22:36',NULL,0,4,25,3600,0,1,'darwin','',NULL);
INSERT INTO `scheduled_queries` VALUES (27,'2017-01-22 20:27:27','2017-01-22 20:27:27',NULL,0,4,29,28800,0,1,'darwin','',NULL);
INSERT INTO `scheduled_queries` VALUES (28,'2017-01-22 20:28:21','2017-01-22 20:28:21',NULL,0,4,28,300,0,1,'linux','',NULL);
INSERT INTO `scheduled_queries` VALUES (29,'2017-01-22 20:28:41','2017-01-22 20:28:41',NULL,0,4,26,3600,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (30,'2017-01-22 20:28:54','2017-01-22 20:28:54',NULL,0,4,27,3600,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (31,'2017-01-22 20:30:27','2017-01-22 20:30:27',NULL,0,4,30,28800,0,1,'darwin',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (32,'2017-01-22 20:32:30','2017-01-22 20:32:30',NULL,0,4,31,300,0,1,'darwin',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (33,'2017-01-22 20:32:45','2017-01-22 20:32:45',NULL,0,4,32,300,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (34,'2017-01-22 20:33:12','2017-01-22 20:33:12',NULL,0,1,31,300,0,1,'darwin',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (35,'2017-01-22 20:33:32','2017-01-22 20:33:32',NULL,0,1,32,300,0,1,'linux','',NULL);
INSERT INTO `scheduled_queries` VALUES (36,'2017-01-22 20:34:59','2017-01-22 20:34:59',NULL,0,4,33,3600,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (37,'2017-01-22 20:35:40','2017-01-22 20:35:40',NULL,0,7,10,3600,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (38,'2017-01-22 20:36:02','2017-01-22 20:36:02',NULL,0,7,11,300,0,1,'',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (39,'2017-01-22 20:36:35','2017-01-22 20:36:35',NULL,0,7,14,3600,0,1,'windows',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (40,'2017-01-22 20:36:58','2017-01-22 20:36:58',NULL,0,7,17,3600,1,0,'windows','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (41,'2017-01-22 20:37:23','2017-01-22 20:37:23',NULL,0,7,20,3600,1,0,'darwin',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (42,'2017-01-22 20:37:46','2017-01-22 20:37:46',NULL,0,7,21,3600,0,1,'','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (43,'2017-01-22 20:38:32','2017-01-22 20:38:32',NULL,0,6,11,3600,1,0,'',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (44,'2017-01-22 20:38:53','2017-01-22 20:38:53',NULL,0,6,31,300,0,1,'darwin',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (45,'2017-01-22 20:39:10','2017-01-22 20:39:10',NULL,0,6,32,300,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (46,'2017-01-22 20:39:51','2017-01-22 20:39:51',NULL,0,6,15,3600,0,1,'windows','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (47,'2017-01-22 20:40:08','2017-01-22 20:40:08',NULL,0,6,19,3600,1,0,'',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (48,'2017-01-22 20:40:31','2017-01-22 20:40:31',NULL,0,6,20,3600,0,1,'darwin',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (49,'2017-01-22 20:40:53','2017-01-22 20:40:53',NULL,0,6,14,3600,0,1,'windows','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (50,'2017-01-22 20:41:14','2017-01-22 20:41:14',NULL,0,6,21,3600,0,1,NULL,'2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (51,'2017-01-22 20:44:12','2017-01-22 20:44:12',NULL,0,5,16,86400,1,0,'windows','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (52,'2017-01-22 20:44:36','2017-01-22 20:44:36',NULL,0,5,16,60,0,1,'windows','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (53,'2017-01-22 20:45:13','2017-01-22 20:45:13',NULL,0,5,15,86400,1,0,'windows','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (54,'2017-01-22 20:45:30','2017-01-22 20:45:30',NULL,0,5,15,60,0,1,'windows','2.2.1',NULL);
INSERT INTO `scheduled_queries` VALUES (55,'2017-01-22 20:46:01','2017-01-22 20:46:01',NULL,0,5,32,86400,1,0,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (56,'2017-01-22 20:46:08','2017-01-22 20:46:08',NULL,0,5,32,60,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (57,'2017-01-22 20:46:28','2017-01-22 20:46:28',NULL,0,5,31,86400,1,0,'darwin',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (58,'2017-01-22 20:46:40','2017-01-22 20:46:40',NULL,0,5,31,60,0,1,'darwin',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (59,'2017-01-22 20:47:13','2017-01-22 20:47:13',NULL,0,8,4,3600,0,1,'',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (60,'2017-01-22 20:47:28','2017-01-22 20:47:28',NULL,0,8,8,300,0,1,'linux','',NULL);
INSERT INTO `scheduled_queries` VALUES (61,'2017-01-22 20:47:47','2017-01-22 20:47:47',NULL,0,8,9,300,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (62,'2017-01-22 20:48:09','2017-01-22 20:48:09',NULL,0,8,10,300,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (63,'2017-01-22 20:48:27','2017-01-22 20:48:27',NULL,0,8,11,300,0,1,'',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (64,'2017-01-22 20:48:42','2017-01-22 20:48:42',NULL,0,8,12,300,0,1,'',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (65,'2017-01-22 20:49:07','2017-01-22 20:49:07',NULL,0,8,19,3600,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (66,'2017-01-22 20:49:32','2017-01-22 20:49:32',NULL,0,8,23,3600,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (67,'2017-01-22 20:50:00','2017-01-22 20:50:00',NULL,0,8,28,60,0,1,'linux',NULL,NULL);
INSERT INTO `scheduled_queries` VALUES (68,'2017-01-22 20:50:19','2017-01-22 20:50:19',NULL,0,8,34,3600,0,1,'darwin',NULL,NULL);
/*!40000 ALTER TABLE `scheduled_queries` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sessions`
--

DROP TABLE IF EXISTS `sessions`;
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
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sessions`
--

LOCK TABLES `sessions` WRITE;
/*!40000 ALTER TABLE `sessions` DISABLE KEYS */;
INSERT INTO `sessions` VALUES (1,'2017-01-20 08:09:01','2017-01-22 20:51:16',1,'qRDbkVCGURs3Auh+3RN5SZF1umFouMQIU7LXT6mzLge04jMRT8Z+FcIfrKYyU28X7G5RkhJH3T9ee9Uby2TFQQ==');
/*!40000 ALTER TABLE `sessions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `username` varchar(255) NOT NULL,
  `password` varbinary(255) NOT NULL,
  `salt` varchar(255) NOT NULL,
  `name` varchar(255) NOT NULL DEFAULT '',
  `email` varchar(255) NOT NULL,
  `admin` tinyint(1) NOT NULL DEFAULT '0',
  `enabled` tinyint(1) NOT NULL DEFAULT '0',
  `admin_forced_password_reset` tinyint(1) NOT NULL DEFAULT '0',
  `gravatar_url` varchar(255) NOT NULL DEFAULT '',
  `position` varchar(255) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_unique_username` (`username`),
  UNIQUE KEY `idx_user_unique_email` (`email`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `users`
--

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
INSERT INTO `users` VALUES (1,'2017-01-18 21:43:48','2017-01-18 21:44:42',NULL,0,'administrator','$2a$12$KPbYHDTqvraN72M9csSRP.SENlgc5Q10zzMH2Wlr5JCEXHwv0P0AS','iqZ/6SSoXgWezAlM7HXJ7Vph96CsYDxb','John Doe','demo-admin@kolide.co',1,1,0,'','Security Engineer');
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yara_file_paths`
--

DROP TABLE IF EXISTS `yara_file_paths`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yara_file_paths` (
  `file_integrity_monitoring_id` int(11) NOT NULL DEFAULT '0',
  `yara_signature_id` int(11) NOT NULL DEFAULT '0',
  PRIMARY KEY (`file_integrity_monitoring_id`,`yara_signature_id`),
  KEY `fk_yara_signature_id` (`yara_signature_id`),
  CONSTRAINT `fk_file_integrity_monitoring_id` FOREIGN KEY (`file_integrity_monitoring_id`) REFERENCES `file_integrity_monitorings` (`id`),
  CONSTRAINT `fk_yara_signature_id` FOREIGN KEY (`yara_signature_id`) REFERENCES `yara_signatures` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yara_file_paths`
--

LOCK TABLES `yara_file_paths` WRITE;
/*!40000 ALTER TABLE `yara_file_paths` DISABLE KEYS */;
/*!40000 ALTER TABLE `yara_file_paths` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yara_signature_paths`
--

DROP TABLE IF EXISTS `yara_signature_paths`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yara_signature_paths` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `file_path` varchar(255) NOT NULL DEFAULT '',
  `yara_signature_id` int(11) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `fk_yara_signature` (`yara_signature_id`),
  CONSTRAINT `fk_yara_signature` FOREIGN KEY (`yara_signature_id`) REFERENCES `yara_signatures` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yara_signature_paths`
--

LOCK TABLES `yara_signature_paths` WRITE;
/*!40000 ALTER TABLE `yara_signature_paths` DISABLE KEYS */;
/*!40000 ALTER TABLE `yara_signature_paths` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `yara_signatures`
--

DROP TABLE IF EXISTS `yara_signatures`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `yara_signatures` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `signature_name` varchar(128) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_yara_signatures_unique_name` (`signature_name`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `yara_signatures`
--

LOCK TABLES `yara_signatures` WRITE;
/*!40000 ALTER TABLE `yara_signatures` DISABLE KEYS */;
/*!40000 ALTER TABLE `yara_signatures` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed
