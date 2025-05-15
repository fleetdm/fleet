package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241110152841, Down_20241110152841)
}

func Up_20241110152841(tx *sql.Tx) error {
	// Add columns
	_, err := tx.Exec(`ALTER TABLE mdm_configuration_profile_labels ADD COLUMN require_all BOOL NOT NULL DEFAULT false`)
	if err != nil {
		return fmt.Errorf("failed to add require_all to mdm_configuration_profile_labels: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE mdm_declaration_labels ADD COLUMN require_all BOOL NOT NULL DEFAULT false`)
	if err != nil {
		return fmt.Errorf("failed to add require_all to mdm_declaration_labels: %w", err)
	}

	// Set require_all to true if exclude was false (this means that it represents an "include all"
	// label filter
	_, err = tx.Exec(`UPDATE mdm_configuration_profile_labels SET require_all = true WHERE exclude = false`)
	if err != nil {
		return fmt.Errorf("failed to migrate include all records in mdm_configuration_profile_labels: %w", err)
	}

	_, err = tx.Exec(`UPDATE mdm_declaration_labels SET require_all = true WHERE exclude = false`)
	if err != nil {
		return fmt.Errorf("failed to migrate include all records in mdm_declaration_labels: %w", err)
	}

	return nil
}

func Down_20241110152841(tx *sql.Tx) error {
	return nil
}
