package migration

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up_20161118193812, Down_20161118193812)
}

func Up_20161118193812(tx *sql.Tx) error {
	sqlStatement := "CREATE TABLE `app_configs` (" +
		"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
		"`org_name` varchar(255) DEFAULT NULL," +
		"`org_logo_url` varchar(255) DEFAULT NULL," +
		"`kolide_server_url` varchar(255) DEFAULT NULL," +
		"PRIMARY KEY (`id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	_, err := tx.Exec(sqlStatement)
	return err

}

func Down_20161118193812(tx *sql.Tx) error {
	sqlStatement := "DROP TABLE IF EXISTS `app_configs`;"
	_, err := tx.Exec(sqlStatement)
	return err
}
