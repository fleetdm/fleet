package osv

import (
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

	// Run the cleanup (should remove old file but keep current)
	err := removeOldOSVArtifacts(today, tmpDir)
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

func TestSyncOSV_FaultTolerance(t *testing.T) {
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

	result, err := SyncOSV(tmpDir, versions, date, release)
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Contains(t, result.Skipped, "2504")
	require.Contains(t, result.Failed, "2204")
}

func TestSyncOSV_ChecksumMatch(t *testing.T) {
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

	result, err := SyncOSV(tmpDir, []string{"2204"}, date, release)
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Contains(t, result.Skipped, "2204")
	require.Empty(t, result.Downloaded)
	require.Empty(t, result.Failed)
}
