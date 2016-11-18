package migration

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up_20161118212604, Down_20161118212604)
}

func Up_20161118212604(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE `options` (" +
			"`id` int(10) unsigned NOT NULL AUTO_INCREMENT," +
			"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
			"`updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
			"`key` varchar(255) NOT NULL," +
			"`value` varchar(255) NOT NULL," +
			"`platform` varchar(255) DEFAULT NULL," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE KEY `idx_option_unique_key` (`key`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8;",
	)
	return err
}

func Down_20161118212604(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `options`;")
	return err
}
