package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261005162628, Down_20261005162628)
}

func Up_20261005162628(tx *sql.Tx) error {
	// '' was never a legitimate value, but code that checks GlobalRole != nil treats it as a global role.
	if _, err := tx.Exec(`UPDATE users SET global_role = NULL, updated_at = updated_at WHERE global_role = ''`); err != nil {
		return fmt.Errorf("normalize empty global_role in users: %w", err)
	}
	if _, err := tx.Exec(`UPDATE invites SET global_role = NULL, updated_at = updated_at WHERE global_role = ''`); err != nil {
		return fmt.Errorf("normalize empty global_role in invites: %w", err)
	}
	return nil
}

func Down_20261005162628(tx *sql.Tx) error {
	return nil
}
