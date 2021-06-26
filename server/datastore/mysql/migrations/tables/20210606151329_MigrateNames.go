package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210606151329, Down_20210606151329)
}

func Up_20210606151329(tx *sql.Tx) error {
	sql := `
        ALTER TABLE app_configs
        CHANGE kolide_server_url server_url varchar(255) NOT NULL DEFAULT ''
    `

	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "rename server_url")
	}
	return nil
}

func Down_20210606151329(tx *sql.Tx) error {
	return nil
}
