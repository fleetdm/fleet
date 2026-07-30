package fleet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVulnExposureFilterSettingsValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		in      *VulnExposureFilterSettings
		wantErr bool
		// substrings expected somewhere in the accumulated error
		errContains []string
	}{
		{
			name: "nil is valid",
			in:   nil,
		},
		{
			name: "empty struct is valid (all absent)",
			in:   &VulnExposureFilterSettings{},
		},
		{
			name: "valid full payload",
			in: &VulnExposureFilterSettings{
				SoftwareFilters:        &[]string{"os", "browsers", "office", "adobe"},
				CVSSMin:                new(0.0),
				CVSSMax:                new(10.0),
				EPSSMin:                new(0.0),
				EPSSMax:                new(100.0),
				HasKnownExploit:        new(true),
				ExcludeVulnerabilities: &[]string{"CVE-2025-50897", "cve-2024-1234"},
			},
		},
		{
			name:        "invalid software category",
			in:          &VulnExposureFilterSettings{SoftwareFilters: &[]string{"os", "bogus"}},
			wantErr:     true,
			errContains: []string{"software_filters", "bogus"},
		},
		{
			name:        "explicit empty software_filters is rejected (must select at least one)",
			in:          &VulnExposureFilterSettings{SoftwareFilters: &[]string{}},
			wantErr:     true,
			errContains: []string{"software_filters", "at least one"},
		},
		{
			name:        "cvss out of range",
			in:          &VulnExposureFilterSettings{CVSSMax: new(11.0)},
			wantErr:     true,
			errContains: []string{"cvss_max"},
		},
		{
			name:        "cvss min greater than max",
			in:          &VulnExposureFilterSettings{CVSSMin: new(8.0), CVSSMax: new(2.0)},
			wantErr:     true,
			errContains: []string{"cvss_min"},
		},
		{
			name:        "epss out of range",
			in:          &VulnExposureFilterSettings{EPSSMin: new(-1.0)},
			wantErr:     true,
			errContains: []string{"epss_min"},
		},
		{
			name:        "epss min greater than max",
			in:          &VulnExposureFilterSettings{EPSSMin: new(80.0), EPSSMax: new(20.0)},
			wantErr:     true,
			errContains: []string{"epss_min"},
		},
		{
			name:        "invalid CVE identifier",
			in:          &VulnExposureFilterSettings{ExcludeVulnerabilities: &[]string{"not-a-cve"}},
			wantErr:     true,
			errContains: []string{"exclude_vulnerabilities", "not-a-cve"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			invalid := &InvalidArgumentError{}
			c.in.Validate("org_settings.features", invalid)
			if c.wantErr {
				require.True(t, invalid.HasErrors(), "expected validation errors")
				msg := invalid.Error()
				for _, sub := range c.errContains {
					assert.Contains(t, msg, sub)
				}
			} else {
				assert.False(t, invalid.HasErrors(), "unexpected validation errors: %v", invalid)
			}
		})
	}
}

func TestVulnExposureFilterSettingsCopyIsIndependent(t *testing.T) {
	t.Parallel()

	assert.Nil(t, (*VulnExposureFilterSettings)(nil).Copy())

	orig := &VulnExposureFilterSettings{
		SoftwareFilters:        &[]string{"os", "browsers"},
		CVSSMin:                new(9.0),
		EPSSMax:                new(100.0),
		HasKnownExploit:        new(true),
		ExcludeVulnerabilities: &[]string{"CVE-2025-50897"},
	}
	clone := orig.Copy()
	require.Equal(t, orig, clone)
	require.NotNil(t, clone)
	require.NotNil(t, clone.SoftwareFilters)
	require.NotNil(t, clone.CVSSMin)
	require.NotNil(t, clone.HasKnownExploit)
	require.NotNil(t, clone.ExcludeVulnerabilities)

	// Mutating the clone's slices/scalars must not affect the original.
	(*clone.SoftwareFilters)[0] = "adobe"
	*clone.CVSSMin = 1.0
	*clone.HasKnownExploit = false
	(*clone.ExcludeVulnerabilities)[0] = "CVE-0000-0000"

	require.NotNil(t, orig.CVSSMin)
	assert.Equal(t, "os", (*orig.SoftwareFilters)[0])
	assert.InDelta(t, 9.0, *orig.CVSSMin, 0.0001)
	assert.True(t, *orig.HasKnownExploit)
	assert.Equal(t, "CVE-2025-50897", (*orig.ExcludeVulnerabilities)[0])
}
