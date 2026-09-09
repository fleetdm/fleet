package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260909192031, Down_20260909192031)
}

// Go binaries are no longer translated to CPEs: they are matched by module path against the
// Go vulnerability database instead. CPEs already stored for them were guesses built from the
// binary's name, which is unrelated to its module, so a binary named "air" carried the Adobe
// AIR CPE and every Adobe AIR CVE with it.
//
// The vulnerability cron only deletes a CPE for software it still iterates, and it no longer
// iterates this source, so these rows have to go here. Their software_cve rows are dropped by
// the cron's own stale-vulnerability pass once the CPE they came from is gone.
func Up_20260909192031(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		DELETE cpe FROM software_cpe cpe
		JOIN software s ON s.id = cpe.software_id
		WHERE s.source = 'go_binaries'
	`); err != nil {
		return fmt.Errorf("deleting software CPEs for Go binaries: %w", err)
	}

	return nil
}

func Down_20260909192031(tx *sql.Tx) error {
	return nil
}
