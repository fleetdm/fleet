package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20251217000000, Down_20251217000000)
}

func Up_20251217000000(tx *sql.Tx) error {
	if _, err := tx.Exec("ALTER TABLE `host_certificate_templates` ADD COLUMN `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL"); err != nil {
		return errors.Wrap(err, "adding name column to host_certificate_templates")
	}

	// Populate name from certificate_templates for existing rows
	if _, err := tx.Exec(`
		UPDATE host_certificate_templates hct
		INNER JOIN certificate_templates ct ON ct.id = hct.certificate_template_id
		SET hct.name = ct.name
	`); err != nil {
		return errors.Wrap(err, "populating name column in host_certificate_templates")
	}

	return nil
}

func Down_20251217000000(tx *sql.Tx) error {
	return nil
}
