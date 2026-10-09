package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260901200001, Down_20260901200001)
}

func Up_20260901200001(tx *sql.Tx) error {
	// Temporarily remove ON UPDATE from updated_at so the ENUM modification
	// does not reset every row's timestamp.
	if _, err := tx.Exec(`
		ALTER TABLE mdm_android_commands
			MODIFY COLUMN updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
	`); err != nil {
		return fmt.Errorf("drop updated_at ON UPDATE: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE mdm_android_commands
			MODIFY COLUMN status ENUM('Pending','Acknowledged','Error') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'Pending'
	`); err != nil {
		return fmt.Errorf("normalize mdm_android_commands status casing: %w", err)
	}

	// Restore ON UPDATE on updated_at.
	if _, err := tx.Exec(`
		ALTER TABLE mdm_android_commands
			MODIFY COLUMN updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
	`); err != nil {
		return fmt.Errorf("restore updated_at ON UPDATE: %w", err)
	}

	return nil
}

func Down_20260901200001(tx *sql.Tx) error {
	return nil
}
