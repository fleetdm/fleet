package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260821201620, Down_20260821201620)
}

func Up_20260821201620(tx *sql.Tx) error {
	// last_login_status records the value of the com.microsoft/MDM/LoginStatus device alert ("user", "others", or "none") the device
	// last reported, and last_login_status_at when that value last changed. Windows sends the alert in the first message of every
	// management session, but the row is only written when the value differs, so the timestamp is not a per-session heartbeat.
	// This is the only durable record of the enrollment's user context.
	//
	// NULL means "never observed", which is distinct from "observed as no user": the user-scoped profile gate holds on both,
	// but only a positive "user" observation releases it.
	if _, err := tx.Exec(`
		ALTER TABLE mdm_windows_enrollments
			ADD COLUMN last_login_status VARCHAR(16) NULL,
			ADD COLUMN last_login_status_at DATETIME(6) NULL
	`); err != nil {
		return fmt.Errorf("adding login status columns to mdm_windows_enrollments: %w", err)
	}
	return nil
}

func Down_20260821201620(tx *sql.Tx) error {
	return nil
}
