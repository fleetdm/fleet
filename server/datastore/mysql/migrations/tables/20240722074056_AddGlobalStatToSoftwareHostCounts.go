package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240722074056, Down_20240722074056)
}

func Up_20240722074056(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE software_host_counts
		ADD COLUMN global_stats tinyint unsigned NOT NULL DEFAULT '0',
		DROP PRIMARY KEY,
		ADD PRIMARY KEY (software_id, team_id, global_stats)
	`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add global_stats column to software_host_counts: %w", err)
	}

	// update team counts to have global_stats = 0
	stmt = `
		UPDATE software_host_counts
		SET global_stats = 1
		WHERE team_id = 0
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("update global_stats for team_id = 0: %w", err)
	}

	return nil
}

func Down_20240722074056(tx *sql.Tx) error {
	return nil
}
