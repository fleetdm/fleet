package tables

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20251027140000(t *testing.T) {
	db := applyUpToPrev(t)

	// Set up test data BEFORE running the migration
	// This tests the backfill capability

	// Insert test software for Linux kernels
	_, err := db.Exec(`
		INSERT INTO software (id, name, version, source, bundle_identifier, checksum)
		VALUES (1, 'linux', '5.15.0-1', 'programs', '', 'abc123'),
		       (2, 'linux', '6.1.0-1', 'programs', '', 'def456')
	`)
	require.NoError(t, err)

	// Insert CVEs for the software
	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO software_cve (software_id, cve, source, resolved_in_version, created_at)
		VALUES
			(1, 'CVE-2024-0001', 0, '5.15.0-2', ?),
			(1, 'CVE-2024-0002', 1, '5.15.0-3', ?),
			(2, 'CVE-2024-0003', 0, '6.1.0-2', ?)
	`, now, now, now)
	require.NoError(t, err)

	// Insert kernel host counts (linking kernels to OS versions)
	// team_id 0 represents "no team" (hosts without team assignment)
	_, err = db.Exec(`
		INSERT INTO kernel_host_counts (team_id, os_version_id, software_id, hosts_count)
		VALUES
			(0, 100, 1, 5),
			(0, 101, 2, 3)
	`)
	require.NoError(t, err)

	// Apply migration - this should backfill the data
	applyNext(t, db)

	// Test backfill: Verify per-team Linux kernel vulnerabilities were copied
	t.Run("backfill per-team Linux kernel vulnerabilities", func(t *testing.T) {
		type vuln struct {
			OSVersionID       int            `db:"os_version_id"`
			CVE               string         `db:"cve"`
			TeamID            *uint          `db:"team_id"`
			Source            int            `db:"source"`
			ResolvedInVersion sql.NullString `db:"resolved_in_version"`
		}

		var vulns []vuln
		err := db.Select(&vulns, `
			SELECT os_version_id, cve, team_id, source, resolved_in_version
			FROM operating_system_version_vulnerabilities
			WHERE os_version_id IN (100, 101) AND team_id IS NOT NULL
			ORDER BY os_version_id, cve
		`)
		require.NoError(t, err)
		require.Len(t, vulns, 3, "Should have 3 per-team Linux kernel vulnerabilities")

		// Verify first OS version (100) has 2 CVEs from software_id 1
		require.Equal(t, 100, vulns[0].OSVersionID)
		require.Equal(t, "CVE-2024-0001", vulns[0].CVE)
		require.NotNil(t, vulns[0].TeamID)
		require.Equal(t, uint(0), *vulns[0].TeamID) // team_id = 0 from kernel_host_counts
		require.Equal(t, 0, vulns[0].Source)
		require.True(t, vulns[0].ResolvedInVersion.Valid)
		require.Equal(t, "5.15.0-2", vulns[0].ResolvedInVersion.String)

		require.Equal(t, 100, vulns[1].OSVersionID)
		require.Equal(t, "CVE-2024-0002", vulns[1].CVE)
		require.NotNil(t, vulns[1].TeamID)
		require.Equal(t, uint(0), *vulns[1].TeamID)
		require.Equal(t, 1, vulns[1].Source)
		require.True(t, vulns[1].ResolvedInVersion.Valid)
		require.Equal(t, "5.15.0-3", vulns[1].ResolvedInVersion.String)

		// Verify second OS version (101) has 1 CVE from software_id 2
		require.Equal(t, 101, vulns[2].OSVersionID)
		require.Equal(t, "CVE-2024-0003", vulns[2].CVE)
		require.NotNil(t, vulns[2].TeamID)
		require.Equal(t, uint(0), *vulns[2].TeamID)
		require.Equal(t, 0, vulns[2].Source)
		require.True(t, vulns[2].ResolvedInVersion.Valid)
		require.Equal(t, "6.1.0-2", vulns[2].ResolvedInVersion.String)
	})

	// Test backfill: Verify "all teams" aggregated Linux kernel vulnerabilities
	t.Run("backfill 'all teams' aggregated Linux kernel vulnerabilities", func(t *testing.T) {
		type vuln struct {
			OSVersionID       int            `db:"os_version_id"`
			CVE               string         `db:"cve"`
			TeamID            *uint          `db:"team_id"`
			Source            int            `db:"source"`
			ResolvedInVersion sql.NullString `db:"resolved_in_version"`
		}

		var vulns []vuln
		err := db.Select(&vulns, `
			SELECT os_version_id, cve, team_id, source, resolved_in_version
			FROM operating_system_version_vulnerabilities
			WHERE os_version_id IN (100, 101) AND team_id IS NULL
			ORDER BY os_version_id, cve
		`)
		require.NoError(t, err)
		require.Len(t, vulns, 3, "Should have 3 'all teams' aggregated vulnerabilities")

		// Verify first OS version (100) has 2 CVEs aggregated across all teams
		require.Equal(t, 100, vulns[0].OSVersionID)
		require.Equal(t, "CVE-2024-0001", vulns[0].CVE)
		require.Nil(t, vulns[0].TeamID) // team_id = NULL means "all teams"
		require.Equal(t, 0, vulns[0].Source)
		require.True(t, vulns[0].ResolvedInVersion.Valid)
		require.Equal(t, "5.15.0-2", vulns[0].ResolvedInVersion.String)

		require.Equal(t, 100, vulns[1].OSVersionID)
		require.Equal(t, "CVE-2024-0002", vulns[1].CVE)
		require.Nil(t, vulns[1].TeamID) // team_id = NULL means "all teams"
		require.Equal(t, 1, vulns[1].Source)
		require.True(t, vulns[1].ResolvedInVersion.Valid)
		require.Equal(t, "5.15.0-3", vulns[1].ResolvedInVersion.String)

		// Verify second OS version (101) has 1 CVE aggregated across all teams
		require.Equal(t, 101, vulns[2].OSVersionID)
		require.Equal(t, "CVE-2024-0003", vulns[2].CVE)
		require.Nil(t, vulns[2].TeamID) // team_id = NULL means "all teams"
		require.Equal(t, 0, vulns[2].Source)
		require.True(t, vulns[2].ResolvedInVersion.Valid)
		require.Equal(t, "6.1.0-2", vulns[2].ResolvedInVersion.String)
	})

	// Test unique index constraint
	t.Run("unique index prevents duplicates", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO operating_system_version_vulnerabilities (os_version_id, cve, team_id, source, resolved_in_version, created_at)
			VALUES (100, 'CVE-2024-0001', 0, 0, '5.15.0-4', NOW())
		`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Duplicate entry")
	})

}
