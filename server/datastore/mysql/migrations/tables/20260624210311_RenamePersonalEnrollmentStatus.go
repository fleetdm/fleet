package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260624210311, Down_20260624210311)
}

// Up_20260622163928 renames the enrollment_status VIRTUAL GENERATED column
// value 'On (personal)' to 'On (manual - personal)' to align with the API
// documentation for manual (profile-driven) BYOD enrollment.
//
// Because enrollment_status is a VIRTUAL column (not stored), MySQL recomputes
// its value at read time; there is no stored data to migrate.
func Up_20260624210311(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE host_mdm
		CHANGE COLUMN enrollment_status enrollment_status
			ENUM('On (manual)', 'On (automatic)', 'Pending', 'Off', 'On (manual - personal)')
			CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
			GENERATED ALWAYS AS (
				CASE
					WHEN is_server = 1 THEN NULL
					WHEN enrolled = 1 AND installed_from_dep = 0 AND is_personal_enrollment = 1 THEN 'On (manual - personal)'
					WHEN enrolled = 1 AND installed_from_dep = 0 AND is_personal_enrollment = 0 THEN 'On (manual)'
					WHEN enrolled = 1 AND installed_from_dep = 1 AND is_personal_enrollment = 0 THEN 'On (automatic)'
					WHEN enrolled = 0 AND installed_from_dep = 1 THEN 'Pending'
					WHEN enrolled = 0 AND installed_from_dep = 0 THEN 'Off'
					ELSE NULL
				END
			) VIRTUAL NULL
	`); err != nil {
		return fmt.Errorf("rename enrollment_status personal value: %w", err)
	}
	return nil
}

func Down_20260624210311(tx *sql.Tx) error {
	return nil
}
