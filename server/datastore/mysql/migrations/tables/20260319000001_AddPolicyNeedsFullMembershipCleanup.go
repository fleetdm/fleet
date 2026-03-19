package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20260319000001, Down_20260319000001)
}

func Up_20260319000001(tx *sql.Tx) error {
	if !columnExists(tx, "policies", "needs_full_membership_cleanup ") {
		_, err := tx.Exec(`
        ALTER TABLE policies
        ADD COLUMN needs_full_membership_cleanup TINYINT(1) NOT NULL DEFAULT 0,
        ALGORITHM=INSTANT
    `)
		return err
	}
	return nil
}

func Down_20260319000001(tx *sql.Tx) error {
	return nil
}
