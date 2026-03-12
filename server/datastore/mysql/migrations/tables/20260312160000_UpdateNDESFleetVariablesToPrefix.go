package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20260312160000, Down_20260312160000)
}

func Up_20260312160000(tx *sql.Tx) error {
	// Update the existing NDES SCEP fleet variables from exact match (is_prefix=0)
	// to prefix-based (is_prefix=1) to support multiple NDES SCEP proxy CAs.
	// The variable names now end with an underscore to match the <CA_NAME> suffix pattern.

	// Delete the old exact-match variables
	_, err := tx.Exec(`DELETE FROM fleet_variables WHERE name IN ('FLEET_VAR_NDES_SCEP_CHALLENGE', 'FLEET_VAR_NDES_SCEP_PROXY_URL')`)
	if err != nil {
		return fmt.Errorf("failed to delete old NDES fleet variables: %s", err)
	}

	// Insert the new prefix-based variables
	insStmt := `
	INSERT INTO fleet_variables (
		name, is_prefix, created_at
	) VALUES
		('FLEET_VAR_NDES_SCEP_CHALLENGE_', 1, :created_at),
		('FLEET_VAR_NDES_SCEP_PROXY_URL_', 1, :created_at)
	`
	createdAt := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	stmt, args, err := sqlx.Named(insStmt, map[string]any{"created_at": createdAt})
	if err != nil {
		return fmt.Errorf("failed to prepare insert for NDES fleet_variables: %s", err)
	}
	_, err = tx.Exec(stmt, args...)
	if err != nil {
		return fmt.Errorf("failed to insert NDES fleet_variables: %s", err)
	}

	return nil
}

func Down_20260312160000(tx *sql.Tx) error {
	return nil
}
