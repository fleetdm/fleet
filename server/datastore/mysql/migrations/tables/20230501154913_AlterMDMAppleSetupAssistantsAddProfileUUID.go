package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230501154913, Down_20230501154913)
}

func Up_20230501154913(tx *sql.Tx) error {
	return nil
}

func Down_20230501154913(tx *sql.Tx) error {
	return nil
}
