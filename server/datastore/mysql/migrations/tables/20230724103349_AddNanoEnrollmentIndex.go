package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230724103349, Down_20230724103349)
}

func Up_20230724103349(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE nano_enrollment_queue ADD INDEX (priority DESC, created_at);`)

	return err
}

func Down_20230724103349(tx *sql.Tx) error {
	return nil
}
