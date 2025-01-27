package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250121094700, Down_20250121094700)
}

func Up_20250121094700(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE policies
		ADD COLUMN vpp_apps_teams_id INT UNSIGNED DEFAULT NULL,
		ADD FOREIGN KEY fk_policies_vpp_apps_team_id (vpp_apps_teams_id) REFERENCES vpp_apps_teams (id);
	`); err != nil {
		return fmt.Errorf("failed to add vpp_apps_teams_id to policies: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE host_vpp_software_installs
		ADD COLUMN policy_id INT UNSIGNED DEFAULT NULL,
		ADD FOREIGN KEY fk_host_vpp_software_installs_policy_id (policy_id) REFERENCES policies (id) ON DELETE SET NULL
	`); err != nil {
		return fmt.Errorf("failed to add policy_id to host VPP software installs: %w", err)
	}

	return nil
}

func Down_20250121094700(tx *sql.Tx) error {
	return nil
}
