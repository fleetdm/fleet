package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExtractUbuntuVersion(t *testing.T) {
	tests := []struct {
		name      string
		ecosystem string
		expected  string
	}{
		{
			name:      "Standard Ubuntu LTS",
			ecosystem: "Ubuntu:24.04:LTS",
			expected:  "24.04",
		},
		{
			name:      "Ubuntu Pro",
			ecosystem: "Ubuntu:Pro:22.04:LTS",
			expected:  "22.04",
		},
		{
			name:      "Ubuntu 20.04",
			ecosystem: "Ubuntu:20.04:LTS",
			expected:  "20.04",
		},
		{
			name:      "Ubuntu 18.04",
			ecosystem: "Ubuntu:18.04:LTS",
			expected:  "18.04",
		},
		{
			name:      "No version pattern",
			ecosystem: "Ubuntu:LTS",
			expected:  "",
		},
		{
			name:      "Empty string",
			ecosystem: "",
			expected:  "",
		},
		{
			name:      "Not Ubuntu",
			ecosystem: "Debian:12:stable",
			expected:  "",
		},
		{
			name:      "Version without LTS",
			ecosystem: "Ubuntu:24.04",
			expected:  "24.04",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUbuntuVersion(tt.ecosystem)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractCVEID(t *testing.T) {
	tests := []struct {
		name     string
		osv      *OSVData
		expected string
	}{
		{
			name: "CVE in Upstream field",
			osv: &OSVData{
				ID:       "UBUNTU-CVE-2024-1234",
				Upstream: []string{"CVE-2024-1234", "https://example.com"},
			},
			expected: "CVE-2024-1234",
		},
		{
			name: "CVE as ID",
			osv: &OSVData{
				ID:       "CVE-2024-5678",
				Upstream: []string{},
			},
			expected: "CVE-2024-5678",
		},
		{
			name: "UBUNTU-CVE prefix",
			osv: &OSVData{
				ID:       "UBUNTU-CVE-2024-9999",
				Upstream: []string{},
			},
			expected: "CVE-2024-9999",
		},
		{
			name: "No CVE found",
			osv: &OSVData{
				ID:       "SOME-OTHER-ID",
				Upstream: []string{"https://example.com"},
			},
			expected: "",
		},
		{
			name: "Multiple upstreams with CVE first",
			osv: &OSVData{
				ID:       "UBUNTU-123",
				Upstream: []string{"CVE-2024-1111", "CVE-2024-2222"},
			},
			expected: "CVE-2024-1111",
		},
		{
			name: "Empty upstream, no CVE in ID",
			osv: &OSVData{
				ID:       "USN-1234-1",
				Upstream: []string{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCVEID(tt.osv)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractVersionRange(t *testing.T) {
	tests := []struct {
		name               string
		ranges             []Range
		expectedIntroduced string
		expectedFixed      string
	}{
		{
			name: "Simple range with introduced and fixed",
			ranges: []Range{
				{
					Type: "ECOSYSTEM",
					Events: []Event{
						{Introduced: "1.0.0"},
						{Fixed: "2.0.0"},
					},
				},
			},
			expectedIntroduced: "1.0.0",
			expectedFixed:      "2.0.0",
		},
		{
			name: "Only introduced version",
			ranges: []Range{
				{
					Type: "ECOSYSTEM",
					Events: []Event{
						{Introduced: "1.5.0"},
					},
				},
			},
			expectedIntroduced: "1.5.0",
			expectedFixed:      "",
		},
		{
			name: "Only fixed version",
			ranges: []Range{
				{
					Type: "ECOSYSTEM",
					Events: []Event{
						{Fixed: "3.0.0"},
					},
				},
			},
			expectedIntroduced: "",
			expectedFixed:      "3.0.0",
		},
		{
			name: "Multiple ranges, first ECOSYSTEM wins",
			ranges: []Range{
				{
					Type: "ECOSYSTEM",
					Events: []Event{
						{Introduced: "1.0.0"},
						{Fixed: "2.0.0"},
					},
				},
				{
					Type: "ECOSYSTEM",
					Events: []Event{
						{Introduced: "3.0.0"},
						{Fixed: "4.0.0"},
					},
				},
			},
			expectedIntroduced: "1.0.0",
			expectedFixed:      "2.0.0",
		},
		{
			name: "Non-ECOSYSTEM range ignored",
			ranges: []Range{
				{
					Type: "GIT",
					Events: []Event{
						{Introduced: "abc123"},
						{Fixed: "def456"},
					},
				},
				{
					Type: "ECOSYSTEM",
					Events: []Event{
						{Introduced: "2.0.0"},
						{Fixed: "2.5.0"},
					},
				},
			},
			expectedIntroduced: "2.0.0",
			expectedFixed:      "2.5.0",
		},
		{
			name:               "Empty ranges",
			ranges:             []Range{},
			expectedIntroduced: "",
			expectedFixed:      "",
		},
		{
			name: "Multiple events in single range",
			ranges: []Range{
				{
					Type: "ECOSYSTEM",
					Events: []Event{
						{Introduced: "0"},
						{Fixed: "1.2.3"},
						{Introduced: "2.0.0"}, // This should be ignored (first introduced wins)
					},
				},
			},
			expectedIntroduced: "0",
			expectedFixed:      "1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			introduced, fixed := extractVersionRange(tt.ranges)
			require.Equal(t, tt.expectedIntroduced, introduced)
			require.Equal(t, tt.expectedFixed, fixed)
		})
	}
}

func TestCountTotalCVEs(t *testing.T) {
	tests := []struct {
		name     string
		artifact *ArtifactData
		expected int
	}{
		{
			name: "Multiple packages with unique CVEs",
			artifact: &ArtifactData{
				Vulnerabilities: map[string][]ProcessedVuln{
					"curl": {
						{CVE: "CVE-2024-1234"},
						{CVE: "CVE-2024-5678"},
					},
					"openssl": {
						{CVE: "CVE-2024-9999"},
					},
				},
			},
			expected: 3,
		},
		{
			name: "Duplicate CVEs across packages",
			artifact: &ArtifactData{
				Vulnerabilities: map[string][]ProcessedVuln{
					"emacs": {
						{CVE: "CVE-2024-39331"},
					},
					"emacs-common": {
						{CVE: "CVE-2024-39331"},
					},
					"emacs-el": {
						{CVE: "CVE-2024-39331"},
					},
				},
			},
			expected: 1, // Deduplicated
		},
		{
			name: "Mix of unique and duplicate CVEs",
			artifact: &ArtifactData{
				Vulnerabilities: map[string][]ProcessedVuln{
					"package1": {
						{CVE: "CVE-2024-1111"},
						{CVE: "CVE-2024-2222"},
					},
					"package2": {
						{CVE: "CVE-2024-1111"}, // Duplicate
						{CVE: "CVE-2024-3333"},
					},
				},
			},
			expected: 3, // CVE-2024-1111, CVE-2024-2222, CVE-2024-3333
		},
		{
			name: "Empty artifact",
			artifact: &ArtifactData{
				Vulnerabilities: map[string][]ProcessedVuln{},
			},
			expected: 0,
		},
		{
			name: "Package with no vulnerabilities",
			artifact: &ArtifactData{
				Vulnerabilities: map[string][]ProcessedVuln{
					"safe-package": {},
				},
			},
			expected: 0,
		},
		{
			name: "Single package, single CVE",
			artifact: &ArtifactData{
				Vulnerabilities: map[string][]ProcessedVuln{
					"apache2": {
						{CVE: "CVE-2024-7777"},
					},
				},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countTotalCVEs(tt.artifact)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildVersionFilter(t *testing.T) {
	tests := []struct {
		name                     string
		versions                 string
		excludeVersions          string
		expectedTargetVersions   map[string]bool
		expectedExcludedVersions map[string]bool
	}{
		{
			name:                     "Inclusive mode: single version",
			versions:                 "20.04",
			excludeVersions:          "",
			expectedTargetVersions:   map[string]bool{"20.04": true},
			expectedExcludedVersions: nil,
		},
		{
			name:                     "Inclusive mode: multiple versions",
			versions:                 "20.04,22.04,24.04",
			excludeVersions:          "",
			expectedTargetVersions:   map[string]bool{"20.04": true, "22.04": true, "24.04": true},
			expectedExcludedVersions: nil,
		},
		{
			name:                     "Inclusive mode: with spaces",
			versions:                 "20.04, 22.04, 24.04",
			excludeVersions:          "",
			expectedTargetVersions:   map[string]bool{"20.04": true, "22.04": true, "24.04": true},
			expectedExcludedVersions: nil,
		},
		{
			name:                     "Exclusive mode: single version",
			versions:                 "",
			excludeVersions:          "14.04",
			expectedTargetVersions:   nil,
			expectedExcludedVersions: map[string]bool{"14.04": true},
		},
		{
			name:                     "Exclusive mode: multiple versions",
			versions:                 "",
			excludeVersions:          "14.04,16.04,24.10",
			expectedTargetVersions:   nil,
			expectedExcludedVersions: map[string]bool{"14.04": true, "16.04": true, "24.10": true},
		},
		{
			name:                     "Exclusive mode: with spaces",
			versions:                 "",
			excludeVersions:          "14.04, 16.04, 24.10",
			expectedTargetVersions:   nil,
			expectedExcludedVersions: map[string]bool{"14.04": true, "16.04": true, "24.10": true},
		},
		{
			name:                     "Auto-detect mode: both empty",
			versions:                 "",
			excludeVersions:          "",
			expectedTargetVersions:   nil,
			expectedExcludedVersions: nil,
		},
		{
			name:                     "Inclusive takes precedence: both provided",
			versions:                 "20.04,22.04",
			excludeVersions:          "14.04,16.04",
			expectedTargetVersions:   map[string]bool{"20.04": true, "22.04": true},
			expectedExcludedVersions: nil,
		},
		{
			name:                     "Inclusive mode: trailing comma ignored",
			versions:                 "20.04,22.04,",
			excludeVersions:          "",
			expectedTargetVersions:   map[string]bool{"20.04": true, "22.04": true},
			expectedExcludedVersions: nil,
		},
		{
			name:                     "Inclusive mode: leading comma ignored",
			versions:                 ",20.04,22.04",
			excludeVersions:          "",
			expectedTargetVersions:   map[string]bool{"20.04": true, "22.04": true},
			expectedExcludedVersions: nil,
		},
		{
			name:                     "Inclusive mode: multiple commas ignored",
			versions:                 "20.04,,22.04",
			excludeVersions:          "",
			expectedTargetVersions:   map[string]bool{"20.04": true, "22.04": true},
			expectedExcludedVersions: nil,
		},
		{
			name:                     "Exclusive mode: trailing comma ignored",
			versions:                 "",
			excludeVersions:          "14.04,16.04,",
			expectedTargetVersions:   nil,
			expectedExcludedVersions: map[string]bool{"14.04": true, "16.04": true},
		},
		{
			name:                     "Inclusive mode: only empty strings falls back to auto-detect",
			versions:                 ",,",
			excludeVersions:          "",
			expectedTargetVersions:   nil,
			expectedExcludedVersions: nil,
		},
		{
			name:                     "Inclusive mode: only whitespace falls back to auto-detect",
			versions:                 "  ,  ,  ",
			excludeVersions:          "",
			expectedTargetVersions:   nil,
			expectedExcludedVersions: nil,
		},
		{
			name:                     "Exclusive mode: only empty strings falls back to auto-detect",
			versions:                 "",
			excludeVersions:          ",,",
			expectedTargetVersions:   nil,
			expectedExcludedVersions: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetVersions, excludedVersions := buildVersionFilter(tt.versions, tt.excludeVersions)
			require.Equal(t, tt.expectedTargetVersions, targetVersions)
			require.Equal(t, tt.expectedExcludedVersions, excludedVersions)
		})
	}
}

func TestShouldIncludeInDelta(t *testing.T) {
	tests := []struct {
		name          string
		inputDir      string
		filePath      string
		changedFiles  map[string]struct{}
		expectedMatch bool
	}{
		{
			name:     "Unix path: file in changed set",
			inputDir: "/tmp/ubuntu-osv",
			filePath: "/tmp/ubuntu-osv/osv/cve/CVE-2024-1234.json",
			changedFiles: map[string]struct{}{
				"osv/cve/CVE-2024-1234.json": {},
			},
			expectedMatch: true,
		},
		{
			name:     "Unix path: file not in changed set",
			inputDir: "/tmp/ubuntu-osv",
			filePath: "/tmp/ubuntu-osv/osv/cve/CVE-2024-9999.json",
			changedFiles: map[string]struct{}{
				"osv/cve/CVE-2024-1234.json": {},
			},
			expectedMatch: false,
		},
		{
			name:     "Nested directory: file in changed set",
			inputDir: "/data/osv",
			filePath: "/data/osv/osv/cve/2024/CVE-2024-1111.json",
			changedFiles: map[string]struct{}{
				"osv/cve/2024/CVE-2024-1111.json": {},
			},
			expectedMatch: true,
		},
		{
			name:     "File already has osv/cve prefix in relative path",
			inputDir: "/workspace",
			filePath: "/workspace/osv/cve/CVE-2024-2222.json",
			changedFiles: map[string]struct{}{
				"osv/cve/CVE-2024-2222.json": {},
			},
			expectedMatch: true,
		},
		{
			name:     "Changed files with leading slash (should not match)",
			inputDir: "/tmp/ubuntu-osv",
			filePath: "/tmp/ubuntu-osv/osv/cve/CVE-2024-3333.json",
			changedFiles: map[string]struct{}{
				"/osv/cve/CVE-2024-3333.json": {}, // Wrong: has leading slash
			},
			expectedMatch: false,
		},
		{
			name:     "Changed files without osv/cve prefix (should not match)",
			inputDir: "/tmp/ubuntu-osv",
			filePath: "/tmp/ubuntu-osv/osv/cve/CVE-2024-4444.json",
			changedFiles: map[string]struct{}{
				"CVE-2024-4444.json": {}, // Wrong: missing osv/cve/ prefix
			},
			expectedMatch: false,
		},
		{
			name:          "Empty changed files set",
			inputDir:      "/tmp/ubuntu-osv",
			filePath:      "/tmp/ubuntu-osv/osv/cve/CVE-2024-5555.json",
			changedFiles:  map[string]struct{}{},
			expectedMatch: false,
		},
		{
			name:     "File outside input directory tree (relative path doesn't match)",
			inputDir: "/tmp/ubuntu-osv/subdir",
			filePath: "/tmp/other-dir/osv/cve/CVE-2024-6666.json",
			changedFiles: map[string]struct{}{
				"osv/cve/CVE-2024-6666.json": {},
			},
			expectedMatch: false, // filepath.Rel will work but path won't match
		},
		{
			name:     "Multiple files in changed set, match one",
			inputDir: "/data/osv",
			filePath: "/data/osv/osv/cve/CVE-2024-7777.json",
			changedFiles: map[string]struct{}{
				"osv/cve/CVE-2024-1111.json": {},
				"osv/cve/CVE-2024-7777.json": {},
				"osv/cve/CVE-2024-9999.json": {},
			},
			expectedMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIncludeInDelta(tt.inputDir, tt.filePath, tt.changedFiles)
			require.Equal(t, tt.expectedMatch, result)
		})
	}
}

func TestRun(t *testing.T) {
	// Create temporary directories for input and output
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Create a simple test OSV file
	testOSVData := `{
		"schema_version": "1.0",
		"id": "USN-1234-1",
		"published": "2024-01-01T00:00:00Z",
		"modified": "2024-01-02T00:00:00Z",
		"details": "Test vulnerability",
		"affected": [{
			"package": {
				"ecosystem": "Ubuntu:22.04:LTS",
				"name": "test-package"
			},
			"ranges": [{
				"type": "ECOSYSTEM",
				"events": [
					{"introduced": "0"},
					{"fixed": "1.2.3"}
				]
			}]
		}],
		"upstream": ["CVE-2024-1234"]
	}`

	// Write test file
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "CVE-2024-1234.json"), []byte(testOSVData), 0o644))

	// Create config
	cfg := Config{
		InputDir:              inputDir,
		OutputDir:             outputDir,
		Versions:              "",
		ExcludeVersions:       "",
		ChangedFilesToday:     "",
		ChangedFilesYesterday: "",
		DateStr:               "2024-01-03",
		YesterdayStr:          "2024-01-02",
		GeneratedTimestamp:    "2024-01-03T00:00:00Z",
		RunTime:               time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
	}

	// Run the function
	err := run(cfg)
	require.NoError(t, err)

	// Verify output artifact was created
	expectedFile := filepath.Join(outputDir, "osv-ubuntu-2204-2024-01-03.json.gz")
	require.FileExists(t, expectedFile)

	// Verify artifact content (decompress and check)
	artifact, err := readArtifact(expectedFile)
	require.NoError(t, err)
	require.Equal(t, "1.0", artifact.SchemaVersion)
	require.Equal(t, "22.04", artifact.UbuntuVersion)
	require.Equal(t, 1, artifact.TotalCVEs)
	require.Equal(t, 1, artifact.TotalPackages)
	require.Contains(t, artifact.Vulnerabilities, "test-package")
	require.Len(t, artifact.Vulnerabilities["test-package"], 1)
	require.Equal(t, "CVE-2024-1234", artifact.Vulnerabilities["test-package"][0].CVE)
}

func TestRunWithDeltaGeneration(t *testing.T) {
	// Create temporary directories
	inputDir := t.TempDir()
	outputDir := t.TempDir()
	changedFilesDir := t.TempDir()

	// Create two test OSV files
	testOSVData1 := `{
		"schema_version": "1.0",
		"id": "USN-1234-1",
		"published": "2024-01-01T00:00:00Z",
		"modified": "2024-01-02T00:00:00Z",
		"affected": [{
			"package": {
				"ecosystem": "Ubuntu:22.04:LTS",
				"name": "changed-today-package"
			},
			"ranges": [{
				"type": "ECOSYSTEM",
				"events": [{"introduced": "0"}, {"fixed": "1.0"}]
			}]
		}],
		"upstream": ["CVE-2024-1111"]
	}`

	testOSVData2 := `{
		"schema_version": "1.0",
		"id": "USN-5678-1",
		"published": "2024-01-01T00:00:00Z",
		"modified": "2024-01-02T00:00:00Z",
		"affected": [{
			"package": {
				"ecosystem": "Ubuntu:22.04:LTS",
				"name": "changed-yesterday-package"
			},
			"ranges": [{
				"type": "ECOSYSTEM",
				"events": [{"introduced": "0"}, {"fixed": "2.0"}]
			}]
		}],
		"upstream": ["CVE-2024-2222"]
	}`

	// Write test files
	osvCveDir := filepath.Join(inputDir, "osv", "cve")
	require.NoError(t, os.MkdirAll(osvCveDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(osvCveDir, "CVE-2024-1111.json"), []byte(testOSVData1), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(osvCveDir, "CVE-2024-2222.json"), []byte(testOSVData2), 0o644))

	// Create changed files lists
	changedTodayFile := filepath.Join(changedFilesDir, "changed_today.txt")
	changedYesterdayFile := filepath.Join(changedFilesDir, "changed_yesterday.txt")
	require.NoError(t, os.WriteFile(changedTodayFile, []byte("osv/cve/CVE-2024-1111.json\n"), 0o644))
	require.NoError(t, os.WriteFile(changedYesterdayFile, []byte("osv/cve/CVE-2024-2222.json\n"), 0o644))

	// Create config with delta generation
	cfg := Config{
		InputDir:              inputDir,
		OutputDir:             outputDir,
		Versions:              "",
		ExcludeVersions:       "",
		ChangedFilesToday:     changedTodayFile,
		ChangedFilesYesterday: changedYesterdayFile,
		DateStr:               "2024-01-03",
		YesterdayStr:          "2024-01-02",
		GeneratedTimestamp:    "2024-01-03T00:00:00Z",
		RunTime:               time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
	}

	// Run
	err := run(cfg)
	require.NoError(t, err)

	// Verify full artifact
	fullArtifact, err := readArtifact(filepath.Join(outputDir, "osv-ubuntu-2204-2024-01-03.json.gz"))
	require.NoError(t, err)
	require.Equal(t, 2, fullArtifact.TotalCVEs)
	require.Equal(t, 2, fullArtifact.TotalPackages)

	// Verify today's delta artifact
	todayDelta, err := readArtifact(filepath.Join(outputDir, "osv-ubuntu-2204-delta-2024-01-03.json.gz"))
	require.NoError(t, err)
	require.Equal(t, 1, todayDelta.TotalCVEs)
	require.Equal(t, 1, todayDelta.TotalPackages)
	require.NotNil(t, todayDelta.Vulnerabilities)
	require.Contains(t, todayDelta.Vulnerabilities, "changed-today-package")
	require.NotEmpty(t, todayDelta.Vulnerabilities["changed-today-package"])
	require.Equal(t, "CVE-2024-1111", todayDelta.Vulnerabilities["changed-today-package"][0].CVE)

	// Verify yesterday's delta artifact
	yesterdayDelta, err := readArtifact(filepath.Join(outputDir, "osv-ubuntu-2204-delta-2024-01-02.json.gz"))
	require.NoError(t, err)
	require.Equal(t, 1, yesterdayDelta.TotalCVEs)
	require.Equal(t, 1, yesterdayDelta.TotalPackages)
	require.NotNil(t, yesterdayDelta.Vulnerabilities)
	require.Contains(t, yesterdayDelta.Vulnerabilities, "changed-yesterday-package")
	require.NotEmpty(t, yesterdayDelta.Vulnerabilities["changed-yesterday-package"])
	require.Equal(t, "CVE-2024-2222", yesterdayDelta.Vulnerabilities["changed-yesterday-package"][0].CVE)
}

func TestRunWithVersionFiltering(t *testing.T) {
	tests := []struct {
		name                 string
		versions             string
		excludeVersions      string
		expectedVersionCount int
		expectedVersions     []string
	}{
		{
			name:                 "Inclusive filtering - single version",
			versions:             "22.04",
			excludeVersions:      "",
			expectedVersionCount: 1,
			expectedVersions:     []string{"22.04"},
		},
		{
			name:                 "Inclusive filtering - multiple versions",
			versions:             "20.04,22.04",
			excludeVersions:      "",
			expectedVersionCount: 2,
			expectedVersions:     []string{"20.04", "22.04"},
		},
		{
			name:                 "Exclusive filtering - exclude one version",
			versions:             "",
			excludeVersions:      "24.04",
			expectedVersionCount: 2,
			expectedVersions:     []string{"20.04", "22.04"},
		},
		{
			name:                 "Auto-detect - no filtering",
			versions:             "",
			excludeVersions:      "",
			expectedVersionCount: 3,
			expectedVersions:     []string{"20.04", "22.04", "24.04"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directories
			inputDir := t.TempDir()
			outputDir := t.TempDir()

			// Create test OSV files for different Ubuntu versions
			for _, ver := range []string{"20.04", "22.04", "24.04"} {
				data := fmt.Sprintf(`{
					"schema_version": "1.0",
					"id": "USN-1234-1",
					"published": "2024-01-01T00:00:00Z",
					"modified": "2024-01-02T00:00:00Z",
					"affected": [{
						"package": {
							"ecosystem": "Ubuntu:%s:LTS",
							"name": "test-package-%s"
						},
						"ranges": [{
							"type": "ECOSYSTEM",
							"events": [{"introduced": "0"}, {"fixed": "1.0"}]
						}]
					}],
					"upstream": ["CVE-2024-%s"]
				}`, ver, strings.ReplaceAll(ver, ".", ""), strings.ReplaceAll(ver, ".", ""))

				filename := fmt.Sprintf("CVE-2024-%s.json", strings.ReplaceAll(ver, ".", ""))
				require.NoError(t, os.WriteFile(filepath.Join(inputDir, filename), []byte(data), 0o644))
			}

			// Create config
			cfg := Config{
				InputDir:              inputDir,
				OutputDir:             outputDir,
				Versions:              tt.versions,
				ExcludeVersions:       tt.excludeVersions,
				ChangedFilesToday:     "",
				ChangedFilesYesterday: "",
				DateStr:               "2024-01-03",
				YesterdayStr:          "2024-01-02",
				GeneratedTimestamp:    "2024-01-03T00:00:00Z",
				RunTime:               time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			}

			// Run
			err := run(cfg)
			require.NoError(t, err)

			// Count artifacts created
			files, err := filepath.Glob(filepath.Join(outputDir, "osv-ubuntu-*.json.gz"))
			require.NoError(t, err)
			require.Equal(t, tt.expectedVersionCount, len(files))

			// Verify expected versions were generated
			for _, expectedVer := range tt.expectedVersions {
				verStr := strings.ReplaceAll(expectedVer, ".", "")
				expectedFile := filepath.Join(outputDir, fmt.Sprintf("osv-ubuntu-%s-2024-01-03.json.gz", verStr))
				require.FileExists(t, expectedFile)

				artifact, err := readArtifact(expectedFile)
				require.NoError(t, err)
				require.Equal(t, expectedVer, artifact.UbuntuVersion)
			}
		})
	}
}

func TestExtractRHELVersion(t *testing.T) {
	tests := []struct {
		name      string
		ecosystem string
		expected  string
	}{
		{
			name:      "RHEL 9 appstream",
			ecosystem: "Red Hat:enterprise_linux:9::appstream",
			expected:  "9",
		},
		{
			name:      "RHEL 8 baseos",
			ecosystem: "Red Hat:enterprise_linux:8::baseos",
			expected:  "8",
		},
		{
			name:      "RHEL 8 crb",
			ecosystem: "Red Hat:enterprise_linux:8::crb",
			expected:  "8",
		},
		{
			name:      "RHEL 10 with minor version",
			ecosystem: "Red Hat:enterprise_linux:10.0",
			expected:  "10",
		},
		{
			name:      "RHEL 10.1 with minor version",
			ecosystem: "Red Hat:enterprise_linux:10.1",
			expected:  "10",
		},
		{
			name:      "RHEL 7 software collections",
			ecosystem: "Red Hat:rhel_software_collections:3::el7",
			expected:  "",
		},
		{
			name:      "RHEL EUS not supported",
			ecosystem: "Red Hat:rhel_e4s:8.8::appstream",
			expected:  "",
		},
		{
			name:      "Empty string",
			ecosystem: "",
			expected:  "",
		},
		{
			name:      "Ubuntu ecosystem",
			ecosystem: "Ubuntu:24.04:LTS",
			expected:  "",
		},
		{
			name:      "RHEL 9 no repository suffix",
			ecosystem: "Red Hat:enterprise_linux:9",
			expected:  "9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRHELVersion(tt.ecosystem)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractCVEIDs(t *testing.T) {
	tests := []struct {
		name     string
		osv      *OSVData
		expected []string
	}{
		{
			name: "CVEs in upstream",
			osv: &OSVData{
				ID:       "RHSA-2025:9978",
				Upstream: []string{"CVE-2025-32462"},
			},
			expected: []string{"CVE-2025-32462"},
		},
		{
			name: "Multiple CVEs in upstream",
			osv: &OSVData{
				ID:       "RHSA-2026:0001",
				Upstream: []string{"CVE-2025-59375", "CVE-2025-6965", "CVE-2025-8176", "CVE-2025-9900"},
			},
			expected: []string{"CVE-2025-59375", "CVE-2025-6965", "CVE-2025-8176", "CVE-2025-9900"},
		},
		{
			name: "CVE in related field as fallback",
			osv: &OSVData{
				ID:      "RHSA-2025:1234",
				Related: []string{"CVE-2025-1111"},
			},
			expected: []string{"CVE-2025-1111"},
		},
		{
			name: "CVE as ID fallback",
			osv: &OSVData{
				ID: "CVE-2025-9999",
			},
			expected: []string{"CVE-2025-9999"},
		},
		{
			name: "No CVE found",
			osv: &OSVData{
				ID:       "RHBA-2025:5678",
				Upstream: []string{"https://example.com"},
			},
			expected: nil,
		},
		{
			name: "Non-CVE upstream entries filtered",
			osv: &OSVData{
				ID:       "RHSA-2025:0001",
				Upstream: []string{"https://bugzilla.redhat.com/123", "CVE-2025-4444"},
			},
			expected: []string{"CVE-2025-4444"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCVEIDs(tt.osv)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRunRHEL(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Create a RHEL OSV advisory with one CVE affecting sudo on RHEL 9
	testData := `{
		"schema_version": "1.7.5",
		"id": "RHSA-2025:9978",
		"published": "2025-07-01T10:06:01Z",
		"modified": "2026-03-18T11:30:33Z",
		"upstream": ["CVE-2025-32462"],
		"summary": "sudo security update",
		"affected": [
			{
				"package": {
					"name": "sudo",
					"ecosystem": "Red Hat:enterprise_linux:9::appstream"
				},
				"ranges": [{"type": "ECOSYSTEM", "events": [{"introduced": "0"}, {"fixed": "0:1.9.5p2-10.el9_6.1"}]}]
			},
			{
				"package": {
					"name": "sudo",
					"ecosystem": "Red Hat:enterprise_linux:9::baseos"
				},
				"ranges": [{"type": "ECOSYSTEM", "events": [{"introduced": "0"}, {"fixed": "0:1.9.5p2-10.el9_6.1"}]}]
			}
		]
	}`
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "RHSA-2025-9978.json"), []byte(testData), 0o644))

	cfg := Config{
		Platform:           "rhel",
		InputDir:           inputDir,
		OutputDir:          outputDir,
		DateStr:            "2026-04-08",
		GeneratedTimestamp: "2026-04-08T00:00:00Z",
		RunTime:            time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	}

	err := runRHEL(cfg)
	require.NoError(t, err)

	// Verify artifact was created
	expectedFile := filepath.Join(outputDir, "osv-rhel-9-2026-04-08.json.gz")
	require.FileExists(t, expectedFile)

	artifact, err := readRHELArtifact(expectedFile)
	require.NoError(t, err)
	require.Equal(t, "1.0", artifact.SchemaVersion)
	require.Equal(t, "9", artifact.RHELVersion)
	require.Equal(t, 1, artifact.TotalCVEs)
	require.Contains(t, artifact.Vulnerabilities, "sudo")
	// Deduplication: sudo appears in both appstream and baseos, should be deduplicated
	require.Len(t, artifact.Vulnerabilities["sudo"], 1)
	require.Equal(t, "CVE-2025-32462", artifact.Vulnerabilities["sudo"][0].CVE)
	require.Equal(t, "0:1.9.5p2-10.el9_6.1", artifact.Vulnerabilities["sudo"][0].Fixed)
}

func TestRunRHELMultiCVEAdvisory(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Advisory with 2 CVEs and packages across RHEL 8 and 9
	testData := `{
		"schema_version": "1.7.5",
		"id": "RHSA-2026:0001",
		"published": "2026-01-05T10:11:47Z",
		"modified": "2026-04-03T10:05:48Z",
		"upstream": ["CVE-2025-1111", "CVE-2025-2222"],
		"affected": [
			{
				"package": {"name": "curl", "ecosystem": "Red Hat:enterprise_linux:9::baseos"},
				"ranges": [{"type": "ECOSYSTEM", "events": [{"introduced": "0"}, {"fixed": "0:7.76.1-29.el9_4.2"}]}]
			},
			{
				"package": {"name": "curl", "ecosystem": "Red Hat:enterprise_linux:8::baseos"},
				"ranges": [{"type": "ECOSYSTEM", "events": [{"introduced": "0"}, {"fixed": "0:7.61.1-34.el8_10"}]}]
			}
		]
	}`
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "RHSA-2026-0001.json"), []byte(testData), 0o644))

	cfg := Config{
		Platform:           "rhel",
		InputDir:           inputDir,
		OutputDir:          outputDir,
		DateStr:            "2026-04-08",
		GeneratedTimestamp: "2026-04-08T00:00:00Z",
		RunTime:            time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	}

	err := runRHEL(cfg)
	require.NoError(t, err)

	// Should produce artifacts for both RHEL 8 and 9
	rhel9, err := readRHELArtifact(filepath.Join(outputDir, "osv-rhel-9-2026-04-08.json.gz"))
	require.NoError(t, err)
	require.Equal(t, 2, rhel9.TotalCVEs)
	require.Len(t, rhel9.Vulnerabilities["curl"], 2)

	rhel8, err := readRHELArtifact(filepath.Join(outputDir, "osv-rhel-8-2026-04-08.json.gz"))
	require.NoError(t, err)
	require.Equal(t, 2, rhel8.TotalCVEs)
	require.Len(t, rhel8.Vulnerabilities["curl"], 2)
}

func TestRunRHELVersionFiltering(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	testData := `{
		"schema_version": "1.7.5",
		"id": "RHSA-2025:1234",
		"published": "2025-01-01T00:00:00Z",
		"modified": "2025-01-02T00:00:00Z",
		"upstream": ["CVE-2025-5555"],
		"affected": [
			{"package": {"name": "pkg", "ecosystem": "Red Hat:enterprise_linux:8::baseos"}, "ranges": [{"type": "ECOSYSTEM", "events": [{"introduced": "0"}, {"fixed": "0:1.0-1.el8"}]}]},
			{"package": {"name": "pkg", "ecosystem": "Red Hat:enterprise_linux:9::baseos"}, "ranges": [{"type": "ECOSYSTEM", "events": [{"introduced": "0"}, {"fixed": "0:1.0-1.el9"}]}]}
		]
	}`
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "RHSA-2025-1234.json"), []byte(testData), 0o644))

	cfg := Config{
		Platform:           "rhel",
		InputDir:           inputDir,
		OutputDir:          outputDir,
		Versions:           "9",
		DateStr:            "2026-04-08",
		GeneratedTimestamp: "2026-04-08T00:00:00Z",
		RunTime:            time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	}

	err := runRHEL(cfg)
	require.NoError(t, err)

	// Only RHEL 9 should be generated
	require.FileExists(t, filepath.Join(outputDir, "osv-rhel-9-2026-04-08.json.gz"))
	_, err = os.Stat(filepath.Join(outputDir, "osv-rhel-8-2026-04-08.json.gz"))
	require.True(t, os.IsNotExist(err), "RHEL 8 artifact should not exist when filtering to version 9")
}

func readRHELArtifact(path string) (*RHELArtifactData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzReader.Close()

	var artifact RHELArtifactData
	if err := json.NewDecoder(gzReader).Decode(&artifact); err != nil {
		return nil, err
	}

	return &artifact, nil
}

func readArtifact(path string) (*ArtifactData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzReader.Close()

	var artifact ArtifactData
	if err := json.NewDecoder(gzReader).Decode(&artifact); err != nil {
		return nil, err
	}

	return &artifact, nil
}
