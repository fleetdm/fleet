package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230912101759, Down_20230912101759)
}

func Up_20230912101759(tx *sql.Tx) error {
	stmt := `
          ALTER TABLE cve_meta
          ADD COLUMN description TEXT COLLATE utf8mb4_unicode_ci DEFAULT NULL;
 	 `
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add description to cve_meta: %w", err)
	}
	return nil
}

func Down_20230912101759(tx *sql.Tx) error {
	return nil
}
