package homebrew

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/stretchr/testify/require"
)

// jabberDistribution mirrors the shape of Cisco's real Distribution file: the
// component is named three times and only the definition carries a version.
const jabberDistribution = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<installer-gui-script minSpecVersion="2">
    <pkg-ref id="com.cisco.Jabber"/>
    <choices-outline>
        <line choice="com.cisco.Jabber"/>
    </choices-outline>
    <choice id="com.cisco.Jabber" visible="false">
        <pkg-ref id="com.cisco.Jabber"/>
    </choice>
    <pkg-ref id="com.cisco.Jabber" onConclusion="none" version="15.3.0.311163">#CiscoJabberInstaller.pkg</pkg-ref>
    <pkg-ref id="com.jabra.CiscoJabberPlugin" version="1.0.21.291">#JabraCiscoJabberPlugin.pkg</pkg-ref>
</installer-gui-script>`

// buildFlatPackage assembles a minimal flat package (xar archive) holding the
// given Distribution file, preceded by a filler member so the Distribution sits
// at a non-zero heap offset.
func buildFlatPackage(t *testing.T, distribution string) []byte {
	t.Helper()

	filler := deflate(t, []byte("payload"))
	distributionMember := deflate(t, []byte(distribution))

	toc := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<xar>
  <toc>
    <file id="1">
      <name>Resources</name>
      <type>directory</type>
      <file id="2">
        <name>Payload</name>
        <type>file</type>
        <data>
          <length>%d</length>
          <offset>0</offset>
          <size>7</size>
          <encoding style="application/x-gzip"/>
        </data>
      </file>
    </file>
    <file id="3">
      <name>Distribution</name>
      <type>file</type>
      <data>
        <length>%d</length>
        <offset>%d</offset>
        <size>%d</size>
        <encoding style="application/x-gzip"/>
      </data>
    </file>
  </toc>
</xar>`, len(filler), len(distributionMember), len(filler), len(distribution))

	compressedTOC := deflate(t, []byte(toc))

	header := make([]byte, xarHeaderSize)
	copy(header, xarMagic)
	binary.BigEndian.PutUint16(header[4:6], xarHeaderSize)
	binary.BigEndian.PutUint16(header[6:8], 1)
	binary.BigEndian.PutUint64(header[8:16], uint64(len(compressedTOC)))
	binary.BigEndian.PutUint64(header[16:24], uint64(len(toc)))
	binary.BigEndian.PutUint32(header[24:28], 1)

	var pkg bytes.Buffer
	pkg.Write(header)
	pkg.Write(compressedTOC)
	pkg.Write(filler)
	pkg.Write(distributionMember)
	return pkg.Bytes()
}

func deflate(t *testing.T, in []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	w := zlib.NewWriter(&out)
	_, err := w.Write(in)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return out.Bytes()
}

// servePackage serves pkg with Range support, recording how many bytes it
// handed out.
func servePackage(t *testing.T, pkg []byte, served *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &countingResponseWriter{ResponseWriter: w, served: served}
		http.ServeContent(rec, r, "installer.pkg", time.Time{}, bytes.NewReader(pkg))
	}))
	t.Cleanup(srv.Close)
	return srv
}

type countingResponseWriter struct {
	http.ResponseWriter
	served *int
}

func (w *countingResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	if w.served != nil {
		*w.served += n
	}
	return n, err
}

func testIngester(t *testing.T) *brewIngester {
	t.Helper()
	return &brewIngester{
		logger:           slog.New(slog.DiscardHandler),
		client:           fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
		retryInterval:    time.Millisecond,
		retryMaxAttempts: 2,
	}
}

func TestPkgRefVersion(t *testing.T) {
	pkg := buildFlatPackage(t, jabberDistribution)
	var served int
	srv := servePackage(t, pkg, &served)

	i := testIngester(t)

	version, err := i.pkgRefVersion(context.Background(), srv.URL+"/installer.pkg", "com.cisco.Jabber")
	require.NoError(t, err)
	require.Equal(t, "15.3.0.311163", version)

	// The point of reading via range requests is not downloading the package;
	// guard against a change that fetches the whole thing.
	require.Less(t, served, len(pkg), "should not have served the whole package")

	t.Run("unknown pkg-ref id", func(t *testing.T) {
		_, err := i.pkgRefVersion(context.Background(), srv.URL+"/installer.pkg", "com.example.nope")
		require.ErrorContains(t, err, "no versioned pkg-ref")
	})
}

func TestPkgRefVersionErrors(t *testing.T) {
	t.Run("not a flat package", func(t *testing.T) {
		srv := servePackage(t, []byte("this is a disk image, not a package"), nil)
		_, err := testIngester(t).pkgRefVersion(context.Background(), srv.URL+"/installer.dmg", "com.cisco.Jabber")
		require.ErrorContains(t, err, "xar magic missing")
	})

	t.Run("no Distribution file", func(t *testing.T) {
		pkg := buildFlatPackage(t, jabberDistribution)
		// Rename the Distribution entry inside the compressed table of contents by
		// rebuilding the package without one.
		toc := `<?xml version="1.0" encoding="UTF-8"?><xar><toc><file id="1"><name>PackageInfo</name></file></toc></xar>`
		compressedTOC := deflate(t, []byte(toc))
		header := make([]byte, xarHeaderSize)
		copy(header, xarMagic)
		binary.BigEndian.PutUint16(header[4:6], xarHeaderSize)
		binary.BigEndian.PutUint64(header[8:16], uint64(len(compressedTOC)))
		binary.BigEndian.PutUint64(header[16:24], uint64(len(toc)))
		pkg = append(header, compressedTOC...)

		srv := servePackage(t, pkg, nil)
		_, err := testIngester(t).pkgRefVersion(context.Background(), srv.URL+"/installer.pkg", "com.cisco.Jabber")
		require.ErrorContains(t, err, "no Distribution file")
	})

	t.Run("server ignores range requests", func(t *testing.T) {
		pkg := buildFlatPackage(t, jabberDistribution)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(pkg)
		}))
		t.Cleanup(srv.Close)

		_, err := testIngester(t).pkgRefVersion(context.Background(), srv.URL+"/installer.pkg", "com.cisco.Jabber")
		require.ErrorContains(t, err, "want 206")
	})

	t.Run("installer is unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(srv.Close)

		_, err := testIngester(t).pkgRefVersion(context.Background(), srv.URL+"/installer.pkg", "com.cisco.Jabber")
		require.ErrorContains(t, err, "status 404")
	})
}

func TestTruncateVersion(t *testing.T) {
	cases := []struct {
		version  string
		segments int
		want     string
	}{
		{"15.3.0.311163", 3, "15.3.0"},
		{"15.2.1.310617", 3, "15.2.1"},
		{"15.3.0", 3, "15.3.0"},
		{"15.3", 3, "15.3"},
		{"15", 3, "15"},
		{"", 3, ""},
	}
	for _, c := range cases {
		require.Equal(t, c.want, truncateVersion(c.version, c.segments), c.version)
	}
}

// TestIngestCiscoJabberVersion covers the reason this lookup exists: the cask
// version is a build timestamp, and the manifest has to carry the app version
// osquery reports instead.
func TestIngestCiscoJabberVersion(t *testing.T) {
	pkgSrv := servePackage(t, buildFlatPackage(t, jabberDistribution), nil)

	newBrewServer := func(t *testing.T, installerURL string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			err := json.NewEncoder(w).Encode(brewCask{
				Token:   "cisco-jabber",
				Name:    []string{"Cisco Jabber"},
				URL:     installerURL,
				Version: "20260722023311",
			})
			if err != nil {
				t.Errorf("encoding fixture: %v", err)
			}
		}))
		t.Cleanup(srv.Close)
		return srv
	}

	input := inputApp{
		Token:            "cisco-jabber",
		UniqueIdentifier: "com.cisco.jabber",
		InstallerFormat:  "pkg",
		Name:             "Cisco Jabber",
		Slug:             "cisco-jabber/darwin",
	}

	t.Run("version comes from the installer", func(t *testing.T) {
		i := testIngester(t)
		i.baseURL = newBrewServer(t, pkgSrv.URL+"/Install_Cisco-Jabber-Mac.pkg").URL + "/"

		out, err := i.ingestOne(context.Background(), input)
		require.NoError(t, err)
		require.Equal(t, "15.3.0", out.Version)
		require.Equal(t,
			"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.cisco.jabber' AND version_compare(bundle_short_version, '15.3.0') < 0);",
			out.Queries.Patched,
		)
	})

	t.Run("lookup failure skips the manifest update", func(t *testing.T) {
		badPkgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(badPkgSrv.Close)

		i := testIngester(t)
		i.baseURL = newBrewServer(t, badPkgSrv.URL+"/Install_Cisco-Jabber-Mac.pkg").URL + "/"

		out, err := i.ingestOne(context.Background(), input)
		require.NoError(t, err)
		// An empty manifest tells the caller to leave the published one alone,
		// rather than publish the build timestamp as a version.
		require.True(t, out.IsEmpty(), "expected an empty manifest, got version %q", out.Version)
	})
}
