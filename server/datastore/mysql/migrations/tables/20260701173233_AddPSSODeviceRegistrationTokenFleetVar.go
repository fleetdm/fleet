package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20260701173233, Down_20260701173233)
}

func Up_20260701173233(tx *sql.Tx) error {
	insStmt := `
	INSERT INTO fleet_variables (
		name, is_prefix, created_at
	) VALUES
		('FLEET_VAR_PSSO_DEVICE_REGISTRATION_TOKEN', 0, :created_at)
	`
	// use a constant time so that the generated schema is deterministic
	createdAt := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	stmt, args, err := sqlx.Named(insStmt, map[string]any{"created_at": createdAt})
	if err != nil {
		return fmt.Errorf("failed to prepare insert for FLEET_VAR_PSSO_DEVICE_REGISTRATION_TOKEN: %w", err)
	}
	_, err = tx.Exec(stmt, args...)
	if err != nil {
		return fmt.Errorf("failed to insert FLEET_VAR_PSSO_DEVICE_REGISTRATION_TOKEN into fleet_variables: %w", err)
	}
	return nil
}

func Down_20260701173233(tx *sql.Tx) error {
	return nil
}
