package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210601000007, Down_20210601000007)
}

func Up_20210601000007(tx *sql.Tx) error {
	sql := `
		ALTER TABLE queries
		ADD COLUMN observer_can_run TINYINT(1) NOT NULL DEFAULT FALSE
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add column observer_run")
	}
	return nil
}

func Down_20210601000007(tx *sql.Tx) error {
	return nil
}
