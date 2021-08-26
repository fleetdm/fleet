package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210616163757, Down_20210616163757)
}

func Up_20210616163757(tx *sql.Tx) error {
	sql := `
		ALTER TABLE users
		ADD COLUMN api_only TINYINT(1) NOT NULL DEFAULT 0
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add column api_only")
	}
	return nil
}

func Down_20210616163757(tx *sql.Tx) error {
	return nil
}
