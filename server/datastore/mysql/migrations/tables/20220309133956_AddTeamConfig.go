package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220309133956, Down_20220309133956)
}

func Up_20220309133956(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE teams ADD COLUMN config JSON`); err != nil {
		return errors.Wrap(err, "add config column to teams table")
	}
	if _, err := tx.Exec(`UPDATE teams SET config = JSON_SET('{}', '$.agent_options', agent_options)`); err != nil {
		return errors.Wrap(err, "migrate agent_options")
	}
	if _, err := tx.Exec(`ALTER TABLE teams DROP COLUMN agent_options`); err != nil {
		return errors.Wrap(err, "drop agent_options column in teams table")
	}
	return nil
}

func Down_20220309133956(tx *sql.Tx) error {
	return nil
}
