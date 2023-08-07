package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230807100822, Down_20230807100822)
}

func Up_20230807100822(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE nano_enrollment_queue ADD INDEX (priority DESC, created_at);`)

	return err
}

func Down_20230807100822(tx *sql.Tx) error {
	return nil
}
