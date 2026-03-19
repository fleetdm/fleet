package osv

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestIsPlatformSupported(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		expected bool
	}{
		{
			name:     "Ubuntu lowercase",
			platform: "ubuntu",
			expected: true,
		},
		{
			name:     "Ubuntu mixed case",
			platform: "Ubuntu",
			expected: true,
		},
		{
			name:     "Ubuntu uppercase",
			platform: "UBUNTU",
			expected: true,
		},
		{
			name:     "RHEL not supported",
			platform: "rhel",
			expected: false,
		},
		{
			name:     "Windows not supported",
			platform: "windows",
			expected: false,
		},
		{
			name:     "Empty string not supported",
			platform: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPlatformSupported(tt.platform)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractUbuntuVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "Ubuntu 22.04 LTS",
			version:  "22.04.8 LTS",
			expected: "2204",
		},
		{
			name:     "Ubuntu 20.04 LTS",
			version:  "20.04.1 LTS",
			expected: "2004",
		},
		{
			name:     "Ubuntu 18.04",
			version:  "18.04",
			expected: "1804",
		},
		{
			name:     "Ubuntu 24.04 no LTS suffix",
			version:  "24.04.0",
			expected: "2404",
		},
		{
			name:     "Ubuntu 16.04 with extra spaces",
			version:  "16.04.7  LTS  ",
			expected: "1604",
		},
		{
			name:     "Invalid version - single digit",
			version:  "22",
			expected: "",
		},
		{
			name:     "Empty string",
			version:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUbuntuVersion(tt.version)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeKernelVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "Generic kernel",
			version:  "5.15.0-94-generic",
			expected: "5.15.0-94",
		},
		{
			name:     "Lowlatency kernel",
			version:  "5.15.0-94-lowlatency",
			expected: "5.15.0-94",
		},
		{
			name:     "Generic 64k kernel",
			version:  "6.8.0-31-generic-64k",
			expected: "6.8.0-31",
		},
		{
			name:     "Kernel with only one part",
			version:  "5.15.0",
			expected: "5.15.0",
		},
		{
			name:     "Empty string",
			version:  "",
			expected: "",
		},
		{
			name:     "Already normalized",
			version:  "5.15.0-94",
			expected: "5.15.0-94",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeKernelVersion(tt.version)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestKernelVersionMatches(t *testing.T) {
	tests := []struct {
		name            string
		softwareVersion string
		osvVersion      string
		expected        bool
	}{
		{
			name:            "Exact match after normalization",
			softwareVersion: "5.15.0-94-generic",
			osvVersion:      "5.15.0-94",
			expected:        true,
		},
		{
			name:            "OSV version with build number",
			softwareVersion: "5.15.0-94-generic",
			osvVersion:      "5.15.0-94.104",
			expected:        true,
		},
		{
			name:            "OSV version with different build",
			softwareVersion: "5.15.0-94-lowlatency",
			osvVersion:      "5.15.0-94.103",
			expected:        true,
		},
		{
			name:            "Different kernel versions",
			softwareVersion: "5.15.0-93-generic",
			osvVersion:      "5.15.0-94.104",
			expected:        false,
		},
		{
			name:            "Different major versions",
			softwareVersion: "6.8.0-31-generic",
			osvVersion:      "5.15.0-94.104",
			expected:        false,
		},
		{
			name:            "OSV prefix but not with dot",
			softwareVersion: "5.15.0-94-generic",
			osvVersion:      "5.15.0-941",
			expected:        false,
		},
		{
			name:            "Exact match without normalization needed",
			softwareVersion: "5.15.0-94",
			osvVersion:      "5.15.0-94",
			expected:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := kernelVersionMatches(tt.softwareVersion, tt.osvVersion)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsVulnerable(t *testing.T) {
	tests := []struct {
		name            string
		softwareVersion string
		vuln            OSVVulnerability
		isKernelPackage bool
		expected        bool
	}{
		{
			name:            "Kernel - version in explicit list",
			softwareVersion: "5.15.0-94-generic",
			vuln: OSVVulnerability{
				CVE:      "CVE-2024-1234",
				Versions: []string{"5.15.0-94.104", "5.15.0-93.103"},
			},
			isKernelPackage: true,
			expected:        true,
		},
		{
			name:            "Kernel - version not in list",
			softwareVersion: "5.15.0-95-generic",
			vuln: OSVVulnerability{
				CVE:      "CVE-2024-1234",
				Versions: []string{"5.15.0-94.104", "5.15.0-93.103"},
			},
			isKernelPackage: true,
			expected:        false,
		},
		{
			name:            "Regular package - exact version match",
			softwareVersion: "1.2.3-4ubuntu1",
			vuln: OSVVulnerability{
				CVE:      "CVE-2024-5678",
				Versions: []string{"1.2.3-4ubuntu1", "1.2.3-3ubuntu1"},
			},
			isKernelPackage: false,
			expected:        true,
		},
		{
			name:            "Regular package - no match",
			softwareVersion: "1.2.3-5ubuntu1",
			vuln: OSVVulnerability{
				CVE:      "CVE-2024-5678",
				Versions: []string{"1.2.3-4ubuntu1", "1.2.3-3ubuntu1"},
			},
			isKernelPackage: false,
			expected:        false,
		},
		{
			name:            "Range - vulnerable (in range)",
			softwareVersion: "2.0.0",
			vuln: OSVVulnerability{
				CVE:        "CVE-2024-9999",
				Introduced: "1.0.0",
				Fixed:      "3.0.0",
			},
			isKernelPackage: false,
			expected:        true,
		},
		{
			name:            "Range - not vulnerable (below introduced)",
			softwareVersion: "0.9.0",
			vuln: OSVVulnerability{
				CVE:        "CVE-2024-9999",
				Introduced: "1.0.0",
				Fixed:      "3.0.0",
			},
			isKernelPackage: false,
			expected:        false,
		},
		{
			name:            "Range - not vulnerable (at or above fixed)",
			softwareVersion: "3.0.0",
			vuln: OSVVulnerability{
				CVE:        "CVE-2024-9999",
				Introduced: "1.0.0",
				Fixed:      "3.0.0",
			},
			isKernelPackage: false,
			expected:        false,
		},
		{
			name:            "Range - vulnerable from zero",
			softwareVersion: "0.5.0",
			vuln: OSVVulnerability{
				CVE:   "CVE-2024-8888",
				Fixed: "1.0.0",
			},
			isKernelPackage: false,
			expected:        true,
		},
		{
			name:            "Range - no fixed version (vulnerable)",
			softwareVersion: "2.0.0",
			vuln: OSVVulnerability{
				CVE:        "CVE-2024-7777",
				Introduced: "1.0.0",
			},
			isKernelPackage: false,
			expected:        true,
		},
		{
			name:            "Empty versions list with no range (vulnerable by default)",
			softwareVersion: "1.0.0",
			vuln: OSVVulnerability{
				CVE: "CVE-2024-6666",
			},
			isKernelPackage: false,
			expected:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVulnerable(tt.softwareVersion, tt.vuln, tt.isKernelPackage)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchSoftwareToOSV(t *testing.T) {
	tests := []struct {
		name     string
		software []fleet.Software
		artifact *OSVArtifact
		expected []fleet.SoftwareVulnerability
	}{
		{
			name: "Kernel package mapping",
			software: []fleet.Software{
				{ID: 1, Name: "linux-image-5.15.0-94-generic", Version: "5.15.0-94-generic"},
				{ID: 2, Name: "linux-signed-image-5.15.0-94-generic", Version: "5.15.0-94-generic"},
			},
			artifact: &OSVArtifact{
				Vulnerabilities: map[string][]OSVVulnerability{
					"linux": {
						{CVE: "CVE-2024-1111", Versions: []string{"5.15.0-94.104"}},
					},
				},
			},
			expected: []fleet.SoftwareVulnerability{
				{SoftwareID: 1, CVE: "CVE-2024-1111"},
				{SoftwareID: 2, CVE: "CVE-2024-1111"},
			},
		},
		{
			name: "Regular package exact match",
			software: []fleet.Software{
				{ID: 1, Name: "curl", Version: "7.68.0-1ubuntu2.21"},
			},
			artifact: &OSVArtifact{
				Vulnerabilities: map[string][]OSVVulnerability{
					"curl": {
						{CVE: "CVE-2024-2222", Versions: []string{"7.68.0-1ubuntu2.21"}},
					},
				},
			},
			expected: []fleet.SoftwareVulnerability{
				{SoftwareID: 1, CVE: "CVE-2024-2222"},
			},
		},
		{
			name: "Package not in artifact",
			software: []fleet.Software{
				{ID: 1, Name: "unknown-package", Version: "1.0.0"},
			},
			artifact: &OSVArtifact{
				Vulnerabilities: map[string][]OSVVulnerability{
					"curl": {
						{CVE: "CVE-2024-2222", Versions: []string{"7.68.0-1ubuntu2.21"}},
					},
				},
			},
			expected: []fleet.SoftwareVulnerability{},
		},
		{
			name: "Multiple vulnerabilities for same package",
			software: []fleet.Software{
				{ID: 1, Name: "openssl", Version: "1.1.1f-1ubuntu2.20"},
			},
			artifact: &OSVArtifact{
				Vulnerabilities: map[string][]OSVVulnerability{
					"openssl": {
						{CVE: "CVE-2024-3333", Versions: []string{"1.1.1f-1ubuntu2.20"}},
						{CVE: "CVE-2024-4444", Versions: []string{"1.1.1f-1ubuntu2.20"}},
					},
				},
			},
			expected: []fleet.SoftwareVulnerability{
				{SoftwareID: 1, CVE: "CVE-2024-3333"},
				{SoftwareID: 1, CVE: "CVE-2024-4444"},
			},
		},
		{
			name: "Version doesn't match",
			software: []fleet.Software{
				{ID: 1, Name: "curl", Version: "7.68.0-1ubuntu2.22"},
			},
			artifact: &OSVArtifact{
				Vulnerabilities: map[string][]OSVVulnerability{
					"curl": {
						{CVE: "CVE-2024-2222", Versions: []string{"7.68.0-1ubuntu2.21"}},
					},
				},
			},
			expected: []fleet.SoftwareVulnerability{},
		},
		{
			name: "Kernel headers should not map to linux",
			software: []fleet.Software{
				{ID: 1, Name: "linux-headers-5.15.0-94", Version: "5.15.0-94-generic"},
			},
			artifact: &OSVArtifact{
				Vulnerabilities: map[string][]OSVVulnerability{
					"linux": {
						{CVE: "CVE-2024-1111", Versions: []string{"5.15.0-94.104"}},
					},
				},
			},
			expected: []fleet.SoftwareVulnerability{},
		},
		{
			name: "Range-based vulnerability matching",
			software: []fleet.Software{
				{ID: 1, Name: "apache2", Version: "2.4.41"},
			},
			artifact: &OSVArtifact{
				Vulnerabilities: map[string][]OSVVulnerability{
					"apache2": {
						{CVE: "CVE-2024-5555", Introduced: "2.4.0", Fixed: "2.4.50"},
					},
				},
			},
			expected: []fleet.SoftwareVulnerability{
				{SoftwareID: 1, CVE: "CVE-2024-5555"},
			},
		},
		{
			name: "emacs-common not vulnerable",
			software: []fleet.Software{
				{ID: 60992, Name: "emacs-common", Version: "1:29.3+1-1ubuntu2"},
			},
			artifact: &OSVArtifact{
				Vulnerabilities: map[string][]OSVVulnerability{
					// This avoids OVAL's false positive from USN grouping where
					// CVE-2024-30205 was incorrectly applied to emacs-common
					// because it was grouped with emacs in the same USN advisory
					"emacs": {
						{CVE: "CVE-2024-30205", Versions: []string{"1:26.3+1-1ubuntu2"}},
					},
				},
			},
			expected: []fleet.SoftwareVulnerability{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchSoftwareToOSV(tt.software, tt.artifact)
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}
