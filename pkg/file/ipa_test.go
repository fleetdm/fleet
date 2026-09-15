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
	require.Equal(t, "com.example.ios", meta.BundleIdentifier)
}

// An Expo/React Native ipa embeds CocoaPods resource bundles whose Info.plist
// carries the pod's name and version, so identity has to come from the app
// bundle's own plist and not from whichever plist the zip happens to list last.
func TestExtractZIPMetadataUsesMainAppInfoPlist(t *testing.T) {
	app := infoPlistNamed("com.example.ios", "Example", "3.2.1", true)
	privacy := infoPlistNamed("org.cocoapods.React-Core-privacy", "React-Core_privacy", "0.81.5", false)
	framework := infoPlistNamed("org.cocoapods.hermes-engine", "hermes", "0.81.5", false)
	appex := infoPlistNamed("com.example.ios.Widget", "Widget", "9.9.9", true)

	requireAppMetadata := func(t *testing.T, entries [][2]string) {
		t.Helper()
		tfr, err := fleet.NewKeepFileReader(writeZip(t, entries))
		require.NoError(t, err)
		defer tfr.Close()

		meta, err := file.ExtractZIPMetadata(tfr)
		require.NoError(t, err)
		require.NotNil(t, meta)
		require.Equal(t, "com.example.ios", meta.BundleIdentifier)
		require.Equal(t, []string{"com.example.ios"}, meta.PackageIDs)
		require.Equal(t, "Example", meta.Name)
		require.Equal(t, "3.2.1", meta.Version)
	}

	t.Run("nested plists listed after the app plist", func(t *testing.T) {
		requireAppMetadata(t, [][2]string{
			{"Payload/App.app/Info.plist", app},
			{"Payload/App.app/Frameworks/hermes.framework/Info.plist", framework},
			{"Payload/App.app/React-Core_privacy.bundle/Info.plist", privacy},
			{"Payload/App.app/PlugIns/Widget.appex/Info.plist", appex},
		})
	})

	t.Run("nested plists listed before the app plist", func(t *testing.T) {
		requireAppMetadata(t, [][2]string{
			{"Payload/App.app/React-Core_privacy.bundle/Info.plist", privacy},
			{"Payload/App.app/PlugIns/Widget.appex/Info.plist", appex},
			{"Payload/App.app/Info.plist", app},
		})
	})

	t.Run("ipa still detected when only a nested plist sets LSRequiresIPhoneOS", func(t *testing.T) {
		tfr, err := fleet.NewKeepFileReader(writeZip(t, [][2]string{
			{"Payload/App.app/Info.plist", infoPlistNamed("com.example.ios", "Example", "3.2.1", false)},
			{"Payload/App.app/PlugIns/Widget.appex/Info.plist", appex},
		}))
		require.NoError(t, err)
		defer tfr.Close()

		meta, err := file.ExtractZIPMetadata(tfr)
		require.NoError(t, err)
		require.NotNil(t, meta)
		require.Equal(t, "com.example.ios", meta.BundleIdentifier)
		require.Equal(t, "3.2.1", meta.Version)
	})

	t.Run("merges plists in order when there is no Payload app bundle", func(t *testing.T) {
		// A plist carrying none of the keys, such as a compiled Core Data
		// model, must not clear identity read from an earlier entry.
		keyless := `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict>
<key>NSPersistenceFrameworkVersion</key><string>1234</string></dict></plist>`
		tfr, err := fleet.NewKeepFileReader(writeZip(t, [][2]string{
			{"Wrapped/Payload/App.app/Info.plist", infoPlistNamed("com.example.wrapped", "Wrapped", "1.2.3", true)},
			{"Wrapped/Payload/App.app/Model.momd/VersionInfo.plist", keyless},
		}))
		require.NoError(t, err)
		defer tfr.Close()

		meta, err := file.ExtractZIPMetadata(tfr)
		require.NoError(t, err)
		require.NotNil(t, meta)
		require.Equal(t, "com.example.wrapped", meta.BundleIdentifier)
		require.Equal(t, "Wrapped", meta.Name)
		require.Equal(t, "1.2.3", meta.Version)
	})
}
