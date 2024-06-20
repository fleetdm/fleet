package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240613162201, Down_20240613162201)
}

func Up_20240613162201(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE mdm_windows_enrollments
		ADD INDEX idx_mdm_windows_enrollments_host_uuid (host_uuid)`,
	)
	if err != nil {
		return fmt.Errorf("failed to add index to mdm_windows_enrollments.host_uuid: %w", err)
	}
	return nil
}

func Down_20240613162201(tx *sql.Tx) error {
	return nil
}
