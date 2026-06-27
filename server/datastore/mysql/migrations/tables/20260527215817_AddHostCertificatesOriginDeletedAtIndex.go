package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260527215817, Down_20260527215817)
}

func Up_20260527215817(tx *sql.Tx) error {
	// Supports the unenrolled-host MDM cert sweep, which filters on
	// (origin, deleted_at) and orders by id. InnoDB appends the PK (id) to
	// the secondary index, so this also satisfies the ORDER BY id LIMIT
	// without a filesort.
	return withSteps([]migrationStep{
		basicMigrationStep(
			`CREATE INDEX idx_host_certs_origin_deleted ON host_certificates (origin, deleted_at);`,
			"creating index idx_host_certs_origin_deleted on host_certificates",
		),
	}, tx)
}

func Down_20260527215817(tx *sql.Tx) error {
	return nil
}
