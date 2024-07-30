package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231224070653, Down_20231224070653)
}

func Up_20231224070653(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE operating_system_vulnerabilities
		ADD COLUMN resolved_in_version VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
		ADD COLUMN updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
	`)
	if err != nil {
		return fmt.Errorf("adding operating_system_vulnerabilities columns: %w", err)
	}

	return nil
}

func Down_20231224070653(tx *sql.Tx) error {
	return nil
}
