package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250905190030, Down_20250905190030)
}

func Up_20250905190030(tx *sql.Tx) error {
	// source 2 is OVAL; we switched to get vuln data to ALAS via goval-dictionary, so
	// OVAL vulns need to be purged for CentOS and Fedora packages
	_, err := tx.Exec(`
		 DELETE software_cve FROM software_cve JOIN software ON
		        software.id = software_cve.software_id AND software.vendor IN ('CentOS', 'Fedora Project') AND software_cve.source = 2
		`)
	if err != nil {
		return fmt.Errorf("failed to clear CentOS and Fedora OVAL false-positives: %w", err)
	}
	return nil
}

func Down_20250905190030(tx *sql.Tx) error {
	return nil
}
