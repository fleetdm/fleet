package tables

import (
	"database/sql"
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
		return err
	}
	return nil
}

func Down_20250718091828(tx *sql.Tx) error {
	return nil
}
