package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240129115133, Down_20240129115133)
}

func Up_20240129115133(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE operating_systems
		ADD COLUMN os_version_id INT UNSIGNED DEFAULT NULL
		`
	_, err := tx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("failed to add os_version_id column: %w", err)
	}
	
	return nil
}

func Down_20240129115133(tx *sql.Tx) error {
	return nil
}
