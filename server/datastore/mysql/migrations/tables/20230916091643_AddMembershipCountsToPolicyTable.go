package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230916091643, Down_20230916091643)
}

func Up_20230916091643(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE policies
		ADD COLUMN failing_host_count MEDIUMINT UNSIGNED DEFAULT 0,
		ADD COLUMN passing_host_count MEDIUMINT UNSIGNED DEFAULT 0
	`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add failed_policy_counts and succeeded_policy_counts columns to policies: %w", err)
	}

	return nil
}

func Down_20230916091643(tx *sql.Tx) error {
	return nil
}
