package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220510110838, Down_20220510110838)
}

func Up_20220510110838(tx *sql.Tx) error {
	stm := "CREATE INDEX hosts_platform_idx ON hosts (platform);"

	if _, err := tx.Exec(stm); err != nil {
		return errors.Wrap(err, "creating hosts index")
	}

	return nil
}

func Down_20220510110838(tx *sql.Tx) error {
	return nil
}
