package main

import (
	"archive/zip"
	"bytes"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ddfPage mirrors the Learn page: the current drop first, then older drops and pre-v2 downloads.
const ddfPage = `<html><body>
<p>DDF v2 XML files for Windows 10 and Windows 11 are combined, and provided in a single download:</p>
<ul><li><a href="https://download.microsoft.com/download/015bd9f5-9cca-4821-8a85-a4c5f9a5d0f2/DDFv2Feb2026.zip" data-linktype="external">DDF v2 Files, February 2026</a></li></ul>
<h2>Older DDF files</h2>
<ul>
<li><a href="https://download.microsoft.com/download/2ff2c8b9-2e3f-47af-89e6-11c19b7f0c2a/DDFv2Sept25.zip">September 2025</a></li>
<li><a href="https://download.microsoft.com/download/a/a/a/aaadc008-67d4-4dcd-b864-70c479baf7d6/DDFv2September24.zip">September 2024</a></li>
<li><a href="https://download.microsoft.com/download/6/F/0/6F019079-6EB0-41B5-88E8-D1CE77DBA27B/Windows10_1903_DDF_download.zip">1903</a></li>
</ul></body></html>`

func TestResolveDDF(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		page    string
		want    profilevalidation.WindowsVersion
		wantErr string
	}{
		{
			name: "first link is current",
			page: ddfPage,
			want: profilevalidation.WindowsVersion{
				Release: "DDFv2Feb2026",
				URL:     "https://download.microsoft.com/download/015bd9f5-9cca-4821-8a85-a4c5f9a5d0f2/DDFv2Feb2026.zip",
			},
		},
		{
			name: "other hosts and pre-v2 downloads ignored",
			page: `<a href="https://example.com/download/x/DDFv2Fake.zip">x</a>
				<a href="https://download.microsoft.com/download/6/F/0/6F019079/Windows10_1903_DDF_download.zip">1903</a>
				<a href="https://download.microsoft.com/download/a/b/DDFv2May24.zip">May 2024</a>`,
			want: profilevalidation.WindowsVersion{Release: "DDFv2May24", URL: "https://download.microsoft.com/download/a/b/DDFv2May24.zip"},
		},
		{
			name:    "no links",
			page:    `<html><body>Moved to a new page.</body></html>`,
			wantErr: "no DDFv2 zip linked from",
		},
		{
			name:    "not found",
			status:  http.StatusNotFound,
			wantErr: "unexpected status 404 Not Found",
		},
		{
			name:    "server error",
			status:  http.StatusInternalServerError,
			page:    ddfPage,
			wantErr: "unexpected status 500 Internal Server Error",
		},
		{
			name:    "requires https://download.microsoft.com and no other host can take over",
			page:    `<a href="https://example.com?https://download.microsoft.com/download/015bd9f5-9cca-4821-8a85-a4c5f9a5d0f2/DDFv2Feb2026.zip">x</a>`,
			wantErr: "no DDFv2 zip linked from",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if c.status != 0 {
					w.WriteHeader(c.status)
				}
				_, _ = w.Write([]byte(c.page))
			}))
			defer srv.Close()

			got, err := resolveDDF(t.Context(), slog.New(slog.DiscardHandler), fleethttp.NewClient(), srv.URL)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range slices.Sorted(maps.Keys(files)) {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(files[name]))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

// ddfServer serves zips by path; anything else is a 404.
func ddfServer(t *testing.T, zips map[string][]byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := zips[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

var ddfDrop = map[string]string{
	"DDFDrop012026/":                     "",
	"DDFDrop012026/Policy_DDF_file.xml":  "<MgmtTree/>",
	"DDFDrop012026/Firewall.XML":         "<MgmtTree/>",
	"DDFDrop012026/readme.txt":           "not a DDF",
	"DDFDrop012026/nested/DMClient.xml":  "<MgmtTree/>",
	"DDFDrop012026/schema/DDFv2Msft.xsd": "<xs:schema/>",
}

func TestDownloadDDF(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	good := makeZip(t, ddfDrop)
	srv := ddfServer(t, map[string][]byte{
		"/good.zip":      good,
		"/empty.zip":     makeZip(t, map[string]string{"readme.txt": "x"}),
		"/dup.zip":       makeZip(t, map[string]string{"a/Policy.xml": "<a/>", "b/Policy.xml": "<b/>"}),
		"/notazip.zip":   []byte("<html>not a zip</html>"),
		"/traversal.zip": makeZip(t, map[string]string{"../../evil.xml": "<x/>"}),
	})
	download := func(path, wantSHA string) (string, string, error) {
		dest := t.TempDir()
		got, err := downloadDDF(t.Context(), logger, fleethttp.NewClient(), srv.URL+path, wantSHA, dest)
		return dest, got, err
	}

	t.Run("extracts XML flat, without a pin", func(t *testing.T) {
		dest, got, err := download("/good.zip", "")
		require.NoError(t, err)
		assert.Equal(t, sha256Hex(good), got)
		assert.ElementsMatch(t, []string{"Policy_DDF_file.xml", "Firewall.XML", "DMClient.xml"}, listFiles(t, dest))
	})
	t.Run("matching pin", func(t *testing.T) {
		_, got, err := download("/good.zip", sha256Hex(good))
		require.NoError(t, err)
		assert.Equal(t, sha256Hex(good), got)
	})
	t.Run("mismatched pin extracts nothing", func(t *testing.T) {
		dest, _, err := download("/good.zip", strings.Repeat("0", 64))
		require.ErrorContains(t, err, "has sha256 "+sha256Hex(good)+", but the output pins "+strings.Repeat("0", 64))
		assert.Empty(t, listFiles(t, dest))
	})
	t.Run("no XML", func(t *testing.T) {
		_, _, err := download("/empty.zip", "")
		require.ErrorContains(t, err, "has no XML files")
	})
	t.Run("same name twice", func(t *testing.T) {
		_, _, err := download("/dup.zip", "")
		require.ErrorContains(t, err, "DDF zip has Policy.xml twice (a/Policy.xml and b/Policy.xml)")
	})
	t.Run("traversal flattened into dest", func(t *testing.T) {
		dest, _, err := download("/traversal.zip", "")
		require.NoError(t, err)
		assert.Equal(t, []string{"evil.xml"}, listFiles(t, dest))
	})
	t.Run("not a zip", func(t *testing.T) {
		_, _, err := download("/notazip.zip", "")
		require.ErrorContains(t, err, "open DDF zip")
	})
	t.Run("HTTP error", func(t *testing.T) {
		_, _, err := download("/missing.zip", "")
		require.ErrorContains(t, err, "unexpected status 404 Not Found")
	})
}

func TestRunWindowsWithoutUpdate(t *testing.T) {
	pin := &profilevalidation.WindowsVersion{Release: "DDFv2Feb2026", URL: "https://unused.invalid/x.zip", SHA256: strings.Repeat("a", 64)}
	run := func(dir string, pin *profilevalidation.WindowsVersion) (profilevalidation.WindowsVersion, error) {
		// No client and no -fetch-pinned: any network use would panic.
		return runWindows(t.Context(), slog.New(slog.DiscardHandler), nil, windowsRunOptions{Dir: dir, Pinned: pin})
	}

	_, err := run(t.TempDir(), nil)
	require.ErrorContains(t, err, "no DDF drop pinned in the output; run with -update")

	_, err = run(t.TempDir(), pin)
	require.ErrorContains(t, err, "no DDF download in")
	require.ErrorContains(t, err, "-fetch-pinned")

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, sha256Marker), []byte(strings.Repeat("b", 64)+"\n"), 0o644))
	_, err = run(dir, pin)
	require.ErrorContains(t, err, "holds DDF zip "+strings.Repeat("b", 64)+" but the output pins "+pin.SHA256)

	require.NoError(t, os.WriteFile(filepath.Join(dir, sha256Marker), []byte(pin.SHA256+"\n"), 0o644))
	got, err := run(dir, pin)
	require.NoError(t, err)
	assert.Equal(t, *pin, got)
}

func TestRunWindowsFetchPinned(t *testing.T) {
	zipData := makeZip(t, ddfDrop)
	srv := ddfServer(t, map[string][]byte{"/DDFv2Feb2026.zip": zipData})
	pin := profilevalidation.WindowsVersion{Release: "DDFv2Feb2026", URL: srv.URL + "/DDFv2Feb2026.zip", SHA256: sha256Hex(zipData)}
	run := func(dir string, pin profilevalidation.WindowsVersion) (profilevalidation.WindowsVersion, error) {
		return runWindows(t.Context(), slog.New(slog.DiscardHandler), fleethttp.NewClient(), windowsRunOptions{Dir: dir, Pinned: &pin, FetchPinned: true})
	}

	t.Run("downloads and marks", func(t *testing.T) {
		dir := t.TempDir()
		got, err := run(dir, pin)
		require.NoError(t, err)
		assert.Equal(t, pin, got)
		marker, err := os.ReadFile(filepath.Join(dir, sha256Marker))
		require.NoError(t, err)
		assert.Equal(t, pin.SHA256+"\n", string(marker))
	})
	t.Run("replaces a stale download", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "Stale.xml"), []byte("<old/>"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, sha256Marker), []byte("old\n"), 0o644))
		_, err := run(dir, pin)
		require.NoError(t, err)
		assert.NotContains(t, listFiles(t, dir), "Stale.xml")
	})
	t.Run("changed zip fails and leaves no marker", func(t *testing.T) {
		dir := t.TempDir()
		changed := pin
		changed.SHA256 = strings.Repeat("0", 64)
		_, err := run(dir, changed)
		require.ErrorContains(t, err, "but the output pins "+changed.SHA256)
		assert.NoFileExists(t, filepath.Join(dir, sha256Marker))
	})
}

// microsoftTransport sends download.microsoft.com and learn.microsoft.com requests to a test server,
// since resolveDDF only accepts links on Microsoft's download host.
type microsoftTransport struct{ target *url.URL }

func (m microsoftTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.URL.Scheme, r.URL.Host = m.target.Scheme, m.target.Host
	return http.DefaultTransport.RoundTrip(r)
}

func TestRunWindowsUpdate(t *testing.T) {
	oldSHA := strings.Repeat("0", 64)
	newZip := makeZip(t, ddfDrop)
	const newURL = "https://download.microsoft.com/download/015bd9f5/DDFv2Feb2026.zip"
	page := `<a href="` + newURL + `">February 2026</a><a href="https://download.microsoft.com/download/2ff2c8b9/DDFv2Sept25.zip">Sept</a>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/page":
			_, _ = w.Write([]byte(page))
		case "/download/015bd9f5/DDFv2Feb2026.zip":
			_, _ = w.Write(newZip)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	target, err := url.Parse(srv.URL)
	require.NoError(t, err)
	client := fleethttp.NewClient()
	client.Transport = microsoftTransport{target: target}
	want := profilevalidation.WindowsVersion{Release: "DDFv2Feb2026", URL: newURL, SHA256: sha256Hex(newZip)}

	cases := []struct {
		name    string
		pinned  *profilevalidation.WindowsVersion
		wantLog string
	}{
		{name: "no pin", wantLog: "no DDF pin; using latest"},
		{name: "same drop", pinned: &want, wantLog: "latest DDF drop matches pin"},
		{name: "new drop", pinned: &profilevalidation.WindowsVersion{Release: "DDFv2Sept25", URL: "https://download.microsoft.com/download/2ff2c8b9/DDFv2Sept25.zip", SHA256: oldSHA}, wantLog: "moving DDF pin to latest"},
		{name: "republished in place", pinned: &profilevalidation.WindowsVersion{Release: "DDFv2Feb2026", URL: newURL, SHA256: oldSHA}, wantLog: "level=WARN msg=\"DDF zip changed at the pinned URL\""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var logs bytes.Buffer
			dir := t.TempDir()
			got, err := runWindows(t.Context(), slog.New(slog.NewTextHandler(&logs, nil)), client, windowsRunOptions{
				PageURL: "https://learn.microsoft.com/page", Dir: dir, Pinned: c.pinned, Update: true,
			})
			require.NoError(t, err)
			assert.Equal(t, want, got)
			assert.Contains(t, logs.String(), c.wantLog)
			assert.Contains(t, listFiles(t, dir), "Policy_DDF_file.xml")
		})
	}
}
