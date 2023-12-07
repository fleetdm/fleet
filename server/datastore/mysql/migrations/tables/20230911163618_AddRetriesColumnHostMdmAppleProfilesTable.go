package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230911163618, Down_20230911163618)
}

func Up_20230911163618(tx *sql.Tx) error {
	stmt := `
ALTER TABLE host_mdm_apple_profiles
	ADD COLUMN retries TINYINT(3) UNSIGNED NOT NULL DEFAULT 0`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add retries to host_mdm_apple_profiles: %w", err)
	}
	return nil
}

func Down_20230911163618(tx *sql.Tx) error {
	return nil
}
