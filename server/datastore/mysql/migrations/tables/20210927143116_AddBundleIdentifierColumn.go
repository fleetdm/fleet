package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210927143116, Down_20210927143116)
}

func Up_20210927143116(tx *sql.Tx) error {
	if columnExists(tx, "software", "bundle_identifier") {
		return nil
	}

	if _, err := tx.Exec(`ALTER TABLE software ADD COLUMN bundle_identifier VARCHAR(255) DEFAULT ''`); err != nil {
		return errors.Wrap(err, "add column bundle_identifier")
	}
	return nil
}

func Down_20210927143116(tx *sql.Tx) error {
	return nil
}
