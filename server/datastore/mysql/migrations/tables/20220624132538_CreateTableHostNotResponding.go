package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220624132538, Down_20220624132538)
}

func Up_20220624132538(tx *sql.Tx) error {
	stmt := `
		CREATE TABLE IF NOT EXISTS host_not_responding (
			host_id int(10) UNSIGNED NOT NULL PRIMARY KEY
		)
	`
	if _, err := tx.Exec(stmt); err != nil {
		return errors.Wrap(err, "create host_not_responding table")
	}
	return nil
}

func Down_20220624132538(tx *sql.Tx) error {
	return nil
}
