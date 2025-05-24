package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212630, Down_20161118212630)
}

func Up_20161118212630(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE `packs` (" +
			"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
			"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
			"`updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
			"`deleted_at` timestamp NULL DEFAULT NULL," +
			"`deleted` tinyint(1) NOT NULL DEFAULT FALSE," +
			"`disabled` tinyint(1) NOT NULL DEFAULT FALSE," +
			"`name` varchar(255) NOT NULL," +
			"`description` varchar(255) DEFAULT NULL," +
			"`platform` varchar(255) DEFAULT NULL," +
			"`created_by` int(10) unsigned DEFAULT NULL," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE KEY `idx_pack_unique_name` (`name`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
	)
	return err
}

func Down_20161118212630(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `packs`;")
	return err
}
