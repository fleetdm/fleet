package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20251028103213, Down_20251028103213)
}

func Up_20251028103213(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE in_house_apps
    RENAME COLUMN name TO filename
`)
	if err != nil {
		return errors.Wrapf(err, "in_house_apps table: rename name to filename")
	}
	return nil
}

func Down_20251028103213(tx *sql.Tx) error {
	return nil
}
