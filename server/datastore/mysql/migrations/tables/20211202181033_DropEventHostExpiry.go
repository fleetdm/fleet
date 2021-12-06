package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211202181033, Down_20211202181033)
}

func Up_20211202181033(tx *sql.Tx) error {
	if _, err := tx.Exec(`DROP EVENT IF EXISTS host_expiry;`); err != nil {
		return errors.Wrap(err, "dropping host_expiry event")
	}
	return nil
}

func Down_20211202181033(tx *sql.Tx) error {
	return nil
}
