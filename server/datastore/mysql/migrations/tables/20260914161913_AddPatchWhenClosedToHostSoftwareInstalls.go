package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260914161913, Down_20260914161913)
}

// patch_when_closed snapshots the triggering policy's patch_when_closed value at
// install-activation time. Read paths that need to tell "Patch skipped" apart
// from an ordinary pre-install-query failure must use this column rather than
// joining policies, since policies.patch_when_closed can be toggled and the
// policy row itself is subject to ON DELETE SET NULL (policy_id → NULL), both
// of which would silently reclassify a historical row.
func Up_20260914161913(tx *sql.Tx) error {
	if !columnExists(tx, "host_software_installs", "patch_when_closed") {
		if _, err := tx.Exec(`
			ALTER TABLE host_software_installs
			ADD COLUMN patch_when_closed TINYINT UNSIGNED NOT NULL DEFAULT 0
		`); err != nil {
			return fmt.Errorf("adding patch_when_closed to host_software_installs: %w", err)
		}
	}

	// Preserve current classification of historical rows. Only rows with a live
	// policy_id + policies.patch_when_closed = 1 need the flag flipped on; the
	// column defaults to 0 for the rest. Uses the existing
	// fk_software_install_policy_id index on policy_id, so the join stays
	// index-driven even on large host_software_installs tables.
	if _, err := tx.Exec(`
		UPDATE host_software_installs hsi
		INNER JOIN policies p ON p.id = hsi.policy_id
		SET hsi.patch_when_closed = 1
		WHERE p.patch_when_closed = 1
	`); err != nil {
		return fmt.Errorf("backfilling patch_when_closed on host_software_installs: %w", err)
	}
	return nil
}

func Down_20260914161913(tx *sql.Tx) error {
	return nil
}
