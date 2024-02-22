package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231215122713, Down_20231215122713)
}

func Up_20231215122713(tx *sql.Tx) error {
	// NOTE these queries are duplicated in the mysql method here.  Updates
	// to these queries should be reflected there as well.
	// https://github.com/fleetdm/fleet/blob/main/server/datastore/mysql/policies.go#L1125

	// Update Counts for Inherited Global Policies for each Team
	inheritedStatsQuery := `
		INSERT INTO policy_stats (policy_id, inherited_team_id, passing_host_count, failing_host_count)
		SELECT
			p.id,
			t.id AS inherited_team_id,
			(
				SELECT COUNT(*) 
				FROM policy_membership pm 
				INNER JOIN hosts h ON pm.host_id = h.id 
				WHERE pm.policy_id = p.id AND pm.passes = true AND h.team_id = t.id
			) AS passing_host_count,
			(
				SELECT COUNT(*) 
				FROM policy_membership pm 
				INNER JOIN hosts h ON pm.host_id = h.id 
				WHERE pm.policy_id = p.id AND pm.passes = false AND h.team_id = t.id
			) AS failing_host_count
		FROM policies p
		CROSS JOIN teams t
		WHERE p.team_id IS NULL
		GROUP BY p.id, t.id
		ON DUPLICATE KEY UPDATE 
			updated_at = NOW(),
			passing_host_count = VALUES(passing_host_count),
			failing_host_count = VALUES(failing_host_count);
    `

	// Update Counts for Global and Team Policies
	globalAndTeamStatsQuery := `
		INSERT INTO policy_stats (policy_id, inherited_team_id, passing_host_count, failing_host_count)
		SELECT
			p.id,
			0 AS inherited_team_id, -- using 0 to represent global scope
			COALESCE(SUM(IF(pm.passes IS NULL, 0, pm.passes = 1)), 0), 
			COALESCE(SUM(IF(pm.passes IS NULL, 0, pm.passes = 0)), 0)
		FROM policies p
		LEFT JOIN policy_membership pm ON p.id = pm.policy_id
		GROUP BY p.id
		ON DUPLICATE KEY UPDATE 
			updated_at = NOW(),
			passing_host_count = VALUES(passing_host_count),
			failing_host_count = VALUES(failing_host_count);
    `

	countQuery := `SELECT COUNT(*) FROM policy_stats`
	var count int
	err := tx.QueryRow(countQuery).Scan(&count)
	if err != nil {
		return fmt.Errorf("counting policy_stats: %w", err)
	}

	// Only run if data doesn't already exist
	if count == 0 {
		_, err := tx.Exec(inheritedStatsQuery)
		if err != nil {
			return fmt.Errorf("inserting inherited policy stats: %w", err)
		}

		_, err = tx.Exec(globalAndTeamStatsQuery)
		if err != nil {
			return fmt.Errorf("inserting global and team policy stats: %w", err)
		}
	}

	return nil
}

func Down_20231215122713(tx *sql.Tx) error {
	return nil
}
