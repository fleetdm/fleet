package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260811134710, Down_20260811134710)
}

// Apple TVs synced from ABM get their own default team, mirroring the existing
// macOS/iOS/iPadOS columns. NULL means "No team", which is the default: before
// this column existed Apple TVs fell through to the macOS default team.
func Up_20260811134710(tx *sql.Tx) error {
	if columnExists(tx, "abm_tokens", "tvos_default_team_id") {
		return nil
	}

	_, err := tx.Exec(`ALTER TABLE abm_tokens
		ADD COLUMN tvos_default_team_id INT UNSIGNED NULL,
		ADD CONSTRAINT fk_abm_tokens_tvos_default_team_id
			FOREIGN KEY (tvos_default_team_id) REFERENCES teams(id) ON DELETE SET NULL`)
	if err != nil {
		return fmt.Errorf("adding tvos_default_team_id to abm_tokens: %w", err)
	}

	return nil
}

func Down_20260811134710(_ *sql.Tx) error {
	return nil
}
