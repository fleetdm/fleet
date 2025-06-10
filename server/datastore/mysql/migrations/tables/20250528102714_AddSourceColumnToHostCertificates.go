package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250528102714, Down_20250528102714)
}

func Up_20250528102714(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE host_certificates
		ADD COLUMN source ENUM('system', 'user') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'system'
`)
	if err != nil {
		return fmt.Errorf("failed to add column 'source' to 'host_certificates': %w", err)
	}
	return nil
}

func Down_20250528102714(tx *sql.Tx) error {
	return nil
}
