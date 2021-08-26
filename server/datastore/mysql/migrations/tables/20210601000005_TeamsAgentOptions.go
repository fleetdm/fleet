package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210601000005, Down_20210601000005)
}

func Up_20210601000005(tx *sql.Tx) error {
	sql := `
		ALTER TABLE teams
		ADD COLUMN agent_options JSON
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add column agent_options")
	}

	return nil
}

func Down_20210601000005(tx *sql.Tx) error {
	return nil
}
