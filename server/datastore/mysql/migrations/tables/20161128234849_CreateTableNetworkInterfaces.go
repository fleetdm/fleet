package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161128234849, Down_20161128234849)
}

func Up_20161128234849(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE `network_interfaces` (" +
			"`id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT," +
			"`host_id` INT(10) UNSIGNED NOT NULL," +
			"`mac` varchar(255) NOT NULL DEFAULT ''," +
			"`ip_address` varchar(255) NOT NULL DEFAULT ''," +
			"`broadcast` varchar(255) NOT NULL DEFAULT ''," +
			"`ibytes` BIGINT NOT NULL DEFAULT 0," +
			"`interface` VARCHAR(255) NOT NULL DEFAULT ''," +
			"`ipackets` BIGINT NOT NULL DEFAULT 0," +
			"`last_change` BIGINT NOT NULL DEFAULT 0," +
			"`mask` varchar(255) NOT NULL DEFAULT ''," +
			"`metric` INT NOT NULL DEFAULT 0," +
			"`mtu` INT NOT NULL DEFAULT 0," +
			"`obytes` BIGINT NOT NULL DEFAULT 0," +
			"`ierrors` BIGINT NOT NULL DEFAULT 0," +
			"`oerrors` BIGINT NOT NULL DEFAULT 0," +
			"`opackets` BIGINT NOT NULL DEFAULT 0," +
			"`point_to_point` varchar(255) NOT NULL DEFAULT ''," +
			"`type` INT NOT NULL DEFAULT 0," +
			"PRIMARY KEY (`id`), " +
			"FOREIGN KEY `idx_network_interfaces_hosts_fk` (`host_id`) " +
			"REFERENCES hosts(id) " +
			"ON DELETE CASCADE, " +
			"FULLTEXT KEY `ip_address_search` (`ip_address`)," +
			"UNIQUE KEY `idx_network_interfaces_unique_ip_host_intf` (`ip_address`, `host_id`, `interface`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
	)
	return err
}

func Down_20161128234849(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `network_interfaces`;")
	return err
}
