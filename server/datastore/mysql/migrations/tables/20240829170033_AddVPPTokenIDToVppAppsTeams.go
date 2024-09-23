package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240829170033, Down_20240829170033)
}

func Up_20240829170033(tx *sql.Tx) error {
	stmtAddColumn := `
ALTER TABLE vpp_apps_teams
	ADD COLUMN vpp_token_id int(10) UNSIGNED NOT NULL`

	stmtAssociate := `UPDATE vpp_apps_teams SET vpp_token_id = (SELECT id FROM vpp_tokens LIMIT 1)`

	stmtAddConstraint := `
ALTER TABLE vpp_apps_teams
	ADD CONSTRAINT fk_vpp_apps_teams_vpp_token_id
		FOREIGN KEY (vpp_token_id) REFERENCES vpp_tokens(id) ON DELETE CASCADE`

	if _, err := tx.Exec(stmtAddColumn); err != nil {
		return fmt.Errorf("failed to add vpp_token_id column to table: %w", err)
	}

	// Associate apps with the first token available. If we're
	// migrating from single-token VPP this should be correct.
	if _, err := tx.Exec(stmtAssociate); err != nil {
		return fmt.Errorf("failed to associate vpp apps with first token: %w", err)
	}

	if _, err := tx.Exec(stmtAddConstraint); err != nil {
		return fmt.Errorf("failed to add vpp token id constraint: %w", err)
	}

	return nil
}

func Down_20240829170033(tx *sql.Tx) error {
	return nil
}
