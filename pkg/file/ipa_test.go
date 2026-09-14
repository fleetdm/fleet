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
	return infoPlistNamed(bundleID, "Test", "1.0", iOS)
}

func infoPlistNamed(bundleID, name, version string, iOS bool) string {
	requiresIPhoneOS := ""
	if iOS {
		requiresIPhoneOS = "<key>LSRequiresIPhoneOS</key><true/>"
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>` + bundleID + `</string>
<key>CFBundleName</key><string>` + name + `</string>
<key>CFBundleShortVersionString</key><string>` + version + `</string>
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

	// a framework plist without the key, coming after the app's own plist,
	// neither undoes ipa detection nor replaces the app's metadata
	nestedTfr, err := fleet.NewKeepFileReader(writeZip(t, [][2]string{
		{"Payload/App.app/Info.plist", infoPlist("com.example.ios", true)},
		{"Payload/App.app/Frameworks/Bar.framework/Info.plist", infoPlist("com.example.framework", false)},
	}))
	require.NoError(t, err)
	defer nestedTfr.Close()

	meta, err = file.ExtractZIPMetadata(nestedTfr)
	require.NoError(t, err)
	require.Equal(t, "com.example.ios", meta.BundleIdentifier)
}

// Metadata comes from the app's own Payload/<App>.app/Info.plist. Embedded
// frameworks, extensions and watch apps ship their own, and any of them may be
// written to the archive after the app's.
func TestExtractZIPMetadataAppPlistSelection(t *testing.T) {
	app := infoPlistNamed("com.fleetdm.HelloWorld", "HelloWorld", "1.0", true)
	framework := infoPlistNamed("com.acme.AcmeKit", "AcmeKit", "9.9.9", false)

	tests := []struct {
		name    string
		entries [][2]string
	}{
		{
			name: "framework plist written after the app's",
			entries: [][2]string{
				{"Payload/HelloWorld.app/Info.plist", app},
				{"Payload/HelloWorld.app/Frameworks/AcmeKit.framework/Info.plist", framework},
			},
		},
		{
			name: "framework plist written before the app's",
			entries: [][2]string{
				{"Payload/HelloWorld.app/Frameworks/AcmeKit.framework/Info.plist", framework},
				{"Payload/HelloWorld.app/Info.plist", app},
			},
		},
		{
			name: "file whose name merely ends in Info.plist",
			entries: [][2]string{
				{"Payload/HelloWorld.app/Info.plist", app},
				{"Payload/HelloWorld.app/SomethingInfo.plist", framework},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tfr, err := fleet.NewKeepFileReader(writeZip(t, tt.entries))
			require.NoError(t, err)
			defer tfr.Close()

			meta, err := file.ExtractZIPMetadata(tfr)
			require.NoError(t, err)
			require.Equal(t, "com.fleetdm.HelloWorld", meta.BundleIdentifier)
			require.Equal(t, "HelloWorld", meta.Name)
			require.Equal(t, "1.0", meta.Version)
		})
	}

	t.Run("archive with no app plist", func(t *testing.T) {
		tfr, err := fleet.NewKeepFileReader(writeZip(t, [][2]string{
			{"Payload/HelloWorld.app/Frameworks/AcmeKit.framework/Info.plist", infoPlistNamed("com.acme.AcmeKit", "AcmeKit", "9.9.9", true)},
		}))
		require.NoError(t, err)
		defer tfr.Close()

		meta, err := file.ExtractZIPMetadata(tfr)
		require.ErrorIs(t, err, file.ErrInvalidType)
		require.Nil(t, meta)
	})
}
