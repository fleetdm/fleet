package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212621, Down_20161118212621)
}

func Up_20161118212621(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE `pack_targets` (" +
			"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
			"`pack_id` int(10) unsigned DEFAULT NULL," +
			"`type` int(11) DEFAULT NULL," +
			"`target_id` int(10) unsigned DEFAULT NULL," +
			"PRIMARY KEY (`id`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8;",
	)
	return err
}

func Down_20161118212621(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `pack_targets`;")
	return err
}
