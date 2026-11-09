package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261109135014, Down_20261109135014)
}

// Up_20261109135014 rewrites the enabled_macos_disk_encryption and disabled_macos_disk_encryption activities as
// edited_disk_encryption_settings. Before Fleet 4.92 one setting covered every platform, so no single platform is right and it is stored
// as null. COALESCE prefers fleet_*: a JSON null (unassigned) is a non-NULL SQL value, so only a missing key falls through to team_*.
//
// The index is forced because with stale statistics MySQL scans the whole table instead.
func Up_20261109135014(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		UPDATE activity_past FORCE INDEX (idx_activities_activity_type)
		SET
			activity_type = 'edited_disk_encryption_settings',
			details = JSON_OBJECT(
				'fleet_id',   COALESCE(JSON_EXTRACT(details, '$.fleet_id'),   JSON_EXTRACT(details, '$.team_id')),
				'fleet_name', COALESCE(JSON_EXTRACT(details, '$.fleet_name'), JSON_EXTRACT(details, '$.team_name')),
				'platform',   NULL
			)
		WHERE activity_type IN ('enabled_macos_disk_encryption', 'disabled_macos_disk_encryption')`,
	); err != nil {
		return fmt.Errorf("migrating legacy disk encryption activities: %w", err)
	}
	return nil
}

func Down_20261109135014(tx *sql.Tx) error {
	return nil
}
