package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20250808000000, Down_20250808000000)
}

func Up_20250808000000(tx *sql.Tx) error {
	insStmt := `
	INSERT INTO fleet_variables (
		name, is_prefix, created_at
	) VALUES
		('FLEET_VAR_HOST_UUID', 0, :created_at)
	`
	// use a constant time so that the generated schema is deterministic
	createdAt := time.Date(2025, 8, 8, 0, 0, 0, 0, time.UTC)
	stmt, args, err := sqlx.Named(insStmt, map[string]any{"created_at": createdAt})
	if err != nil {
		return fmt.Errorf("Failed to prepare insert for FLEET_VAR_HOST_UUID: %w", err)
	}
	_, err = tx.Exec(stmt, args...)
	if err != nil {
		return fmt.Errorf("Failed to insert FLEET_VAR_HOST_UUID into fleet_variables: %w", err)
	}

	return nil
}

func Down_20250808000000(_ *sql.Tx) error {
	return nil
}
