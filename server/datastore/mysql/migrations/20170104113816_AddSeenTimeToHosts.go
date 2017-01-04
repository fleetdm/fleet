package migration

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up_20170104113816, Down_20170104113816)
}

func Up_20170104113816(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `hosts` " +
			"ADD COLUMN `seen_time` timestamp NULL DEFAULT NULL;",
	)
	return err
}

func Down_20170104113816(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `hosts` " +
			"DROP COLUMN `seen_time`;",
	)
	return err
}
