package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20260316120011, Down_20260316120011)
}

func Up_20260316120011(tx *sql.Tx) error {
	_, err := tx.Exec(`
        ALTER TABLE policies
        ADD COLUMN needs_full_membership_cleanup TINYINT(1) NOT NULL DEFAULT 0,
        ALGORITHM=INSTANT
    `)
	return err
}

func Down_20260316120011(tx *sql.Tx) error {
	return nil
}
