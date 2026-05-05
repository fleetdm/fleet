package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251117020000, Down_20251117020000)
}

func Up_20251117020000(tx *sql.Tx) error {
	// Drop the old index that doesn't include global_stats
	_, err := tx.Exec(`
			DROP INDEX idx_software_host_counts_team_id_hosts_count_software_id
			ON software_host_counts
		`)
	if err != nil {
		return fmt.Errorf("failed to drop old index: %w", err)
	}

	// Create new optimized index with global_stats and DESC on hosts_count
	// This allows MySQL to:
	// 1. Seek directly to team_id + global_stats combination
	// 2. Read rows in descending hosts_count order (already sorted)
	// 3. Stop after finding LIMIT rows
	_, err = tx.Exec(`
			CREATE INDEX idx_software_host_counts_team_global_hosts_desc
			ON software_host_counts (team_id, global_stats, hosts_count DESC, software_id)
		`)
	if err != nil {
		return fmt.Errorf("failed to create new index: %w", err)
	}

	return nil
}

func Down_20251117020000(_ *sql.Tx) error {
	return nil
}
