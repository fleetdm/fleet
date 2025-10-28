package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251028140000, Down_20251028140000)
}

func Up_20251028140000(tx *sql.Tx) error {
	// Create the pre-aggregated OS version vulnerabilities table
	// This table contains ONLY Linux kernel vulnerabilities
	// team_id semantics:
	//   NULL  = "all teams" (pre-aggregated across all teams)
	//   0     = "no team" (hosts without team assignment)
	//   >0    = specific team ID
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS operating_system_version_vulnerabilities (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			os_version_id INT UNSIGNED NOT NULL,
			cve VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			team_id INT UNSIGNED DEFAULT NULL,
			source SMALLINT DEFAULT 0,
			resolved_in_version VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (id),
		    -- for inserts
			UNIQUE KEY idx_os_version_vulnerabilities_unq_os_version_team_cve ((IFNULL(CAST(team_id AS SIGNED), -1)), os_version_id, cve),
		    -- for reads
		    KEY idx_os_version_vulnerabilities_os_version_team_cve (team_id, os_version_id, cve),
		    -- for cleanup
			KEY idx_os_version_vulnerabilities_updated_at (updated_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	if err != nil {
		return fmt.Errorf("creating operating_system_version_vulnerabilities table: %w", err)
	}

	// Backfill the table with existing data
	// This runs as part of the migration to populate historical data
	// Note: This table contains ONLY Linux kernel vulnerabilities
	// Non-Linux OS vulnerabilities continue to be queried from operating_system_vulnerabilities table
	fmt.Printf("[INFO] Starting backfill of operating_system_version_vulnerabilities table\n")

	// Backfill per-team Linux kernel vulnerabilities
	fmt.Printf("[INFO] Backfilling per-team Linux kernel vulnerabilities...\n")
	result, err := tx.Exec(`
		INSERT INTO operating_system_version_vulnerabilities
			(os_version_id, cve, team_id, source, resolved_in_version, created_at)
		SELECT
			khc.os_version_id,
			sc.cve,
			khc.team_id,
			sc.source,
			sc.resolved_in_version,
			MIN(sc.created_at) as created_at
		FROM kernel_host_counts khc
		JOIN software_cve sc ON sc.software_id = khc.software_id
		WHERE khc.hosts_count > 0
		GROUP BY khc.os_version_id, sc.cve, khc.team_id, sc.source, sc.resolved_in_version
		ON DUPLICATE KEY UPDATE
			source = VALUES(source),
			resolved_in_version = VALUES(resolved_in_version),
			created_at = VALUES(created_at),
			updated_at = CURRENT_TIMESTAMP(6)
	`)
	if err != nil {
		return fmt.Errorf("backfilling per-team Linux kernel vulnerabilities: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("[INFO] Backfilled %d per-team Linux kernel vulnerability entries\n", rowsAffected)

	// Backfill "all teams" aggregated Linux kernel vulnerabilities
	// team_id = NULL represents pre-aggregated data across all teams
	fmt.Printf("[INFO] Backfilling 'all teams' aggregated Linux kernel vulnerabilities...\n")
	result, err = tx.Exec(`
		INSERT INTO operating_system_version_vulnerabilities
			(os_version_id, cve, team_id, source, resolved_in_version, created_at)
		SELECT
			khc.os_version_id,
			sc.cve,
			NULL as team_id,
			sc.source,
			sc.resolved_in_version,
			MIN(sc.created_at) as created_at
		FROM kernel_host_counts khc
		JOIN software_cve sc ON sc.software_id = khc.software_id
		WHERE khc.hosts_count > 0
		GROUP BY khc.os_version_id, sc.cve, sc.source, sc.resolved_in_version
		ON DUPLICATE KEY UPDATE
			source = VALUES(source),
			resolved_in_version = VALUES(resolved_in_version),
			created_at = VALUES(created_at),
			updated_at = CURRENT_TIMESTAMP(6)
	`)
	if err != nil {
		return fmt.Errorf("backfilling 'all teams' Linux kernel vulnerabilities: %w", err)
	}
	rowsAffected, _ = result.RowsAffected()
	fmt.Printf("[INFO] Backfilled %d 'all teams' Linux kernel vulnerability entries\n", rowsAffected)

	fmt.Printf("[INFO] Backfill of operating_system_version_vulnerabilities table completed successfully\n")

	return nil
}

func Down_20251028140000(_ *sql.Tx) error {
	return nil
}
