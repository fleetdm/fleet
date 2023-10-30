package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230807100822, Down_20230807100822)
}

// This migration tracks the nanomdm repo. See https://github.com/micromdm/nanomdm/blob/main/storage/mysql/schema.00009.sql
func Up_20230807100822(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE nano_enrollment_queue ADD INDEX (priority DESC, created_at);`)

	return err
}

func Down_20230807100822(tx *sql.Tx) error {
	return nil
}
