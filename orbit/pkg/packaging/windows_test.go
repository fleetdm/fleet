package packaging

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateVersionInfo(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "somefile.json")

	t.Run("invalid version parts", func(t *testing.T) {
		parts := []string{"1", "a", "3", "c"}
		result, err := createVersionInfo(parts, manifestPath)
		require.ErrorContains(t, err, "error parsing version part")
		require.Nil(t, result)
	})

	t.Run("creates a VersionInfo struct", func(t *testing.T) {
		parts := []string{"1", "2", "3", "0"}
		result, err := createVersionInfo(parts, manifestPath)
		require.NoError(t, err)

		require.NotNil(t, result)

		require.Equal(t, result.FixedFileInfo.FileVersion.Major, 1)
		require.Equal(t, result.FixedFileInfo.FileVersion.Minor, 2)
		require.Equal(t, result.FixedFileInfo.FileVersion.Patch, 3)
		require.Equal(t, result.FixedFileInfo.FileVersion.Build, 0)

		require.Equal(t, result.FixedFileInfo.ProductVersion.Major, 1)
		require.Equal(t, result.FixedFileInfo.ProductVersion.Minor, 2)
		require.Equal(t, result.FixedFileInfo.ProductVersion.Patch, 3)
		require.Equal(t, result.FixedFileInfo.ProductVersion.Build, 0)

		require.Equal(t, result.FixedFileInfo.FileFlagsMask, "3f")
		require.Equal(t, result.FixedFileInfo.FileFlags, "00")
		require.Equal(t, result.FixedFileInfo.FileOS, "040004")
		require.Equal(t, result.FixedFileInfo.FileType, "01")
		require.Equal(t, result.FixedFileInfo.FileSubType, "00")

		require.Equal(t, result.StringFileInfo.Comments, "Fleet osquery")
		require.Equal(t, result.StringFileInfo.CompanyName, "Fleet Device Management (fleetdm.com)")
		require.Equal(t, result.StringFileInfo.FileDescription, "Fleet osquery installer")
		require.Equal(t, result.StringFileInfo.FileVersion, "1.2.3.0")
		require.Equal(t, result.StringFileInfo.LegalCopyright, fmt.Sprintf("%d Fleet Device Management Inc.", time.Now().Year()))
		require.Equal(t, result.StringFileInfo.ProductName, "Fleet osquery")
		require.Equal(t, result.StringFileInfo.ProductVersion, "1.2.3.0")
		require.Equal(t, result.ManifestPath, manifestPath)
	})
}

func TestWriteResourceSyso(t *testing.T) {
	t.Run("removes intermediary manifest.xml file", func(t *testing.T) {
		path := t.TempDir()
		opt := Options{Version: "1.2.3"}

		err := writeResourceSyso(opt, path)
		require.NoError(t, err)

		require.NoFileExists(t, filepath.Join(path, "manifest.xml"))
	})
}

func TestSanitizeVersion(t *testing.T) {
	testCases := []struct {
		Version   string
		Parts     []string
		ErrorsOut bool
	}{
		{Version: "4.13.0", Parts: []string{"4", "13", "0", "0"}},
		{Version: "4.13.0.1", Parts: []string{"4", "13", "0", "1"}},

		// We need to support this form of semantic versioning (with pre-releases)
		// to comply with semantic versioning required by goreleaser to allow building
		// orbit pre-releases.
		{Version: "4.13.0-1", Parts: []string{"4", "13", "0", "1"}},
		{Version: "4.13.0-alpha", Parts: []string{"4", "13", "0", "alpha"}},
		{Version: "4.13.0-", ErrorsOut: true},

		{Version: "4.13.0.1.2", Parts: []string{"4", "13", "0", "1"}},
		{Version: "4", ErrorsOut: true},
		{Version: "4.13", ErrorsOut: true},
		{Version: "bad bad", ErrorsOut: true},
	}

	for _, tC := range testCases {
		result, err := SanitizeVersion(tC.Version)

		if tC.ErrorsOut {
			require.Error(t, err)
		}

		require.Equal(t, tC.Parts, result)
	}
}

func TestDownloadAndExtractZip(t *testing.T) {
	t.Parallel()
	path := t.TempDir()
	client := fleethttp.NewClient()
	err := downloadAndExtractZip(client, wixDownload, path)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(path, "heat.exe"))
	assert.FileExists(t, filepath.Join(path, "candle.exe"))
	assert.FileExists(t, filepath.Join(path, "light.exe"))
}
