package winoffice

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsWindowsOffice(t *testing.T) {
	tests := []struct {
		name     string
		software *fleet.Software
		expected bool
	}{
		{
			name: "programs source",
			software: &fleet.Software{
				Name:   "Microsoft 365 Apps for enterprise",
				Source: "programs",
			},
			expected: true,
		},
		{
			name: "not programs source - chocolatey",
			software: &fleet.Software{
				Name:   "Microsoft 365 Apps",
				Source: "chocolatey",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWindowsOffice(tt.software)
			assert.Equal(t, tt.expected, result)
		})
	}
}

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
