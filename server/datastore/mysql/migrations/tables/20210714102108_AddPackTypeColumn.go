package tables

import (
	"database/sql"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210714102108, Down_20210714102108)
}

func Up_20210714102108(tx *sql.Tx) error {
	sql := `
		ALTER TABLE packs
		ADD COLUMN pack_type varchar(255) DEFAULT NULL
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add pack_type")
	}
	return nil
}

func Down_20210714102108(tx *sql.Tx) error {
	return nil
}
