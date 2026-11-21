package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261121135808, Down_20261121135808)
}

func Up_20261121135808(tx *sql.Tx) error {
	// Marks a policy whose platforms or labels changed, so the policy_membership_cleanup schedule removes the results of
	// hosts it no longer targets.
	if !columnExists(tx, "policies", "needs_partial_membership_cleanup") {
		if _, err := tx.Exec(`
			ALTER TABLE policies
				ADD COLUMN needs_partial_membership_cleanup TINYINT(1) NOT NULL DEFAULT '0' AFTER needs_full_membership_cleanup
		`); err != nil {
			return fmt.Errorf("add needs_partial_membership_cleanup to policies: %w", err)
		}
	}
	return nil
}

func Down_20261121135808(tx *sql.Tx) error {
	return nil
}
