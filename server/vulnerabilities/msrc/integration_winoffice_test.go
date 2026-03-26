package msrc_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc"
	msrcapps "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/apps"
	"github.com/stretchr/testify/require"
)

func TestIntegrationsWinOfficeGenerator(t *testing.T) {
	nettest.Run(t)

	client := fleethttp.NewClient(fleethttp.WithTimeout(60 * time.Second))

	t.Run("scrapes security updates from Microsoft Learn", func(t *testing.T) {
		releases, err := msrc.ScrapeWinOfficeSecurityUpdates(client)
		require.NoError(t, err)
		require.NotEmpty(t, releases, "should have at least one security release")

		// Verify first release has expected structure
		firstRelease := releases[0]
		require.NotEmpty(t, firstRelease.Date, "release should have a date")
		require.NotEmpty(t, firstRelease.Branches, "release should have version branches")
		require.NotEmpty(t, firstRelease.CVEs, "release should have CVEs")

		// Verify we capture multiple version branches per release
		require.GreaterOrEqual(t, len(firstRelease.Branches), 3,
			"should capture multiple version branches (Current Channel, Monthly Enterprise, etc.)")

		// Verify version branch structure
		for _, branch := range firstRelease.Branches {
			require.NotEmpty(t, branch.Version, "branch should have version")
			require.NotEmpty(t, branch.BuildPrefix, "branch should have build prefix")
			require.NotEmpty(t, branch.FullBuild, "branch should have full build")
			require.Regexp(t, regexp.MustCompile(`^\d{4}$`), branch.Version,
				"version should be YYMM format (4 digits)")
			require.Regexp(t, regexp.MustCompile(`^\d+\.\d+$`), branch.FullBuild,
				"full build should be prefix.suffix format")
		}

		// Verify CVE format
		cvePattern := regexp.MustCompile(`^CVE-\d{4}-\d+$`)
		for _, cve := range firstRelease.CVEs {
			require.Regexp(t, cvePattern, cve, "CVE should match expected format")
		}
	})

	t.Run("builds WinOfficeBulletin with mappings", func(t *testing.T) {
		bulletin, err := msrc.FetchWinOfficeBulletin(client)
		require.NoError(t, err)

		// Verify bulletin structure
		require.NotEmpty(t, bulletin.BuildPrefixToVersion, "should have build prefix mappings")
		require.NotEmpty(t, bulletin.CVEToFixedBuilds, "should have CVE to fixed builds")
		require.NotEmpty(t, bulletin.SupportedVersions, "should have supported versions")

		// Verify supported versions are in YYMM format
		for _, version := range bulletin.SupportedVersions {
			require.Regexp(t, regexp.MustCompile(`^\d{4}$`), version,
				"supported version should be YYMM format")
		}

		// Verify CVE mappings have proper build versions
		for cve, fixedBuilds := range bulletin.CVEToFixedBuilds {
			require.NotEmpty(t, cve, "CVE should not be empty")
			require.NotEmpty(t, fixedBuilds, "CVE should have at least one fixed build")
			for version, build := range fixedBuilds {
				require.NotEmpty(t, version, "version should not be empty")
				require.NotEmpty(t, build, "build should not be empty")
				require.Regexp(t, regexp.MustCompile(`^\d+\.\d+$`), build,
					"build should be prefix.suffix format")
			}
		}
	})

	t.Run("generates correctly formatted JSON file", func(t *testing.T) {
		bulletin, err := msrc.FetchWinOfficeBulletin(client)
		require.NoError(t, err)

		// Convert to AppBulletinFile
		bulletinFile := msrc.ConvertWinOfficeToAppBulletin(bulletin)
		require.NotEmpty(t, bulletinFile.Versions, "should have versions")
		require.NotEmpty(t, bulletinFile.BuildPrefixes, "should have build prefixes")
		require.Equal(t, 1, bulletinFile.Version, "should have version 1")

		// Verify at least one supported version exists
		supportedCount := 0
		for _, vb := range bulletinFile.Versions {
			if vb.Supported {
				supportedCount++
			}
		}
		require.Greater(t, supportedCount, 0, "should have at least one supported version")

		// Write to temp directory
		tempDir := t.TempDir()
		now := time.Now()
		err = bulletinFile.SerializeAsWinOffice(now, tempDir)
		require.NoError(t, err)

		// Verify file was created with correct name
		expectedFileName := io.WinOfficeFileName(now)
		filePath := filepath.Join(tempDir, expectedFileName)
		_, err = os.Stat(filePath)
		require.NoError(t, err, "file should exist at expected path")

		// Read and parse the file
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)

		var parsed msrcapps.AppBulletinFile
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err, "should be valid JSON")

		// Verify parsed structure
		require.Equal(t, 1, parsed.Version, "parsed file should have version 1")
		require.NotEmpty(t, parsed.BuildPrefixes, "parsed file should have build prefixes")
		require.NotEmpty(t, parsed.Versions, "parsed file should have versions")

		// Verify build prefixes map to valid versions
		for prefix, version := range parsed.BuildPrefixes {
			require.NotEmpty(t, prefix, "build prefix should not be empty")
			require.Regexp(t, regexp.MustCompile(`^\d{4}$`), version,
				"version should be YYMM format")
		}

		// Verify version bulletins structure
		cvePattern := regexp.MustCompile(`^CVE-\d{4}-\d+$`)
		versionPattern := regexp.MustCompile(`^16\.0\.\d+\.\d+$`)

		for version, vb := range parsed.Versions {
			require.Regexp(t, regexp.MustCompile(`^\d{4}$`), version,
				"version key should be YYMM format")
			require.NotEmpty(t, vb.SecurityUpdates, "version should have security updates")

			for _, update := range vb.SecurityUpdates {
				require.Regexp(t, cvePattern, update.CVE,
					"CVE should match expected format")
				require.Regexp(t, versionPattern, update.FixedBuild,
					"fixed build should be 16.0.X.Y format")
			}
		}
	})

	t.Run("MatchHostVersion detects vulnerabilities correctly", func(t *testing.T) {
		bulletin, err := msrc.FetchWinOfficeBulletin(client)
		require.NoError(t, err)

		// Get a CVE from the bulletin to test with
		var testCVE string
		var expectedFixedBuild string
		for cve, builds := range bulletin.CVEToFixedBuilds {
			testCVE = cve
			for _, build := range builds {
				expectedFixedBuild = build
				break
			}
			break
		}
		require.NotEmpty(t, testCVE, "should have at least one CVE to test")

		// Extract build prefix from the fixed build (e.g., "19725.20172" -> "19725")
		// This ensures we test with a consistent build prefix
		fixedBuildParts := strings.Split(expectedFixedBuild, ".")
		require.Len(t, fixedBuildParts, 2, "fixed build should be prefix.suffix format")
		buildPrefix := fixedBuildParts[0]
		fixedSuffix := fixedBuildParts[1]

		// Test with vulnerable version (older suffix than fixed)
		vulnerableVersion := "16.0." + buildPrefix + ".10000"
		isVulnerable, fixedVersion, err := bulletin.MatchHostVersion(vulnerableVersion, testCVE)
		require.NoError(t, err)
		require.True(t, isVulnerable, "older version should be vulnerable")
		require.Equal(t, "16.0."+expectedFixedBuild, fixedVersion, "should report correct fixed version")

		// Test with fixed version (exact match)
		fixedHostVersion := "16.0." + expectedFixedBuild
		isVulnerable, fixedVersion, err = bulletin.MatchHostVersion(fixedHostVersion, testCVE)
		require.NoError(t, err)
		require.False(t, isVulnerable, "fixed version should not be vulnerable")
		require.Empty(t, fixedVersion, "should not report fixed version when not vulnerable")

		// Test with newer version (same prefix, higher suffix)
		// Use a suffix that's definitely higher than the fixed suffix
		newerSuffix := fixedSuffix + "9"
		newerVersion := "16.0." + buildPrefix + "." + newerSuffix
		isVulnerable, _, err = bulletin.MatchHostVersion(newerVersion, testCVE)
		require.NoError(t, err)
		require.False(t, isVulnerable, "newer version should not be vulnerable")

		// Test with unknown CVE
		isVulnerable, _, err = bulletin.MatchHostVersion(vulnerableVersion, "CVE-9999-99999")
		require.NoError(t, err)
		require.False(t, isVulnerable, "unknown CVE should not be vulnerable")
	})
}
