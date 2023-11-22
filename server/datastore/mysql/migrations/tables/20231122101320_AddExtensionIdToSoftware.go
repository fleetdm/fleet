package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231122101320, Down_20231122101320)
}

func Up_20231122101320(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE software
		ADD COLUMN extension_id varchar(255) NOT NULL DEFAULT '';
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add extension_id to software: %w", err)
	}

	return nil
}

func Down_20231122101320(tx *sql.Tx) error {
	return nil
}
