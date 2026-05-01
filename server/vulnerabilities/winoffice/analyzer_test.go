package winoffice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOfficeVersion(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		wantPrefix string
		wantSuffix string
		wantErr    bool
	}{
		{
			name:       "valid version",
			version:    "16.0.19725.20204",
			wantPrefix: "19725",
			wantSuffix: "20204",
		},
		{
			name:    "invalid version - too few parts",
			version: "16.0.19725",
			wantErr: true,
		},
		{
			name:    "invalid version - wrong prefix",
			version: "17.0.19725.20204",
			wantErr: true,
		},
		{
			name:    "invalid version - no prefix",
			version: "19725.20204",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, suffix, err := parseOfficeVersion(tt.version)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantPrefix, prefix)
				assert.Equal(t, tt.wantSuffix, suffix)
			}
		})
	}
}

func TestCompareBuildSuffix(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"a less than b", "20100", "20200", -1},
		{"a greater than b", "20300", "20200", 1},
		{"equal", "20200", "20200", 0},
		{"different lengths - shorter is less", "200", "20200", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareBuildSuffix(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func testBulletin() *BulletinFile {
	return &BulletinFile{
		Version: 1,
		BuildPrefixes: map[string]string{
			"17928": "ltsc2024",
			"17932": "ltsc2024",
			"19725": "2602",
		},
		Versions: map[string]*VersionBulletin{
			"ltsc2024": {
				SecurityUpdates: []SecurityUpdate{
					{CVE: "CVE-2024-0001", ResolvedInVersion: "16.0.17928.20500"},
					{CVE: "CVE-2024-0002", ResolvedInVersion: "16.0.17932.20100"},
				},
			},
			"2602": {
				SecurityUpdates: []SecurityUpdate{
					{CVE: "CVE-2024-0003", ResolvedInVersion: "16.0.19725.20200"},
				},
			},
		},
	}
}

func TestCheckVersion(t *testing.T) {
	bulletin := testBulletin()

	tests := []struct {
		name     string
		version  string
		wantCVEs []string
	}{
		{
			name:     "nil bulletin returns nil",
			version:  "16.0.19725.20100",
			wantCVEs: nil,
		},
		{
			name:     "invalid version format",
			version:  "not-a-version",
			wantCVEs: nil,
		},
		{
			name:     "unknown build prefix",
			version:  "16.0.99999.20100",
			wantCVEs: nil,
		},
		{
			name:     "same prefix - host older than fix",
			version:  "16.0.19725.20100",
			wantCVEs: []string{"CVE-2024-0003"},
		},
		{
			name:     "same prefix - host equal to fix",
			version:  "16.0.19725.20200",
			wantCVEs: nil,
		},
		{
			name:     "same prefix - host newer than fix",
			version:  "16.0.19725.20300",
			wantCVEs: nil,
		},
		{
			name:    "cross-prefix - fix has newer prefix, host vulnerable",
			version: "16.0.17928.20100",
			// CVE-2024-0001 has same prefix 17928, host 20100 < fix 20500 → vulnerable
			// CVE-2024-0002 has prefix 17932 > 17928 → vulnerable (newer prefix)
			wantCVEs: []string{"CVE-2024-0001", "CVE-2024-0002"},
		},
		{
			name:    "cross-prefix - fix has older prefix, host not vulnerable",
			version: "16.0.17932.20050",
			// CVE-2024-0001 has prefix 17928 < 17932 → not vulnerable (older prefix)
			// CVE-2024-0002 has same prefix 17932, host 20050 < fix 20100 → vulnerable
			wantCVEs: []string{"CVE-2024-0002"},
		},
		{
			name:    "cross-prefix - host on newer prefix, fully patched suffix",
			version: "16.0.17932.20200",
			// CVE-2024-0001 has prefix 17928 < 17932 → not vulnerable
			// CVE-2024-0002 has same prefix 17932, host 20200 > fix 20100 → not vulnerable
			wantCVEs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bulletin
			if tt.name == "nil bulletin returns nil" {
				b = nil
			}
			vulns := CheckVersion(tt.version, b)
			if tt.wantCVEs == nil {
				assert.Empty(t, vulns)
			} else {
				require.Len(t, vulns, len(tt.wantCVEs))
				var gotCVEs []string
				for _, v := range vulns {
					gotCVEs = append(gotCVEs, v.CVE)
				}
				assert.Equal(t, tt.wantCVEs, gotCVEs)
			}
		})
	}
}

func TestCheckVersionNumericPrefixComparison(t *testing.T) {
	// Regression test: prefixes of different lengths must compare numerically, not lexicographically.
	// "9999" < "10000" numerically, but "9999" > "10000" lexicographically.
	bulletin := &BulletinFile{
		Version: 1,
		BuildPrefixes: map[string]string{
			"9999":  "branch1",
			"10000": "branch1",
		},
		Versions: map[string]*VersionBulletin{
			"branch1": {
				SecurityUpdates: []SecurityUpdate{
					{CVE: "CVE-2024-9999", ResolvedInVersion: "16.0.10000.20100"},
				},
			},
		},
	}

	// Host on prefix 9999 (numerically smaller) with fix on prefix 10000 (numerically larger).
	// Host should be vulnerable because 10000 > 9999.
	vulns := CheckVersion("16.0.9999.20050", bulletin)
	require.Len(t, vulns, 1)
	assert.Equal(t, "CVE-2024-9999", vulns[0].CVE)

	// Host on prefix 10000 with fix on prefix 9999 would mean fix is for older prefix.
	// But we need a bulletin where the fix has prefix 9999.
	bulletin2 := &BulletinFile{
		Version: 1,
		BuildPrefixes: map[string]string{
			"9999":  "branch1",
			"10000": "branch1",
		},
		Versions: map[string]*VersionBulletin{
			"branch1": {
				SecurityUpdates: []SecurityUpdate{
					{CVE: "CVE-2024-8888", ResolvedInVersion: "16.0.9999.20100"},
				},
			},
		},
	}

	// Host on prefix 10000 (numerically larger), fix on prefix 9999 (numerically smaller).
	// Host should NOT be vulnerable.
	vulns = CheckVersion("16.0.10000.20050", bulletin2)
	assert.Empty(t, vulns)
}

func TestCheckVersionResolvedVersionPointer(t *testing.T) {
	bulletin := testBulletin()

	vulns := CheckVersion("16.0.19725.20100", bulletin)
	require.Len(t, vulns, 1)
	assert.Equal(t, "CVE-2024-0003", vulns[0].CVE)
	require.NotNil(t, vulns[0].ResolvedInVersion)
	assert.Equal(t, "16.0.19725.20200", *vulns[0].ResolvedInVersion)
	assert.Equal(t, uint(0), vulns[0].SoftwareID)
}
