package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260805161502, Down_20260805161502)
}

func Up_20260805161502(tx *sql.Tx) error {
	// The row is keyed by host_uuid and deliberately outlives the hosts row, so a host deleted from the UI while the
	// device is still enrolled keeps its break-glass password. It must not outlive the *enrollment* that produced it:
	// a re-enrolling device may have been re-imaged, in which case the password no longer opens any account and Fleet
	// would keep reporting it as verified. Soft-delete rather than hard-delete, mirroring host_recovery_key_passwords,
	// so no lifecycle event destroys a password; reads filter deleted = 0 and the next successful escrow clears it.
	//
	// No index: every access path is already served by the primary key (host_uuid) or by an existing index
	// (command_uuid, auto_rotate_at), and a two-value column has no useful selectivity.
	_, err := tx.Exec(`
		ALTER TABLE host_managed_local_account_passwords
		ADD COLUMN deleted TINYINT(1) NOT NULL DEFAULT '0'
	`)
	if err != nil {
		return fmt.Errorf("adding host_managed_local_account_passwords.deleted: %w", err)
	}

	return nil
}

func Down_20260805161502(tx *sql.Tx) error {
	return nil
}
