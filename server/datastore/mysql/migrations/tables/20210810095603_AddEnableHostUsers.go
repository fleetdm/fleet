package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210810095603, Down_20210810095603)
}

func Up_20210810095603(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE app_configs ADD COLUMN enable_host_users BOOL DEFAULT TRUE`); err != nil {
		return errors.Wrap(err, "add column enable_host_users")
	}
	return nil
}

func Down_20210810095603(tx *sql.Tx) error {
	return nil
}
