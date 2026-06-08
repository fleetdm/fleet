package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260608070039, Down_20260608070039)
}

// Up_20260529144132 adds the supervised column to hosts to track whether
// iOS/iPadOS devices are supervised. NULL means not applicable (non-Apple),
// 0 means unsupervised, 1 means supervised.
func Up_20260608070039(tx *sql.Tx) error {
	if columnExists(tx, "hosts", "supervised") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE hosts
		ADD COLUMN supervised TINYINT(1) DEFAULT NULL,
		ALGORITHM=INSTANT
	`); err != nil {
		return fmt.Errorf("add supervised to hosts: %w", err)
	}
	return nil
}

func Down_20260608070039(tx *sql.Tx) error {
	return nil
}
