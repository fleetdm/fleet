package file_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func infoPlist(bundleID string, iOS bool) string {
	requiresIPhoneOS := ""
	if iOS {
		requiresIPhoneOS = "<key>LSRequiresIPhoneOS</key><true/>"
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>` + bundleID + `</string>
<key>CFBundleName</key><string>Test</string>
<key>CFBundleShortVersionString</key><string>1.0</string>
` + requiresIPhoneOS + `
</dict></plist>`
}

// writeZip builds a zip at a temp path with the given entries in order.
func writeZip(t *testing.T, entries [][2]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pkg.zip")
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, e := range entries {
		w, err := zw.Create(e[0])
		require.NoError(t, err)
		_, err = w.Write([]byte(e[1]))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return path
}

func TestExtractZIPMetadata(t *testing.T) {
	// a valid ipa returns metadata without error
	tfr, err := fleet.NewKeepFileReader(filepath.Join("testdata", "software-installers", "ipa_test.ipa"))
	require.NoError(t, err)
	defer tfr.Close()

	meta, err := file.ExtractZIPMetadata(tfr)
	require.NoError(t, err)
	require.NotNil(t, meta)

	// a zip-based package with no Info.plist at all is not an ipa. This has the
	// same magic bytes as a Windows .zip installer and likewise has no Info.plist,
	// so it covers that case too; we use a real .msix here.
	msixTfr, err := fleet.NewKeepFileReader(filepath.Join("testdata", "software-installers", "msix_test.msix"))
	require.NoError(t, err)
	defer msixTfr.Close()

	meta, err = file.ExtractZIPMetadata(msixTfr)
	require.ErrorIs(t, err, file.ErrInvalidType)
	require.Nil(t, meta)

	// the same msix renamed to a .msi extension still returns invalid type
	obfuscatedPath := filepath.Join(t.TempDir(), "not-really-an.msi")
	require.NoError(t, file.Copy(filepath.Join("testdata", "software-installers", "msix_test.msix"), obfuscatedPath, 0o644))

	obfuscatedTfr, err := fleet.NewKeepFileReader(obfuscatedPath)
	require.NoError(t, err)
	defer obfuscatedTfr.Close()

	meta, err = file.ExtractZIPMetadata(obfuscatedTfr)
	require.ErrorIs(t, err, file.ErrInvalidType)
	require.Nil(t, meta)

	// a macOS .app zip has an Info.plist but no LSRequiresIPhoneOS, so it is not an ipa
	macTfr, err := fleet.NewKeepFileReader(writeZip(t, [][2]string{
		{"MacApp.app/Contents/Info.plist", infoPlist("com.example.mac", false)},
	}))
	require.NoError(t, err)
	defer macTfr.Close()

	meta, err = file.ExtractZIPMetadata(macTfr)
	require.ErrorIs(t, err, file.ErrInvalidType)
	require.Nil(t, meta)

	// once LSRequiresIPhoneOS is seen it stays set, so a framework plist without
	// the key coming after the app plist doesn't undo ipa detection
	latchTfr, err := fleet.NewKeepFileReader(writeZip(t, [][2]string{
		{"Payload/App.app/Info.plist", infoPlist("com.example.ios", true)},
		{"Payload/App.app/Frameworks/Bar.framework/Info.plist", infoPlist("com.example.framework", false)},
	}))
	require.NoError(t, err)
	defer latchTfr.Close()

	meta, err = file.ExtractZIPMetadata(latchTfr)
	require.NoError(t, err)
	require.NotNil(t, meta)
}
