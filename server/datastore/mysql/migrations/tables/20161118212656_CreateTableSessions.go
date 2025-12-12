package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212656, Down_20161118212656)
}

func Up_20161118212656(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE `sessions` (" +
			"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
			"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
			"`accessed_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
			"`user_id` int(10) unsigned NOT NULL," +
			"`key` varchar(255) NOT NULL," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE KEY `idx_session_unique_key` (`key`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
	)
	return err
}

func Down_20161118212656(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `sessions`;")
	return err
}
