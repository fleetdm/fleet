package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230918132351, Down_20230918132351)
}

func Up_20230918132351(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE software_cve
		ADD COLUMN resolved_in_version VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL;
	`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add resolved_in_version column to software_cve: %w", err)
	}

	return nil
}

func Down_20230918132351(tx *sql.Tx) error {
	return nil
}
