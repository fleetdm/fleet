package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260316120000, Down_20260316120000)
}

func Up_20260316120000(tx *sql.Tx) error {
	// The FK is unnecessary because kernel_host_counts is fully rebuilt on each vulnerability cron run via a swap table,
	// and CREATE TABLE ... LIKE does not copy foreign keys. Keeping the FK would require restoring it after every swap,
	// which can fail if a referenced software_title is deleted between the SELECT and the ALTER TABLE.
	// Orphaned rows are harmless because API queries (ListKernelsByOS) JOIN back to software_titles, excluding any
	// rows that reference deleted titles.
	if _, err := tx.Exec(`ALTER TABLE kernel_host_counts DROP FOREIGN KEY kernel_host_counts_ibfk_1`); err != nil {
		return fmt.Errorf("dropping kernel_host_counts foreign key: %w", err)
	}
	return nil
}

func Down_20260316120000(_ *sql.Tx) error {
	return nil
}
