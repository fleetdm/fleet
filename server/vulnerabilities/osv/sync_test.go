package osv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestRemoveOldOSVArtifacts(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	today := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)

	// Create some test files
	currentFile := filepath.Join(tmpDir, "osv-ubuntu-2204-2026-03-30.json.gz")
	oldFile := filepath.Join(tmpDir, "osv-ubuntu-2204-2026-03-29.json.gz")
	otherFile := filepath.Join(tmpDir, "some-other-file.json")

	for _, file := range []string{currentFile, oldFile, otherFile} {
		err := os.WriteFile(file, []byte("test"), 0o644)
		require.NoError(t, err)
	}

	// Create a directory that matches the OSV pattern
	osvDir := filepath.Join(tmpDir, "osv-ubuntu-test.json.gz")
	err := os.Mkdir(osvDir, 0o755)
	require.NoError(t, err)

	// Run the cleanup with 2204 as successfully downloaded (should remove old file but keep current)
	err = removeOldOSVArtifacts(today, tmpDir, []string{"2204"})
	require.NoError(t, err)

	// Check that old file was removed
	_, err = os.Stat(oldFile)
	require.True(t, os.IsNotExist(err))

	// Check that current file still exists
	_, err = os.Stat(currentFile)
	require.NoError(t, err)

	// Check that other file still exists (should not be touched)
	_, err = os.Stat(otherFile)
	require.NoError(t, err)

	// Check that directory still exists (should be skipped, not removed)
	stat, err := os.Stat(osvDir)
	require.NoError(t, err)
	require.True(t, stat.IsDir())
}

func TestRemoveOldOSVArtifactsPreservesFailedVersions(t *testing.T) {
	tmpDir := t.TempDir()
	today := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)

	// Create artifacts for two versions, both old and new
	files := []string{
		"osv-ubuntu-2204-2026-03-30.json.gz", // Today's 22.04 (just downloaded)
		"osv-ubuntu-2204-2026-03-29.json.gz", // Yesterday's 22.04 (should be removed)
		"osv-ubuntu-2404-2026-03-29.json.gz", // Yesterday's 24.04 (should be KEPT - download failed)
	}

	for _, file := range files {
		err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0o644)
		require.NoError(t, err)
	}

	// Only 2204 downloaded successfully, 2404 failed
	err := removeOldOSVArtifacts(today, tmpDir, []string{"2204"})
	require.NoError(t, err)

	// Today's 2204 should still exist
	_, err = os.Stat(filepath.Join(tmpDir, "osv-ubuntu-2204-2026-03-30.json.gz"))
	require.NoError(t, err)

	// Old 2204 should be removed (new one downloaded)
	_, err = os.Stat(filepath.Join(tmpDir, "osv-ubuntu-2204-2026-03-29.json.gz"))
	require.True(t, os.IsNotExist(err))

	// Old 2404 should be PRESERVED (download failed, need last-known-good)
	_, err = os.Stat(filepath.Join(tmpDir, "osv-ubuntu-2404-2026-03-29.json.gz"))
	require.NoError(t, err, "old 2404 artifact should be preserved when download fails")
}

func TestRemoveOldOSVArtifactsWithSkippedVersions(t *testing.T) {
	tmpDir := t.TempDir()
	today := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)

	// Artifact for 2204 already exists with correct checksum (will be skipped)
	files := []string{
		"osv-ubuntu-2204-2026-03-30.json.gz", // Today's 22.04 (already exists, will be skipped)
		"osv-ubuntu-2204-2026-03-29.json.gz", // Yesterday's 22.04 (should be REMOVED even though skipped)
		"osv-ubuntu-2204-2026-03-28.json.gz", // Day before yesterday's 22.04 (should be REMOVED)
	}

	for _, file := range files {
		err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0o644)
		require.NoError(t, err)
	}

	err := removeOldOSVArtifacts(today, tmpDir, []string{"2204"})
	require.NoError(t, err)

	// Today's 2204 should still exist
	_, err = os.Stat(filepath.Join(tmpDir, "osv-ubuntu-2204-2026-03-30.json.gz"))
	require.NoError(t, err, "current artifact should be preserved")

	// Old 2204 files should be REMOVED (this is the bug fix!)
	_, err = os.Stat(filepath.Join(tmpDir, "osv-ubuntu-2204-2026-03-29.json.gz"))
	require.True(t, os.IsNotExist(err), "old artifact from yesterday should be removed even when version was skipped")

	_, err = os.Stat(filepath.Join(tmpDir, "osv-ubuntu-2204-2026-03-28.json.gz"))
	require.True(t, os.IsNotExist(err), "old artifact from day before should be removed even when version was skipped")
}

func TestRemoveOldOSVArtifactsDateBoundaryRace(t *testing.T) {
	tmpDir := t.TempDir()
	// now is April 10 but the release only created April 9 artifacts.
	today := time.Date(2026, 4, 10, 0, 5, 0, 0, time.UTC)

	files := []string{
		"osv-ubuntu-2404-2026-04-09.json.gz", // Yesterday's artifact (only one available)
	}

	for _, file := range files {
		err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0o644)
		require.NoError(t, err)
	}

	// 2404 is in NotInRelease, not Skipped
	// so removeOldOSVArtifacts should not touch it
	err := removeOldOSVArtifacts(today, tmpDir, []string{})
	require.NoError(t, err)

	// Yesterday's artifact must still exist since the version wasn't in the successful set
	_, err = os.Stat(filepath.Join(tmpDir, "osv-ubuntu-2404-2026-04-09.json.gz"))
	require.NoError(t, err, "old artifact must be preserved when version is not in release")
}

func TestGetNeededUbuntuVersions(t *testing.T) {
	tests := []struct {
		name     string
		osVers   *fleet.OSVersions
		expected []string
	}{
		{
			name: "multiple ubuntu versions",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "ubuntu", Version: "22.04.8 LTS"},
					{Platform: "ubuntu", Version: "20.04.1 LTS"},
				},
			},
			expected: []string{"2204", "2004"},
		},
		{
			name: "duplicate ubuntu versions",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "ubuntu", Version: "22.04.8 LTS"},
					{Platform: "ubuntu", Version: "22.04.3 LTS"},
					{Platform: "ubuntu", Version: "22.04.1 LTS"},
				},
			},
			expected: []string{"2204"},
		},
		{
			name: "non-ubuntu platforms ignored",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "ubuntu", Version: "22.04.8 LTS"},
					{Platform: "rhel", Version: "8.5"},
					{Platform: "windows", Version: "10.0.19041"},
				},
			},
			expected: []string{"2204"},
		},
		{
			name: "no ubuntu platforms",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "rhel", Version: "8.5"},
					{Platform: "windows", Version: "10.0.19041"},
				},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNeededUbuntuVersions(tt.osVers)
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestGetNeededRHELVersions(t *testing.T) {
	tests := []struct {
		name     string
		osVers   *fleet.OSVersions
		expected []string
	}{
		{
			name: "multiple RHEL versions",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "rhel", Name: "Red Hat Enterprise Linux 8.10.0", Version: "8.10.0"},
					{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.4.0", Version: "9.4.0"},
				},
			},
			expected: []string{"8", "9"},
		},
		{
			name: "duplicate major versions deduplicated",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.2.0", Version: "9.2.0"},
					{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.4.0", Version: "9.4.0"},
				},
			},
			expected: []string{"9"},
		},
		{
			name: "Fedora skipped",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.4.0", Version: "9.4.0"},
					{Platform: "rhel", Name: "Fedora Linux 36.0.0", Version: "36.0.0"},
				},
			},
			expected: []string{"9"},
		},
		{
			name: "non-RHEL platforms ignored",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.4.0", Version: "9.4.0"},
					{Platform: "ubuntu", Name: "Ubuntu 22.04.8 LTS", Version: "22.04.8 LTS"},
					{Platform: "windows", Name: "Windows 10", Version: "10.0.19041"},
				},
			},
			expected: []string{"9"},
		},
		{
			name: "no RHEL platforms",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "ubuntu", Name: "Ubuntu 22.04.8 LTS", Version: "22.04.8 LTS"},
				},
			},
			expected: []string{},
		},
		{
			name: "only Fedora",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "rhel", Name: "Fedora Linux 36.0.0", Version: "36.0.0"},
				},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNeededRHELVersions(tt.osVers)
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestRemoveOldRHELOSVArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	today := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	files := []string{
		"osv-rhel-9-2026-04-08.json.gz",      // today — keep
		"osv-rhel-9-2026-04-07.json.gz",      // yesterday — remove
		"osv-rhel-8-2026-04-07.json.gz",      // yesterday, different version, not in successful — keep
		"osv-ubuntu-2204-2026-04-07.json.gz", // ubuntu — not touched
		"some-other-file.json",               // unrelated — not touched
	}

	for _, file := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0o644))
	}

	err := removeOldRHELOSVArtifacts(today, tmpDir, []string{"9"})
	require.NoError(t, err)

	// Today's RHEL 9 — kept
	_, err = os.Stat(filepath.Join(tmpDir, "osv-rhel-9-2026-04-08.json.gz"))
	require.NoError(t, err)

	// Yesterday's RHEL 9 — removed (successfully downloaded today)
	_, err = os.Stat(filepath.Join(tmpDir, "osv-rhel-9-2026-04-07.json.gz"))
	require.True(t, os.IsNotExist(err))

	// Yesterday's RHEL 8 — kept (not in successful list, last-known-good)
	_, err = os.Stat(filepath.Join(tmpDir, "osv-rhel-8-2026-04-07.json.gz"))
	require.NoError(t, err)

	// Ubuntu artifact — not touched
	_, err = os.Stat(filepath.Join(tmpDir, "osv-ubuntu-2204-2026-04-07.json.gz"))
	require.NoError(t, err)

	// Other file — not touched
	_, err = os.Stat(filepath.Join(tmpDir, "some-other-file.json"))
	require.NoError(t, err)
}

func TestRHELOSVFilename(t *testing.T) {
	date := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		version  string
		expected string
	}{
		{"9", "osv-rhel-9-2026-04-08.json.gz"},
		{"8", "osv-rhel-8-2026-04-08.json.gz"},
		{"10", "osv-rhel-10-2026-04-08.json.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			require.Equal(t, tt.expected, rhelOSVFilename(tt.version, date))
		})
	}
}

func TestOSVFilename(t *testing.T) {
	date := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		ubuntuVersion string
		expected      string
	}{
		{"2204", "osv-ubuntu-2204-2026-03-30.json.gz"},
		{"2004", "osv-ubuntu-2004-2026-03-30.json.gz"},
		{"1804", "osv-ubuntu-1804-2026-03-30.json.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.ubuntuVersion, func(t *testing.T) {
			result := osvFilename(tt.ubuntuVersion, date)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestComputeFileSHA256(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Write test content
	testContent := []byte("test content")
	err := os.WriteFile(testFile, testContent, 0o644)
	require.NoError(t, err)

	// Compute SHA256
	digest, err := computeFileSHA256(testFile)
	require.NoError(t, err)

	// Expected digest for "test content"
	// echo -n "test content" | sha256sum
	// 6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72
	expected := "sha256:6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
	require.Equal(t, expected, digest)

	_, err = computeFileSHA256(filepath.Join(tmpDir, "nonexistent.txt"))
	require.Error(t, err)
}

func TestSyncOSVFaultTolerance(t *testing.T) {
	tmpDir := t.TempDir()
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	// Create a mock release with only some artifacts available
	release := &ReleaseInfo{
		TagName: "cve-202604010000",
		Assets: map[string]*AssetInfo{
			"osv-ubuntu-2204-2026-04-01.json.gz": {
				Name:   "osv-ubuntu-2204-2026-04-01.json.gz",
				ID:     12345,
				Digest: "sha256:abc123",
			},
		},
	}

	versions := []string{"2204", "2504"}

	// Mock download function that always fails
	mockDownload := func(ctx context.Context, assetID int64, dstPath string) error {
		return errors.New("mock download failure")
	}

	result, err := syncOSVWithDownloader(context.Background(), tmpDir, versions, date, release, mockDownload, osvFilename)
	require.Error(t, err)
	require.NotNil(t, result)

	require.Contains(t, result.NotInRelease, "2504", "2504 artifact not in release")
	require.Contains(t, result.Failed, "2204", "2204 download failed, should be in Failed")
}

func TestSyncOSVChecksumMatch(t *testing.T) {
	tmpDir := t.TempDir()
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	// Create a test file with known content
	testContent := []byte("test content")
	filename := "osv-ubuntu-2204-2026-04-01.json.gz"
	testFile := filepath.Join(tmpDir, filename)
	err := os.WriteFile(testFile, testContent, 0o644)
	require.NoError(t, err)

	// Compute the digest
	digest, err := computeFileSHA256(testFile)
	require.NoError(t, err)

	// Create a mock release with matching digest
	release := &ReleaseInfo{
		TagName: "cve-202604010000",
		Assets: map[string]*AssetInfo{
			filename: {
				Name:   filename,
				ID:     12345,
				Digest: digest,
			},
		},
	}

	result, err := SyncOSV(context.Background(), tmpDir, []string{"2204"}, date, release)
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Contains(t, result.Skipped, "2204")
	require.Empty(t, result.Downloaded)
	require.Empty(t, result.Failed)
}
