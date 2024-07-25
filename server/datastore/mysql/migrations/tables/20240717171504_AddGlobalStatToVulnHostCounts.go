package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240717171504, Down_20240717171504)
}

func Up_20240717171504(tx *sql.Tx) error {
	stmt := `
	ALTER TABLE vulnerability_host_counts
	ADD COLUMN global_stats tinyint(1) NOT NULL DEFAULT 0
	`
	_, err := tx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("failed to add global_stats column: %w", err)
	}

	stmt = `
	ALTER TABLE vulnerability_host_counts
	DROP INDEX cve_team_id
	`
	_, err = tx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("failed to drop index cve_team_id: %w", err)
	}

	stmt = `
	CREATE UNIQUE INDEX cve_team_id_global_stats
	ON vulnerability_host_counts (cve, team_id, global_stats)
	`
	_, err = tx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("failed to create index cve_team_id_global_stats: %w", err)
	}

	stmt = `
	UPDATE vulnerability_host_counts
	SET global_stats = 1
	WHERE team_id = 0
	`
	_, err = tx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("failed to update global_stats for team_id = 0: %w", err)
	}

	return nil
}

func Down_20240717171504(tx *sql.Tx) error {
	return nil
}
