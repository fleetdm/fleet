package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260916091319, Down_20260916091319)
}

func Up_20260916091319(tx *sql.Tx) error {
	if columnExists(tx, "host_mdm_commands", "command_uuid") {
		return nil
	}
	// NULL marks rows written before Fleet recorded which command a tracking
	// row refers to; code treats them with the pre-UUID semantics until they
	// age out.
	if _, err := tx.Exec(`
		ALTER TABLE host_mdm_commands
		ADD COLUMN command_uuid VARCHAR(127) COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL
	`); err != nil {
		return fmt.Errorf("adding command_uuid to host_mdm_commands: %w", err)
	}
	return nil
}

func Down_20260916091319(tx *sql.Tx) error {
	return nil
}
