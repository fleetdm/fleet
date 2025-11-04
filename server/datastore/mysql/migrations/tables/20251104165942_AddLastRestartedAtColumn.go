package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20251104165942, Down_20251104165942)
}

func Up_20251104165942(tx *sql.Tx) error {
	return nil
}

func Down_20251104165942(tx *sql.Tx) error {
	return nil
}
