package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261113214109, Down_20261113214109)
}

func Up_20261113214109(tx *sql.Tx) error {
	// IF EXISTS keeps the migration safe to re-run after MySQL's implicit DDL commit.
	if _, err := tx.Exec(`DROP TABLE IF EXISTS mdm_apple_installers`); err != nil {
		return fmt.Errorf("drop mdm_apple_installers: %w", err)
	}
	return nil
}

func Down_20261113214109(tx *sql.Tx) error {
	return nil
}
