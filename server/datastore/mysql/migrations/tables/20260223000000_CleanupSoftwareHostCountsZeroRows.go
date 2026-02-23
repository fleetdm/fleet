package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260223000000, Down_20260223000000)
}

func Up_20260223000000(tx *sql.Tx) error {
	// Delete any accumulated zero-count rows from software_host_counts and software_titles_host_counts.
	// After this migration, the sync process uses an atomic swap table pattern that never produces zero-count rows.
	// Add CHECK constraints to prevent zero-count rows from being inserted in the future.

	return withSteps([]migrationStep{
		basicMigrationStep(
			`DELETE FROM software_host_counts WHERE hosts_count = 0`,
			"deleting zero-count rows from software_host_counts",
		),
		basicMigrationStep(
			`ALTER TABLE software_host_counts ADD CONSTRAINT ck_software_host_counts_positive CHECK (hosts_count > 0)`,
			"adding CHECK constraint to software_host_counts",
		),
		basicMigrationStep(
			`DELETE FROM software_titles_host_counts WHERE hosts_count = 0`,
			"deleting zero-count rows from software_titles_host_counts",
		),
		basicMigrationStep(
			`ALTER TABLE software_titles_host_counts ADD CONSTRAINT ck_software_titles_host_counts_positive CHECK (hosts_count > 0)`,
			"adding CHECK constraint to software_titles_host_counts",
		),
	}, tx)
}

func Down_20260223000000(tx *sql.Tx) error {
	return nil
}
