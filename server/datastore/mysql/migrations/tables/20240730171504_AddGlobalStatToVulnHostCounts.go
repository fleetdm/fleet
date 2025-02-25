package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240730171504, Down_20240730171504)
}

func Up_20240730171504(tx *sql.Tx) error {
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

	// Insert "no team" counts
	stmt = `
	INSERT INTO vulnerability_host_counts (cve, team_id, host_count, global_stats)
	SELECT
		vhc1.cve,
		0 AS team_id,
		GREATEST(vhc1.host_count - COALESCE(SUM(vhc2.host_count), 0), 0) AS host_count,
		0 AS global_stats
	FROM
		vulnerability_host_counts vhc1
	LEFT JOIN
		vulnerability_host_counts vhc2 ON vhc1.cve = vhc2.cve AND vhc2.team_id != 0 AND vhc2.global_stats = 0
	WHERE
		vhc1.team_id = 0 AND vhc1.global_stats = 1
	GROUP BY
		vhc1.cve, vhc1.host_count
	`
	_, err = tx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("failed to insert no team counts: %w", err)
	}

	return nil
}

func Down_20240730171504(tx *sql.Tx) error {
	return nil
}
