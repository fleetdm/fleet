package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20170519105647, Down_20170519105647)
}

func Up_20170519105647(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `decorators` " +
			"ADD COLUMN `name` VARCHAR(128) " +
			"CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci " +
			"NOT NULL DEFAULT '' AFTER `built_in`;",
	)

	return err
}

func Down_20170519105647(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE `decorators` DROP COLUMN `name` ;")
	return err
}
