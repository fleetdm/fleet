package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250527110207, Down_20250527110207)
}

func Up_20250527110207(tx *sql.Tx) error {
	// Rename the users table to users_all
	_, err := tx.Exec(`
		ALTER TABLE users
		RENAME TO users_all
	`)
	if err != nil {
		return fmt.Errorf("failed to rename 'users' table to 'users_all': %w", err)
	}
	// Adding soft delete columns to the users table
	_, err = tx.Exec(`
		ALTER TABLE users_all
		ADD COLUMN deleted_at DATETIME NULL DEFAULT NULL,
		ADD COLUMN deleted_by_user_id INT(10) UNSIGNED NULL DEFAULT NULL
		`)
	if err != nil {
		return fmt.Errorf("failed to add columns 'deleted_at' and 'deleted_by_user_id': %w", err)
	}
	// Create a users view that only shows non-deleted users
	_, err = tx.Exec(`
		CREATE VIEW users AS
		SELECT *
		FROM users_all
		WHERE deleted_at IS NULL
	`)
	return nil
}

func Down_20250527110207(tx *sql.Tx) error {
	return nil
}
