package osv

import (
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
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
			name:     "RHEL lowercase",
			platform: "rhel",
			expected: true,
		},
		{
			name:     "RHEL uppercase",
			platform: "RHEL",
			expected: true,
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
		{
			name:     "Interim release 23.10",
			version:  "23.10",
			expected: "2310",
		},
		{
			name:     "Interim release 24.10 with patch",
			version:  "24.10.1",
			expected: "2410",
		},
		{
			name:     "Version with codename suffix",
			version:  "22.04.1 LTS (Jammy Jellyfish)",
			expected: "2204",
		},
		{
			name:     "Very old version 14.04",
			version:  "14.04.6 LTS",
			expected: "1404",
		},
		{
			name:     "Future version 25.04",
			version:  "25.04 LTS",
			expected: "2504",
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
				{SoftwareID: 1, CVE: "CVE-2024-5555", ResolvedInVersion: ptr.String("2.4.50")},
			},
		},
		{
			name: "Range-based vulnerability matching with multiple fixed versions",
			software: []fleet.Software{
				{ID: 1, Name: "apache2", Version: "2.4.41"},
			},
			artifact: &OSVArtifact{
				Vulnerabilities: map[string][]OSVVulnerability{
					"apache2": {
						{CVE: "CVE-2024-5555", Introduced: "2.4.0", Fixed: "2.4.50"},
						{CVE: "CVE-2024-6666", Introduced: "2.4.10", Fixed: "2.4.48"},
					},
				},
			},
			expected: []fleet.SoftwareVulnerability{
				{SoftwareID: 1, CVE: "CVE-2024-5555", ResolvedInVersion: ptr.String("2.4.50")},
				{SoftwareID: 1, CVE: "CVE-2024-6666", ResolvedInVersion: ptr.String("2.4.48")},
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

func TestFindLatestOSVArtifactForVersion(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test artifacts for different Ubuntu versions and dates
	artifacts := []struct {
		filename string
		age      int // days old (to set mod time)
	}{
		{"osv-ubuntu-2204-2026-03-28.json.gz", 3},       // Older 22.04
		{"osv-ubuntu-2204-2026-03-30.json.gz", 1},       // Newer 22.04 (should be selected)
		{"osv-ubuntu-2204-2026-03-29.json.gz", 2},       // Middle 22.04
		{"osv-ubuntu-2004-2026-03-30.json.gz", 1},       // 20.04 (different version)
		{"osv-ubuntu-1804-2026-03-30.json.gz", 1},       // 18.04 (different version)
		{"other-file.json.gz", 0},                       // Non-OSV file
		{"osv-ubuntu-2204-delta-2026-03-30.json.gz", 1}, // Delta file (should be ignored by pattern)
	}

	for _, a := range artifacts {
		path := filepath.Join(tmpDir, a.filename)
		err := os.WriteFile(path, []byte("test"), 0o644)
		require.NoError(t, err)

		// Set modification time to simulate different ages
		modTime := time.Now().Add(-time.Duration(a.age) * 24 * time.Hour)
		err = os.Chtimes(path, modTime, modTime)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		ubuntuVersion string
		expectedFile  string
		expectError   bool
	}{
		{
			name:          "finds latest 22.04 artifact",
			ubuntuVersion: "2204",
			expectedFile:  "osv-ubuntu-2204-2026-03-30.json.gz", // Most recent
		},
		{
			name:          "finds latest 20.04 artifact",
			ubuntuVersion: "2004",
			expectedFile:  "osv-ubuntu-2004-2026-03-30.json.gz",
		},
		{
			name:          "finds latest 18.04 artifact",
			ubuntuVersion: "1804",
			expectedFile:  "osv-ubuntu-1804-2026-03-30.json.gz",
		},
		{
			name:          "returns error for non-existent version",
			ubuntuVersion: "2404",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := findLatestOSVArtifactForVersion(tmpDir, tt.ubuntuVersion)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no OSV artifact found")
			} else {
				require.NoError(t, err)
				require.Equal(t, filepath.Join(tmpDir, tt.expectedFile), result)
			}
		})
	}
}

func TestLoadOSVArtifactZeroTimeUsesLatest(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifacts with different dates
	artifacts := []struct {
		filename string
		age      int
	}{
		{"osv-ubuntu-2204-2026-03-28.json.gz", 3}, // Older
		{"osv-ubuntu-2204-2026-03-30.json.gz", 1}, // Newer (should be selected)
	}

	for _, a := range artifacts {
		path := filepath.Join(tmpDir, a.filename)

		// Create a minimal valid gzipped JSON artifact
		f, err := os.Create(path)
		require.NoError(t, err)

		gz := gzip.NewWriter(f)
		_, err = gz.Write([]byte(`{"schema_version":"1.0.0","ubuntu_version":"2204","generated":"2026-03-30T00:00:00Z","total_cves":0,"total_packages":0,"vulnerabilities":{}}`))
		require.NoError(t, err)
		gz.Close()
		f.Close()

		// Set modification time
		modTime := time.Now().Add(-time.Duration(a.age) * 24 * time.Hour)
		err = os.Chtimes(path, modTime, modTime)
		require.NoError(t, err)
	}

	// Test with zero time (simulates DisableDataSync=true)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ver := fleet.OSVersion{Name: "Ubuntu 22.04.8 LTS", Version: "22.04.8 LTS"}

	artifact, err := loadOSVArtifact(ctx, ver, tmpDir, logger, time.Time{})
	require.NoError(t, err)
	require.NotNil(t, artifact)

	// Verify it loaded successfully (artifact should have schema_version)
	require.Equal(t, "1.0.0", artifact.SchemaVersion)
}

func TestExtractRHELMajorVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"RHEL 9.4.0", "9.4.0", "9"},
		{"RHEL 8.10.0", "8.10.0", "8"},
		{"RHEL 7.9.0", "7.9.0", "7"},
		{"Major only", "9", "9"},
		{"Empty string", "", ""},
		{"Whitespace", "  9.4.0  ", "9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, extractRHELMajorVersion(tt.input))
		})
	}
}

func TestIsVulnerableRPM(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		release  string
		vuln     OSVVulnerability
		expected bool
	}{
		{
			name:    "vulnerable - older than fixed",
			version: "2.1.3", release: "3.el9",
			vuln:     OSVVulnerability{Fixed: "0:2.1.3-4.el9_1", Introduced: "0"},
			expected: true,
		},
		{
			name:    "not vulnerable - at fixed version",
			version: "2.1.3", release: "4.el9_1",
			vuln:     OSVVulnerability{Fixed: "0:2.1.3-4.el9_1", Introduced: "0"},
			expected: false,
		},
		{
			name:    "not vulnerable - newer than fixed",
			version: "2.1.3", release: "5.el9_2",
			vuln:     OSVVulnerability{Fixed: "0:2.1.3-4.el9_1", Introduced: "0"},
			expected: false,
		},
		{
			name:    "vulnerable - no release field",
			version: "1.0.0", release: "",
			vuln:     OSVVulnerability{Fixed: "0:2.0.0-1.el9", Introduced: "0"},
			expected: true,
		},
		{
			name:    "vulnerable - no fixed version (still affected)",
			version: "1.0.0", release: "1.el9",
			vuln:     OSVVulnerability{Introduced: "0"},
			expected: true,
		},
		{
			name:    "not vulnerable - below introduced",
			version: "0.9.0", release: "1.el9",
			vuln:     OSVVulnerability{Fixed: "0:2.0.0-1.el9", Introduced: "0:1.0.0-1.el9"},
			expected: false,
		},
		{
			name:    "vulnerable - kernel version with epoch",
			version: "5.14.0", release: "503.26.1.el9_5",
			vuln:     OSVVulnerability{Fixed: "0:5.14.0-611.8.1.el9_7", Introduced: "0"},
			expected: true,
		},
		{
			name:    "not vulnerable - kernel at fixed",
			version: "5.14.0", release: "611.8.1.el9_7",
			vuln:     OSVVulnerability{Fixed: "0:5.14.0-611.8.1.el9_7", Introduced: "0"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, isVulnerableRPM(tt.version, tt.release, tt.vuln))
		})
	}
}

func TestMatchSoftwareToRHELOSV(t *testing.T) {
	artifact := &RHELOSVArtifact{
		RHELVersion: "9",
		Vulnerabilities: map[string][]OSVVulnerability{
			"curl": {
				{CVE: "CVE-2024-1234", Fixed: "0:7.76.1-29.el9_4.2", Introduced: "0"},
			},
			"kernel": {
				{CVE: "CVE-2025-5678", Fixed: "0:5.14.0-611.8.1.el9_7", Introduced: "0"},
			},
		},
	}

	t.Run("regular package match", func(t *testing.T) {
		software := []fleet.Software{
			{ID: 1, Name: "curl", Version: "7.76.1", Release: "26.el9_3.2"},
		}
		result := matchSoftwareToRHELOSV(software, artifact)
		require.Len(t, result, 1)
		require.Equal(t, "CVE-2024-1234", result[0].CVE)
		require.Equal(t, uint(1), result[0].SoftwareID)
		require.NotNil(t, result[0].ResolvedInVersion)
		require.Equal(t, "0:7.76.1-29.el9_4.2", *result[0].ResolvedInVersion)
	})

	t.Run("package not in artifact", func(t *testing.T) {
		software := []fleet.Software{
			{ID: 2, Name: "nginx", Version: "1.0", Release: "1.el9"},
		}
		result := matchSoftwareToRHELOSV(software, artifact)
		require.Empty(t, result)
	})

	t.Run("kernel-core maps to kernel", func(t *testing.T) {
		software := []fleet.Software{
			{ID: 3, Name: "kernel-core", Version: "5.14.0", Release: "503.26.1.el9_5"},
		}
		result := matchSoftwareToRHELOSV(software, artifact)
		require.Len(t, result, 1)
		require.Equal(t, "CVE-2025-5678", result[0].CVE)
		require.NotNil(t, result[0].ResolvedInVersion)
		require.Equal(t, "0:5.14.0-611.8.1.el9_7", *result[0].ResolvedInVersion)
	})

	t.Run("kernel-modules maps to kernel", func(t *testing.T) {
		software := []fleet.Software{
			{ID: 4, Name: "kernel-modules", Version: "5.14.0", Release: "503.26.1.el9_5"},
		}
		result := matchSoftwareToRHELOSV(software, artifact)
		require.Len(t, result, 1)
		require.Equal(t, "CVE-2025-5678", result[0].CVE)
	})

	t.Run("kernel-debug-core maps to kernel", func(t *testing.T) {
		software := []fleet.Software{
			{ID: 5, Name: "kernel-debug-core", Version: "5.14.0", Release: "503.26.1.el9_5"},
		}
		result := matchSoftwareToRHELOSV(software, artifact)
		require.Len(t, result, 1)
		require.Equal(t, "CVE-2025-5678", result[0].CVE)
	})

	t.Run("patched kernel not vulnerable", func(t *testing.T) {
		software := []fleet.Software{
			{ID: 6, Name: "kernel-core", Version: "5.14.0", Release: "611.8.1.el9_7"},
		}
		result := matchSoftwareToRHELOSV(software, artifact)
		require.Empty(t, result)
	})

	t.Run("patched curl not vulnerable", func(t *testing.T) {
		software := []fleet.Software{
			{ID: 7, Name: "curl", Version: "7.76.1", Release: "29.el9_4.2"},
		}
		result := matchSoftwareToRHELOSV(software, artifact)
		require.Empty(t, result)
	})
}
