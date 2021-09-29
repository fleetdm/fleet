package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210924114500, Down_20210924114500)
}

func Up_20210924114500(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE software ADD COLUMN bundle_identifier VARCHAR(255) DEFAULT ''`); err != nil {
		return errors.Wrap(err, "add column team_id")
	}
	return nil
}

func Down_20210924114500(tx *sql.Tx) error {
	return nil
}
