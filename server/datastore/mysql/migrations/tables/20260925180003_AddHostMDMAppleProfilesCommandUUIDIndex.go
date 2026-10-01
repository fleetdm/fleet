package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260925180003, Down_20260925180003)
}

func Up_20260925180003(tx *sql.Tx) error {
	// The largest reference guard: without this, every cleanup batch scans
	// host_mdm_apple_profiles to check whether a command is still a host's profile.
	return addIndexesTx(tx, "host_mdm_apple_profiles", indexDef{"idx_hmap_command_uuid", "command_uuid"})
}

func Down_20260925180003(tx *sql.Tx) error {
	return nil
}
