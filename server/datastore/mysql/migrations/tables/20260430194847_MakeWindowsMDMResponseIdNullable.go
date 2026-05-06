package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260430194847, Down_20260430194847)
}

func Up_20260430194847(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE windows_mdm_command_results DROP FOREIGN KEY windows_mdm_command_results_ibfk_3`); err != nil {
		return fmt.Errorf("dropping windows_mdm_command_results FK on response_id: %w", err)
	}

	if _, err := tx.Exec(`ALTER TABLE windows_mdm_command_results MODIFY COLUMN response_id int unsigned NULL`); err != nil {
		return fmt.Errorf("making response_id nullable: %w", err)
	}

	return nil
}

func Down_20260430194847(_ *sql.Tx) error {
	return nil
}
