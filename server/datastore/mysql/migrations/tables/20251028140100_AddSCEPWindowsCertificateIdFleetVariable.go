package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20251028140100, Down_20251028140100)
}

func Up_20251028140100(tx *sql.Tx) error {
	insStmt := `
	INSERT INTO fleet_variables (
		name, is_prefix, created_at
	) VALUES
		('FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID', 0, :created_at)
	`
	// use a constant time so that the generated schema is deterministic
	createdAt := time.Date(2025, 10, 22, 0, 0, 0, 0, time.UTC)
	stmt, args, err := sqlx.Named(insStmt, map[string]any{"created_at": createdAt})
	if err != nil {
		return fmt.Errorf("Failed to prepare insert for FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID: %s", err)
	}
	_, err = tx.Exec(stmt, args...)
	if err != nil {
		return fmt.Errorf("failed to insert FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID into fleet_variables: %s", err)
	}
	return nil
}

func Down_20251028140100(tx *sql.Tx) error {
	return nil
}
