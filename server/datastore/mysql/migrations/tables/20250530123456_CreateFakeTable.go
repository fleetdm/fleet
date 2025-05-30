package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250530123456, Down_20250530123456)
}

func Up_20250530123456(tx *sql.Tx) error {
	// create host_mdm_apple_bootstrap_packages table
	_, err := tx.Exec(`
          CREATE TABLE fake_table (
	        host_uuid    varchar(127) NOT NULL,
            command_uuid varchar(127) NOT NULL,

            PRIMARY KEY (host_uuid),
            FOREIGN KEY (command_uuid) REFERENCES nano_commands (command_uuid) ON DELETE CASCADE
          )`)
	if err != nil {
		return fmt.Errorf("create fake_table: %w", err)
	}

	return nil
}

func Down_20250530123456(tx *sql.Tx) error {
	return nil
}
