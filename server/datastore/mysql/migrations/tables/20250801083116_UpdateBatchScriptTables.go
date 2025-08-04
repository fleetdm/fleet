package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250801083116, Down_20250801083116)
}

func Up_20250801083116(tx *sql.Tx) error {
	// Rename the existing table from batch_script_executions to batch_activities
	if _, err := tx.Exec(`
ALTER TABLE batch_script_executions RENAME TO batch_activities; 
`); err != nil {
		return fmt.Errorf("failed to rename batch_script_executions: %w", err)
	}

	// Add new columns to the renamed table
	if _, err := tx.Exec(`
ALTER TABLE batch_activities
ADD COLUMN job_id int unsigned AFTER user_id,
ADD COLUMN num_canceled int unsigned NULL DEFAULT NULL AFTER job_id,
ADD COLUMN num_incompatible int unsigned NULL DEFAULT NULL AFTER job_id,
ADD COLUMN num_errored int unsigned NULL DEFAULT NULL AFTER job_id,
ADD COLUMN num_ran int unsigned NULL DEFAULT NULL AFTER job_id,
ADD COLUMN num_pending int unsigned NULL DEFAULT NULL AFTER job_id,
ADD COLUMN num_targeted int unsigned NULL DEFAULT NULL AFTER job_id,
ADD COLUMN activity_type varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci AFTER job_id,
ADD COLUMN status varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci AFTER job_id,
ADD COLUMN completed_at datetime NULL DEFAULT NULL AFTER updated_at,
ADD COLUMN canceled_at datetime NULL DEFAULT NULL AFTER completed_at;
`); err != nil {
		return fmt.Errorf("failed to add columns to batch_activities: %w", err)
	}

	// Update existing rows to have status `started`
	if _, err := tx.Exec(`
UPDATE batch_activities 
SET status = 'started' WHERE status IS NULL;
`); err != nil {
		return fmt.Errorf("failed to update status in batch_activities: %w", err)
	}

	// Update existing rows to have activity_type `script`
	if _, err := tx.Exec(`
UPDATE batch_activities 
SET activity_type = 'script' WHERE activity_type IS NULL;
`); err != nil {
		return fmt.Errorf("failed to update activity_type in batch_activities: %w", err)
	}

	// Add an index on the new `status` column
	if _, err := tx.Exec(`
CREATE INDEX idx_batch_activities_status ON batch_activities (status);
`); err != nil {
		return fmt.Errorf("failed to create index on status in batch_activities: %w", err)
	}

	// Rename batch_script_execution_host_results to batch_activity_host_results
	if _, err := tx.Exec(`
ALTER TABLE batch_script_execution_host_results RENAME TO batch_activity_host_results;
`); err != nil {
		return fmt.Errorf("failed to rename batch_script_execution_host_results: %w", err)
	}

	return nil
}

func Down_20250801083116(tx *sql.Tx) error {
	return nil
}
