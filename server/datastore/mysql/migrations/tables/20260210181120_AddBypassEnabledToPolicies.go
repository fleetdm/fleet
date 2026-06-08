package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260210181120, Down_20260210181120)
}

func Up_20260210181120(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE policies ADD COLUMN conditional_access_bypass_enabled TINYINT(1) NOT NULL DEFAULT 1`); err != nil {
		return fmt.Errorf("adding bypass_enabled to policies table: %w", err)
	}
	return nil
}

func Down_20260210181120(tx *sql.Tx) error {
	return nil
}
