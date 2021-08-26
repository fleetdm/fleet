package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210818182258, Down_20210818182258)
}

func Up_20210818182258(tx *sql.Tx) error {
	sql := `
		ALTER TABLE hosts
		ADD COLUMN gigs_disk_space_available FLOAT NOT NULL DEFAULT 0,
        ADD COLUMN percent_disk_space_available FLOAT NOT NULL DEFAULT 0
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add columns for disk_space")
	}
	return nil
}

func Down_20210818182258(tx *sql.Tx) error {
	return nil
}
