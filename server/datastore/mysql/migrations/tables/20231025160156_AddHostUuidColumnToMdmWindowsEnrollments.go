package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231025160156, Down_20231025160156)
}

func Up_20231025160156(tx *sql.Tx) error {
	stmt := `
          ALTER TABLE mdm_windows_enrollments
          ADD COLUMN host_uuid varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
 	 `
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add host_uuid to mdm_windows_enrollments: %w", err)
	}
	return nil
}

func Down_20231025160156(tx *sql.Tx) error {
	return nil
}
