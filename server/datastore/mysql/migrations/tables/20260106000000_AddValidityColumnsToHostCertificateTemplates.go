package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260106000000, Down_20260106000000)
}

func Up_20260106000000(tx *sql.Tx) error {
	// Add not_valid_before column for certificate validity tracking
	// Using DATETIME (not DATETIME(6)) since X.509 cert validity has max 1-second precision per RFC 5280
	if _, err := tx.Exec(`
		ALTER TABLE host_certificate_templates
		ADD COLUMN not_valid_before DATETIME NULL
	`); err != nil {
		return errors.Wrap(err, "adding not_valid_before column to host_certificate_templates")
	}

	// Add not_valid_after column for certificate expiration tracking (used for renewal)
	// Using DATETIME (not DATETIME(6)) since X.509 cert validity has max 1-second precision per RFC 5280
	if _, err := tx.Exec(`
		ALTER TABLE host_certificate_templates
		ADD COLUMN not_valid_after DATETIME NULL
	`); err != nil {
		return errors.Wrap(err, "adding not_valid_after column to host_certificate_templates")
	}

	// Add serial column to store certificate serial number
	if _, err := tx.Exec(`
		ALTER TABLE host_certificate_templates
		ADD COLUMN serial VARCHAR(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL
	`); err != nil {
		return errors.Wrap(err, "adding serial column to host_certificate_templates")
	}

	// Add index on not_valid_after for efficient renewal queries
	if _, err := tx.Exec(`
		ALTER TABLE host_certificate_templates
		ADD INDEX idx_host_certificate_templates_not_valid_after (not_valid_after)
	`); err != nil {
		return errors.Wrap(err, "adding index on not_valid_after to host_certificate_templates")
	}

	return nil
}

func Down_20260106000000(tx *sql.Tx) error {
	return nil
}
