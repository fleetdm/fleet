package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261124143542, Down_20261124143542)
}

func Up_20261124143542(tx *sql.Tx) error {
	// DDL isn't transactional, so each step is skipped if an earlier attempt already applied it.
	//
	// Existing rows keep a NULL device_family: it's only set from what Apple Business reports.
	if !columnExists(tx, "host_dep_assignments", "device_family") {
		if _, err := tx.Exec(`
			ALTER TABLE host_dep_assignments
			ADD COLUMN device_family VARCHAR(63) COLLATE utf8mb4_unicode_ci NULL`); err != nil {
			return fmt.Errorf("adding device_family to host_dep_assignments: %w", err)
		}
	}

	// NULL means the tvOS profile hasn't been worked out yet, an empty string that tvOS uses the regular profile.
	for _, table := range []string{"mdm_apple_default_setup_assistants", "mdm_apple_setup_assistant_profiles"} {
		if columnExists(tx, table, "tvos_profile_uuid") {
			continue
		}
		if _, err := tx.Exec(fmt.Sprintf(`
			ALTER TABLE %s
			ADD COLUMN tvos_profile_uuid VARCHAR(255) COLLATE utf8mb4_unicode_ci NULL`, table)); err != nil {
			return fmt.Errorf("adding tvos_profile_uuid to %s: %w", table, err)
		}
	}

	return nil
}

func Down_20261124143542(_ *sql.Tx) error {
	return nil
}
