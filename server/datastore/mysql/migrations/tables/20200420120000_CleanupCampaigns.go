package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up20200420120000, Down20200420120000)
}

func Up20200420120000(tx *sql.Tx) error {
	if _, err := tx.Exec(
		"DROP TABLE `distributed_query_executions` ",
	); err != nil {
		return errors.Wrap(err, "drop distributed_query_executions table ")
	}

	return nil
}

func Down20200420120000(tx *sql.Tx) error {
	return nil
}
