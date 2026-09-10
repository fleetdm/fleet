package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260908184939, Down_20260908184939)
}

// Up_20260908184939 indexes nano_commands.created_at for the Apple MDM
// command cleanup's orphan scan, which walks commands in creation order.
func Up_20260908184939(tx *sql.Tx) error {
	return addIndexesTx(tx, "nano_commands", indexDef{"idx_nano_commands_created_at", "created_at"})
}

func Down_20260908184939(tx *sql.Tx) error {
	return nil
}
