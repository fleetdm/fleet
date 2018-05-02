package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up20170831234303, Down20170831234303)
}

func Up20170831234303(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `app_configs` " +
			"ADD COLUMN `fim_file_accesses` VARCHAR(255) NOT NULL DEFAULT '';",
	)
	return err
}

func Down20170831234303(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `app_configs` DROP COLUMN `fim_file_accesses` ;",
	)
	return err
}
