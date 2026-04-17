package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260417202812, Down_20260417202812)
}

func Up_20260417202812(tx *sql.Tx) error {
	// Make tpm_pin_set nullable so we can distinguish between "not yet reported" (NULL),
	// "confirmed absent" (false), and "present" (true). Previously defaulted to false,
	// which made new hosts appear as "Action required" before their first osquery report.
	if _, err := tx.Exec(`ALTER TABLE host_disks MODIFY COLUMN tpm_pin_set TINYINT(1) NULL DEFAULT NULL`); err != nil {
		return fmt.Errorf("making tpm_pin_set nullable: %w", err)
	}
	// Reset all existing rows to NULL so hosts re-report their actual PIN status.
	// Explicitly set updated_at = updated_at to prevent the ON UPDATE CURRENT_TIMESTAMP trigger from firing.
	if _, err := tx.Exec(`UPDATE host_disks SET tpm_pin_set = NULL, updated_at = updated_at`); err != nil {
		return fmt.Errorf("resetting tpm_pin_set to NULL: %w", err)
	}
	return nil
}

func Down_20260417202812(tx *sql.Tx) error {
	return nil
}
