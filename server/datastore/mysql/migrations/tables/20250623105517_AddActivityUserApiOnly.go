package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20250623105517, Down_20250623105517)
}

func Up_20250623105517(tx *sql.Tx) error {
	const stmt = `ALTER TABLE activities ADD COLUMN api_only boolean`
	if _, err := tx.Exec(stmt); err != nil {
		return err
	}
	return nil
}

func Down_20250623105517(tx *sql.Tx) error {
	return nil
}
