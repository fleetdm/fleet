package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20170127014618, Down_20170127014618)
}

func Up_20170127014618(tx *sql.Tx) error {
	sqlStatement := "CREATE TABLE `licenses` ( " +
		"`id` int(10) NOT NULL AUTO_INCREMENT, " +
		"`updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, " +
		"`revoked` tinyint(1) unsigned NOT NULL DEFAULT FALSE, " +
		"`key` text NOT NULL, " +
		"`token` text, " +
		"PRIMARY KEY (`id`) " +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	_, err := tx.Exec(sqlStatement)
	return err
}

func Down_20170127014618(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `licenses`;")
	return err
}
