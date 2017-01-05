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
			"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
			"`updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
			"`expires_at` timestamp NOT NULL DEFAULT '1970-01-01 00:00:01'," +
			"`user_id` int(10) unsigned NOT NULL," +
			"`token` varchar(1024) NOT NULL," +
			"PRIMARY KEY (`id`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8;",
	)
	return err
}

func Down_20161118212641(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `password_reset_requests`;")
	return err
}
