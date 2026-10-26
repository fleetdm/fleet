package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261026185706, Down_20261026185706)
}

func Up_20261026185706(tx *sql.Tx) error {
	if !indexExistsTx(tx, "vpp_token_teams", "idx_vpp_token_teams_team_id") {
		return nil
	}

	// One statement so the team_id foreign key always has a leading index.
	if _, err := tx.Exec(`
		ALTER TABLE vpp_token_teams
		DROP INDEX idx_vpp_token_teams_team_id,
		ADD UNIQUE KEY idx_vpp_token_teams_team_id_vpp_token_id (team_id, vpp_token_id)
	`); err != nil {
		return fmt.Errorf("replacing vpp_token_teams team_id unique key: %w", err)
	}
	return nil
}

func Down_20261026185706(tx *sql.Tx) error {
	return nil
}
