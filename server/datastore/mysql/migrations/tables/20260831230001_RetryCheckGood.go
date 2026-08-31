package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260831230001, Down_20260831230001)
}

func Up_20260831230001(tx *sql.Tx) error {
	if !columnExists(tx, "users", "retry_check_good") {
		if _, err := tx.Exec(`ALTER TABLE users ADD COLUMN retry_check_good TINYINT(1) NOT NULL DEFAULT '0'`); err != nil {
			return fmt.Errorf("adding retry_check_good column to users table: %w", err)
		}
	}

	if _, err := tx.Exec(`UPDATE users SET retry_check_good = 1`); err != nil {
		return fmt.Errorf("backfilling retry_check_good: %w", err)
	}

	return nil
}

func Down_20260831230001(tx *sql.Tx) error {
	return nil
}
