package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260814183816, Down_20260814183816)
}

func Up_20260814183816(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE mdm_android_commands
	ADD COLUMN raw_command MEDIUMTEXT COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER command_type,
	ADD COLUMN raw_result  MEDIUMTEXT COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER error_message
`)
	if err != nil {
		return fmt.Errorf("alter table mdm_android_commands: %w", err)
	}
	return nil
}

func Down_20260814183816(tx *sql.Tx) error {
	return nil
}
