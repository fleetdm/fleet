package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260902205155, Down_20260902205155)
}

func Up_20260902205155(tx *sql.Tx) error {
	// Add denormalized sort columns to the hosts table. These columns cache
	// values from LEFT-JOINed adjacency tables so that ORDER BY can use an
	// index on hosts instead of requiring a full filesort across the join.
	//
	// ADD COLUMN with a DEFAULT is an INSTANT DDL operation in MySQL 8.0 and
	// does not rewrite existing rows.
	_, err := tx.Exec(`
		ALTER TABLE hosts
			ADD COLUMN sort_seen_time              TIMESTAMP    NULL,
			ADD COLUMN sort_software_updated_at     TIMESTAMP    NULL,
			ADD COLUMN sort_gigs_disk_space_available   DECIMAL(10,2) NOT NULL DEFAULT 0,
			ADD COLUMN sort_gigs_total_disk_space       DECIMAL(10,2) NOT NULL DEFAULT 0,
			ADD COLUMN sort_percent_disk_space_available DECIMAL(10,2) NOT NULL DEFAULT 0,
			ADD COLUMN sort_total_issues_count      INT UNSIGNED NOT NULL DEFAULT 0,
			ADD COLUMN sort_team_name               VARCHAR(255) NULL,
			ALGORITHM=INSTANT
	`)
	if err != nil {
		return fmt.Errorf("add sort columns to hosts: %w", err)
	}

	// Batch-populate the new columns from the existing adjacency tables.
	// We process in batches of 10 000 to bound lock duration and undo-log
	// size on large deployments.
	const batchSize = 10_000

	// Populate sort_seen_time from host_seen_times (fallback to created_at).
	if err := batchUpdateSortColumn(tx, batchSize,
		`UPDATE hosts h
		 LEFT JOIN host_seen_times hst ON h.id = hst.host_id
		 SET h.sort_seen_time = COALESCE(hst.seen_time, h.created_at)
		 WHERE h.id BETWEEN ? AND ?`,
	); err != nil {
		return fmt.Errorf("populate sort_seen_time: %w", err)
	}

	// Populate sort_software_updated_at from host_updates (fallback to created_at).
	if err := batchUpdateSortColumn(tx, batchSize,
		`UPDATE hosts h
		 LEFT JOIN host_updates hu ON h.id = hu.host_id
		 SET h.sort_software_updated_at = COALESCE(hu.software_updated_at, h.created_at)
		 WHERE h.id BETWEEN ? AND ?`,
	); err != nil {
		return fmt.Errorf("populate sort_software_updated_at: %w", err)
	}

	// Populate disk space columns from host_disks.
	if err := batchUpdateSortColumn(tx, batchSize,
		`UPDATE hosts h
		 LEFT JOIN host_disks hd ON h.id = hd.host_id
		 SET h.sort_gigs_disk_space_available   = COALESCE(hd.gigs_disk_space_available, 0),
		     h.sort_gigs_total_disk_space       = COALESCE(hd.gigs_total_disk_space, 0),
		     h.sort_percent_disk_space_available = COALESCE(hd.percent_disk_space_available, 0)
		 WHERE h.id BETWEEN ? AND ?`,
	); err != nil {
		return fmt.Errorf("populate sort disk columns: %w", err)
	}

	// Populate sort_total_issues_count from host_issues.
	if err := batchUpdateSortColumn(tx, batchSize,
		`UPDATE hosts h
		 LEFT JOIN host_issues hi ON h.id = hi.host_id
		 SET h.sort_total_issues_count = COALESCE(hi.total_issues_count, 0)
		 WHERE h.id BETWEEN ? AND ?`,
	); err != nil {
		return fmt.Errorf("populate sort_total_issues_count: %w", err)
	}

	// Populate sort_team_name from teams.
	if err := batchUpdateSortColumn(tx, batchSize,
		`UPDATE hosts h
		 LEFT JOIN teams t ON h.team_id = t.id
		 SET h.sort_team_name = t.name
		 WHERE h.id BETWEEN ? AND ?`,
	); err != nil {
		return fmt.Errorf("populate sort_team_name: %w", err)
	}

	// Create indexes. These are INPLACE online DDL operations that do not
	// block concurrent DML.
	indexes := []string{
		`CREATE INDEX idx_hosts_sort_seen_time ON hosts (sort_seen_time)`,
		`CREATE INDEX idx_hosts_sort_software_updated_at ON hosts (sort_software_updated_at)`,
		`CREATE INDEX idx_hosts_sort_gigs_disk_space_available ON hosts (sort_gigs_disk_space_available)`,
		`CREATE INDEX idx_hosts_sort_gigs_total_disk_space ON hosts (sort_gigs_total_disk_space)`,
		`CREATE INDEX idx_hosts_sort_percent_disk_space_available ON hosts (sort_percent_disk_space_available)`,
		`CREATE INDEX idx_hosts_sort_total_issues_count ON hosts (sort_total_issues_count)`,
		`CREATE INDEX idx_hosts_sort_team_name ON hosts (sort_team_name)`,
	}
	for _, ddl := range indexes {
		if _, err := tx.Exec(ddl); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	return nil
}

// batchUpdateSortColumn executes stmt in batches of batchSize rows at a time.
// The statement must contain two ? placeholders for the inclusive ID range.
func batchUpdateSortColumn(tx *sql.Tx, batchSize int, stmt string) error {
	var maxID int
	if err := tx.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM hosts`).Scan(&maxID); err != nil {
		return fmt.Errorf("get max host id: %w", err)
	}
	for start := 1; start <= maxID; start += batchSize {
		end := start + batchSize - 1
		if _, err := tx.Exec(stmt, start, end); err != nil {
			return fmt.Errorf("batch [%d, %d]: %w", start, end, err)
		}
	}
	return nil
}

func Down_20260902205155(tx *sql.Tx) error {
	return nil
}
