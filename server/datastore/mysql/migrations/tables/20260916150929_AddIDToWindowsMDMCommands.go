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
	// AUTO_INCREMENT column to be indexed.
	//
	// ALGORITHM=COPY on purpose: it numbers existing rows in primary-key order on every server, so a primary
	// and its read replica (which replays this DDL itself) agree on the ids. INPLACE numbers rows in sort
	// chunks whose boundaries depend on innodb_ddl_buffer_size, so differently tuned servers disagree. COPY
	// measured about 1.7x the INPLACE rebuild time. MySQL requires a lock for an AUTO_INCREMENT column either
	// way, so writes block for the duration; pinning both makes the migration fail loudly rather than degrade.
	if _, err := tx.Exec(`
		ALTER TABLE windows_mdm_commands
		ALGORITHM=COPY, LOCK=SHARED,
		ADD COLUMN id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		ADD UNIQUE KEY idx_windows_mdm_commands_id (id)
	`); err != nil {
		return fmt.Errorf("adding id to windows_mdm_commands: %w", err)
	}
	return nil
}

func Down_20260916150929(tx *sql.Tx) error {
	return nil
}
