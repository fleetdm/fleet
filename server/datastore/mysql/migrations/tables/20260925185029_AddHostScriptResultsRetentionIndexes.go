package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260925185029, Down_20260925185029)
}

func Up_20260925185029(tx *sql.Tx) error {
	// Without this index the sweep's steady-state "nothing left" case scans the
	// whole table every hour. created_at is stamped once on insert, so the
	// index only grows at its right edge.
	if err := addIndexesTx(tx, "host_script_results",
		indexDef{"idx_host_script_results_created_at", "created_at"}); err != nil {
		return err
	}

	// The sweep keeps a script result that a batch run still points at, and
	// this is the only column it probes that had no index.
	return addIndexesTx(tx, "batch_activity_host_results",
		indexDef{"idx_bahr_host_execution_id", "host_execution_id"})
}

func Down_20260925185029(tx *sql.Tx) error {
	return nil
}
