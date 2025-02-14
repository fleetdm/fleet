package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250214205657, Down_20250214205657)
}

func Up_20250214205657(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE software_cve ADD INDEX idx_software_cve_cve (cve);`)
	if err != nil {
		return fmt.Errorf("failed to add index to software_cve.cve: %w", err)
	}
	return nil
}

func Down_20250214205657(tx *sql.Tx) error {
	return nil
}
