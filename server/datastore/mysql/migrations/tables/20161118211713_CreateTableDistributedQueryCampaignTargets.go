package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118211713, Down_20161118211713)
}

func Up_20161118211713(tx *sql.Tx) error {
	sqlStatement := "CREATE TABLE `distributed_query_campaign_targets` (" +
		"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
		"`type` int(11) DEFAULT NULL," +
		"`distributed_query_campaign_id` int(10) unsigned DEFAULT NULL," +
		"`target_id` int(10) unsigned DEFAULT NULL," +
		"PRIMARY KEY (`id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	_, err := tx.Exec(sqlStatement)
	return err

}

func Down_20161118211713(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `distributed_query_campaign_targets`;")
	return err
}
