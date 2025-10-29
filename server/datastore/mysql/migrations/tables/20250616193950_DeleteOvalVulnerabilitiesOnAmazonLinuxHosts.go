package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250616193950, Down_20250616193950)
}

func Up_20250616193950(tx *sql.Tx) error {
	// source 2 is OVAL; as of v4.56.0 Fleet switched from (incorrect) RHEL 6 OVAL
	// as a source for Amazon Linux 2 vuln data to ALAS via goval-dictionary, so
	// OVAL vulns need to be purged for Amazon Linux packages
	_, err := tx.Exec(`
	 DELETE software_cve FROM software_cve JOIN software ON
	        software.id = software_cve.software_id AND software.vendor = 'amazon linux' AND software_cve.source = 2
	`)
	if err != nil {
		return fmt.Errorf("failed to clear Amazon Linux OVAL false-positives: %w", err)
	}

	return nil
}

func Down_20250616193950(tx *sql.Tx) error {
	return nil
}
