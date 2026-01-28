package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250817154557, Down_20250817154557)
}

func Up_20250817154557(tx *sql.Tx) error {
	// Add index to optimize kernel vulnerability queries
	// This index supports efficient joins between software_cve and kernel_host_counts
	// when filtering by os_version_id and hosts_count > 0

	createIndexStmt := `
		CREATE INDEX idx_kernel_host_counts_os_version_software
		ON kernel_host_counts (os_version_id, software_id, hosts_count)
	`

	if _, err := tx.Exec(createIndexStmt); err != nil {
		return fmt.Errorf("failed to create kernel_host_counts index for vulnerability queries: %w", err)
	}

	return nil
}

func Down_20250817154557(_ *sql.Tx) error {
	return nil
}
