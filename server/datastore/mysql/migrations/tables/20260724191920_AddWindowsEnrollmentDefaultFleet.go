package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260724191920, Down_20260724191920)
}

func Up_20260724191920(tx *sql.Tx) error {
	// Single-row config table holding the default team (fleet) that new user-driven Windows MDM
	// enrollments are assigned to. team_id is NULL when no default is configured; the row is
	// deleted or nulled when the referenced team is deleted (ON DELETE SET NULL).
	_, err := tx.Exec(`
		CREATE TABLE windows_enrollment_config (
			id INT UNSIGNED NOT NULL PRIMARY KEY,
			team_id INT UNSIGNED DEFAULT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			CONSTRAINT fk_windows_enrollment_config_team_id
				FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE SET NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("create windows_enrollment_config: %w", err)
	}

	// hardware_serial is the SMBIOS serial the device reports over OMA-DM (DevDetail). It is
	// persisted while the enrollment is still unlinked (host_uuid = '') so the orbit enrollment
	// path can reverse-link the enrollment to the host it just created. NULL until the device
	// answers the DevDetail query; placeholder serials are never stored.
	_, err = tx.Exec(`
		ALTER TABLE mdm_windows_enrollments
		ADD COLUMN hardware_serial VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
		ADD INDEX idx_mdm_windows_enrollments_hardware_serial (hardware_serial)
	`)
	if err != nil {
		return fmt.Errorf("add hardware_serial to mdm_windows_enrollments: %w", err)
	}

	return nil
}

func Down_20260724191920(tx *sql.Tx) error {
	return nil
}
