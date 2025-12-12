package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212538, Down_20161118212538)
}

func Up_20161118212538(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE `invites` (" +
			"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
			"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
			"`updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
			"`deleted_at` timestamp NULL DEFAULT NULL," +
			"`deleted` tinyint(1) NOT NULL DEFAULT FALSE," +
			"`invited_by` int(10) unsigned NOT NULL," +
			"`email` varchar(255) NOT NULL," +
			"`admin` tinyint(1) DEFAULT NULL," +
			"`name` varchar(255) DEFAULT NULL," +
			"`position` varchar(255) DEFAULT NULL," +
			"`token` varchar(255) NOT NULL," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE KEY `idx_invite_unique_email` (`email`)," +
			"UNIQUE KEY `idx_invite_unique_key` (`token`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
	)
	return err
}

func Down_20161118212538(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `invites`;")
	return err
}
