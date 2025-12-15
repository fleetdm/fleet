package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251209221850, Down_20251209221850)
}

func Up_20251209221850(tx *sql.Tx) error {
	// Make fleet_challenge nullable (it will be NULL for pending records,
	// and populated when transitioning to delivering).
	_, err := tx.Exec(`
		ALTER TABLE host_certificate_templates
		MODIFY COLUMN fleet_challenge char(32) COLLATE utf8mb4_unicode_ci NULL
	`)
	if err != nil {
		return fmt.Errorf("make fleet_challenge nullable: %w", err)
	}

	// Add default 'pending' to status column.
	_, err = tx.Exec(`
		ALTER TABLE host_certificate_templates
		MODIFY COLUMN status varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending'
	`)
	if err != nil {
		return fmt.Errorf("add default to status column: %w", err)
	}

	return nil
}

func Down_20251209221850(tx *sql.Tx) error {
	return nil
}
