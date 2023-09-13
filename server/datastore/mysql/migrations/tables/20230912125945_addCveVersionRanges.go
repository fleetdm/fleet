package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230912125945, Down_20230912125945)
}

func Up_20230912125945(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE software_cve
		ADD COLUMN version_ranges JSON
	`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add version_ranges to software_cve: %w", err)
	}
	return nil
}

func Down_20230912125945(tx *sql.Tx) error {
	return nil
}
