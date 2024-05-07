package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240430111727, Down_20240430111727)
}

func Up_20240430111727(tx *sql.Tx) error {
	// This cleanup correspond to the following bug: https://github.com/fleetdm/fleet/issues/18079.
	// The following deletes "team query results" that do not match the host's team.
	_, err := tx.Exec(`
		DELETE qr
		FROM query_results qr
		JOIN queries q ON (q.id=qr.query_id)
		JOIN hosts h ON (h.id=qr.host_id)
		WHERE q.team_id IS NOT NULL AND q.team_id != COALESCE(h.team_id, 0);
	`)
	if err != nil {
		return fmt.Errorf("failed to delete query_results %w", err)
	}
	return nil
}

func Down_20240430111727(tx *sql.Tx) error {
	return nil
}
