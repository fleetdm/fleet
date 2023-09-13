package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230913141311, Down_20230913141311)
}

func Up_20230913141311(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE software_cve
		ADD COLUMN resolved_in_version VARCHAR(50) NOT NULL DEFAULT ''
	`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add resolved_in_version column to software_cve: %w", err)
	}

	return nil
}

func Down_20230913141311(tx *sql.Tx) error {
	return nil
}
