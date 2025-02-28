package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230411102858, Down_20230411102858)
}

func Up_20230411102858(tx *sql.Tx) error {
	// create host_mdm_apple_bootstrap_packages table
	_, err := tx.Exec(`
          CREATE TABLE host_mdm_apple_bootstrap_packages (
	        host_uuid    varchar(127) NOT NULL,
            command_uuid varchar(127) NOT NULL,

            PRIMARY KEY (host_uuid),
            FOREIGN KEY (command_uuid) REFERENCES nano_commands (command_uuid) ON DELETE CASCADE
          ) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("create host_mdm_apple_bootstrap_packages: %w", err)
	}

	// add created_at and updated_at columns to mdm_apple_bootstrap_packages
	_, err = tx.Exec(`
		  ALTER TABLE mdm_apple_bootstrap_packages
		    ADD COLUMN created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
		    ADD COLUMN updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP`)
	if err != nil {
		return fmt.Errorf("add created_at and updated_at columns to mdm_apple_bootstrap_packages: %w", err)
	}

	return nil
}

func Down_20230411102858(tx *sql.Tx) error {
	return nil
}
