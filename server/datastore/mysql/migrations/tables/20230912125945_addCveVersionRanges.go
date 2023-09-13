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
		ADD COLUMN versionStartIncluding VARCHAR(255) COLLATE utf8mb4_unicode_ci,
		ADD COLUMN versionEndExcluding VARCHAR(255) COLLATE utf8mb4_unicode_ci`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add versionStartIncluding and versionEndExcluding to software_cve: %w", err)
	}
	return nil
}

func Down_20230912125945(tx *sql.Tx) error {
	return nil
}
