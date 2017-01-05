package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212549, Down_20161118212549)
}

func Up_20161118212549(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE `label_query_executions` (" +
			"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
			"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
			"`updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
			"`matches` tinyint(1) NOT NULL DEFAULT FALSE," +
			"`label_id` int(10) unsigned DEFAULT NULL," +
			"`host_id` int(10) unsigned DEFAULT NULL," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE KEY `idx_lqe_label_host` (`label_id`,`host_id`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8;",
	)
	return err
}

func Down_20161118212549(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `label_query_executions`;")
	return err
}
