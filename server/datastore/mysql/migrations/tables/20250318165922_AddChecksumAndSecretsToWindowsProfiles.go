package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250318165922, Down_20250318165922)
}

func Up_20250318165922(tx *sql.Tx) error {
	if columnsExists(tx, "mdm_windows_configuration_profiles", "checksum", "secrets_updated_at") && columnsExists(tx, "host_mdm_windows_profiles",
		"checksum", "secrets_updated_at", "created_at", "updated_at") {
		return nil
	}
	_, err := tx.Exec(
		`ALTER TABLE mdm_windows_configuration_profiles
			ADD COLUMN checksum BINARY(16) AS (UNHEX(MD5(syncml))) STORED,
			ADD COLUMN secrets_updated_at DATETIME(6) NULL,
			MODIFY COLUMN created_at TIMESTAMP(6) NOT NULL DEFAULT NOW(6),
			MODIFY COLUMN uploaded_at TIMESTAMP(6) NULL DEFAULT NULL;

		ALTER TABLE host_mdm_windows_profiles
			ADD COLUMN checksum BINARY(16) NOT NULL DEFAULT 0,
			ADD COLUMN secrets_updated_at DATETIME(6) NULL,
			ADD COLUMN created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
			ADD COLUMN updated_at DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6);

		UPDATE host_mdm_windows_profiles hmwp
			SET checksum = COALESCE((SELECT checksum FROM mdm_windows_configuration_profiles mwcp WHERE mwcp.profile_uuid = hmwp.profile_uuid),0);`)
	if err != nil {
		return fmt.Errorf("error adding checksum column to mdm_windows_configuration_profiles and host_mdm_windows_profiles: %w", err)
	}
	return nil
}

func Down_20250318165922(_ *sql.Tx) error {
	return nil
}
