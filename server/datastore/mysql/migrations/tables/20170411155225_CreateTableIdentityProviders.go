package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20170411155225, Down_20170411155225)
}

func Up_20170411155225(tx *sql.Tx) error {
	statement :=
		"CREATE TABLE `identity_providers` ( " +
			"`id` int(11) NOT NULL AUTO_INCREMENT, " +
			"`sso_url` varchar(1024) NOT NULL DEFAULT '', " +
			"`issuer_uri` varchar(1024) NOT NULL DEFAULT '', " +
			"`cert` text NOT NULL, " +
			"`name` varchar(128) NOT NULL DEFAULT '', " +
			"`image_url` varchar(1024) NOT NULL DEFAULT '', " +
			"`created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, " +
			"`updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, " +
			"`deleted_at` timestamp NULL DEFAULT NULL, " +
			"`deleted` tinyint(1) NOT NULL DEFAULT FALSE, " +
			"PRIMARY KEY (`id`), " +
			"UNIQUE KEY `idx_unique_identity_providers_name` (`name`) USING BTREE " +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	_, err := tx.Exec(statement)
	return err
}

func Down_20170411155225(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `identity_providers`;")
	return err
}
