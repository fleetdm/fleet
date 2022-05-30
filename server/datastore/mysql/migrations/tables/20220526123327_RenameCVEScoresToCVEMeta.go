package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220526123327, Down_20220526123327)
}

func Up_20220526123327(tx *sql.Tx) error {
	_, err := tx.Exec(`
RENAME TABLE
    cve_scores TO cve_meta
`)
	if err != nil {
		return errors.Wrapf(err, "rename table")
	}

	_, err = tx.Exec(`
ALTER TABLE cve_meta
    ADD published TIMESTAMP NULL DEFAULT NULL
`)
	if err != nil {
		return errors.Wrapf(err, "add column")
	}

	return nil
}

func Down_20220526123327(tx *sql.Tx) error {
	return nil
}
