package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20220921164931, Down_20220921164931)
}

func Up_20220921164931(tx *sql.Tx) error {
	return nil
}

func Down_20220921164931(tx *sql.Tx) error {
	return nil
}
