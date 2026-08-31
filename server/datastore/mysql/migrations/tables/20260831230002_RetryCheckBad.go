package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260831230002, Down_20260831230002)
}

func Up_20260831230002(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE users ADD COLUMN retry_check_bad TINYINT(1) NOT NULL DEFAULT '0'`); err != nil {
		return fmt.Errorf("adding retry_check_bad column to users table: %w", err)
	}

	if _, err := tx.Exec(`UPDATE users SET retry_check_bad = 1`); err != nil {
		return fmt.Errorf("backfilling retry_check_bad: %w", err)
	}

	return nil
}

func Down_20260831230002(tx *sql.Tx) error {
	return nil
}
