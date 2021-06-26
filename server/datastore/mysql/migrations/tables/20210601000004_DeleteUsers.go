package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210601000004, Down_20210601000004)
}

func Up_20210601000004(tx *sql.Tx) error {
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

func Down_20210601000004(tx *sql.Tx) error {
	return nil
}
