package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261030212758, Down_20261030212758)
}

func Up_20261030212758(tx *sql.Tx) error {
	if !columnExists(tx, "policies", "patch_deadline") {
		if _, err := tx.Exec(`
			ALTER TABLE policies
			ADD COLUMN patch_deadline TINYINT(1) NOT NULL DEFAULT 0,
			ALGORITHM=INSTANT
		`); err != nil {
			return fmt.Errorf("add patch_deadline to policies: %w", err)
		}
	}
	return nil
}

func Down_20261030212758(tx *sql.Tx) error {
	return nil
}
