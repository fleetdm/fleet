package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240313143416, Down_20240313143416)
}

func Up_20240313143416(tx *sql.Tx) error {
	stmt := `ALTER TABLE scripts DROP COLUMN script_contents`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("remove scripts.script_contents column: %w", err)
	}

	stmt = `ALTER TABLE host_script_results DROP COLUMN script_contents`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("remove host_script_results.script_contents column: %w", err)
	}

	return nil
}

func Down_20240313143416(tx *sql.Tx) error {
	return nil
}
