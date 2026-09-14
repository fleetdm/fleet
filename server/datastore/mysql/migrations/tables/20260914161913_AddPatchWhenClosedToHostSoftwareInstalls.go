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

	// Preserve current classification of historical rows. Drive from the small
	// side (policies with patch_when_closed = 1 is a tiny set — admins configure
	// them deliberately) so the UPDATE pins to the fk_software_install_policy_id
	// index via IN() rather than gambling on the optimizer picking the right
	// JOIN drive on a large host_software_installs table. Rows without a
	// matching policy keep the column at its DEFAULT 0.
	if _, err := tx.Exec(`
		UPDATE host_software_installs hsi
		SET hsi.patch_when_closed = 1
		WHERE hsi.policy_id IN (
			SELECT id FROM policies WHERE patch_when_closed = 1
		)
	`); err != nil {
		return fmt.Errorf("backfilling patch_when_closed on host_software_installs: %w", err)
	}
	return nil
}

func Down_20260914161913(tx *sql.Tx) error {
	return nil
}
