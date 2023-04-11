package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230411102858, Down_20230411102858)
}

func Up_20230411102858(tx *sql.Tx) error {
	_, err := tx.Exec(`
          CREATE TABLE host_mdm_apple_bootstrap_packages (
	    host_uuid    varchar(127) NOT NULL,
            command_uuid varchar(127) NOT NULL,

            PRIMARY KEY (host_uuid, command_uuid),
            FOREIGN KEY (command_uuid) REFERENCES nano_commands (command_uuid) ON DELETE CASCADE
          )`)

	if err != nil {
		return fmt.Errorf("create host_mdm_apple_bootstrap_packages: %w", err)
	}

	return nil
}

func Down_20230411102858(tx *sql.Tx) error {
	return nil
}
