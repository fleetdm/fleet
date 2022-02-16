package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211013133706, Down_20211013133706)
}

func Up_20211013133706(tx *sql.Tx) error {
	if columnExists(tx, "policies", "resolution") {
		return nil
	}

	if _, err := tx.Exec(`ALTER TABLE policies ADD COLUMN resolution TEXT`); err != nil {
		return errors.Wrap(err, "add column resolution to policies")
	}
	return nil
}

func Down_20211013133706(tx *sql.Tx) error {
	return nil
}
