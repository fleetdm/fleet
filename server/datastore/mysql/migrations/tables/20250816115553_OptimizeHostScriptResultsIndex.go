package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250816115553, Down_20250816115553)
}

func Up_20250816115553(tx *sql.Tx) error {
	// Replace the inefficient index with an optimized one that includes canceled status
	// and has created_at in descending order to better support "find latest" queries
	// This addresses performance issues with hosts having 40,000+ script execution records
	// https://github.com/fleetdm/fleet/issues/32295

	// First, create the new index
	// Including canceled in the index allows for more efficient filtering and index-only scans
	// DESC on created_at helps with queries that need the most recent records
	createIndexStmt := `
		CREATE INDEX idx_host_script_canceled_created_at 
		ON host_script_results(host_id, script_id, canceled, created_at DESC)
	`
	if _, err := tx.Exec(createIndexStmt); err != nil {
		return fmt.Errorf("failed to create optimized index: %w", err)
	}

	// Then drop the old, less efficient index
	dropIndexStmt := `
		DROP INDEX idx_host_script_created_at ON host_script_results
	`
	if _, err := tx.Exec(dropIndexStmt); err != nil {
		return fmt.Errorf("failed to drop old index: %w", err)
	}

	return nil
}

func Down_20250816115553(_ *sql.Tx) error {
	// Not used
	return nil
}
