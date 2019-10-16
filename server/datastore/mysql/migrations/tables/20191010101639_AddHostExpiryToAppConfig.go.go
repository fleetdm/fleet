package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up20191010101639, Down20191010101639)
}

func Up20191010101639(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `app_configs` " +
			"ADD COLUMN `host_expiry_enabled` TINYINT(1) NOT NULL DEFAULT FALSE, " +
			"ADD COLUMN `host_expiry_window` int DEFAULT 0;",
	)
	return err
}

func Down20191010101639(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `app_configs` " +
			"DROP COLUMN `host_expiry_enabled`, " +
			"DROP COLUMN `host_expiry_window`;",
	)
	return err
}
