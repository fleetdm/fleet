package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260403000000, Down_20260403000000)
}

func Up_20260403000000(tx *sql.Tx) error {
	// Add variables_updated_at column to host_mdm_apple_declarations
	if !columnExists(tx, "host_mdm_apple_declarations", "variables_updated_at") {
		_, err := tx.Exec(`
		ALTER TABLE host_mdm_apple_declarations
		ADD COLUMN variables_updated_at DATETIME(6) NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to add variables_updated_at column to host_mdm_apple_declarations table: %s", err)
		}
	}

	// Add apple_declaration_uuid column to mdm_configuration_profile_variables, mirroring
	// the existing apple_profile_uuid and windows_profile_uuid columns.
	if !columnExists(tx, "mdm_configuration_profile_variables", "apple_declaration_uuid") {
		_, err := tx.Exec(`
		ALTER TABLE mdm_configuration_profile_variables
			ADD COLUMN apple_declaration_uuid VARCHAR(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			ADD UNIQUE KEY idx_mdm_configuration_profile_variables_declaration_variable (apple_declaration_uuid, fleet_variable_id),
			ADD CONSTRAINT fk_mdm_configuration_profile_variables_apple_declaration_uuid
				FOREIGN KEY (apple_declaration_uuid) REFERENCES mdm_apple_declarations (apple_declaration_uuid) ON DELETE CASCADE,
			DROP CHECK ck_mdm_configuration_profile_variables_apple_or_windows,
			ADD CONSTRAINT ck_mdm_configuration_profile_variables_exactly_one CHECK (
				(IF(apple_profile_uuid IS NULL, 0, 1) + IF(windows_profile_uuid IS NULL, 0, 1) + IF(apple_declaration_uuid IS NULL, 0, 1)) = 1
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to add apple_declaration_uuid column to mdm_configuration_profile_variables table: %s", err)
		}
	}

	return nil
}

func Down_20260403000000(tx *sql.Tx) error {
	return nil
}
