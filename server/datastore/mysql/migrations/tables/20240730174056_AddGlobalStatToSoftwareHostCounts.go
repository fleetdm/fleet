package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240730174056, Down_20240730174056)
}

func Up_20240730174056(tx *sql.Tx) error {
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

	// Insert "no team" counts
	stmt = `
		INSERT INTO software_host_counts (software_id, hosts_count, team_id, global_stats)
		SELECT
			sthc1.software_id,
			GREATEST(sthc1.hosts_count - COALESCE(SUM(sthc2.hosts_count), 0),0) AS hosts_count,
			0 AS team_id,
			0 AS global_stats
		FROM
			software_host_counts sthc1
		LEFT JOIN
			software_host_counts sthc2 ON sthc1.software_id = sthc2.software_id AND sthc2.team_id != 0 AND sthc2.global_stats = 0
		WHERE
			sthc1.team_id = 0 AND sthc1.global_stats = 1
		GROUP BY
			sthc1.software_id, sthc1.hosts_count
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("insert no team counts: %w", err)
	}

	return nil
}

func Down_20240730174056(tx *sql.Tx) error {
	return nil
}
