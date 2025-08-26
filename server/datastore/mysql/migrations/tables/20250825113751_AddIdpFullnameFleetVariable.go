package tables

import (
	"database/sql"
	"fmt"
	"time"
)

func init() {
	MigrationClient.AddMigration(Up_20250825113751, Down_20250825113751)
}

func Up_20250825113751(tx *sql.Tx) error {
	// use a constant time so that the generated schema is deterministic
	createdAt := time.Date(2025, 8, 25, 0, 0, 0, 0, time.UTC)
	if _, err := tx.Exec("INSERT INTO fleet_variables (name, is_prefix, created_at) VALUES ('FLEET_VAR_HOST_END_USER_IDP_FULL_NAME', 0, ?)", createdAt); err != nil {
		return fmt.Errorf("inserting fullname idp fleet variable: %w", err)
	}
	return nil
}

func Down_20250825113751(tx *sql.Tx) error {
	return nil
}
