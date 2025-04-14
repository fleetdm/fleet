package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250414140429, Down_20250414140429)
}

func Up_20250414140429(tx *sql.Tx) error {
	if _, err := tx.Exec(`CREATE INDEX idx_software_installers_storage_id ON software_installers (storage_id)`); err != nil {
		return fmt.Errorf("creating software_installers.storage_id index: %w", err)
	}

	return nil
}

func Down_20250414140429(tx *sql.Tx) error {
	return nil
}
