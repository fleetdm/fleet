package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260922120749, Down_20260922120749)
}

func Up_20260922120749(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE mdm_apple_configuration_profiles
  ADD COLUMN self_service TINYINT(1) NOT NULL DEFAULT 0,
  ADD COLUMN hidden       TINYINT(1) NOT NULL DEFAULT 0;`); err != nil {
		return fmt.Errorf("failed to add self_service and hidden columns to mdm_apple_configuration_profiles: %w", err)
	}

	if _, err := tx.Exec(`ALTER TABLE mdm_apple_declarations
  ADD COLUMN hidden TINYINT(1) NOT NULL DEFAULT 0;`); err != nil {
		return fmt.Errorf("failed to add hidden column to mdm_apple_declarations: %w", err)
	}

	if _, err := tx.Exec(`ALTER TABLE mdm_windows_configuration_profiles
  ADD COLUMN hidden TINYINT(1) NOT NULL DEFAULT 0;`); err != nil {
		return fmt.Errorf("failed to add hidden column to mdm_windows_configuration_profiles: %w", err)
	}

	if _, err := tx.Exec(`ALTER TABLE mdm_android_configuration_profiles
  ADD COLUMN hidden TINYINT(1) NOT NULL DEFAULT 0;`); err != nil {
		return fmt.Errorf("failed to add hidden column to mdm_android_configuration_profiles: %w", err)
	}

	if _, err := tx.Exec(`CREATE TABLE host_mdm_profile_opt_ins (
		  host_uuid    VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
		  profile_uuid VARCHAR(37)  COLLATE utf8mb4_unicode_ci NOT NULL,
		  created_at   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		  PRIMARY KEY (host_uuid, profile_uuid),
		  KEY idx_host_mdm_profile_opt_ins_profile_uuid (profile_uuid)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`); err != nil {
		return fmt.Errorf("failed to create host_mdm_profile_opt_ins table: %w", err)
	}
	return nil
}

func Down_20260922120749(tx *sql.Tx) error {
	return nil
}
