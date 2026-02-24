package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260225000000, Down_20260225000000)
}

func Up_20260225000000(tx *sql.Tx) error {
	// Drop the old index that doesn't include global_stats or DESC on hosts_count
	_, err := tx.Exec(`
		DROP INDEX idx_software_titles_host_counts_team_counts_title
		ON software_titles_host_counts
	`)
	if err != nil {
		return fmt.Errorf("failed to drop old index: %w", err)
	}

	// Create new covering index that includes global_stats.
	// The optimized title query filters by (team_id, global_stats) then joins
	// software_titles for the name-based secondary sort, so the index is used
	// for filtering, not ordering.
	_, err = tx.Exec(`
		CREATE INDEX idx_software_titles_host_counts_team_global_hosts
		ON software_titles_host_counts (team_id, global_stats, hosts_count, software_title_id)
	`)
	if err != nil {
		return fmt.Errorf("failed to create new index: %w", err)
	}

	return nil
}

func Down_20260225000000(_ *sql.Tx) error {
	return nil
}
