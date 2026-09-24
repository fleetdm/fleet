package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260807151355, Down_20260807151355)
}

// Up_20260805182836 adds an index supporting the Android command reconciler's batch query
// (ListPendingMDMAndroidCommands), which reads
//
//	WHERE status = 'pending' AND created_at < ? ORDER BY created_at, command_uuid LIMIT ?
//
// mdm_android_commands only had the primary key, the operation_name unique key, and a host_uuid
// key, none of which lead with status, so that query was a full table scan. The table grows with
// every Lock/Wipe/Clear-passcode ever issued while the pending rows the cron wants are a small
// slice of it, so the scan gets steadily more expensive as the table grows.
//
// status (equality) leads, created_at (range) follows -- the order MySQL needs to use both
// predicates from one index. InnoDB appends the primary key (command_uuid) to every secondary
// index, so this also satisfies the ORDER BY and the LIMIT can stop early instead of sorting.
//
// ALGORITHM=INPLACE, LOCK=NONE so the index builds without blocking command inserts.
func Up_20260807151355(tx *sql.Tx) error {
	stmt := `ALTER TABLE mdm_android_commands
		ADD INDEX idx_mdm_android_commands_status_created_at (status, created_at),
		ALGORITHM=INPLACE, LOCK=NONE`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("failed to add idx_mdm_android_commands_status_created_at: %w", err)
	}
	return nil
}

func Down_20260807151355(tx *sql.Tx) error {
	return nil
}
