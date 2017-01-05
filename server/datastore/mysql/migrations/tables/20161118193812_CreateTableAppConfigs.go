package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20161118193812, Down_20161118193812)
}

func Up_20161118193812(tx *sql.Tx) error {
	sqlStatement := "CREATE TABLE `app_configs` (" +
		"`id` INT(10) UNSIGNED NOT NULL DEFAULT 1," +
		"`org_name` VARCHAR(255) NOT NULL DEFAULT ''," +
		"`org_logo_url` VARCHAR(255) NOT NULL DEFAULT ''," +
		"`kolide_server_url` VARCHAR(255) NOT NULL DEFAULT ''," +
		"`smtp_configured` TINYINT(1) NOT NULL DEFAULT FALSE," +
		"`smtp_sender_address` VARCHAR(255) NOT NULL DEFAULT ''," +
		"`smtp_server` VARCHAR(255) NOT NULL DEFAULT ''," +
		"`smtp_port` INT UNSIGNED NOT NULL DEFAULT 587," +
		"`smtp_authentication_type` INT NOT NULL DEFAULT 0," +
		"`smtp_enable_ssl_tls` TINYINT(1) NOT NULL DEFAULT TRUE," +
		"`smtp_authentication_method` INT NOT NULL DEFAULT 0," +
		"`smtp_domain` VARCHAR(255) NOT NULL DEFAULT ''," +
		"`smtp_user_name` VARCHAR(255) NOT NULL DEFAULT ''," +
		"`smtp_password` VARCHAR(255) NOT NULL DEFAULT ''," +
		"`smtp_verify_ssl_certs` TINYINT(1) NOT NULL DEFAULT TRUE, " +
		"`smtp_enable_start_tls` TINYINT(1) NOT NULL DEFAULT TRUE, " +
		"`smtp_enabled` TINYINT(1) NOT NULL DEFAULT FALSE, " +
		"PRIMARY KEY (`id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	if _, err := tx.Exec(sqlStatement); err != nil {
		return err
	}
	// create an app config record with defaults because there is only one, and it
	// always needs to exist
	if _, err := tx.Exec("INSERT INTO app_configs VALUES ()"); err != nil {
		return err
	}

	return nil
}

func Down_20161118193812(tx *sql.Tx) error {
	sqlStatement := "DROP TABLE IF EXISTS `app_configs`;"
	_, err := tx.Exec(sqlStatement)
	return err
}
