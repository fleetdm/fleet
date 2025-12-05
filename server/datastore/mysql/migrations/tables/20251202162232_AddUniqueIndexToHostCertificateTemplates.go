package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20251202162232, Down_20251202162232)
}

func Up_20251202162232(tx *sql.Tx) error {
	// Add a unique index to:
	// 1. Ensure data integrity: no duplicate records for the same host and certificate template
	// 2. Improve query performance for lookups by these columns
	_, err := tx.Exec(`
		ALTER TABLE host_certificate_templates
		ADD UNIQUE INDEX idx_host_certificate_templates_host_template (host_uuid, certificate_template_id)
	`)
	return err
}

func Down_20251202162232(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE host_certificate_templates
		DROP INDEX idx_host_certificate_templates_host_template
	`)
	return err
}
