package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240307162019, Down_20240307162019)
}

func Up_20240307162019(tx *sql.Tx) error {
	_, err := tx.Exec(`
	  ALTER TABLE nano_commands
	  ADD COLUMN user_id int(10) unsigned DEFAULT NULL,
	  ADD COLUMN fleet_initiated tinyint(1) DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add user_id and fleet_initiated to nano_commands: %w", err)
	}

	_, err = tx.Exec(`
	  ALTER TABLE windows_mdm_commands
	  ADD COLUMN user_id int(10) unsigned DEFAULT NULL,
	  ADD COLUMN fleet_initiated tinyint(1) DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add user_id and fleet_initiated to windows_mdm_commands: %w", err)
	}

	return nil
}

func Down_20240307162019(tx *sql.Tx) error {
	return nil
}
