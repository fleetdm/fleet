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

	// Create junction table for declaration-to-fleet-variable associations
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS mdm_declaration_variables (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT,
		declaration_uuid VARCHAR(37) NOT NULL,
		fleet_variable_id INT UNSIGNED NOT NULL,
		created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		PRIMARY KEY (id),
		UNIQUE KEY idx_mdm_declaration_variables_decl_variable (declaration_uuid, fleet_variable_id),
		KEY mdm_declaration_variables_fleet_variable_id (fleet_variable_id),
		CONSTRAINT fk_mdm_declaration_variables_declaration_uuid FOREIGN KEY (declaration_uuid) REFERENCES mdm_apple_declarations (declaration_uuid) ON DELETE CASCADE,
		CONSTRAINT mdm_declaration_variables_fleet_variable_id FOREIGN KEY (fleet_variable_id) REFERENCES fleet_variables (id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("failed to create mdm_declaration_variables table: %s", err)
	}

	return nil
}

func Down_20260403000000(tx *sql.Tx) error {
	return nil
}
