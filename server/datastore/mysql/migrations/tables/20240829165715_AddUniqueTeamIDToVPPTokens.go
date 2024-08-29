package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240829165715, Down_20240829165715)
}

func Up_20240829165715(tx *sql.Tx) error {
	stmt := `ALTER TABLE vpp_tokens ADD UNIQUE KEY idx_vpp_tokens_team_id (team_id)`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("adding unique constraint to team_id on vpp_tokens: %w", err)
	}
	return nil
}

func Down_20240829165715(tx *sql.Tx) error {
	return nil
}
