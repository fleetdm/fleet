package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260707140102, Down_20260707140102)
}

func Up_20260707140102(tx *sql.Tx) error {
	// Supports RetrieveNextCommand, which filters a single enrollment's queue by
	// (id, active) and orders by (priority DESC, created_at). Without an
	// id-leading index the optimizer picks the global (priority DESC, created_at)
	// index to satisfy the ORDER BY ... LIMIT 1 and scans the entire index; this
	// index scopes to the enrollment first and returns rows already sorted, so
	// LIMIT 1 stops at the first match. InnoDB appends the remaining PK column
	// (command_uuid) to the secondary index, making it covering for the join to
	// nano_commands.
	return withSteps([]migrationStep{
		basicMigrationStep(
			`CREATE INDEX idx_neq_next_command ON nano_enrollment_queue (id, active, priority DESC, created_at);`,
			"creating index idx_neq_next_command on nano_enrollment_queue",
		),
	}, tx)
}

func Down_20260707140102(tx *sql.Tx) error {
	return nil
}
