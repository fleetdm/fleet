package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220309133956, Down_20220309133956)
}

func Up_20220309133956(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE teams ADD COLUMN config JSON DEFAULT NULL`); err != nil {
		return errors.Wrap(err, "add config column to teams table")
	}
	return nil
}

func Down_20220309133956(tx *sql.Tx) error {
	return nil
}
