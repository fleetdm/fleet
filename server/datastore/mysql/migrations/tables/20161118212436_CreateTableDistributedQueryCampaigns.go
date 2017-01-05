package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212436, Down_20161118212436)
}

func Up_20161118212436(tx *sql.Tx) error {
	sqlStatement := "CREATE TABLE `distributed_query_campaigns` (" +
		"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
		"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
		"`updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
		"`deleted_at` timestamp NULL DEFAULT NULL," +
		"`deleted` tinyint(1) NOT NULL DEFAULT FALSE," +
		"`query_id` int(10) unsigned DEFAULT NULL," +
		"`status` int(11) DEFAULT NULL," +
		"`user_id` int(10) unsigned DEFAULT NULL," +
		"PRIMARY KEY (`id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	_, err := tx.Exec(sqlStatement)
	return err
}

func Down_20161118212436(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `distributed_query_campaigns`;")
	return err
}
