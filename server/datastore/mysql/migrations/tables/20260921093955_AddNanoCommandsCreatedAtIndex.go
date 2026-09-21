package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260921093955, Down_20260921093955)
}

func Up_20260921093955(tx *sql.Tx) error {
	// The orphan sweep walks commands oldest-first by created_at.
	return addIndexesTx(tx, "nano_commands", indexDef{"idx_nano_commands_created_at", "created_at"})
}

func Down_20260921093955(tx *sql.Tx) error {
	return nil
}
