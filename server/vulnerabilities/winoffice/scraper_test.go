package winoffice

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSecurityMarkdown(t *testing.T) {
	t.Run("parses single release with multiple versions", func(t *testing.T) {
		markdown := `# Security Updates

## March 11, 2026

Current Channel: Version 2602 (Build 19725.20172), Monthly Enterprise Channel: Version 2512 (Build 19530.20260)

- [CVE-2026-12345](https://example.com) Remote code execution
- [CVE-2026-12346](https://example.com) Elevation of privilege
`
		releases, err := parseSecurityMarkdown(strings.NewReader(markdown))
		require.NoError(t, err)
		require.Len(t, releases, 1)

		rel := releases[0]
		assert.Equal(t, "March 11, 2026", rel.Date)
		assert.Len(t, rel.Branches, 2)
		assert.Len(t, rel.CVEs, 2)

		// Check branches
		assert.Equal(t, "2602", rel.Branches[0].Version)
		assert.Equal(t, "19725", rel.Branches[0].BuildPrefix)
		assert.Equal(t, "19725.20172", rel.Branches[0].FullBuild)

		assert.Equal(t, "2512", rel.Branches[1].Version)
		assert.Equal(t, "19530", rel.Branches[1].BuildPrefix)

		// Check CVEs
		assert.Contains(t, rel.CVEs, "CVE-2026-12345")
		assert.Contains(t, rel.CVEs, "CVE-2026-12346")
	})

	t.Run("parses multiple releases", func(t *testing.T) {
		markdown := `# Security Updates

## March 11, 2026

Current Channel: Version 2602 (Build 19725.20172)

- [CVE-2026-12345](https://example.com) First CVE

## February 11, 2026

Current Channel: Version 2601 (Build 19628.20204)

- [CVE-2026-11111](https://example.com) Second CVE
`
		releases, err := parseSecurityMarkdown(strings.NewReader(markdown))
		require.NoError(t, err)
		require.Len(t, releases, 2)

		assert.Equal(t, "March 11, 2026", releases[0].Date)
		assert.Equal(t, "February 11, 2026", releases[1].Date)
	})

	t.Run("skips retail versions", func(t *testing.T) {
		markdown := `# Security Updates

## March 11, 2026

Current Channel: Version 2602 (Build 19725.20172), Office 2024 Retail: Version 2602 (Build 19725.20172)

- [CVE-2026-12345](https://example.com) CVE
`
		releases, err := parseSecurityMarkdown(strings.NewReader(markdown))
		require.NoError(t, err)
		require.Len(t, releases, 1)

		// Should only have one branch (retail skipped)
		assert.Len(t, releases[0].Branches, 1)
		assert.Equal(t, "2602", releases[0].Branches[0].Version)
	})

	t.Run("keeps minimum build suffix for same version", func(t *testing.T) {
		markdown := `# Security Updates

## March 11, 2026

Current Channel: Version 2602 (Build 19725.20172), Monthly Enterprise Channel: Version 2602 (Build 19725.20170)

- [CVE-2026-12345](https://example.com) CVE
`
		releases, err := parseSecurityMarkdown(strings.NewReader(markdown))
		require.NoError(t, err)
		require.Len(t, releases, 1)

		// Should only have one branch with minimum build suffix
		assert.Len(t, releases[0].Branches, 1)
		assert.Equal(t, "2602", releases[0].Branches[0].Version)
		assert.Equal(t, "19725.20170", releases[0].Branches[0].FullBuild)
	})

	t.Run("skips releases without CVEs", func(t *testing.T) {
		markdown := `# Security Updates

## March 11, 2026

Current Channel: Version 2602 (Build 19725.20172)

No security updates this month.

## February 11, 2026

Current Channel: Version 2601 (Build 19628.20204)

- [CVE-2026-11111](https://example.com) CVE
`
		releases, err := parseSecurityMarkdown(strings.NewReader(markdown))
		require.NoError(t, err)
		require.Len(t, releases, 1)

		assert.Equal(t, "February 11, 2026", releases[0].Date)
	})

	t.Run("parses LTSC versions", func(t *testing.T) {
		markdown := `# Security Updates

## March 11, 2026

Current Channel: Version 2602 (Build 19725.20172), Office LTSC 2024 Volume Licensed: Version 2408 (Build 17932.20700)

- [CVE-2026-12345](https://example.com) CVE
`
		releases, err := parseSecurityMarkdown(strings.NewReader(markdown))
		require.NoError(t, err)
		require.Len(t, releases, 1)
		require.Len(t, releases[0].Branches, 2)

		// Find LTSC version
		var ltscBranch *VersionBranch
		for i := range releases[0].Branches {
			if releases[0].Branches[i].Version == "2408" {
				ltscBranch = &releases[0].Branches[i]
				break
			}
		}
		require.NotNil(t, ltscBranch)
		assert.Equal(t, "17932", ltscBranch.BuildPrefix)
		assert.Equal(t, "17932.20700", ltscBranch.FullBuild)
	})
}

func TestCompareBuildVersions(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"equal", "19725.20172", "19725.20172", 0},
		{"a less than b - different prefix", "19628.20204", "19725.20172", -1},
		{"a greater than b - different prefix", "19725.20172", "19628.20204", 1},
		{"a less than b - same prefix", "19725.20170", "19725.20172", -1},
		{"a greater than b - same prefix", "19725.20172", "19725.20170", 1},
		{"different suffix lengths", "19725.20170", "19725.201720", -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := compareBuildVersions(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildBulletin(t *testing.T) {
	t.Run("builds mappings from releases", func(t *testing.T) {
		releases := []SecurityRelease{
			{
				Date: "March 11, 2026",
				Branches: []VersionBranch{
					{Version: "2602", BuildPrefix: "19725", FullBuild: "19725.20172"},
					{Version: "2512", BuildPrefix: "19530", FullBuild: "19530.20260"},
				},
				CVEs: []string{"CVE-2026-12345", "CVE-2026-12346"},
			},
			{
				Date: "February 11, 2026",
				Branches: []VersionBranch{
					{Version: "2601", BuildPrefix: "19628", FullBuild: "19628.20204"},
					{Version: "2512", BuildPrefix: "19530", FullBuild: "19530.20200"},
				},
				CVEs: []string{"CVE-2026-11111"},
			},
		}

		bulletin := BuildBulletin(releases)

		// Check build prefix mappings
		assert.Equal(t, "2602", bulletin.BuildPrefixToVersion["19725"])
		assert.Equal(t, "2512", bulletin.BuildPrefixToVersion["19530"])
		assert.Equal(t, "2601", bulletin.BuildPrefixToVersion["19628"])

		// Check CVE mappings
		assert.Equal(t, "19725.20172", bulletin.CVEToResolvedVersions["CVE-2026-12345"]["2602"])
		assert.Equal(t, "19530.20260", bulletin.CVEToResolvedVersions["CVE-2026-12345"]["2512"])
		assert.Equal(t, "19628.20204", bulletin.CVEToResolvedVersions["CVE-2026-11111"]["2601"])
	})

	t.Run("first fix wins for same CVE", func(t *testing.T) {
		releases := []SecurityRelease{
			{
				Date: "March 11, 2026",
				Branches: []VersionBranch{
					{Version: "2602", BuildPrefix: "19725", FullBuild: "19725.20172"},
				},
				CVEs: []string{"CVE-2026-12345"},
			},
			{
				Date: "February 11, 2026",
				Branches: []VersionBranch{
					{Version: "2602", BuildPrefix: "19725", FullBuild: "19725.20100"},
				},
				CVEs: []string{"CVE-2026-12345"}, // Same CVE, different build
			},
		}

		bulletin := BuildBulletin(releases)

		// Should have first (March) build, not February
		assert.Equal(t, "19725.20172", bulletin.CVEToResolvedVersions["CVE-2026-12345"]["2602"])
	})
}

func TestToBulletinFile(t *testing.T) {
	t.Run("converts bulletin to file format", func(t *testing.T) {
		bulletin := &Bulletin{
			BuildPrefixToVersion: map[string]string{
				"19725": "2602",
				"19530": "2512",
			},
			CVEToResolvedVersions: map[string]map[string]string{
				"CVE-2026-12345": {
					"2602": "19725.20172",
					"2512": "19530.20260",
				},
			},
		}

		file := bulletin.ToBulletinFile()

		assert.Equal(t, 1, file.Version)
		assert.Equal(t, "2602", file.BuildPrefixes["19725"])
		assert.Equal(t, "2512", file.BuildPrefixes["19530"])

		// Check version 2602
		require.NotNil(t, file.Versions["2602"])
		found := false
		for _, su := range file.Versions["2602"].SecurityUpdates {
			if su.CVE == "CVE-2026-12345" {
				found = true
				assert.Equal(t, "16.0.19725.20172", su.ResolvedInVersion)
			}
		}
		assert.True(t, found, "CVE should be in version 2602")
	})

	t.Run("adds upgrade paths for dropped versions", func(t *testing.T) {
		bulletin := &Bulletin{
			BuildPrefixToVersion: map[string]string{
				"19725": "2602",
				"19530": "2512",
				"19000": "2400", // Dropped version
			},
			CVEToResolvedVersions: map[string]map[string]string{
				"CVE-2026-12345": {
					"2602": "19725.20172",
					"2512": "19530.20260",
					// No fix for 2400
				},
			},
		}

		file := bulletin.ToBulletinFile()

		// Dropped version 2400 should have upgrade path to 2512 (oldest newer version)
		require.NotNil(t, file.Versions["2400"])
		found := false
		for _, su := range file.Versions["2400"].SecurityUpdates {
			if su.CVE == "CVE-2026-12345" {
				found = true
				// Should point to 2512's fix (oldest version > 2400 with a fix)
				assert.Equal(t, "16.0.19530.20260", su.ResolvedInVersion)
			}
		}
		assert.True(t, found, "CVE should be in dropped version with upgrade path")
	})
}
