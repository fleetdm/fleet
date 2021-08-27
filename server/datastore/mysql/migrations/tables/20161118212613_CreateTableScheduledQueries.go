package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212613, Down_20161118212613)
}

func Up_20161118212613(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE IF NOT EXISTS `scheduled_queries` (" +
			"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
			"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
			"`updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
			"`deleted_at` timestamp NULL DEFAULT NULL," +
			"`deleted` tinyint(1) NOT NULL DEFAULT FALSE," +
			"`pack_id` int(10) unsigned DEFAULT NULL," +
			"`query_id` int(10) unsigned DEFAULT NULL," +
			"`interval` int(10) unsigned DEFAULT NULL," +
			"`snapshot` tinyint(1) DEFAULT NULL," +
			"`removed` tinyint(1) DEFAULT NULL," +
			"`platform` varchar(255) DEFAULT NULL," +
			"`version` varchar(255) DEFAULT NULL," +
			"`shard` int(10) unsigned DEFAULT NULL," +
			"PRIMARY KEY (`id`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8;",
	)
	return err
}

func Down_20161118212613(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `scheduled_queries`;")
	return err
}
