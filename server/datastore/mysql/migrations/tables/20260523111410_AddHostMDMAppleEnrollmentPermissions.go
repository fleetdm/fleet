package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260523111410, Down_20260523111410)
}

func Up_20260523111410(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE host_mdm_apple_enrollment_permissions (
			host_id       INT UNSIGNED NOT NULL,
			access_rights INT          NOT NULL DEFAULT 8191,
			delivered_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (host_id),
			CONSTRAINT fk_hmaep_host
				FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("create host_mdm_apple_enrollment_permissions: %w", err)
	}

	// Backfill existing non-DEP enrolled Apple hosts. Before this feature
	// every enrollment profile was delivered with AccessRights=8191 (all
	// permissions), so that is the correct starting value for all rows that
	// existed prior to this migration.
	_, err = tx.Exec(`
		INSERT INTO host_mdm_apple_enrollment_permissions (host_id, access_rights)
		SELECT hm.host_id, 8191
		FROM host_mdm hm
		WHERE hm.enrolled = 1
		  AND hm.installed_from_dep = 0
	`)
	if err != nil {
		return fmt.Errorf("backfill host_mdm_apple_enrollment_permissions: %w", err)
	}

	return nil
}

func Down_20260523111410(tx *sql.Tx) error {
	return nil
}
