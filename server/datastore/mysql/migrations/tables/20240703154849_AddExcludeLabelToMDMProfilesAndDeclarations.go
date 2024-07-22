package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240703154849, Down_20240703154849)
}

func Up_20240703154849(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE mdm_configuration_profile_labels ADD COLUMN exclude TINYINT(1) NOT NULL DEFAULT 0`)
	if err != nil {
		return fmt.Errorf("failed to add exclude boolean to mdm_configuration_profile_labels: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE mdm_declaration_labels ADD COLUMN exclude TINYINT(1) NOT NULL DEFAULT 0`)
	if err != nil {
		return fmt.Errorf("failed to add exclude boolean to mdm_declaration_labels: %w", err)
	}
	return nil
}

func Down_20240703154849(tx *sql.Tx) error {
	return nil
}
