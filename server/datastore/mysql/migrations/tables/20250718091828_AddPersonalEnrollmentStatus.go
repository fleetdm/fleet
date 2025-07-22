package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20250718091828, Down_20250718091828)
}

func Up_20250718091828(tx *sql.Tx) error {
	if _, err := tx.Exec(`
ALTER TABLE host_mdm
	ADD COLUMN is_personal_enrollment TINYINT(1) NOT NULL DEFAULT '0',
	CHANGE COLUMN enrollment_status enrollment_status ENUM('On (manual)', 'On (automatic)', 'Pending', 'Off', 'On (personal)') COLLATE utf8mb4_unicode_ci
GENERATED ALWAYS AS (
	CASE
		WHEN is_server = 1 THEN NULL
		WHEN enrolled = 1 AND installed_from_dep = 0 AND is_personal_enrollment = 1 THEN 'On (personal)'
		WHEN enrolled = 1 AND installed_from_dep = 0 AND is_personal_enrollment = 0 THEN 'On (manual)'
		WHEN enrolled = 1 AND installed_from_dep = 1 AND is_personal_enrollment = 0 THEN 'On (automatic)'
		WHEN enrolled = 0 AND installed_from_dep = 1 THEN 'Pending'
		WHEN enrolled = 0 AND installed_from_dep = 0 THEN 'Off'
		ELSE NULL
	END
) VIRTUAL NULL`); err != nil {
		return errors.Wrap(err, "add is_personal_enrollment column and modify enrollment_status on host_mdm table")
	}

	// Remove the old index and create a new one that includes is_personal_enrollment.
	if _, err := tx.Exec(`ALTER TABLE host_mdm DROP INDEX host_mdm_enrolled_installed_from_dep_idx, ADD INDEX host_mdm_enrolled_installed_from_dep_is_personal_enrollment_idx (enrolled, installed_from_dep, is_personal_enrollment);`); err != nil {
		return errors.Wrap(err, "create enrollment status index")
	}
	return nil
}

func Down_20250718091828(tx *sql.Tx) error {
	return nil
}
