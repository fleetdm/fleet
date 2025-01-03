package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241231112624, Down_20241231112624)
}

func Up_20241231112624(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE host_mdm_apple_declarations
    	RENAME COLUMN checksum TO token`)
	if err != nil {
		return fmt.Errorf("failed to alter mdm_apple_configuration_profiles table: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE mdm_apple_declarations
    	DROP COLUMN checksum`)
	if err != nil {
		return fmt.Errorf("failed to alter mdm_apple_configuration_profiles table: %w", err)
	}

	return nil
}

func Down_20241231112624(_ *sql.Tx) error {
	return nil
}
