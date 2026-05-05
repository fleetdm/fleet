package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260126150840, Down_20260126150840)
}

func Up_20260126150840(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE mdm_windows_enrollments
	ADD COLUMN credentials_hash BINARY(16),
	ADD COLUMN credentials_acknowledged BOOLEAN NOT NULL DEFAULT FALSE;`)
	return err
}

func Down_20260126150840(tx *sql.Tx) error {
	return nil
}
