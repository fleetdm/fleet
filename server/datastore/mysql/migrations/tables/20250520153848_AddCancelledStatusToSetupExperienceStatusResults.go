package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250520153848, Down_20250520153848)
}

func Up_20250520153848(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE setup_experience_status_results 
		MODIFY COLUMN status ENUM('pending', 'running', 'success', 'failure', 'cancelled') COLLATE utf8mb4_unicode_ci NOT NULL;
	`)
	if err != nil {
		return fmt.Errorf("failed to add cancelled status to setup_experience_status_results table: %w", err)
	}
	return nil
}

func Down_20250520153848(tx *sql.Tx) error {
	return nil
}
