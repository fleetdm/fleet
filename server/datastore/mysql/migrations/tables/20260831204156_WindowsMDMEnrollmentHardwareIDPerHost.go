package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260831204156, Down_20260831204156)
}

func Up_20260831204156(tx *sql.Tx) error {
	// The old UNIQUE KEY on mdm_hardware_id alone assumed one HWDevID identifies one device forever. HWDevID is derived from OS state in
	// the Windows image, so machines sharing an image lineage that was never generalized with sysprep report the same value. Under the old
	// key, the second such machine to enroll took over the first one's enrollment row and silently stopped Fleet from managing it. Keying
	// on (mdm_hardware_id, host_uuid) lets both hosts hold their own enrollment. Existing rows need no dedup pass: uniqueness on
	// (mdm_hardware_id) already implies uniqueness on the pair.
	if indexExistsTx(tx, "mdm_windows_enrollments", "idx_type") {
		if _, err := tx.Exec(`ALTER TABLE mdm_windows_enrollments DROP INDEX idx_type`); err != nil {
			return fmt.Errorf("dropping idx_type from mdm_windows_enrollments: %w", err)
		}
	}

	// The pair's leftmost prefix indexes mdm_hardware_id on its own, so the collision lookup needs no separate index.
	if !indexExistsTx(tx, "mdm_windows_enrollments", "idx_mdm_windows_enrollments_hardware_id_host_uuid") {
		if _, err := tx.Exec(`
			ALTER TABLE mdm_windows_enrollments
				ADD UNIQUE KEY idx_mdm_windows_enrollments_hardware_id_host_uuid (mdm_hardware_id, host_uuid)`); err != nil {
			return fmt.Errorf("adding idx_mdm_windows_enrollments_hardware_id_host_uuid to mdm_windows_enrollments: %w", err)
		}
	}

	return nil
}

func Down_20260831204156(tx *sql.Tx) error {
	return nil
}
