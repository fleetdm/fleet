package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260902083032, Down_20260902083032)
}

// Up_20260902083032 adds an index supporting the APNs sweep cron, which walks
// enabled enrollments in primary-key order
//
//	WHERE enabled = 1 AND id > ? ORDER BY id LIMIT ?
//
// and counts them (COUNT(*) WHERE enabled = 1) to size its batches.
// nano_enrollments only had the primary key and device_id/user_id/type keys,
// none of which lead with enabled. InnoDB appends the primary key (id) to
// every secondary index, so this is effectively (enabled, id) and serves both
// the ordered walk and the count as index-only operations.
//
// ALGORITHM=INPLACE, LOCK=NONE so the index builds without blocking check-ins,
// which update this table's last_seen_at constantly.
func Up_20260902083032(tx *sql.Tx) error {
	stmt := `ALTER TABLE nano_enrollments
		ADD INDEX idx_nano_enrollments_enabled (enabled),
		ALGORITHM=INPLACE, LOCK=NONE`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("failed to add idx_nano_enrollments_enabled: %w", err)
	}
	return nil
}

func Down_20260902083032(tx *sql.Tx) error {
	return nil
}
