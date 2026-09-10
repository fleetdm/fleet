package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260909192031, Down_20260909192031)
}

// Go binaries are no longer translated to CPEs: they are matched by module path against the
// Go vulnerability database instead. The vulnerability cron only deletes a CPE for software it still iterates, and it no longer
// iterates this source, so those rows have to go here.
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
