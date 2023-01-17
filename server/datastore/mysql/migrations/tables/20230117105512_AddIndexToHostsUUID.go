package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230117105512, Down_20230117105512)
}

func Up_20230117105512(tx *sql.Tx) error {
	_, err := tx.Exec("CREATE INDEX idx_hosts_uuid ON hosts (uuid)")
	return errors.Wrap(err, "creating hosts uuid index")
}

func Down_20230117105512(tx *sql.Tx) error {
	return nil
}
