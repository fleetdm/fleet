package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251020175220, Down_20251020175220)
}

func Up_20251020175220(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE vpp_apps_teams DROP FOREIGN KEY fk_vpp_apps_teams_vpp_token_id`)
	if err != nil {
		return fmt.Errorf("failed to drop fk from vpp_apps_table: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE vpp_apps_teams MODIFY vpp_token_id INT UNSIGNED`)
	if err != nil {
		return fmt.Errorf("failed to make vpp_apps_table.vpp_token_id nullable: %w", err)
	}

	return nil
}

func Down_20251020175220(tx *sql.Tx) error {
	return nil
}
