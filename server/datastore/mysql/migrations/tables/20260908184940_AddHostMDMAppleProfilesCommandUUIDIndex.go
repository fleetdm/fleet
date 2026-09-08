package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260908184940, Down_20260908184940)
}

// Up_20260908184940 indexes host_mdm_apple_profiles.command_uuid so the Apple
// MDM command cleanup can check whether a command is still a host's current
// profile command without scanning the table.
func Up_20260908184940(tx *sql.Tx) error {
	return addIndexesTx(tx, "host_mdm_apple_profiles", indexDef{"idx_hmap_command_uuid", "command_uuid"})
}

func Down_20260908184940(tx *sql.Tx) error {
	return nil
}
