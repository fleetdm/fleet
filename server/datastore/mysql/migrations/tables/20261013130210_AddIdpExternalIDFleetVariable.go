package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20261013130210, Down_20261013130210)
}

func Up_20261013130210(tx *sql.Tx) error {
	insStmt := `
	INSERT INTO fleet_variables (
		name, is_prefix, created_at
	) VALUES
		('FLEET_VAR_HOST_END_USER_IDP_EXTERNAL_ID', 0, :created_at)
	`
	// use a constant time so that the generated schema is deterministic
	createdAt := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	stmt, args, err := sqlx.Named(insStmt, map[string]any{"created_at": createdAt})
	if err != nil {
		return fmt.Errorf("failed to prepare insert for FLEET_VAR_HOST_END_USER_IDP_EXTERNAL_ID: %w", err)
	}
	_, err = tx.Exec(stmt, args...)
	if err != nil {
		return fmt.Errorf("failed to insert FLEET_VAR_HOST_END_USER_IDP_EXTERNAL_ID into fleet_variables: %w", err)
	}
	return nil
}

func Down_20261013130210(tx *sql.Tx) error {
	return nil
}
