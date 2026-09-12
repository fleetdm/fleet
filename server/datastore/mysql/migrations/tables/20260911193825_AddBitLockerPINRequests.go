package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260911193825, Down_20260911193825)
}

func Up_20260911193825(tx *sql.Tx) error {
	// An end user submits a BitLocker startup PIN from the My device page and fleetd, running as SYSTEM, applies it on
	// their behalf so a standard user does not need local admin rights. The PIN is relayed through the server, so it is
	// held here encrypted with the server private key for the seconds between the submission and the agent's next config
	// poll, then cleared. One row per host, replaced on resubmission, and it never holds a secret once delivered.
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS host_bitlocker_pin_requests (
			host_id INT UNSIGNED NOT NULL PRIMARY KEY,
			-- NULL once the agent has collected the PIN, so a terminal row carries no secret.
			pin_encrypted TEXT NULL DEFAULT NULL,
			status ENUM('pending', 'delivered', 'set', 'failed') NOT NULL DEFAULT 'pending',
			-- Width matches host_disks.bitlocker_protection_error, the other agent-reported reason string.
			client_error VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at TIMESTAMP(6) NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("create host_bitlocker_pin_requests: %w", err)
	}

	// Two columns on mdm_windows_enrollments, both read by the single row fetch the orbit-config check-in already does
	// for every Windows MDM host (see GetMDMWindowsHostConfigState), so neither costs an extra query per poll. This is
	// the same reasoning that put has_pending_commands on this row.
	//
	//   - fleetd_bitlocker_pin_capable: the last-observed X-Fleet-Capabilities CapabilityWindowsBitLockerPIN flag,
	//     persisted by the orbit-config endpoint exactly as fleetd_sync_capable is. The device and Fleet Desktop
	//     endpoints carry no capability header, so they gate on this column instead of re-deriving it. Default 0 means
	//     an agent that has not reported is treated as unable to apply a PIN, keeping the instructions modal in place
	//     until it upgrades.
	//   - bitlocker_pin_request_pending: denormalizes "this host has a PIN waiting to be collected" from
	//     host_bitlocker_pin_requests, which is keyed by host_id and would otherwise need its own lookup on every
	//     poll. Kept in step with that table inside one transaction. A stale true is harmless, because the agent's
	//     collect request simply finds nothing and the flag is cleared; a stale false would strand a submission, which
	//     is why the write that queues a PIN and the write that sets this flag commit together.
	for _, col := range []struct{ name, definition string }{
		{"fleetd_bitlocker_pin_capable", "TINYINT(1) NOT NULL DEFAULT 0"},
		{"bitlocker_pin_request_pending", "TINYINT(1) NOT NULL DEFAULT 0"},
	} {
		if columnExists(tx, "mdm_windows_enrollments", col.name) {
			continue
		}
		if _, err := tx.Exec(fmt.Sprintf(
			`ALTER TABLE mdm_windows_enrollments ADD COLUMN %s %s`, col.name, col.definition,
		)); err != nil {
			return fmt.Errorf("add %s to mdm_windows_enrollments: %w", col.name, err)
		}
	}

	return nil
}

func Down_20260911193825(tx *sql.Tx) error {
	return nil
}
