package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212758, Down_20161118212758)
}

func Up_20161118212758(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE `queries` (" +
			"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
			"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
			"`updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
			"`deleted_at` timestamp NULL DEFAULT NULL," +
			"`deleted` tinyint(1) NOT NULL DEFAULT FALSE," +
			"`saved` tinyint(1) NOT NULL DEFAULT FALSE," +
			"`name` varchar(255) NOT NULL," +
			"`description` varchar(255) DEFAULT NULL," +
			"`query` varchar(255) NOT NULL," +
			"`author_id` int(10) unsigned DEFAULT NULL," +
			"PRIMARY KEY (`id`)," +
			"FOREIGN KEY (`author_id`) REFERENCES `users`(`id`) ON DELETE SET NULL" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
	)
	return err
}

func Down_20161118212758(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `queries`;")
	return err
}
