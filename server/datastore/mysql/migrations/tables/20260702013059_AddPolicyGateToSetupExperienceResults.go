package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260702013059, Down_20260702013059)
}

func Up_20260702013059(tx *sql.Tx) error {
	// policy_gated marks a Windows/Linux setup-experience software item whose installer has at least one team policy with an
	// install-software automation pointing at it. Such an item is gated: it is skipped only if every in-scope gating policy
	// passes, and installed if any fails. The set of gating policies is derived from the installer at decision time, so this is
	// only a marker (no specific policy is stored, which also means deleting one of several gating policies does not un-gate the
	// item). It is internal (json:"-"), so this is not an API change.
	_, err := tx.Exec(`
ALTER TABLE setup_experience_status_results
	ADD COLUMN policy_gated TINYINT(1) NOT NULL DEFAULT 0
`)
	if err != nil {
		return fmt.Errorf("add policy_gated to setup_experience_status_results: %w", err)
	}
	return nil
}

func Down_20260702013059(tx *sql.Tx) error {
	return nil
}
