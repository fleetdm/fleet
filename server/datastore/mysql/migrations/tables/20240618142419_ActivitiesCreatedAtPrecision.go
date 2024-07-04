package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240618142419, Down_20240618142419)
}

func Up_20240618142419(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE activities MODIFY COLUMN created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)")
	if err != nil {
		return fmt.Errorf("failed to modify column created_at in activities table: %w", err)
	}
	return nil
}

func Down_20240618142419(_ *sql.Tx) error {
	return nil
}
