package externalrefs

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// buildFlatPkgWithDistribution assembles a minimal, synthetic flat package
// (xar) containing only a top-level Distribution file, in the same byte
// layout real pkgs use: a fixed header, a zlib-compressed table of contents,
// then a heap holding the zlib-compressed Distribution XML. It stands in for
// a real ~20MB munkitools pkg so the test doesn't need to fetch or embed one.
func buildFlatPkgWithDistribution(t *testing.T, distributionXML string) []byte {
	t.Helper()
	return buildFlatPkgWithFile(t, "Distribution", distributionXML)
}

func buildFlatPkgWithFile(t *testing.T, name, contents string) []byte {
	t.Helper()

	distCompressed := zlibCompress(t, []byte(contents))

	tocXML := `<?xml version="1.0" encoding="UTF-8"?>
<xar>
 <toc>
  <file>
   <data>
    <offset>0</offset>
    <length>` + strconv.Itoa(len(distCompressed)) + `</length>
   </data>
   <type>file</type>
   <name>` + name + `</name>
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

func TestMunkiPackageVersions(t *testing.T) {
	t.Run("real-shaped Distribution file", func(t *testing.T) {
		// Mirrors munkitools 7.4.1: the core id appears three times (an unversioned
		// <choice> reference, the versioned declaration, and an empty
		// <bundle-version> wrapper), and the app's helper bundles are listed
		// before Managed Software Center itself, whose version lags the release.
		const distributionXML = `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<installer-script minSpecVersion="2">
    <choices-outline>
        <line choice="core"/>
    </choices-outline>
    <choice id="core" title="Munki core tools">
        <pkg-ref id="com.googlecode.munki.core"/>
    </choice>
    <pkg-ref id="com.googlecode.munki.core" auth="Root" version="7.4.1.5894" installKBytes="21302">#munkitools_core.pkg</pkg-ref>
    <pkg-ref id="com.googlecode.munki.app" auth="Root" version="7.4.0.5891" installKBytes="9037">#munkitools_app.pkg</pkg-ref>
    <product id="com.googlecode.munki" version="7.4.1.5894"/>
    <pkg-ref id="com.googlecode.munki.core">
        <bundle-version/>
    </pkg-ref>
    <pkg-ref id="com.googlecode.munki.app">
        <bundle-version>
            <bundle CFBundleShortVersionString="7.0.5.5680" CFBundleVersion="5680" id="com.googlecode.munki.MunkiStatus" path="Applications/Managed Software Center.app/Contents/Helpers/MunkiStatus.app"/>
            <bundle CFBundleShortVersionString="7.4.0.5891" CFBundleVersion="5891" id="com.googlecode.munki.ManagedSoftwareCenter" path="Applications/Managed Software Center.app"/>
        </bundle-version>
    </pkg-ref>
</installer-script>`

		url := serveRangedPkg(t, buildFlatPkgWithDistribution(t, distributionXML))

		got, err := MunkiPackageVersions(t.Context(), url)
		require.NoError(t, err)
		require.Equal(t, MunkiVersions{Core: "7.4.1.5894", App: "7.4.0.5891"}, got)
	})

	t.Run("attribute order doesn't matter", func(t *testing.T) {
		// version before id, unlike the real Distribution, to guard against
		// order-dependent parsing.
		const distributionXML = `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<installer-script minSpecVersion="2">
    <pkg-ref auth="Root" version="7.4.2.5903" id="com.googlecode.munki.core">#munkitools_core.pkg</pkg-ref>
    <pkg-ref id="com.googlecode.munki.app">
        <bundle-version>
            <bundle id="com.googlecode.munki.ManagedSoftwareCenter" CFBundleVersion="5904" CFBundleShortVersionString="7.4.2.5904"/>
        </bundle-version>
    </pkg-ref>
</installer-script>`

		url := serveRangedPkg(t, buildFlatPkgWithDistribution(t, distributionXML))

		got, err := MunkiPackageVersions(t.Context(), url)
		require.NoError(t, err)
		require.Equal(t, MunkiVersions{Core: "7.4.2.5903", App: "7.4.2.5904"}, got)
	})

	t.Run("app without a short version falls back to CFBundleVersion", func(t *testing.T) {
		// Matches what the apps table and Fleet's inventory report in that case.
		const distributionXML = `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<installer-script minSpecVersion="2">
    <pkg-ref id="com.googlecode.munki.core" version="7.4.1.5894">#munkitools_core.pkg</pkg-ref>
    <pkg-ref id="com.googlecode.munki.app">
        <bundle-version>
            <bundle CFBundleVersion="5891" id="com.googlecode.munki.ManagedSoftwareCenter"/>
        </bundle-version>
    </pkg-ref>
</installer-script>`

		url := serveRangedPkg(t, buildFlatPkgWithDistribution(t, distributionXML))

		got, err := MunkiPackageVersions(t.Context(), url)
		require.NoError(t, err)
		require.Equal(t, MunkiVersions{Core: "7.4.1.5894", App: "5891"}, got)
	})

	t.Run("no versioned core pkg-ref", func(t *testing.T) {
		const distributionXML = `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<installer-script minSpecVersion="2">
    <pkg-ref id="com.googlecode.munki.core"/>
    <pkg-ref id="com.googlecode.munki.app">
        <bundle-version>
            <bundle CFBundleShortVersionString="7.4.0.5891" id="com.googlecode.munki.ManagedSoftwareCenter"/>
        </bundle-version>
    </pkg-ref>
</installer-script>`

		url := serveRangedPkg(t, buildFlatPkgWithDistribution(t, distributionXML))

		_, err := MunkiPackageVersions(t.Context(), url)
		require.ErrorContains(t, err, "com.googlecode.munki.core")
	})

	t.Run("no Managed Software Center bundle", func(t *testing.T) {
		// Only a helper bundle: its version must not be used as the app's.
		const distributionXML = `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<installer-script minSpecVersion="2">
    <pkg-ref id="com.googlecode.munki.core" version="7.4.1.5894">#munkitools_core.pkg</pkg-ref>
    <pkg-ref id="com.googlecode.munki.app">
        <bundle-version>
            <bundle CFBundleShortVersionString="7.0.5.5680" id="com.googlecode.munki.MunkiStatus"/>
        </bundle-version>
    </pkg-ref>
</installer-script>`

		url := serveRangedPkg(t, buildFlatPkgWithDistribution(t, distributionXML))

		_, err := MunkiPackageVersions(t.Context(), url)
		require.ErrorContains(t, err, "com.googlecode.munki.ManagedSoftwareCenter")
	})

	t.Run("no Distribution file", func(t *testing.T) {
		tocCompressed := zlibCompress(t, []byte(`<xar><toc></toc></xar>`))

		header := make([]byte, xarHeaderSize)
		copy(header[0:4], "xar!")
		binary.BigEndian.PutUint16(header[4:6], xarHeaderSize)
		binary.BigEndian.PutUint64(header[8:16], uint64(len(tocCompressed)))

		url := serveRangedPkg(t, slices.Concat(header, tocCompressed))

		_, err := MunkiPackageVersions(t.Context(), url)
		require.Error(t, err)
	})
}

func TestZlibInflate_CapsDecompressedSize(t *testing.T) {
	huge := bytes.Repeat([]byte("a"), maxDecompressedLen+1)
	compressed := zlibCompress(t, huge)

	_, err := zlibInflate(compressed)
	require.Error(t, err)
}

func TestZlibInflate_AllowsNormalSize(t *testing.T) {
	small := []byte("<xar><toc></toc></xar>")
	compressed := zlibCompress(t, small)

	out, err := zlibInflate(compressed)
	require.NoError(t, err)
	require.Equal(t, small, out)
}
