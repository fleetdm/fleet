package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240601174138, Down_20240601174138)
}

func Up_20240601174138(tx *sql.Tx) error {
	stmt := `
ALTER TABLE mdm_apple_configuration_profiles MODIFY COLUMN mobileconfig MEDIUMBLOB NOT NULL;
	`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("changing data type for mdm_apple_configuration_profiles.mobileconfig: %w", err)
	}

	return nil
}

func Down_20240601174138(tx *sql.Tx) error {
	return nil
}
