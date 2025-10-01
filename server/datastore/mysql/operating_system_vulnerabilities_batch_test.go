package mysql

import (
	"context"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListVulnsByMultipleOSVersions(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"BatchQuery", testListVulnsByMultipleOSVersionsBatchQuery},
		{"EmptyInput", testListVulnsByMultipleOSVersionsEmptyInput},
		{"WithCVSS", testListVulnsByMultipleOSVersionsWithCVSS},
		{"WithTeamFilter", testListVulnsByMultipleOSVersionsWithTeamFilter},
		{"NonExistentOS", testListVulnsByMultipleOSVersionsNonExistentOS},
		{"MixedPlatforms", testListVulnsByMultipleOSVersionsMixedPlatforms},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testListVulnsByMultipleOSVersionsBatchQuery(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create test OS versions
	osVersions := setupTestOSVersionsWithVulns(t, ds, ctx)

	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, osVersions, false, nil)
	require.NoError(t, err)
	require.NotNil(t, vulnsMap)

	// Check we got results for all OS versions
	assert.Len(t, vulnsMap, 3)

	for i, osV := range osVersions {
		key := osV.NameOnly + "-" + osV.Version
		vulns, ok := vulnsMap[key]
		assert.True(t, ok, "Should have vulnerabilities for %s", key)
		assert.Len(t, vulns, 2, "Should have 2 vulnerabilities for %s", key)

		// Check CVE values
		expectedCVE1 := "CVE-2024-000" + string(rune('1'+i))
		expectedCVE2 := "CVE-2024-100" + string(rune('1'+i))

		cves := []string{vulns[0].CVE, vulns[1].CVE}
		assert.Contains(t, cves, expectedCVE1)
		assert.Contains(t, cves, expectedCVE2)
	}
}

func testListVulnsByMultipleOSVersionsEmptyInput(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, []fleet.OSVersion{}, false, nil)
	require.NoError(t, err)
	assert.Empty(t, vulnsMap)
}

func testListVulnsByMultipleOSVersionsWithCVSS(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create test OS versions
	osVersions := setupTestOSVersionsWithVulns(t, ds, ctx)

	// Add CVE metadata for one vulnerability
	_, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO cve_meta (cve, cvss_score, epss_probability, cisa_known_exploit, published, description)
		VALUES ('CVE-2024-0001', 7.5, 0.05, 0, NOW(), 'Test vulnerability')
		ON DUPLICATE KEY UPDATE cvss_score = VALUES(cvss_score)
	`)
	require.NoError(t, err)

	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, osVersions[:1], true, nil)
	require.NoError(t, err)

	key := osVersions[0].NameOnly + "-" + osVersions[0].Version
	vulns := vulnsMap[key]
	require.NotEmpty(t, vulns)

	// Find the vulnerability with metadata
	foundMetadata := false
	for _, vuln := range vulns {
		if vuln.CVE == "CVE-2024-0001" {
			foundMetadata = true
			assert.NotNil(t, vuln.CVSSScore)
			if vuln.CVSSScore != nil {
				assert.Equal(t, 7.5, **vuln.CVSSScore)
			}
			assert.NotNil(t, vuln.EPSSProbability)
			if vuln.EPSSProbability != nil {
				assert.InDelta(t, 0.05, **vuln.EPSSProbability, 0.001)
			}
			break
		}
	}
	assert.True(t, foundMetadata, "Should find vulnerability with CVSS metadata")
}

func testListVulnsByMultipleOSVersionsWithTeamFilter(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create test OS versions
	osVersions := setupTestOSVersionsWithVulns(t, ds, ctx)

	// Create kernel vulnerabilities with team-specific data
	setupKernelVulnsWithTeam(t, ds, ctx, osVersions[0])

	teamID := uint(1)

	// Test with team filter
	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, osVersions[:1], false, &teamID)
	require.NoError(t, err)

	key := osVersions[0].NameOnly + "-" + osVersions[0].Version
	vulns := vulnsMap[key]

	// Should have OS vulnerabilities plus kernel vulnerabilities for the team
	assert.GreaterOrEqual(t, len(vulns), 2)
}

func testListVulnsByMultipleOSVersionsNonExistentOS(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create OS versions that don't exist in the database
	// Don't set ID so it will be looked up by name/version
	nonExistentOS := []fleet.OSVersion{
		{
			NameOnly: "NonExistent",
			Version:  "1.0.0",
			Platform: "unknown",
		},
	}

	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, nonExistentOS, false, nil)
	require.NoError(t, err)

	// Should return empty map for non-existent OS
	assert.Empty(t, vulnsMap)
}

func testListVulnsByMultipleOSVersionsMixedPlatforms(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create OS versions for different platforms
	osVersions := []fleet.OSVersion{}
	platforms := []string{"linux", "darwin", "windows"}

	for i, platform := range platforms {
		osName := fmt.Sprintf("OS_%s", platform)
		osVersion := fmt.Sprintf("1.0.%d", i)

		os := fleet.OperatingSystem{
			Name:          osName,
			Version:       osVersion,
			Platform:      platform,
			KernelVersion: "test-kernel",
		}

		err := ds.UpdateHostOperatingSystem(ctx, uint(100+i), os) //nolint:gosec
		require.NoError(t, err)

		// Get the created OS
		stmt := `SELECT id, os_version_id FROM operating_systems WHERE name = ? AND version = ? LIMIT 1`
		var result struct {
			ID          uint `db:"id"`
			OSVersionID uint `db:"os_version_id"`
		}
		err = sqlx.GetContext(ctx, ds.writer(ctx), &result, stmt, osName, osVersion)
		require.NoError(t, err)

		osVersionObj := fleet.OSVersion{
			ID:          result.ID,
			OSVersionID: result.OSVersionID,
			NameOnly:    osName,
			Version:     osVersion,
			Platform:    platform,
		}
		osVersions = append(osVersions, osVersionObj)

		// Add a vulnerability for each OS
		vulns := []fleet.OSVulnerability{
			{
				OSID:              result.ID,
				CVE:               fmt.Sprintf("CVE-2024-PLAT-%d", i),
				ResolvedInVersion: ptr.String(osVersion + ".1"),
			},
		}
		_, err = ds.InsertOSVulnerabilities(ctx, vulns, fleet.UbuntuOVALSource)
		require.NoError(t, err)
	}

	// Test batch query with mixed platforms
	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, osVersions, false, nil)
	require.NoError(t, err)

	// Should get results for all platforms
	assert.Len(t, vulnsMap, 3)

	for i, osV := range osVersions {
		key := osV.NameOnly + "-" + osV.Version
		vulns, ok := vulnsMap[key]
		assert.True(t, ok, "Should have vulnerabilities for %s", key)
		assert.Len(t, vulns, 1)
		assert.Equal(t, fmt.Sprintf("CVE-2024-PLAT-%d", i), vulns[0].CVE)
	}
}

// Helper function to set up test OS versions with vulnerabilities
func setupTestOSVersionsWithVulns(t *testing.T, ds *Datastore, ctx context.Context) []fleet.OSVersion {
	var osVersions []fleet.OSVersion

	for i := 0; i < 3; i++ {
		osName := "Ubuntu"
		osVersion := "22.04." + string(rune('0'+i))

		os := fleet.OperatingSystem{
			Name:          osName,
			Version:       osVersion,
			Platform:      "linux",
			KernelVersion: "5.15.0-generic",
		}

		err := ds.UpdateHostOperatingSystem(ctx, uint(i+1), os) //nolint:gosec
		require.NoError(t, err)

		// Get the created OS
		stmt := `SELECT id, os_version_id FROM operating_systems WHERE name = ? AND version = ? LIMIT 1`
		var osID uint
		var osVersionID uint
		err = ds.writer(ctx).QueryRowContext(ctx, stmt, osName, osVersion).Scan(&osID, &osVersionID)
		require.NoError(t, err)

		osVersionObj := fleet.OSVersion{
			ID:          osID,
			OSVersionID: osVersionID,
			NameOnly:    osName,
			Version:     osVersion,
			Platform:    "linux",
		}
		osVersions = append(osVersions, osVersionObj)

		// Add vulnerabilities for this OS
		vulns := []fleet.OSVulnerability{
			{
				OSID:              osID,
				CVE:               "CVE-2024-000" + string(rune('1'+i)),
				ResolvedInVersion: ptr.String(osVersion + ".1"),
			},
			{
				OSID:              osID,
				CVE:               "CVE-2024-100" + string(rune('1'+i)),
				ResolvedInVersion: nil,
			},
		}
		_, err = ds.InsertOSVulnerabilities(ctx, vulns, fleet.UbuntuOVALSource)
		require.NoError(t, err)
	}

	return osVersions
}

// Helper function to set up kernel vulnerabilities with team-specific data
func setupKernelVulnsWithTeam(t *testing.T, ds *Datastore, ctx context.Context, osVersion fleet.OSVersion) {
	// Create a software title and software entry
	_, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO software_titles (name, source, is_kernel)
		VALUES ('linux-kernel', 'deb_packages', TRUE)
		ON DUPLICATE KEY UPDATE is_kernel = TRUE
	`)
	require.NoError(t, err)

	var titleID uint
	err = sqlx.GetContext(ctx, ds.writer(ctx), &titleID,
		`SELECT id FROM software_titles WHERE name = 'linux-kernel'`)
	require.NoError(t, err)

	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO software (name, version, source, title_id, checksum)
		VALUES ('linux-kernel', '5.15.0', 'deb_packages', ?, 'test-checksum')
		ON DUPLICATE KEY UPDATE id = id
	`, titleID)
	require.NoError(t, err)

	var softwareID uint
	err = sqlx.GetContext(ctx, ds.writer(ctx), &softwareID,
		`SELECT id FROM software WHERE name = 'linux-kernel' AND version = '5.15.0'`)
	require.NoError(t, err)

	// Add kernel CVE
	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO software_cve (software_id, cve, resolved_in_version)
		VALUES (?, 'CVE-2024-KERNEL-001', '5.15.1')
		ON DUPLICATE KEY UPDATE resolved_in_version = VALUES(resolved_in_version)
	`, softwareID)
	require.NoError(t, err)

	// Add kernel_host_counts for team 1
	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO kernel_host_counts (software_title_id, software_id, os_version_id, hosts_count, team_id)
		VALUES (?, ?, ?, 100, 1)
		ON DUPLICATE KEY UPDATE hosts_count = VALUES(hosts_count)
	`, titleID, softwareID, osVersion.OSVersionID)
	require.NoError(t, err)
}
