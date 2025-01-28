package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240905105135, Down_20240905105135)
}

func Up_20240905105135(tx *sql.Tx) error {
	// The AUTO_INCREMENT columns are used to determine if a row was updated by an INSERT ... ON DUPLICATE KEY UPDATE statement.
	// This is needed because we are currently using CLIENT_FOUND_ROWS option to determine if a row was found.
	// And in order to find if the row was updated, we need to check LAST_INSERT_ID().
	// MySQL docs: https://dev.mysql.com/doc/refman/8.4/en/insert-on-duplicate.html

	if !columnExists(tx, "mdm_windows_configuration_profiles", "auto_increment") {
		if _, err := tx.Exec(`
ALTER TABLE mdm_windows_configuration_profiles
ADD COLUMN auto_increment BIGINT NOT NULL AUTO_INCREMENT UNIQUE
`); err != nil {
			return fmt.Errorf("failed to add auto_increment to mdm_windows_configuration_profiles: %w", err)
		}
	}

	if !columnExists(tx, "mdm_apple_declarations", "auto_increment") {
		if _, err := tx.Exec(`
ALTER TABLE mdm_apple_declarations
ADD COLUMN auto_increment BIGINT NOT NULL AUTO_INCREMENT UNIQUE
`); err != nil {
			return fmt.Errorf("failed to add auto_increment to mdm_apple_declarations: %w", err)
		}
	}
	return nil
}

func Down_20240905105135(tx *sql.Tx) error {
	return nil
}
