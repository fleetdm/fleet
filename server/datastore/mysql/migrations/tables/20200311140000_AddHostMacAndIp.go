package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up20200311140000, Down20200311140000)
}

func Up20200311140000(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `hosts` " +
			"ADD COLUMN `primary_ip` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"ADD COLUMN `primary_mac` varchar(17) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''," +
			"ADD FULLTEXT KEY `host_ip_mac_search` (`primary_ip`,`primary_mac`)",
	)
	return err
}

func Down20200311140000(tx *sql.Tx) error {
	return nil
}
