package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212649, Down_20161118212649)
}

func Up_20161118212649(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE `users` (" +
			"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
			"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
			"`updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
			"`deleted_at` timestamp NULL DEFAULT NULL," +
			"`deleted` tinyint(1) NOT NULL DEFAULT FALSE," +
			"`username` varchar(255) NOT NULL," +
			"`password` varbinary(255) NOT NULL," +
			"`salt` varchar(255) NOT NULL," +
			"`name` varchar(255) NOT NULL DEFAULT ''," +
			"`email` varchar(255) NOT NULL," +
			"`admin` tinyint(1) NOT NULL DEFAULT FALSE," +
			"`enabled` tinyint(1) NOT NULL DEFAULT FALSE," +
			"`admin_forced_password_reset` tinyint(1) NOT NULL DEFAULT FALSE," +
			"`gravatar_url` varchar(255) NOT NULL DEFAULT ''," +
			"`position` varchar(255) NOT NULL DEFAULT ''," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE KEY `idx_user_unique_username` (`username`)," +
			"UNIQUE KEY `idx_user_unique_email` (`email`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
	)
	return err
}

func Down_20161118212649(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `users`;")
	return err
}
