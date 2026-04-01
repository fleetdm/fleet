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
		err := os.WriteFile(file, []byte("test"), 0644)
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

func TestGetExistingOSVArtifacts(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	today := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)

	// Create some test files
	currentFile := filepath.Join(tmpDir, "osv-ubuntu-2204-2026-03-30.json.gz")
	currentFile2 := filepath.Join(tmpDir, "osv-ubuntu-2004-2026-03-30.json.gz")
	oldFile := filepath.Join(tmpDir, "osv-ubuntu-2204-2026-03-29.json.gz")
	otherFile := filepath.Join(tmpDir, "some-other-file.json")

	for _, file := range []string{currentFile, currentFile2, oldFile, otherFile} {
		err := os.WriteFile(file, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Get existing artifacts for today
	existing, err := getExistingOSVArtifacts(today, tmpDir)
	require.NoError(t, err)

	// Should only find current files
	_, exists1 := existing[filepath.Base(currentFile)]
	require.True(t, exists1)
	_, exists2 := existing[filepath.Base(currentFile2)]
	require.True(t, exists2)
	_, exists3 := existing[filepath.Base(oldFile)]
	require.False(t, exists3)
	_, exists4 := existing[filepath.Base(otherFile)]
	require.False(t, exists4)
	require.Len(t, existing, 2)
}

func TestWhatToDownloadOSV(t *testing.T) {
	date := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		osVers   *fleet.OSVersions
		existing map[string]struct{}
		expected []string
	}{
		{
			name: "no existing artifacts",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "ubuntu", Version: "22.04.8 LTS"},
					{Platform: "ubuntu", Version: "20.04.1 LTS"},
				},
			},
			existing: map[string]struct{}{},
			expected: []string{"2204", "2004"},
		},
		{
			name: "some existing artifacts",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "ubuntu", Version: "22.04.8 LTS"},
					{Platform: "ubuntu", Version: "20.04.1 LTS"},
				},
			},
			existing: map[string]struct{}{
				"osv-ubuntu-2204-2026-03-30.json.gz": {},
			},
			expected: []string{"2004"},
		},
		{
			name: "all artifacts exist",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "ubuntu", Version: "22.04.8 LTS"},
					{Platform: "ubuntu", Version: "20.04.1 LTS"},
				},
			},
			existing: map[string]struct{}{
				"osv-ubuntu-2204-2026-03-30.json.gz": {},
				"osv-ubuntu-2004-2026-03-30.json.gz": {},
			},
			expected: []string{},
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
			existing: map[string]struct{}{},
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
			existing: map[string]struct{}{},
			expected: []string{"2204"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := whatToDownloadOSV(tt.osVers, tt.existing, date)
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
