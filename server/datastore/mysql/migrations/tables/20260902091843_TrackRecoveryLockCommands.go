package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260902091843, Down_20260902091843)
}

func Up_20260902091843(tx *sql.Tx) error {
	/**
	- pending_set_command_uuid covers the pending set operation for the recovery lock, both for clearing and setting a new password
	- pending_verify_command_uuid covers the pending verify operation for the recovery lock
	- set_command_uuid covers the last set operation for the recovery lock
	- verify_command_uuid covers the last verify operation for the recovery lock
	*/
	_, err := tx.Exec(`ALTER TABLE host_recovery_key_passwords
		ADD COLUMN pending_set_command_uuid VARCHAR(127) NULL COLLATE utf8mb4_unicode_ci,
		ADD COLUMN pending_verify_command_uuid VARCHAR(127) NULL COLLATE utf8mb4_unicode_ci,
		ADD COLUMN set_command_uuid VARCHAR(127) NULL COLLATE utf8mb4_unicode_ci,
		ADD COLUMN verify_command_uuid VARCHAR(127) NULL COLLATE utf8mb4_unicode_ci,
		ADD COLUMN retry INT NOT NULL DEFAULT 0,
		DROP COLUMN pending_error_message,
		MODIFY COLUMN encrypted_password BLOB NULL;
		`)
	return err
}

func Down_20260902091843(tx *sql.Tx) error {
	return nil
}
