package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240226082255, Down_20240226082255)
}

func Up_20240226082255(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE teams DROP INDEX idx_name`)
	if err != nil {
		return fmt.Errorf("failed to drop teams idx_name: %w", err)
	}

	// Add a new virtual name column with binary collation
	_, err = tx.Exec(`ALTER TABLE teams ADD COLUMN name_bin VARCHAR(255) COLLATE utf8mb4_bin GENERATED ALWAYS AS (name) VIRTUAL`)
	if err != nil {
		return fmt.Errorf("failed to add virtual column to teams: %w", err)
	}

	// Put index on the new virtual column -- this is needed to support emojis.
	_, err = tx.Exec(`CREATE UNIQUE INDEX idx_name_bin ON teams (name_bin)`)
	if err != nil {
		return fmt.Errorf("failed to add idx_name to teams: %w", err)
	}

	return nil
}

func Down_20240226082255(_ *sql.Tx) error {
	return nil
}
