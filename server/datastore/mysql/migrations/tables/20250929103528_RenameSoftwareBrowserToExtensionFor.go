package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20250929103528, Down_20250929103528)
}

func Up_20250929103528(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE software
    RENAME COLUMN browser TO extension_for
`)
	if err != nil {
		return errors.Wrapf(err, "software table: rename browser to extension_for")
	}

	_, err = tx.Exec(`
ALTER TABLE software_titles
    RENAME COLUMN browser TO extension_for
`)
	if err != nil {
		return errors.Wrapf(err, "software_titles table: rename browser to extension_for")
	}

	return nil
}

func Down_20250929103528(tx *sql.Tx) error {
	return nil
}
