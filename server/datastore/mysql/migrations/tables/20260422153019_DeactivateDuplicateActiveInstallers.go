package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260422153019, Down_20260422153019)
}

func Up_20260422153019(tx *sql.Tx) error {
	// Find (global_or_team_id, title_id) groups with more than one active
	// non-FMA installer. This can happen when a user uploads a new version of
	// the same software via the UI — the old unique index
	// (global_or_team_id, title_id) was replaced with
	// (global_or_team_id, title_id, version), allowing multiple versions, but
	// the upload path did not deactivate the old installer.
	_, err := tx.Exec(`
		CREATE TEMPORARY TABLE tmp_active_installers AS
		SELECT MAX(id) AS keep_id, global_or_team_id, title_id
		FROM software_installers
		WHERE is_active = 1 AND fleet_maintained_app_id IS NULL
		GROUP BY global_or_team_id, title_id
		HAVING COUNT(*) > 1`)
	if err != nil {
		return fmt.Errorf("creating temp table for duplicate active installers: %w", err)
	}

	// Re-point policies from the old (soon-to-be-deactivated) installers to the newest one.
	_, err = tx.Exec(`
		UPDATE policies p
		JOIN software_installers si ON si.id = p.software_installer_id
		JOIN tmp_active_installers tmp
			ON tmp.global_or_team_id = si.global_or_team_id
			AND tmp.title_id = si.title_id
			AND si.id != tmp.keep_id
		SET p.software_installer_id = tmp.keep_id`)
	if err != nil {
		return fmt.Errorf("re-pointing policies to newest installer: %w", err)
	}

	// Deactivate the older duplicate installers, keeping only the newest.
	_, err = tx.Exec(`
		UPDATE software_installers si
		JOIN tmp_active_installers tmp
			ON tmp.global_or_team_id = si.global_or_team_id
			AND tmp.title_id = si.title_id
			AND si.id != tmp.keep_id
		SET si.is_active = 0`)
	if err != nil {
		return fmt.Errorf("deactivating duplicate active installers: %w", err)
	}

	_, err = tx.Exec(`DROP TEMPORARY TABLE tmp_active_installers`)
	if err != nil {
		return fmt.Errorf("dropping temp table: %w", err)
	}

	return nil
}

func Down_20260422153019(tx *sql.Tx) error {
	return nil
}
