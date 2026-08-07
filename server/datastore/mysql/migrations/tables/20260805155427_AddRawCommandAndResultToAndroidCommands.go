package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260805155427, Down_20260805155427)
}

func Up_20260805155427(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE mdm_android_commands
	ADD COLUMN raw_command TEXT COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER command_type,
	ADD COLUMN raw_result  TEXT COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER error_message
`)
	if err != nil {
		return fmt.Errorf("alter table mdm_android_commands: %w", err)
	}
	return nil
}

func Down_20260805155427(tx *sql.Tx) error {
	return nil
}
