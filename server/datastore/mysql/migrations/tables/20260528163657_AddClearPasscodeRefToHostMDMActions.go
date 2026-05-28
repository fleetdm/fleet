package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260528163657, Down_20260528163657)
}

// Up_20260528163657 adds clear_passcode_ref to host_mdm_actions, mirroring lock_ref and wipe_ref. For Android hosts the column
// points at mdm_android_commands.command_uuid for a RESET_PASSWORD command and is the signal
// HostLockWipeStatus.IsPendingClearPasscode reads to flip a host into the "clearing passcode" device status while the AMAPI
// command is in flight.
//
// Other platforms keep this column NULL. Story to add Apple support: #46286
func Up_20260528163657(tx *sql.Tx) error {
	if columnExists(tx, "host_mdm_actions", "clear_passcode_ref") {
		return nil
	}
	if _, err := tx.Exec(`
ALTER TABLE host_mdm_actions
	ADD COLUMN clear_passcode_ref VARCHAR(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL
`); err != nil {
		return fmt.Errorf("add clear_passcode_ref to host_mdm_actions: %w", err)
	}
	return nil
}

func Down_20260528163657(tx *sql.Tx) error {
	return nil
}
