package externalrefs

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/require"
)

// buildFlatPkgWithDistribution assembles a minimal, synthetic flat package
// (xar) containing only a top-level Distribution file, in the same byte
// layout real pkgs use: a fixed header, a zlib-compressed table of contents,
// then a heap holding the zlib-compressed Distribution XML. It stands in for
// a real ~20MB munkitools pkg so the test doesn't need to fetch or embed one.
func buildFlatPkgWithDistribution(t *testing.T, distributionXML string) []byte {
	t.Helper()

	distCompressed := zlibCompress(t, []byte(distributionXML))

	tocXML := `<?xml version="1.0" encoding="UTF-8"?>
<xar>
 <toc>
  <file>
   <data>
    <offset>0</offset>
    <length>` + strconv.Itoa(len(distCompressed)) + `</length>
   </data>
   <type>file</type>
   <name>Distribution</name>
  </file>
 </toc>
</xar>`
	tocCompressed := zlibCompress(t, []byte(tocXML))

	header := make([]byte, xarHeaderSize)
	copy(header[0:4], "xar!")
	binary.BigEndian.PutUint16(header[4:6], xarHeaderSize)
	binary.BigEndian.PutUint16(header[6:8], 1)
	binary.BigEndian.PutUint64(header[8:16], uint64(len(tocCompressed)))
	binary.BigEndian.PutUint64(header[16:24], uint64(len(tocXML)))

	var pkg bytes.Buffer
	pkg.Write(header)
	pkg.Write(tocCompressed)
	pkg.Write(distCompressed)
	return pkg.Bytes()
}

func zlibCompress(t *testing.T, b []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write(b)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func serveRangedPkg(t *testing.T, pkg []byte) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "munkitools.pkg", time.Time{}, bytes.NewReader(pkg))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func TestMunkiAppVersionFromPkg(t *testing.T) {
	const distributionXML = `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<installer-script minSpecVersion="2">
    <pkg-ref id="com.googlecode.munki.app">
        <bundle-version>
            <bundle CFBundleShortVersionString="1.2.0.4567" CFBundleVersion="4567" id="com.googlecode.munki.MunkiStatus" path="Applications/Managed Software Center.app/Contents/Helpers/MunkiStatus.app"/>
            <bundle CFBundleShortVersionString="1.0.0.4321" CFBundleVersion="4321" id="com.googlecode.munki.ManagedSoftwareCenter" path="Applications/Managed Software Center.app"/>
        </bundle-version>
    </pkg-ref>
</installer-script>`

	pkg := buildFlatPkgWithDistribution(t, distributionXML)
	url := serveRangedPkg(t, pkg)

	app := &maintained_apps.FMAManifestApp{
		Version:      "1.3.0.9999",
		InstallerURL: url,
	}

	got, err := MunkiAppVersionFromPkg(app)
	require.NoError(t, err)
	require.Equal(t, "1.0.0.4321", got.Version)
}

func TestMunkiAppVersionFromPkg_NoDistributionFile(t *testing.T) {
	// A TOC with no Distribution entry at all shouldn't happen for a real
	// munkitools pkg, but the function must fail clearly rather than panic
	// or silently keep a wrong version.
	tocCompressed := zlibCompress(t, []byte(`<xar><toc></toc></xar>`))

	header := make([]byte, xarHeaderSize)
	copy(header[0:4], "xar!")
	binary.BigEndian.PutUint16(header[4:6], xarHeaderSize)
	binary.BigEndian.PutUint64(header[8:16], uint64(len(tocCompressed)))

	pkg := append(header, tocCompressed...)
	url := serveRangedPkg(t, pkg)

	app := &maintained_apps.FMAManifestApp{
		Version:      "1.3.0.9999",
		InstallerURL: url,
	}

	_, err := MunkiAppVersionFromPkg(app)
	require.Error(t, err)
}
