package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260925155809, Down_20260925155809)
}

// Up_20260925155809 gives enrollments orphaned before host deletion started
// touching updated_at a fresh retention window, so the first stale-enrollment
// cleanup after upgrade does not reap devices whose host was deleted recently.
func Up_20260925155809(tx *sql.Tx) error {
	_, err := tx.Exec(`
		UPDATE mdm_windows_enrollments e
		SET e.updated_at = CURRENT_TIMESTAMP
		WHERE e.host_uuid <> ''
		  AND NOT EXISTS (SELECT 1 FROM hosts h WHERE h.uuid = e.host_uuid)`)
	if err != nil {
		return fmt.Errorf("touching orphaned windows mdm enrollments: %w", err)
	}
	return nil
}

func Down_20260925155809(tx *sql.Tx) error {
	return nil
}
