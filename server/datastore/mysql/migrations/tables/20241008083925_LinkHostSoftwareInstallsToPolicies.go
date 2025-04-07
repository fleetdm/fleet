package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241008083925, Down_20241008083925)
}

func Up_20241008083925(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE host_software_installs
		ADD COLUMN policy_id INT UNSIGNED DEFAULT NULL,
		ADD FOREIGN KEY fk_software_install_policy_id (policy_id) REFERENCES policies (id) ON DELETE SET NULL
	`); err != nil {
		return fmt.Errorf("failed to add policy_id to host software installs: %w", err)
	}

	return nil
}

func Down_20241008083925(tx *sql.Tx) error {
	return nil
}
