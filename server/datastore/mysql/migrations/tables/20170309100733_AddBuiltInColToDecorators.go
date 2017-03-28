package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20170309100733, Down_20170309100733)
}

func Up_20170309100733(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE `decorators` " +
		"ADD COLUMN built_in TINYINT(1) NOT NULL DEFAULT FALSE;")
	return err
}

func Down_20170309100733(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE `decorators` " +
		"DROP COLUMN `built_in`;")
	return err
}
