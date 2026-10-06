package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260925160000, Down_20260925160000)
}

func Up_20260925160000(tx *sql.Tx) error {
	if columnExists(tx, "patch_notification_apps", "updated_in_inventory") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE patch_notification_apps
		ADD COLUMN updated_in_inventory TINYINT(1) NOT NULL DEFAULT 0,
		ALGORITHM=INSTANT
	`); err != nil {
		return fmt.Errorf("adding updated_in_inventory to patch_notification_apps table: %w", err)
	}
	return nil
}

func Down_20260925160000(tx *sql.Tx) error {
	return nil
}
