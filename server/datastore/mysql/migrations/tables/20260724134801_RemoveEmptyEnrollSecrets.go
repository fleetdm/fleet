package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260724134801, Down_20260724134801)
}

func Up_20260724134801(tx *sql.Tx) error {
	// Remove empty or whitespace-only enroll secrets. These can never be used
	// to enroll a host (the server now rejects blank secrets), so deleting them
	// neutralizes any blank secret that predates the create/update validation.
	// A team left without any secret simply falls back to the global enroll
	// secret for MDM provisioning, matching the "team has no secret" behavior.
	if _, err := tx.Exec(`DELETE FROM enroll_secrets WHERE TRIM(secret) = ''`); err != nil {
		return fmt.Errorf("deleting empty enroll secrets: %w", err)
	}
	return nil
}

func Down_20260724134801(tx *sql.Tx) error {
	return nil
}
