package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212641, Down_20161118212641)
}

func Up_20161118212641(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE `password_reset_requests` (" +
			"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
			"`expires_at` timestamp NOT NULL," +
			"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
			"`updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
			"`user_id` int(10) unsigned NOT NULL," +
			"`token` varchar(1024) NOT NULL," +
			"PRIMARY KEY (`id`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
	)
	return err
}

func Down_20161118212641(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `password_reset_requests`;")
	return err
}
