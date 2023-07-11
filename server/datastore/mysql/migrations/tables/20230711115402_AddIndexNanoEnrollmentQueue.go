package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230711115402, Down_20230711115402)
}

func Up_20230711115402(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE nano_enrollment_queue ADD INDEX (priority DESC, created_at);`)

	return err
}

func Down_20230711115402(tx *sql.Tx) error {
	return nil
}
