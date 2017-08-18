package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up20170519105648, Down20170519105648)
}

func Up20170519105648(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `app_configs` " +
			"ADD COLUMN `fim_interval` " +
			"INT NOT NULL DEFAULT 300 AFTER `enable_sso`;",
	)
	return err
}

func Down20170519105648(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `app_configs` DROP COLUMN `fim_interval` ;",
	)
	return err
}
