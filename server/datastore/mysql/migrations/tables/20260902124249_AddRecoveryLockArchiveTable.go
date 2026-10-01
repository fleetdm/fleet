package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260902124249, Down_20260902124249)
}

func Up_20260902124249(tx *sql.Tx) error {
	// Every recovery lock password Fleet generates is archived here before the command
	// carrying it goes on the wire, because that is the only point at which "could this
	// password have reached the device?" is answerable for all of them. A SetRecoveryLock
	// can be applied by the device without Fleet learning: the result response is lost, the
	// device checks out mid-command, or a re-enrollment clears the queue. Retries also
	// generate a fresh password each time, so an operation can put several on the wire.
	//
	// Rows are pruned on proof rather than on age: when a device confirms which password it
	// holds, everything older is provably not on the device (see SetRecoveryLockVerified).
	// A healthy host therefore keeps one row no matter how often it rotates, and history
	// only accumulates across the window where Fleet is uncertain.
	//
	// Deliberately not tied to the host lifecycle: a device can still hold a lock after it
	// unenrolls or is deleted from Fleet, which is exactly when an admin needs the password.
	if _, err := tx.Exec(`
		CREATE TABLE host_recovery_key_password_archive (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			host_uuid varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			-- Byte-identical to the host_recovery_key_passwords ciphertext it was generated
			-- for, which is how the prune identifies the row a device just confirmed.
			encrypted_password BLOB NOT NULL,
			set_command_uuid varchar(127) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			PRIMARY KEY (id),
			KEY idx_recovery_lock_archive_host_uuid (host_uuid, id)
		)
	`); err != nil {
		return fmt.Errorf("creating host_recovery_key_password_archive table: %w", err)
	}

	// Seed the archive with each host's last known password. Hosts set up before the
	// archive existed would otherwise have no candidate on file at all until their next
	// rotation. Soft-deleted rows are included on purpose: a host that unenrolled is the
	// most likely one to still be sitting on a lock Fleet handed it.
	//
	// One pass over host_recovery_key_passwords, which holds at most one row per
	// Apple-silicon host with the feature enabled — it does not scale with software,
	// certificates, or command history, so a single INSERT ... SELECT is safe here.
	if _, err := tx.Exec(`
		INSERT INTO host_recovery_key_password_archive (host_uuid, encrypted_password, set_command_uuid)
		SELECT host_uuid, encrypted_password, set_command_uuid
		FROM host_recovery_key_passwords
		WHERE encrypted_password IS NOT NULL
	`); err != nil {
		return fmt.Errorf("seeding host_recovery_key_password_archive: %w", err)
	}

	return nil
}

func Down_20260902124249(tx *sql.Tx) error {
	return nil
}
