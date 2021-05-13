package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210513115729, Down_20210513115729)
}

func Up_20210513115729(tx *sql.Tx) error {
	sql := `
		ALTER TABLE hosts
		ADD COLUMN refetch_requested TINYINT(1) NOT NULL DEFAULT 0
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add column refetch_requested")
	}
	return nil
}

func Down_20210513115729(tx *sql.Tx) error {
	return nil
}
