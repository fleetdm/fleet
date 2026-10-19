package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261019094346, Down_20261019094346)
}

func Up_20261019094346(tx *sql.Tx) error {
	// The new key is added before the old one is dropped because the team_id foreign key needs an index that starts with team_id.
	if !indexExistsTx(tx, "vpp_token_teams", "idx_vpp_token_teams_team_id_vpp_token_id") {
		if _, err := tx.Exec(`ALTER TABLE vpp_token_teams ADD UNIQUE KEY idx_vpp_token_teams_team_id_vpp_token_id (team_id, vpp_token_id)`); err != nil {
			return fmt.Errorf("add idx_vpp_token_teams_team_id_vpp_token_id: %w", err)
		}
	}
	if indexExistsTx(tx, "vpp_token_teams", "idx_vpp_token_teams_team_id") {
		if _, err := tx.Exec(`ALTER TABLE vpp_token_teams DROP KEY idx_vpp_token_teams_team_id`); err != nil {
			return fmt.Errorf("drop idx_vpp_token_teams_team_id: %w", err)
		}
	}
	return nil
}

func Down_20261019094346(tx *sql.Tx) error {
	return nil
}
