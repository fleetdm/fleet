package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20250626130239, Down_20250626130239)
}

func Up_20250626130239(tx *sql.Tx) error {
	if _, err := tx.Exec(`
			ALTER TABLE scim_users
			ADD COLUMN department VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL
		`); err != nil {
		return fmt.Errorf("failed to add 'department' column to 'scim_users': %w", err)
	}

	insStmt := `INSERT INTO fleet_variables
	(name, is_prefix, created_at)
	VALUES ('FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT', 0, :created_at)`
	// use a constant time so that the generated schema is deterministic
	createdAt := time.Date(2025, 6, 27, 0, 0, 0, 0, time.UTC)
	stmt, args, err := sqlx.Named(insStmt, map[string]any{"created_at": createdAt})
	if err != nil {
		return fmt.Errorf("failed to prepare insert of FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT to fleet_variables: %s", err)
	}
	_, err = tx.Exec(stmt, args...)
	if err != nil {
		return fmt.Errorf("failed to insert FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT into fleet_variables: %s", err)
	}
	return nil
}

func Down_20250626130239(tx *sql.Tx) error {
	return nil
}
