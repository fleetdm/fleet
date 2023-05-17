package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230503101418, Down_20230503101418)
}

func Up_20230503101418(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE jobs ADD COLUMN not_before TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
`)
	return errors.Wrap(err, "add not_before")
}

func Down_20230503101418(tx *sql.Tx) error {
	return nil
}
