package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212515, Down_20161118212515)
}

func Up_20161118212515(tx *sql.Tx) error {
	sqlStatement := "CREATE TABLE `distributed_query_executions` (" +
		"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
		"`host_id` int(10) unsigned DEFAULT NULL," +
		"`distributed_query_campaign_id` int(10) unsigned DEFAULT NULL," +
		"`status` int(11) DEFAULT NULL," +
		"`error` varchar(1024) DEFAULT NULL," +
		"`execution_duration` bigint(20) DEFAULT NULL," +
		"UNIQUE KEY `idx_dqe_unique_host_dqc_id` (`host_id`, `distributed_query_campaign_id`)," +
		"PRIMARY KEY (`id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	_, err := tx.Exec(sqlStatement)
	return err
}

func Down_20161118212515(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `distributed_query_executions`;")
	return err
}
