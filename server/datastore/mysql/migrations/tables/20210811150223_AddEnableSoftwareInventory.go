package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210811150223, Down_20210811150223)
}

func Up_20210811150223(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE app_configs ADD COLUMN enable_software_inventory BOOL DEFAULT FALSE`); err != nil {
		return errors.Wrap(err, "add column enable_software_inventory")
	}
	return nil
}

func Down_20210811150223(tx *sql.Tx) error {
	return nil
}
