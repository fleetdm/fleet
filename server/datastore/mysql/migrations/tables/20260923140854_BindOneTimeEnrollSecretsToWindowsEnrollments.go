package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260923140854, Down_20260923140854)
}

// Windows MDM mints a one-time enroll secret before a hosts row exists: the automatic enrollment flows carry no Fleet host UUID
// (authBinarySecurityToken returns an empty one), so the enrollment is inserted unlinked and only acquires host_uuid later. The
// secret therefore binds to the MDM enrollment rather than to a host, and host_id stays NULL until the agent enrolls with it.
//
// The foreign key is what invalidates a secret when the device re-enrolls: MDMWindowsDeleteEnrolledDeviceOnReenrollment deletes
// the enrollment row, and the cascade takes the secrets with it. That is the Windows equivalent of the explicit delete the Apple
// path does on re-enroll, and it cannot be forgotten by a future caller.
func Up_20260923140854(tx *sql.Tx) error {
	const table = "host_one_time_enroll_secrets"

	// Each piece is guarded separately so a run that failed partway can be retried. Adding a column that is already there, or a
	// key or constraint of a name already taken, is an error rather than a no-op in MySQL.
	if !columnExists(tx, table, "mdm_windows_enrollment_id") {
		// mdm_windows_enrollments.id is INT UNSIGNED, so the referencing column must match exactly or the FK is rejected.
		if _, err := tx.Exec(`
			ALTER TABLE host_one_time_enroll_secrets
				ADD COLUMN mdm_windows_enrollment_id INT UNSIGNED NULL AFTER host_id`); err != nil {
			return fmt.Errorf("adding mdm_windows_enrollment_id to %s: %w", table, err)
		}
	}

	if !indexExistsTx(tx, table, "idx_hotes_mdm_windows_enrollment_id") {
		if _, err := tx.Exec(`
			ALTER TABLE host_one_time_enroll_secrets
				ADD KEY idx_hotes_mdm_windows_enrollment_id (mdm_windows_enrollment_id)`); err != nil {
			return fmt.Errorf("adding idx_hotes_mdm_windows_enrollment_id to %s: %w", table, err)
		}
	}

	if !constraintExists(tx, table, "fk_hotes_mdm_windows_enrollment_id") {
		if _, err := tx.Exec(`
			ALTER TABLE host_one_time_enroll_secrets
				ADD CONSTRAINT fk_hotes_mdm_windows_enrollment_id
					FOREIGN KEY (mdm_windows_enrollment_id) REFERENCES mdm_windows_enrollments (id) ON DELETE CASCADE`); err != nil {
			return fmt.Errorf("adding fk_hotes_mdm_windows_enrollment_id to %s: %w", table, err)
		}
	}

	return nil
}

func Down_20260923140854(tx *sql.Tx) error {
	return nil
}
