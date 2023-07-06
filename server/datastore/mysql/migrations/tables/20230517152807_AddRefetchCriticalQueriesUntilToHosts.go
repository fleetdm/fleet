package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230517152807, Down_20230517152807)
}

func Up_20230517152807(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE hosts ADD COLUMN refetch_critical_queries_until TIMESTAMP NULL;
`)
	return errors.Wrap(err, "add refetch_critical_queries_until")
}

func Down_20230517152807(tx *sql.Tx) error {
	return nil
}
