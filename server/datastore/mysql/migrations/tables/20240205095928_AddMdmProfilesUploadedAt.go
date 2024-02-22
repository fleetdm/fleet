package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240205095928, Down_20240205095928)
}

func Up_20240205095928(tx *sql.Tx) error {
	// in theory it could be made non-null, as there were default clauses for the
	// previous column definition, but this is too risky given that it could've
	// been set (either by mistake or manually) to NULL and the migration would
	// fail.
	_, err := tx.Exec(`
ALTER TABLE mdm_windows_configuration_profiles
	CHANGE COLUMN updated_at uploaded_at TIMESTAMP NULL
`)
	if err != nil {
		return fmt.Errorf("failed to alter mdm_windows_configuration_profiles table: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE mdm_apple_configuration_profiles
	CHANGE COLUMN updated_at uploaded_at TIMESTAMP NULL
`)
	if err != nil {
		return fmt.Errorf("failed to alter mdm_apple_configuration_profiles table: %w", err)
	}
	return nil
}

func Down_20240205095928(tx *sql.Tx) error {
	return nil
}
