package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260504193725, Down_20260504193725)
}

func Up_20260504193725(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE certificate_templates
		ADD COLUMN subject_alternative_name TEXT NULL
	`)
	if err != nil {
		return fmt.Errorf("add subject_alternative_name column to certificate_templates: %w", err)
	}
	return nil
}

func Down_20260504193725(tx *sql.Tx) error {
	return nil
}
