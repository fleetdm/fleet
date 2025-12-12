package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250716152435, Down_20250716152435)
}

func Up_20250716152435(tx *sql.Tx) error {
	if _, err := tx.Exec(`
ALTER TABLE software_installers 
ADD COLUMN upgrade_code VARCHAR(48) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ""
		`); err != nil {
		return fmt.Errorf("failed to alter software_installers: %w", err)
	}

	return nil
}

func Down_20250716152435(tx *sql.Tx) error {
	return nil
}
