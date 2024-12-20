package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241219180042, Down_20241219180042)
}

func Up_20241219180042(tx *sql.Tx) error {
	// The new 'url' column will only be set for software uploaded in batch via GitOps.
	if _, err := tx.Exec(`
		ALTER TABLE software_installers
		CHANGE COLUMN url url VARCHAR(4095) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '';
	`); err != nil {
		return fmt.Errorf("failed to lengthen url in software_installers: %w", err)
	}
	return nil

	return nil
}

func Down_20241219180042(tx *sql.Tx) error {
	return nil
}
