package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260323161023, Down_20260323161023)
}

func Up_20260323161023(tx *sql.Tx) error {
	stmt := `
          ALTER TABLE software_installers
          ADD COLUMN patch_query TEXT COLLATE utf8mb4_unicode_ci DEFAULT NULL;
 	 `
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add patch_query to software_installers: %w", err)
	}

	// text type column cannot have a default value, so set it to empty here
	// all subsequent inserts to this column should not be null
	updateStmt := `UPDATE software_installers SET patch_query = '';`
	if _, err := tx.Exec(updateStmt); err != nil {
		return fmt.Errorf("setting patch_query to empty: %w", err)
	}
	return nil
}

func Down_20260323161023(tx *sql.Tx) error {
	return nil
}
