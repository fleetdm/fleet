package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260923183241, Down_20260923183241)
}

func Up_20260923183241(tx *sql.Tx) error {
	// The retention sweep deletes oldest-first by created_at; without an index
	// the steady-state "nothing left" case scans the whole table every hour.
	// Both columns are monotonic, so the index only grows at its right edge.
	// INPLACE/NONE so a server that cannot build it online fails instead of
	// silently copying a table that can hold hundreds of millions of rows.
	if !indexExistsTx(tx, "windows_mdm_responses", "idx_windows_mdm_responses_created_at") {
		if _, err := tx.Exec(`
			ALTER TABLE windows_mdm_responses
			ALGORITHM=INPLACE, LOCK=NONE,
			ADD KEY idx_windows_mdm_responses_created_at (created_at)
		`); err != nil {
			return fmt.Errorf("adding created_at index to windows_mdm_responses: %w", err)
		}
	}

	if !indexExistsTx(tx, "windows_mdm_commands", "idx_windows_mdm_commands_created_at") {
		if _, err := tx.Exec(`
			ALTER TABLE windows_mdm_commands
			ALGORITHM=INPLACE, LOCK=NONE,
			ADD KEY idx_windows_mdm_commands_created_at (created_at)
		`); err != nil {
			return fmt.Errorf("adding created_at index to windows_mdm_commands: %w", err)
		}
	}

	return nil
}

func Down_20260923183241(tx *sql.Tx) error {
	return nil
}
