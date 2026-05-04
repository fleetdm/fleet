package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260409153715, Down_20260409153715)
}

func Up_20260409153715(tx *sql.Tx) error {
	// Add variables_updated_at to host_mdm_apple_declarations to track when
	// variables last changed so declaration tokens can be regenerated/updated.
	_, err := tx.Exec(`ALTER TABLE host_mdm_apple_declarations ADD COLUMN variables_updated_at DATETIME(6) DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("adding variables_updated_at to host_mdm_apple_declarations: %w", err)
	}

	// Extend mdm_configuration_profile_variables to support Apple declarations:
	// add column, unique index, foreign key, and update the check constraint
	// to ensure exactly one of the three UUID columns is non-null.
	_, err = tx.Exec(`
		ALTER TABLE mdm_configuration_profile_variables
			ADD COLUMN apple_declaration_uuid VARCHAR(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			ADD UNIQUE KEY idx_mdm_config_profile_vars_apple_decl_variable (apple_declaration_uuid, fleet_variable_id),
			ADD CONSTRAINT fk_mdm_configuration_profile_variables_apple_declaration_uuid
				FOREIGN KEY (apple_declaration_uuid) REFERENCES mdm_apple_declarations (declaration_uuid) ON DELETE CASCADE,
			DROP CHECK ck_mdm_configuration_profile_variables_apple_or_windows,
			ADD CONSTRAINT ck_mdm_configuration_profile_variables_exactly_one
				CHECK (IF(apple_profile_uuid IS NULL, 0, 1) + IF(windows_profile_uuid IS NULL, 0, 1) + IF(apple_declaration_uuid IS NULL, 0, 1) = 1)
	`)
	if err != nil {
		return fmt.Errorf("extending mdm_configuration_profile_variables for declarations: %w", err)
	}

	return nil
}

func Down_20260409153715(tx *sql.Tx) error {
	return nil
}
