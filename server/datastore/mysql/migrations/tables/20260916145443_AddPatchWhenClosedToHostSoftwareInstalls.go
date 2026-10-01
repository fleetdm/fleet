package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260916145443, Down_20260916145443)
}

// patch_when_closed snapshots the triggering policy's flag at install-activation
// time. Post-hoc reads use this snapshot rather than joining policies, since the
// policy flag can be toggled and hsi.policy_id is ON DELETE SET NULL, either of
// which would silently reclassify historical rows. Historical (completed) rows
// keep DEFAULT 0 and render as "Failed" (pre-PR behavior).
//
// One narrow backfill: in-flight rows (execution_status = 'pending_install')
// whose policy currently has patch_when_closed = 1. Fleetd reads the snapshot
// via GetSoftwareInstallDetails, so without this the first result reported by
// a pre-upgrade pending install would misclassify (fleetd sees 0, skips the
// app_open_query path, and the classifier calls it "Failed"). Scoped to
// pending rows only, so there's no historical classification to invalidate
// and no risk of relabeling an already-recorded failure.
func Up_20260916145443(tx *sql.Tx) error {
	if !columnExists(tx, "host_software_installs", "patch_when_closed") {
		// ALGORITHM=INSTANT keeps the ALTER O(1) on MySQL 8.0+. Fail fast if the
		// server can't do instant add-column rather than silently rebuilding the
		// table under a lock on a large host_software_installs.
		if _, err := tx.Exec(`
			ALTER TABLE host_software_installs
			ADD COLUMN patch_when_closed TINYINT UNSIGNED NOT NULL DEFAULT 0,
			ALGORITHM=INSTANT
		`); err != nil {
			return fmt.Errorf("adding patch_when_closed to host_software_installs: %w", err)
		}
	}

	if _, err := tx.Exec(`
		UPDATE host_software_installs
		SET patch_when_closed = 1
		WHERE execution_status = 'pending_install'
			AND policy_id IN (SELECT id FROM policies WHERE patch_when_closed = 1)
	`); err != nil {
		return fmt.Errorf("backfilling patch_when_closed on in-flight host_software_installs: %w", err)
	}
	return nil
}

func Down_20260916145443(tx *sql.Tx) error {
	return nil
}
