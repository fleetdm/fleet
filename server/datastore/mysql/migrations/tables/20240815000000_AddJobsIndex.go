package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240815000000, Down_20240815000000)
}

func Up_20240815000000(tx *sql.Tx) error {
	if _, err := tx.Exec(`CREATE INDEX idx_jobs_state_not_before_updated_at ON jobs (state, not_before, updated_at);`); err != nil {
		return fmt.Errorf("creating jobs index: %w", err)
	}
	return nil
}

func Down_20240815000000(tx *sql.Tx) error {
	return nil
}
