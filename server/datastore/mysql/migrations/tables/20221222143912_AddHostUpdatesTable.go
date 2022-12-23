package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20221222143912, Down_20221222143912)
}

func Up_20221222143912(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE host_updates (
			host_id int(10) unsigned NOT NULL PRIMARY KEY,
			software_updated_at timestamp NULL
		)
	`)
	if err != nil {
		return errors.Wrapf(err, "create table host_updates")
	}

	return nil
}

func Down_20221222143912(tx *sql.Tx) error {
	return nil
}
