package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210408153510, Down_20210408153510)
}

func Up_20210408153510(tx *sql.Tx) error {
	query := "DELETE FROM users WHERE NOT enabled"
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "delete disabled users")
	}

	query = "ALTER TABLE users DROP COLUMN enabled"
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "drop enabled")
	}
	return nil
}

func Down_20210408153510(tx *sql.Tx) error {
	return nil
}
