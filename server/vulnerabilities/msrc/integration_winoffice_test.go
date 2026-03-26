package msrc_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
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
		require.NotEmpty(t, bulletinFile.Products, "should have products")

		// Add mappings
		bulletinFile = bulletinFile.WithMappings(msrcapps.DefaultMappings())
		require.NotEmpty(t, bulletinFile.Mappings, "should have mappings")

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
		require.NotEmpty(t, parsed.Mappings, "parsed file should have mappings")
		require.NotEmpty(t, parsed.Products, "parsed file should have products")

		// Verify mappings
		require.GreaterOrEqual(t, len(parsed.Mappings), 1, "should have at least one mapping")
		for _, mapping := range parsed.Mappings {
			require.NotEmpty(t, mapping.Match.Name, "mapping should have name patterns")
			require.NotEmpty(t, mapping.ProductID, "mapping should have product ID")
		}

		// Verify products
		cvePattern := regexp.MustCompile(`^CVE-\d{4}-\d+$`)
		versionPattern := regexp.MustCompile(`^16\.0\.\d+\.\d+$`)

		for _, product := range parsed.Products {
			require.NotEmpty(t, product.ProductID, "product should have ID")
			require.NotEmpty(t, product.Product, "product should have name")
			require.NotEmpty(t, product.SecurityUpdates, "product should have security updates")

			for _, update := range product.SecurityUpdates {
				require.Regexp(t, cvePattern, update.CVE,
					"CVE should match expected format")
				require.Regexp(t, versionPattern, update.FixedVersion,
					"fixed version should be 16.0.X.Y format")
			}
		}
	})

	t.Run("MatchHostVersion detects vulnerabilities correctly", func(t *testing.T) {
		bulletin, err := msrc.FetchWinOfficeBulletin(client)
		require.NoError(t, err)

		// Get a CVE from the bulletin to test with
		var testCVE string
		var expectedFixedBuild string
		var testVersion string
		for cve, builds := range bulletin.CVEToFixedBuilds {
			testCVE = cve
			for version, build := range builds {
				testVersion = version
				expectedFixedBuild = build
				break
			}
			break
		}
		require.NotEmpty(t, testCVE, "should have at least one CVE to test")

		// Get build prefix for this version
		var buildPrefix string
		for prefix, version := range bulletin.BuildPrefixToVersion {
			if version == testVersion {
				buildPrefix = prefix
				break
			}
		}
		require.NotEmpty(t, buildPrefix, "should find build prefix for test version")

		// Test with vulnerable version (older than fixed)
		vulnerableVersion := "16.0." + buildPrefix + ".10000"
		isVulnerable, fixedVersion, err := bulletin.MatchHostVersion(vulnerableVersion, testCVE)
		require.NoError(t, err)
		require.True(t, isVulnerable, "older version should be vulnerable")
		require.Equal(t, "16.0."+expectedFixedBuild, fixedVersion, "should report correct fixed version")

		// Test with fixed version
		fixedHostVersion := "16.0." + expectedFixedBuild
		isVulnerable, fixedVersion, err = bulletin.MatchHostVersion(fixedHostVersion, testCVE)
		require.NoError(t, err)
		require.False(t, isVulnerable, "fixed version should not be vulnerable")
		require.Empty(t, fixedVersion, "should not report fixed version when not vulnerable")

		// Test with newer version
		newerVersion := "16.0." + buildPrefix + ".99999"
		isVulnerable, _, err = bulletin.MatchHostVersion(newerVersion, testCVE)
		require.NoError(t, err)
		require.False(t, isVulnerable, "newer version should not be vulnerable")

		// Test with unknown CVE
		isVulnerable, _, err = bulletin.MatchHostVersion(vulnerableVersion, "CVE-9999-99999")
		require.NoError(t, err)
		require.False(t, isVulnerable, "unknown CVE should not be vulnerable")
	})
}
