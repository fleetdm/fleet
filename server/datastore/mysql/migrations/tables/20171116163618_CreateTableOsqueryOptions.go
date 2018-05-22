package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20171116163618, Down_20171116163618)
}

func Up_20171116163618(tx *sql.Tx) error {
	sqlStatement := "CREATE TABLE `osquery_options` (" +
		"`id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT," +
		"`override_type` INT(1) NOT NULL, " +
		"`override_identifier` VARCHAR(255) NOT NULL DEFAULT ''," +
		"`options` JSON NOT NULL," +
		"PRIMARY KEY (`id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	_, err := tx.Exec(sqlStatement)
	return err
}

func Down_20171116163618(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `osquery_options`;")
	return err
}
