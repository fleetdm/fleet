package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241022140321, Down_20241022140321)
}

func Up_20241022140321(tx *sql.Tx) error {
	if !columnsExists(tx, "host_mdm", "enrollment_status", "created_at", "updated_at") {
		if _, err := tx.Exec(`
ALTER TABLE host_mdm
ADD COLUMN enrollment_status ENUM('On (manual)', 'On (automatic)', 'Pending', 'Off') COLLATE utf8mb4_unicode_ci
GENERATED ALWAYS AS (
	CASE
		WHEN is_server = 1 THEN NULL
		WHEN enrolled = 1 AND installed_from_dep = 0 THEN 'On (manual)'
		WHEN enrolled = 1 AND installed_from_dep = 1 THEN 'On (automatic)'
		WHEN enrolled = 0 AND installed_from_dep = 1 THEN 'Pending'
		WHEN enrolled = 0 AND installed_from_dep = 0 THEN 'Off'
		ELSE NULL
	END
) VIRTUAL NULL,
ADD COLUMN created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
ADD COLUMN updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
		`); err != nil {
			return fmt.Errorf("failed to alter host_mdm: %w", err)
		}
	}

	return nil
}

func Down_20241022140321(_ *sql.Tx) error {
	return nil
}
