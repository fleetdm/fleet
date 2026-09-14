package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260909192031, Down_20260909192031)
}

// Go binaries are now matched by module path against the Go vulnerability database. The
// vulnerability cron only deletes CPEs for software it still iterates, so the rows it left
// behind have to go here.
func Up_20260909192031(tx *sql.Tx) error {
	return withSteps([]migrationStep{
		basicMigrationStep(
			`DELETE cpe FROM software_cpe cpe
			JOIN software s ON s.id = cpe.software_id
			WHERE s.source = 'go_binaries'`,
			"deleting software CPEs for Go binaries",
		),
		basicMigrationStep(
			`DELETE sc FROM software_cve sc
			JOIN software s ON s.id = sc.software_id
			WHERE s.source = 'go_binaries'`,
			"deleting software vulnerabilities for Go binaries",
		),
	}, tx)
}

func Down_20260909192031(tx *sql.Tx) error {
	return nil
}
