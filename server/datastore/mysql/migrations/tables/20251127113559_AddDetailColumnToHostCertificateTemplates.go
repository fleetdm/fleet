package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251127113559, Down_20251127113559)
}

func Up_20251127113559(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE host_certificate_templates
		ADD COLUMN detail TEXT
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add detail to host_certificate_templates: %w", err)
	}

	return nil
}

func Down_20251127113559(tx *sql.Tx) error {
	return nil
}
