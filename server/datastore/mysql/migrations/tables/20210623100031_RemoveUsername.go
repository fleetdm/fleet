package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210623100031, Down_20210623100031)
}

func Up_20210623100031(tx *sql.Tx) error {
	sql := `
		UPDATE users
		SET name = username
		WHERE name IS NULL OR name = ''
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "replace empty names with username")
	}

	sql = `
		ALTER TABLE users
		DROP COLUMN username
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "drop username")
	}

	return nil
}

func Down_20210623100031(tx *sql.Tx) error {
	return nil
}
