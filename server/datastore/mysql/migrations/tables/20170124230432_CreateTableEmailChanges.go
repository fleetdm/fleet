package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20170124230432, Down_20170124230432)
}

func Up_20170124230432(tx *sql.Tx) error {
	sqlStatement := "CREATE TABLE `email_changes` ( " +
		"`id` int(10) unsigned NOT NULL AUTO_INCREMENT, " +
		"`user_id` int(10) unsigned NOT NULL, " +
		"`token` varchar(128) NOT NULL, " +
		"`new_email` varchar(255) NOT NULL, " +
		"PRIMARY KEY (`id`), " +
		"UNIQUE KEY `idx_unique_email_changes_token` (`token`) USING BTREE, " +
		"KEY `fk_email_changes_users` (`user_id`), " +
		"CONSTRAINT `fk_email_changes_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE " +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;"
	_, err := tx.Exec(sqlStatement)
	return err
}

func Down_20170124230432(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `email_changes`;")

	return err
}
