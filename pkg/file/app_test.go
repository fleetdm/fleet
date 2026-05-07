package file_test

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testInfoPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleIdentifier</key>
  <string>com.example.Demo</string>
  <key>CFBundleName</key>
  <string>Demo</string>
  <key>CFBundleShortVersionString</key>
  <string>1.2.3</string>
  <key>CFBundleIconFile</key>
  <string>AppIcon</string>
</dict>
</plist>`

// synthICNS builds a tiny icns container holding a single PNG-tagged chunk so
// that ExtractPNGFromICNS returns non-nil data.
func synthICNS(t *testing.T) []byte {
	t.Helper()
	png := append([]byte{0x89, 0x50, 0x4E, 0x47}, bytes.Repeat([]byte{0x00}, 16)...)
	var body bytes.Buffer
	body.WriteString("ic10")
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(8+len(png))) //nolint:gosec // dismiss G115
	body.Write(lenBuf)
	body.Write(png)

	var out bytes.Buffer
	out.WriteString("icns")
	binary.BigEndian.PutUint32(lenBuf, uint32(8+body.Len())) //nolint:gosec // dismiss G115
	out.Write(lenBuf)
	out.Write(body.Bytes())
	return out.Bytes()
}

// buildAppZip writes a .app-bundle-shaped zip to the given writer.
// Set includeIcon=false to omit Contents/Resources/AppIcon.icns.
func buildAppZip(t *testing.T, appName, plistBody string, includeIcon bool) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	write := func(name string, body []byte) {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write(body)
		require.NoError(t, err)
	}

	write(appName+"/Contents/Info.plist", []byte(plistBody))
	if includeIcon {
		write(appName+"/Contents/Resources/AppIcon.icns", synthICNS(t))
	}
	// Sanity: a nested Info.plist in a framework should be ignored (depth != 3).
	write(appName+"/Contents/Frameworks/Nested.framework/Info.plist", []byte("ignored"))

	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func tfrFrom(t *testing.T, data []byte) *fleet.TempFileReader {
	t.Helper()
	tfr, err := fleet.NewTempFileReader(bytes.NewReader(data), t.TempDir)
	require.NoError(t, err)
	return tfr
}

func TestExtractZipAppMetadata(t *testing.T) {
	t.Run("parses Info.plist and icon", func(t *testing.T) {
		data := buildAppZip(t, "Demo.app", testInfoPlist, true)
		meta, err := file.ExtractZipAppMetadata(tfrFrom(t, data))
		require.NoError(t, err)
		assert.Equal(t, "Demo", meta.Name)
		assert.Equal(t, "1.2.3", meta.Version)
		assert.Equal(t, "com.example.Demo", meta.BundleIdentifier)
		assert.Equal(t, []string{"com.example.Demo"}, meta.PackageIDs)
		assert.NotEmpty(t, meta.SHASum)
		assert.NotEmpty(t, meta.IconPNG, "should extract icon PNG from icns")
	})

	t.Run("missing icon is non-fatal", func(t *testing.T) {
		data := buildAppZip(t, "Demo.app", testInfoPlist, false)
		meta, err := file.ExtractZipAppMetadata(tfrFrom(t, data))
		require.NoError(t, err)
		assert.Equal(t, "com.example.Demo", meta.BundleIdentifier)
		assert.Empty(t, meta.IconPNG)
	})

	t.Run("no .app directory", func(t *testing.T) {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		w, err := zw.Create("readme.txt")
		require.NoError(t, err)
		_, err = io.WriteString(w, "no app here")
		require.NoError(t, err)
		require.NoError(t, zw.Close())

		_, err = file.ExtractZipAppMetadata(tfrFrom(t, buf.Bytes()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no .app/Contents/Info.plist")
	})
}

func TestExtractInstallerMetadataWithHint_RoutesByExtension(t *testing.T) {
	data := buildAppZip(t, "Demo.app", testInfoPlist, true)

	t.Run("zip hint uses ExtractZipAppMetadata", func(t *testing.T) {
		meta, err := file.ExtractInstallerMetadataWithHint(tfrFrom(t, data), "demo.zip")
		require.NoError(t, err)
		assert.Equal(t, "com.example.Demo", meta.BundleIdentifier)
		assert.Equal(t, "zip", meta.Extension)
	})

	t.Run("unknown hint falls through to magic-byte detection", func(t *testing.T) {
		// A plain zip of random stuff has no magic-byte handler (other than ipa),
		// so we expect a known failure mode rather than a zip-routed success.
		_, err := file.ExtractInstallerMetadataWithHint(tfrFrom(t, data), "demo.unknown")
		// Either an unsupported-type error or an ipa-parse error is acceptable — we
		// only need to confirm we did NOT go through the zip helper.
		require.Error(t, err)
		assert.NotContains(t, strings.ToLower(err.Error()), "no .app/contents/info.plist")
	})
}
