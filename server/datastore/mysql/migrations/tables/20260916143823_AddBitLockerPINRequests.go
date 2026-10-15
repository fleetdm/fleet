package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260916143823, Down_20260916143823)
}

func Up_20260916143823(tx *sql.Tx) error {
	// An end user submits a BitLocker startup PIN from the My device page and fleetd, running as SYSTEM, applies it on
	// their behalf so a standard user does not need local admin rights. The server hands the PIN off to the agent, so it
	// is held here encrypted with the server private key until the agent collects it on its next config poll.
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS host_bitlocker_pin_requests (
			host_id INT UNSIGNED NOT NULL PRIMARY KEY,
			-- Identifies which submission the agent collected, so a delayed outcome for an earlier PIN cannot be
			-- recorded against a newer one the user submitted in the meantime.
			request_uuid BINARY(16) NOT NULL,
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

func Down_20260916143823(tx *sql.Tx) error {
	return nil
}
