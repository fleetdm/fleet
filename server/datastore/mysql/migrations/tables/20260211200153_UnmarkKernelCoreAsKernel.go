package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260211200153, Down_20260211200153)
}

func Up_20260211200153(tx *sql.Tx) error {
	// Previously, both "kernel" and "kernel-core" were marked as kernels for RPM-based distributions.
	// However, on RHEL-family systems (RHEL, AlmaLinux, CentOS, Rocky, Fedora), both packages exist:
	// - "kernel-core"
	// - "kernel"
	//
	// To avoid showing duplicate kernels in the OS versions API, we now only mark "kernel" as the
	// kernel package for all RPM-based distributions (including Amazon Linux and RHEL-family).
	// This unmarks "kernel-core" as a kernel.
	if _, err := tx.Exec(`
UPDATE software_titles
SET is_kernel = 0
WHERE name = 'kernel-core' AND source = 'rpm_packages'
	`); err != nil {
		return fmt.Errorf("failed to unmark kernel-core as kernel: %w", err)
	}

	// Clean up kernel_host_counts entries for kernel-core titles that are no longer kernels
	if _, err := tx.Exec(`
DELETE khc FROM kernel_host_counts khc
JOIN software_titles st ON st.id = khc.software_title_id
WHERE st.name = 'kernel-core' AND st.source = 'rpm_packages'
	`); err != nil {
		return fmt.Errorf("failed to clean up kernel_host_counts for kernel-core: %w", err)
	}

	return nil
}

func Down_20260211200153(tx *sql.Tx) error {
	return nil
}
