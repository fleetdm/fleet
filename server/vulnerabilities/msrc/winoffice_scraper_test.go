package msrc

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWinOfficeSecurityMarkdown_MinimumBuildSuffix(t *testing.T) {
	// Simulates the March 2026 edge case where the same version appears
	// in different channels with different build suffixes on the same date.
	// The parser should keep the minimum build suffix.
	markdown := `
## March 10, 2026

Current Channel: Version 2602 (Build 19725.20172) Monthly Enterprise Channel: Version 2602 (Build 19725.20170)

[CVE-2026-12345](https://example.com)
`

	releases, err := parseWinOfficeSecurityMarkdown(strings.NewReader(markdown))
	require.NoError(t, err)
	require.Len(t, releases, 1)

	release := releases[0]
	assert.Equal(t, "March 10, 2026", release.Date)
	assert.Len(t, release.Branches, 1, "should deduplicate version 2602")

	// Should keep the minimum build suffix (20170, not 20172)
	branch := release.Branches[0]
	assert.Equal(t, "2602", branch.Version)
	assert.Equal(t, "19725", branch.BuildPrefix)
	assert.Equal(t, "19725.20170", branch.FullBuild, "should keep minimum build suffix")
}

func TestParseWinOfficeSecurityMarkdown_SameBuildSuffix(t *testing.T) {
	// When channels have the same build suffix (the common case),
	// the result should be the same regardless of order.
	markdown := `
## January 14, 2025

Current Channel: Version 2412 (Build 18324.20190) Monthly Enterprise Channel: Version 2412 (Build 18324.20190)

[CVE-2025-12345](https://example.com)
`

	releases, err := parseWinOfficeSecurityMarkdown(strings.NewReader(markdown))
	require.NoError(t, err)
	require.Len(t, releases, 1)

	release := releases[0]
	assert.Len(t, release.Branches, 1, "should deduplicate identical versions")
	assert.Equal(t, "18324.20190", release.Branches[0].FullBuild)
}

func TestParseWinOfficeSecurityMarkdown_MultipleVersions(t *testing.T) {
	// Multiple different versions should all be captured
	markdown := `
## March 10, 2026

Current Channel: Version 2602 (Build 19725.20172) Monthly Enterprise Channel: Version 2512 (Build 19530.20260) Monthly Enterprise Channel: Version 2511 (Build 19426.20314)

[CVE-2026-12345](https://example.com)
`

	releases, err := parseWinOfficeSecurityMarkdown(strings.NewReader(markdown))
	require.NoError(t, err)
	require.Len(t, releases, 1)

	release := releases[0]
	assert.Len(t, release.Branches, 3, "should capture all different versions")

	versions := make(map[string]string)
	for _, b := range release.Branches {
		versions[b.Version] = b.FullBuild
	}

	assert.Equal(t, "19725.20172", versions["2602"])
	assert.Equal(t, "19530.20260", versions["2512"])
	assert.Equal(t, "19426.20314", versions["2511"])
}

func TestCompareBuildVersions(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{
			name:     "a less than b - same prefix",
			a:        "19725.20170",
			b:        "19725.20172",
			expected: -1,
		},
		{
			name:     "a greater than b - same prefix",
			a:        "19725.20172",
			b:        "19725.20170",
			expected: 1,
		},
		{
			name:     "equal",
			a:        "19725.20170",
			b:        "19725.20170",
			expected: 0,
		},
		{
			name:     "different prefix - a less",
			a:        "19530.20260",
			b:        "19725.20172",
			expected: -1,
		},
		{
			name:     "different prefix - a greater",
			a:        "19725.20172",
			b:        "19530.20260",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareBuildVersions(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}
