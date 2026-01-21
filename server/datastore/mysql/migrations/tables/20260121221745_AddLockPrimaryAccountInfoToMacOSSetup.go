package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260121221745, Down_20260121221745)
}

func Up_20260121221745(tx *sql.Tx) error {
	return nil
}

func Down_20260121221745(tx *sql.Tx) error {
	return nil
}
