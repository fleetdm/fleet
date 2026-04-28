package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260403120000, Down_20260403120000)
}

func Up_20260403120000(tx *sql.Tx) error {
	if columnExists(tx, "policies", "needs_full_membership_cleanup") {
		return nil
	}
	_, err := tx.Exec(`
        ALTER TABLE policies
        ADD COLUMN needs_full_membership_cleanup TINYINT(1) NOT NULL DEFAULT 0,
        ALGORITHM=INSTANT
    `)
	if err != nil {
		return fmt.Errorf("failed to add needs_full_membership_cleanup column to policies table: %w", err)
	}
	return nil
}

func Down_20260403120000(tx *sql.Tx) error {
	return nil
}
