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
		{"MultipleLinuxOSWithManyKernels", testListVulnsByMultipleLinuxOSWithManyKernels},
		{"WithMaxVulnerabilities", testListVulnsByMultipleOSVersionsWithMaxVulnerabilities},
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

	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, osVersions, false, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, vulnsMap)

	// Check we got results for all OS versions
	assert.Len(t, vulnsMap, 3)

	for i, osV := range osVersions {
		key := osV.NameOnly + "-" + osV.Version
		vulnData, ok := vulnsMap[key]
		assert.True(t, ok, "Should have vulnerabilities for %s", key)
		assert.Len(t, vulnData.Vulnerabilities, 2, "Should have 2 vulnerabilities for %s", key)
		assert.Equal(t, 2, vulnData.Count, "Count should match vulnerabilities length")

		// Check CVE values
		expectedCVE1 := "CVE-2024-000" + string(rune('1'+i))
		expectedCVE2 := "CVE-2024-100" + string(rune('1'+i))

		cves := []string{vulnData.Vulnerabilities[0].CVE, vulnData.Vulnerabilities[1].CVE}
		assert.Contains(t, cves, expectedCVE1)
		assert.Contains(t, cves, expectedCVE2)
	}
}

func testListVulnsByMultipleOSVersionsEmptyInput(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, []fleet.OSVersion{}, false, nil, nil)
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

	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, osVersions[:1], true, nil, nil)
	require.NoError(t, err)

	key := osVersions[0].NameOnly + "-" + osVersions[0].Version
	vulnData := vulnsMap[key]
	require.NotEmpty(t, vulnData.Vulnerabilities)

	// Find the vulnerability with metadata
	foundMetadata := false
	for _, vuln := range vulnData.Vulnerabilities {
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

	// Create a Linux OS version for testing kernel vulnerabilities
	linuxOS := setupLinuxOSWithKernelVulns(t, ds, ctx)

	teamID := uint(1)

	// Test with team filter
	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, []fleet.OSVersion{linuxOS}, false, &teamID, nil)
	require.NoError(t, err)

	key := linuxOS.NameOnly + "-" + linuxOS.Version
	vulnData := vulnsMap[key]

	// Should have kernel vulnerabilities for the team
	assert.GreaterOrEqual(t, len(vulnData.Vulnerabilities), 1)
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

	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, nonExistentOS, false, nil, nil)
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

		// Add vulnerabilities based on platform type
		if platform == "linux" {
			// For Linux, add kernel vulnerabilities
			setupKernelVulnsWithTeam(t, ds, ctx, osVersionObj)
		} else {
			// For non-Linux, add OS vulnerabilities to operating_system_vulnerabilities table
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
	}

	// Test batch query with mixed platforms
	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, osVersions, false, nil, nil)
	require.NoError(t, err)

	// Should get results for all platforms
	assert.Len(t, vulnsMap, 3)

	for i, osV := range osVersions {
		key := osV.NameOnly + "-" + osV.Version
		vulnData, ok := vulnsMap[key]
		assert.True(t, ok, "Should have key in map for %s", key)

		// All platforms should have vulnerabilities
		assert.GreaterOrEqual(t, len(vulnData.Vulnerabilities), 1, "Platform %s should have at least 1 vulnerability", osV.Platform)

		if osV.Platform == "linux" {
			// Linux should have kernel vulnerability
			foundKernelCVE := false
			for _, vuln := range vulnData.Vulnerabilities {
				if vuln.CVE == "CVE-2024-KERNEL-001" {
					foundKernelCVE = true
					break
				}
			}
			assert.True(t, foundKernelCVE, "Linux should have kernel vulnerability CVE-2024-KERNEL-001")
		} else {
			// Non-Linux should have OS vulnerability
			assert.Len(t, vulnData.Vulnerabilities, 1, "Non-Linux platforms should have 1 vulnerability")
			assert.Equal(t, fmt.Sprintf("CVE-2024-PLAT-%d", i), vulnData.Vulnerabilities[0].CVE)
		}
	}

	// Test with maxVulnerabilities
	maxVulns := 1
	vulnsMapWithMax, err := ds.ListVulnsByMultipleOSVersions(ctx, osVersions, false, nil, &maxVulns)
	require.NoError(t, err)

	for i, osV := range osVersions {
		key := osV.NameOnly + "-" + osV.Version
		vulnData, ok := vulnsMapWithMax[key]
		assert.True(t, ok, "Should have key in map for %s", key)

		if osV.Platform == "linux" {
			// Linux should have count from kernel vulnerabilities
			assert.Greater(t, vulnData.Count, 0, "Linux platform count should be > 0 with maxVulnerabilities")
		} else {
			// Non-Linux should also have correct count, not 0
			// BUG REPRODUCTION: This will fail with current code because non-Linux count becomes 0
			// when len(linuxOSVersionMap) > 0
			assert.Equal(t, 1, vulnData.Count, "Non-Linux platform %s should have count=1, not 0 (bug: switch on global len(linuxOSVersionMap))", osV.Platform)
			assert.Len(t, vulnData.Vulnerabilities, 1, "Non-Linux should have 1 limited vulnerability")
			assert.Equal(t, fmt.Sprintf("CVE-2024-PLAT-%d", i), vulnData.Vulnerabilities[0].CVE)
		}
	}
}

// Helper function to set up test OS versions with vulnerabilities
// Uses non-Linux platforms since operating_system_vulnerabilities table doesn't contain Linux vulns
func setupTestOSVersionsWithVulns(t *testing.T, ds *Datastore, ctx context.Context) []fleet.OSVersion {
	var osVersions []fleet.OSVersion

	// Use non-Linux platforms for testing operating_system_vulnerabilities
	platforms := []string{"darwin", "windows", "chrome"}
	osNames := []string{"macOS", "Microsoft Windows 11 Enterprise", "Google Chrome OS"}

	for i := 0; i < 3; i++ {
		osName := osNames[i]
		osVersion := "1.0." + string(rune('0'+i))

		os := fleet.OperatingSystem{
			Name:          osName,
			Version:       osVersion,
			Platform:      platforms[i],
			KernelVersion: "test-kernel",
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
			Platform:    platforms[i],
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

// Helper function to set up a Linux OS with kernel vulnerabilities
func setupLinuxOSWithKernelVulns(t *testing.T, ds *Datastore, ctx context.Context) fleet.OSVersion {
	osName := "Ubuntu"
	osVersion := "22.04.0"

	os := fleet.OperatingSystem{
		Name:          osName,
		Version:       osVersion,
		Platform:      "linux",
		KernelVersion: "5.15.0-generic",
	}

	err := ds.UpdateHostOperatingSystem(ctx, 1, os)
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

	// Set up kernel vulnerabilities
	setupKernelVulnsWithTeam(t, ds, ctx, osVersionObj)

	return osVersionObj
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

	// Refresh the pre-aggregated table
	err = ds.RefreshOSVersionVulnerabilities(ctx)
	require.NoError(t, err)
}

// Test comprehensive scenario: multiple Linux OS versions with multiple kernels and shared vulnerabilities
func testListVulnsByMultipleLinuxOSWithManyKernels(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create software title for kernels
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

	// Create 3 Linux OS versions: Ubuntu 22.04.0, 22.04.1, 22.04.2
	var osVersions []fleet.OSVersion
	osVersionNames := []string{"22.04.0", "22.04.1", "22.04.2"}

	for idx, osVersionName := range osVersionNames {
		os := fleet.OperatingSystem{
			Name:          "Ubuntu",
			Version:       osVersionName,
			Platform:      "linux",
			KernelVersion: "5.15.0-generic",
		}

		err := ds.UpdateHostOperatingSystem(ctx, uint(1000+idx), os) //nolint:gosec
		require.NoError(t, err)

		// Get the created OS
		stmt := `SELECT id, os_version_id FROM operating_systems WHERE name = ? AND version = ? LIMIT 1`
		var osID uint
		var osVersionID uint
		err = ds.writer(ctx).QueryRowContext(ctx, stmt, "Ubuntu", osVersionName).Scan(&osID, &osVersionID)
		require.NoError(t, err)

		osVersions = append(osVersions, fleet.OSVersion{
			ID:          osID,
			OSVersionID: osVersionID,
			NameOnly:    "Ubuntu",
			Version:     osVersionName,
			Platform:    "linux",
		})

		// Create 3-4 different kernels for each OS version
		// Make kernel versions unique per OS to avoid sharing software entries
		baseKernelNum := idx * 10 // OS 0: 0-2, OS 1: 10-13, OS 2: 20-22
		kernelVersions := []string{
			fmt.Sprintf("5.15.0-%d", 100+baseKernelNum),
			fmt.Sprintf("5.15.0-%d", 101+baseKernelNum),
			fmt.Sprintf("5.15.0-%d", 102+baseKernelNum),
		}
		if idx == 1 {
			// Second OS version gets an extra kernel
			kernelVersions = append(kernelVersions, fmt.Sprintf("5.15.0-%d", 103+baseKernelNum))
		}

		for kernelIdx, kernelVersion := range kernelVersions {
			// Create kernel software entry
			_, err = ds.writer(ctx).ExecContext(ctx, `
				INSERT INTO software (name, version, source, title_id, checksum)
				VALUES ('linux-kernel', ?, 'deb_packages', ?, ?)
				ON DUPLICATE KEY UPDATE id = id
			`, kernelVersion, titleID, fmt.Sprintf("cs%d%d", idx, kernelIdx))
			require.NoError(t, err)

			var softwareID uint
			err = sqlx.GetContext(ctx, ds.writer(ctx), &softwareID,
				`SELECT id FROM software WHERE name = 'linux-kernel' AND version = ?`, kernelVersion)
			require.NoError(t, err)

			// Define CVEs for this kernel
			// Some CVEs are shared across kernels (CVE-2024-SHARED-*), some are unique
			cves := []string{
				"CVE-2024-SHARED-1", // Shared across all kernels
				"CVE-2024-SHARED-2", // Shared across all kernels
				fmt.Sprintf("CVE-2024-OS%d-KERNEL%d-1", idx, kernelIdx), // Unique to this OS+kernel
			}

			// Kernel 0 and 1 share an additional CVE
			if kernelIdx <= 1 {
				cves = append(cves, "CVE-2024-SHARED-K0-K1")
			}

			// Add CVEs for this kernel
			for _, cve := range cves {
				_, err = ds.writer(ctx).ExecContext(ctx, `
					INSERT INTO software_cve (software_id, cve, resolved_in_version)
					VALUES (?, ?, NULL)
					ON DUPLICATE KEY UPDATE resolved_in_version = VALUES(resolved_in_version)
				`, softwareID, cve)
				require.NoError(t, err)
			}

			// Add kernel_host_counts linking this kernel to this OS version
			_, err = ds.writer(ctx).ExecContext(ctx, `
				INSERT INTO kernel_host_counts (software_title_id, software_id, os_version_id, hosts_count, team_id)
				VALUES (?, ?, ?, 50, 0)
				ON DUPLICATE KEY UPDATE hosts_count = VALUES(hosts_count)
			`, titleID, softwareID, osVersionID)
			require.NoError(t, err)
		}
	}

	// Refresh the pre-aggregated table
	err = ds.RefreshOSVersionVulnerabilities(ctx)
	require.NoError(t, err)

	// Now query for vulnerabilities
	vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, osVersions, false, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, vulnsMap)

	// Should have results for all 3 OS versions
	assert.Len(t, vulnsMap, 3)

	// Verify deduplication: each OS version should have the shared CVEs plus unique ones
	for idx, osV := range osVersions {
		key := osV.NameOnly + "-" + osV.Version
		vulnData, ok := vulnsMap[key]
		assert.True(t, ok, "Should have vulnerabilities for %s", key)

		// Count expected CVEs
		// - 2 shared across all kernels (CVE-2024-SHARED-1, CVE-2024-SHARED-2)
		// - 1 shared between kernel 0 and 1 (CVE-2024-SHARED-K0-K1)
		// - N unique CVEs (one per kernel)
		numKernels := 3
		if idx == 1 {
			numKernels = 4 // Second OS version has 4 kernels
		}

		expectedUniqueCVEs := numKernels                // One unique CVE per kernel
		expectedTotalCVEs := 2 + 1 + expectedUniqueCVEs // 2 fully shared + 1 k0-k1 shared + unique

		assert.Len(t, vulnData.Vulnerabilities, expectedTotalCVEs, "OS version %s should have %d CVEs", key, expectedTotalCVEs)

		// Verify the shared CVEs are present
		cveSet := make(map[string]bool)
		for _, vuln := range vulnData.Vulnerabilities {
			cveSet[vuln.CVE] = true
		}

		assert.True(t, cveSet["CVE-2024-SHARED-1"], "Should have CVE-2024-SHARED-1")
		assert.True(t, cveSet["CVE-2024-SHARED-2"], "Should have CVE-2024-SHARED-2")
		assert.True(t, cveSet["CVE-2024-SHARED-K0-K1"], "Should have CVE-2024-SHARED-K0-K1")

		// Verify at least one unique CVE is present
		foundUnique := false
		expectedPrefix := fmt.Sprintf("CVE-2024-OS%d-KERNEL", idx)
		for cve := range cveSet {
			if len(cve) > len(expectedPrefix) && cve[:len(expectedPrefix)] == expectedPrefix {
				foundUnique = true
				break
			}
		}
		assert.True(t, foundUnique, "Should have at least one unique CVE for OS %d", idx)
	}
}

func testListVulnsByMultipleOSVersionsWithMaxVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create test OS versions with vulnerabilities
	osVersions := setupTestOSVersionsWithVulns(t, ds, ctx)
	linuxOS := setupLinuxOSWithKernelVulns(t, ds, ctx)

	testCases := []struct {
		name           string
		maxVulns       *int
		osVersions     []fleet.OSVersion
		expectError    bool
		errorMessage   string
		validateResult func(t *testing.T, vulnsMap map[string]fleet.OSVulnerabilitiesWithCount)
	}{
		{
			name:         "negative value returns error",
			maxVulns:     ptr.Int(-1),
			osVersions:   osVersions,
			expectError:  true,
			errorMessage: "max_vulnerabilities must be >= 0",
		},
		{
			name:       "max = 0 returns empty array with count",
			maxVulns:   ptr.Int(0),
			osVersions: osVersions,
			validateResult: func(t *testing.T, vulnsMap map[string]fleet.OSVulnerabilitiesWithCount) {
				assert.Len(t, vulnsMap, 3)
				for _, osV := range osVersions {
					key := osV.NameOnly + "-" + osV.Version
					vulnData, ok := vulnsMap[key]
					assert.True(t, ok, "Should have entry for %s", key)
					assert.Len(t, vulnData.Vulnerabilities, 0, "Should have 0 vulnerabilities when max=0")
					assert.Equal(t, 2, vulnData.Count, "Count should still show total of 2 vulnerabilities")
				}
			},
		},
		{
			name:       "max = 1 limits to 1 vulnerability",
			maxVulns:   ptr.Int(1),
			osVersions: osVersions,
			validateResult: func(t *testing.T, vulnsMap map[string]fleet.OSVulnerabilitiesWithCount) {
				assert.Len(t, vulnsMap, 3)
				for _, osV := range osVersions {
					key := osV.NameOnly + "-" + osV.Version
					vulnData, ok := vulnsMap[key]
					assert.True(t, ok, "Should have vulnerabilities for %s", key)
					assert.Len(t, vulnData.Vulnerabilities, 1, "Should have exactly 1 vulnerability (limited)")
					assert.Equal(t, 2, vulnData.Count, "Count should show total of 2 vulnerabilities")
				}
			},
		},
		{
			name:       "max exceeds total returns all vulnerabilities",
			maxVulns:   ptr.Int(10),
			osVersions: osVersions,
			validateResult: func(t *testing.T, vulnsMap map[string]fleet.OSVulnerabilitiesWithCount) {
				assert.Len(t, vulnsMap, 3)
				for _, osV := range osVersions {
					key := osV.NameOnly + "-" + osV.Version
					vulnData, ok := vulnsMap[key]
					assert.True(t, ok, "Should have vulnerabilities for %s", key)
					assert.Len(t, vulnData.Vulnerabilities, 2, "Should have all 2 vulnerabilities when max exceeds total")
					assert.Equal(t, 2, vulnData.Count, "Count should show 2 vulnerabilities")
				}
			},
		},
		{
			name:       "max = 1 for Linux kernel vulnerabilities",
			maxVulns:   ptr.Int(1),
			osVersions: []fleet.OSVersion{linuxOS},
			validateResult: func(t *testing.T, vulnsMap map[string]fleet.OSVulnerabilitiesWithCount) {
				key := linuxOS.NameOnly + "-" + linuxOS.Version
				vulnData, ok := vulnsMap[key]
				assert.True(t, ok, "Should have vulnerabilities for Linux OS")
				assert.Len(t, vulnData.Vulnerabilities, 1, "Should have exactly 1 vulnerability (limited)")
				assert.GreaterOrEqual(t, vulnData.Count, 1, "Count should be at least 1")
				if vulnData.Count > 1 {
					assert.Equal(t, 1, len(vulnData.Vulnerabilities), "Should be limited to 1 vulnerability even though total is %d", vulnData.Count)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vulnsMap, err := ds.ListVulnsByMultipleOSVersions(ctx, tc.osVersions, false, nil, tc.maxVulns)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMessage)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, vulnsMap)
			tc.validateResult(t, vulnsMap)
		})
	}
}
