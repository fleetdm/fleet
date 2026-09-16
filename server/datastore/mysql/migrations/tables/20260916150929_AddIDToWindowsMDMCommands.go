package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260916150929, Down_20260916150929)
}

func Up_20260916150929(tx *sql.Tx) error {
	if columnExists(tx, "windows_mdm_commands", "id") {
		return nil
	}
	// AUTO_INCREMENT rather than TIMESTAMP(6): a multi-row INSERT stamps every row with the statement start
	// time, so timestamps would still tie. command_uuid stays the primary key, since MySQL only requires the
	// AUTO_INCREMENT column to be indexed. Adding an AUTO_INCREMENT column rebuilds the table and MySQL
	// refuses LOCK=NONE for it, so writes to this table block for the duration; pinning the algorithm and lock
	// makes the migration fail loudly instead of falling back to a full COPY.
	if _, err := tx.Exec(`
		ALTER TABLE windows_mdm_commands
		ADD COLUMN id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		ADD UNIQUE KEY idx_windows_mdm_commands_id (id),
		ALGORITHM=INPLACE, LOCK=SHARED
	`); err != nil {
		return fmt.Errorf("adding id to windows_mdm_commands: %w", err)
	}
	return nil
}

func Down_20260916150929(tx *sql.Tx) error {
	return nil
}
