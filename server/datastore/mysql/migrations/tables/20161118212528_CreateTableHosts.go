package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212528, Down_20161118212528)
}

func Up_20161118212528(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE `hosts` (" +
			"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
			"`osquery_host_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL," +
			"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
			"`updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
			"`deleted_at` timestamp NULL DEFAULT NULL," +
			"`deleted` tinyint(1) NOT NULL DEFAULT FALSE," +
			"`detail_update_time` timestamp NULL DEFAULT NULL," +
			"`node_key` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL," +
			"`host_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`uuid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`platform` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`osquery_version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`os_version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`build` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`platform_like` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`code_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`uptime` bigint(20) NOT NULL DEFAULT 0," +
			"`physical_memory` bigint(20) NOT NULL DEFAULT 0," +
			"`cpu_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`cpu_subtype` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`cpu_brand` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`cpu_physical_cores` int NOT NULL DEFAULT 0," +
			"`cpu_logical_cores` int NOT NULL DEFAULT 0," +
			"`hardware_vendor` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`hardware_model` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`hardware_version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`hardware_serial` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`computer_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"`primary_ip_id` INT(10) UNSIGNED DEFAULT NULL, " +
			"PRIMARY KEY (`id`)," +
			"UNIQUE KEY `idx_host_unique_nodekey` (`node_key`)," +
			"UNIQUE KEY `idx_osquery_host_id` (`osquery_host_id`)," +
			"FULLTEXT KEY `hosts_search` (`host_name`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
	)
	return err
}

func Down_20161118212528(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `hosts`;")
	return err
}
