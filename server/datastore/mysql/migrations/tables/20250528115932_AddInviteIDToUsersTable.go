package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250528115932, Down_20250528115932)
}

func Up_20250528115932(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE users ADD COLUMN invite_id int unsigned")
	if err != nil {
		return fmt.Errorf("adding invite_id to users table: %w", err)
	}
	_, err = tx.Exec("ALTER TABLE users ADD UNIQUE (invite_id)")
	if err != nil {
		return fmt.Errorf("adding unique contraint to invite_id on users table: %w", err)
	}
	return nil
}

func Down_20250528115932(tx *sql.Tx) error {
	return nil
}
