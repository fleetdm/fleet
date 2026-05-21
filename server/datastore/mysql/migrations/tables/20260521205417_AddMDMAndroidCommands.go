package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260521205417, Down_20260521205417)
}

// Up_20260521205417 creates the mdm_android_commands table used to track AMAPI commands issued by
// Fleet via EnterprisesDevicesService.IssueCommand (Lock, Wipe, Clear passcode for Android hosts —
// see #41683).
//
// host_mdm_actions.{lock_ref, wipe_ref} stores the Fleet-generated command_uuid for Android hosts,
// mirroring how those columns point into nano_commands for Apple and mdm_windows_commands for Windows.
// The operation_name column holds the full AMAPI operation name
// (enterprises/X/devices/Y/operations/Z, ~70+ chars) which is the key used to correlate Pub/Sub
// COMMAND notifications back to the originating Fleet command.
func Up_20260521205417(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE mdm_android_commands (
	command_uuid     VARCHAR(36)   NOT NULL,
	host_uuid        VARCHAR(255)  NOT NULL,
	operation_name   VARCHAR(255)  NOT NULL,
	command_type     VARCHAR(32)   NOT NULL,
	status           VARCHAR(32)   NOT NULL,
	error_code       VARCHAR(64)   DEFAULT NULL,
	error_message    TEXT          DEFAULT NULL,
	request_payload  JSON          DEFAULT NULL,

	-- Using DATETIME(6) instead of TIMESTAMP to prevent future Y2K38 issues
	created_at       DATETIME(6)   NOT NULL DEFAULT NOW(6),
	updated_at       DATETIME(6)   NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),

	PRIMARY KEY (command_uuid),
	INDEX idx_mdm_android_commands_host_uuid (host_uuid),
	-- UNIQUE on operation_name: this column is the correlation key Pub/Sub COMMAND notifications
	-- use to find the originating Fleet command (GetMDMAndroidCommandByOperationName). Duplicates
	-- would make the lookup ambiguous and could silently corrupt status updates.
	UNIQUE INDEX idx_mdm_android_commands_operation_name (operation_name)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("create table mdm_android_commands: %w", err)
	}
	return nil
}

func Down_20260521205417(tx *sql.Tx) error {
	return nil
}
