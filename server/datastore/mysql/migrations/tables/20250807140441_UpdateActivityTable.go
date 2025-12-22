package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250807140441, Down_20250807140441)
}

func Up_20250807140441(tx *sql.Tx) error {
	// Update batch activities table to add new columns and rename existing ones
	if _, err := tx.Exec(`
ALTER TABLE batch_activities
ADD COLUMN started_at datetime NULL DEFAULT NULL AFTER updated_at,
ADD COLUMN canceled bool DEFAULT false AFTER finished_at,
RENAME COLUMN completed_at TO finished_at,
DROP COLUMN canceled_at;
`); err != nil {
		return fmt.Errorf("failed to add columns to batch_activities: %w", err)
	}
	return nil
}

func Down_20250807140441(tx *sql.Tx) error {
	return nil
}
