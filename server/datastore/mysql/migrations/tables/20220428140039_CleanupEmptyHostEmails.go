package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220428140039, Down_20220428140039)
}

func Up_20220428140039(tx *sql.Tx) error {
	_, err := tx.Exec(`
      DELETE FROM
        host_emails
      WHERE
        email = ''`)

	if err != nil {
		return errors.Wrap(err, "delete empty host emails")
	}

	return nil
}

func Down_20220428140039(tx *sql.Tx) error {
	return nil
}
