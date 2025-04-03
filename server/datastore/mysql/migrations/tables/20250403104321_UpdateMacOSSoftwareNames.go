package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250403104321, Down_20250403104321)
}

func Up_20250403104321(tx *sql.Tx) error {
	nameStmt := `UPDATE software_titles SET name = TRIM( TRAILING '.app' FROM name ) WHERE source = 'apps'`
	_, err := tx.Exec(nameStmt)
	if err != nil {
		return fmt.Errorf("updating software_titles.name: %w", err)
	}

	return nil
}

func Down_20250403104321(tx *sql.Tx) error {
	return nil
}
