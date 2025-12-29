package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20251229000000, Down_20251229000000)
}

func Up_20251229000000(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE host_certificate_templates
		ADD COLUMN uuid BINARY(16) NULL
	`)
	if err != nil {
		return errors.Wrap(err, "add uuid column to host_certificate_templates")
	}
	return nil
}

func Down_20251229000000(_ *sql.Tx) error {
	return nil
}
