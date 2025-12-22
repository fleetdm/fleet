package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20250827113140, Down_20250827113140)
}

func Up_20250827113140(tx *sql.Tx) error {
	// Add index on distributed_query_campaign_id column to improve query performance
	// This is the most common WHERE clause used when querying this table
	_, err := tx.Exec(`
		ALTER TABLE distributed_query_campaign_targets
		ADD INDEX idx_distributed_query_campaign_targets_campaign_id (distributed_query_campaign_id)
	`)
	return err
}

func Down_20250827113140(_ *sql.Tx) error {
	return nil
}
