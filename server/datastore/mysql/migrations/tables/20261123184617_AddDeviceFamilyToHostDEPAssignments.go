package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261123184617, Down_20261123184617)
}

func Up_20261123184617(tx *sql.Tx) error {
	// DDL isn't transactional, so each step is skipped if an earlier attempt already applied it.
	if !columnExists(tx, "host_dep_assignments", "device_family") {
		if _, err := tx.Exec(`
			ALTER TABLE host_dep_assignments
			ADD COLUMN device_family VARCHAR(63) COLLATE utf8mb4_unicode_ci NULL`); err != nil {
			return fmt.Errorf("adding device_family to host_dep_assignments: %w", err)
		}
	}

	// Apple TVs synced before tvOS was a platform were created as macOS hosts. Their model is either Apple's model
	// name from the sync or, once enrolled, the product identifier reported at check-in.
	if _, err := tx.Exec(`
		UPDATE host_dep_assignments hda
		JOIN hosts h ON h.id = hda.host_id
		SET hda.device_family = 'AppleTV'
		WHERE h.platform IN ('darwin', 'tvos')
			AND (h.hardware_model LIKE 'AppleTV%' OR h.hardware_model LIKE 'Apple TV%')`); err != nil {
		return fmt.Errorf("backfilling host_dep_assignments.device_family: %w", err)
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

func Down_20261123184617(_ *sql.Tx) error {
	return nil
}
