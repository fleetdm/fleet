package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250729143159, Down_20250729143159)
}

func Up_20250729143159(tx *sql.Tx) error {
	if _, err := tx.Exec(`
ALTER TABLE software_titles
	ADD COLUMN is_kernel TINYINT(1) NOT NULL DEFAULT '0'`); err != nil {
		return fmt.Errorf("failed to add software_titles.is_kernel column: %w", err)
	}
	return nil
}

func Down_20250729143159(tx *sql.Tx) error {
	return nil
}
