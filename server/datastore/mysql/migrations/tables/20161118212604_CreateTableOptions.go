package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161118212604, Down_20161118212604)
}

func Up_20161118212604(tx *sql.Tx) error {

	_, err := tx.Exec(
		"CREATE TABLE `options` (" +
			"`id` INT UNSIGNED NOT NULL AUTO_INCREMENT," +
			"`name` VARCHAR(255) NOT NULL," +
			"`type` INT UNSIGNED NOT NULL," +
			"`value` VARCHAR(255) NOT NULL," +
			"`read_only` TINYINT(1) NOT NULL DEFAULT FALSE," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE KEY `idx_option_unique_name` (`name`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8;",
	)

	return err
}

func Down_20161118212604(tx *sql.Tx) error {

	_, err := tx.Exec("DROP TABLE IF EXISTS `options`;")

	return err
}
