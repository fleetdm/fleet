package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240312103753, Down_20240312103753)
}

func Up_20240312103753(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE mdm_windows_enrollments
		ADD INDEX idx_mdm_windows_enrollments_mdm_device_id (mdm_device_id)`,
	)
	if err != nil {
		return fmt.Errorf("failed to add index to mdm_windows_enrollments.mdm_device_id: %w", err)
	}
	return nil
}

func Down_20240312103753(tx *sql.Tx) error {
	return nil
}
