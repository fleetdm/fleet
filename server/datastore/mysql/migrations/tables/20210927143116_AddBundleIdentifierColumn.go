package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210927143116, Down_20210927143116)
}

func columnExists(tx *sql.Tx, table, column string) bool {
	var count int
	err := tx.QueryRow(
		`SELECT count(*) FROM information_schema.columns WHERE COLUMN_NAME = ? AND table_name = ? LIMIT 1;`,
		column, table,
	).Scan(&count)
	if err != nil {
		return false
	}

	return count == 1
}

func Up_20210927143116(tx *sql.Tx) error {
	if columnExists(tx, "software", "bundle_identifier") {
		return nil
	}

	if _, err := tx.Exec(`ALTER TABLE software ADD COLUMN bundle_identifier VARCHAR(255) DEFAULT ''`); err != nil {
		return errors.Wrap(err, "add column bundle_identifier")
	}
	return nil
}

func Down_20210927143116(tx *sql.Tx) error {
	return nil
}
