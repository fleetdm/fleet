package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260901200005, Down_20260901200005)
}

func Up_20260901200005(tx *sql.Tx) error {
	// Without this index the sweep's steady-state "nothing left" case scans the
	// whole table every hour. Every other column it probes is already indexed.
	return addIndexesTx(tx, "host_software_installs",
		indexDef{"idx_host_software_installs_created_at", "created_at"})
}

func Down_20260901200005(tx *sql.Tx) error {
	return nil
}
