package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260506171058, Down_20260506171058)
}

func Up_20260506171058(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE certificate_templates
		ADD COLUMN subject_alternative_name TEXT
		CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL
	`)
	if err != nil {
		return fmt.Errorf("add subject_alternative_name column to certificate_templates: %w", err)
	}
	return nil
}

func Down_20260506171058(tx *sql.Tx) error {
	return nil
}
