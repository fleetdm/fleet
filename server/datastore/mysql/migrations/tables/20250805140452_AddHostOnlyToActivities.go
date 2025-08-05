package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250805140452, Down_20250805140452)
}

func Up_20250805140452(tx *sql.Tx) error {
	stmt := `ALTER TABLE activities ADD COLUMN host_only BOOLEAN NOT NULL DEFAULT false`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("adding host_only column to activities table: %w", err)
	}
	return nil
}

func Down_20250805140452(tx *sql.Tx) error {
	return nil
}
