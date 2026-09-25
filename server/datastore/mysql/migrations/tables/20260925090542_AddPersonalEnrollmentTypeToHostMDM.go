package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260925090542, Down_20260925090542)
}

func Up_20260925090542(tx *sql.Tx) error {
	if !columnExists(tx, "host_mdm", "personal_enrollment_type") {
		if _, err := tx.Exec(`
ALTER TABLE host_mdm
	ADD COLUMN personal_enrollment_type ENUM('account_driven', 'work_profile', 'manual_profile')
		CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL
		AFTER is_personal_enrollment`); err != nil {
			return fmt.Errorf("add personal_enrollment_type column to host_mdm: %w", err)
		}
	}

	backfill := incrementalMigrationStep(
		func(tx *sql.Tx) (uint64, error) {
			var total uint64
			err := tx.QueryRow(`SELECT COUNT(*) FROM host_mdm WHERE is_personal_enrollment = 1 AND personal_enrollment_type IS NULL`).Scan(&total)
			return total, err
		},
		backfillPersonalEnrollmentType,
	)
	if err := backfill(tx); err != nil {
		return fmt.Errorf("backfill personal_enrollment_type: %w", err)
	}

	if _, err := tx.Exec(`
ALTER TABLE host_mdm
	CHANGE COLUMN enrollment_status enrollment_status
		ENUM('On (manual)', 'On (automatic)', 'Pending', 'Off', 'On (manual - personal)', 'On (personal)')
		CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
		GENERATED ALWAYS AS (
			CASE
				WHEN is_server = 1 THEN NULL
				WHEN enrolled = 1 AND installed_from_dep = 0 AND is_personal_enrollment = 1
					AND personal_enrollment_type = 'manual_profile' THEN 'On (manual - personal)'
				WHEN enrolled = 1 AND installed_from_dep = 0 AND is_personal_enrollment = 1 THEN 'On (personal)'
				WHEN enrolled = 1 AND installed_from_dep = 0 AND is_personal_enrollment = 0 THEN 'On (manual)'
				WHEN enrolled = 1 AND installed_from_dep = 1 AND is_personal_enrollment = 0 THEN 'On (automatic)'
				WHEN enrolled = 0 AND installed_from_dep = 1 THEN 'Pending'
				WHEN enrolled = 0 AND installed_from_dep = 0 THEN 'Off'
				ELSE NULL
			END
		) VIRTUAL NULL`); err != nil {
		return fmt.Errorf("update enrollment_status on host_mdm: %w", err)
	}

	if err := addIndexesTx(tx, "host_mdm", indexDef{
		name:    "host_mdm_enrolled_dep_personal_type_idx",
		columns: "enrolled, installed_from_dep, is_personal_enrollment, personal_enrollment_type",
	}); err != nil {
		return err
	}
	if indexExistsTx(tx, "host_mdm", "host_mdm_enrolled_installed_from_dep_is_personal_enrollment_idx") {
		if _, err := tx.Exec(`ALTER TABLE host_mdm DROP INDEX host_mdm_enrolled_installed_from_dep_is_personal_enrollment_idx, ALGORITHM=INPLACE, LOCK=NONE`); err != nil {
			return fmt.Errorf("drop host_mdm enrollment index: %w", err)
		}
	}

	return nil
}

var personalEnrollmentTypeBackfillBatchSize = 5000

func backfillPersonalEnrollmentType(tx *sql.Tx, increment incrementCountFn) error {
	var maxHostID sql.NullInt64
	if err := tx.QueryRow(`SELECT MAX(host_id) FROM host_mdm`).Scan(&maxHostID); err != nil {
		return fmt.Errorf("selecting max host_mdm host_id: %w", err)
	}

	batchSize := int64(personalEnrollmentTypeBackfillBatchSize)
	for start := int64(0); start < maxHostID.Int64; start += batchSize {
		// Unenrolling disables nano enrollments rather than deleting them, so
		// the join must not filter on ne.enabled.
		res, err := tx.Exec(`
UPDATE host_mdm hm
JOIN hosts h ON h.id = hm.host_id
LEFT JOIN nano_enrollments ne ON ne.id = h.uuid AND ne.type = 'User Enrollment (Device)'
SET hm.personal_enrollment_type = CASE
	WHEN h.platform = 'android' THEN 'work_profile'
	WHEN ne.id IS NOT NULL THEN 'account_driven'
	ELSE 'manual_profile'
END
WHERE hm.is_personal_enrollment = 1
	AND hm.personal_enrollment_type IS NULL
	AND hm.host_id > ? AND hm.host_id <= ?`, start, start+batchSize)
		if err != nil {
			return fmt.Errorf("backfilling personal_enrollment_type for host_id range (%d, %d]: %w", start, start+batchSize, err)
		}
		affected, _ := res.RowsAffected()
		for range affected {
			increment()
		}
	}
	return nil
}

func Down_20260925090542(tx *sql.Tx) error {
	return nil
}
