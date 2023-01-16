package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20221227163856, Down_20221227163856)
}

func Up_20221227163856(tx *sql.Tx) error {
	// software_updated_at is NULL in case we want to add more
	// update related columns to this table in the future.
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

func Down_20221227163856(tx *sql.Tx) error {
	return nil
}
