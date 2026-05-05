package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260324161944, Down_20260324161944)
}

func Up_20260324161944(tx *sql.Tx) error {
	stmt := `
          ALTER TABLE software_installers
          ADD COLUMN patch_query TEXT COLLATE utf8mb4_unicode_ci NOT NULL;
 	 `
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add patch_query to software_installers: %w", err)
	}

	return nil
}

func Down_20260324161944(tx *sql.Tx) error {
	return nil
}
