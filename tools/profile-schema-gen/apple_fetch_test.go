package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTarball builds a codeload-style tarball: every entry under "device-management-<sha>/".
func makeTarball(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "device-management-" + testSHA + "/", Typeflag: tar.TypeDir, Mode: 0o755}))
	for name, content := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: "device-management-" + testSHA + "/" + name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(content))}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func serveTarball(t *testing.T, body []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/apple/device-management/tar.gz/"+testSHA, r.URL.Path)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func download(t *testing.T, srv *httptest.Server, dest string) error {
	return downloadAppleTarball(t.Context(), slog.New(slog.DiscardHandler), fleethttp.NewClient(), srv.URL, testSHA, dest)
}

func listFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		files = append(files, filepath.ToSlash(rel))
		return err
	})
	require.NoError(t, err)
	return files
}

func TestDownloadAppleTarballExtractsOnlyWanted(t *testing.T) {
	srv := serveTarball(t, makeTarball(t, map[string]string{
		"mdm/profiles/com.apple.applicationaccess.yaml":                  "payload: {}",
		"declarative/declarations/configurations/passcode.settings.yaml": "payload: {}",
		"declarative/declarations/assets/credentials/acme.yaml":          "payload: {}",
		"mdm/profiles/README.md":                                         "not yaml",
		"mdm/commands/device.erase.yaml":                                 "other dir",
		"docs/schema.md":                                                 "other dir",
	}))
	dest := t.TempDir()
	require.NoError(t, download(t, srv, dest))

	assert.ElementsMatch(t, []string{
		"mdm/profiles/com.apple.applicationaccess.yaml",
		"declarative/declarations/configurations/passcode.settings.yaml",
		"declarative/declarations/assets/credentials/acme.yaml",
	}, listFiles(t, dest))
	content, err := os.ReadFile(filepath.Join(dest, "mdm/profiles/com.apple.applicationaccess.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "payload: {}", string(content))
}

func TestDownloadAppleTarballRejectsTraversal(t *testing.T) {
	srv := serveTarball(t, makeTarball(t, map[string]string{
		"mdm/profiles/../../../escaped.yaml": "evil",
	}))
	dest := filepath.Join(t.TempDir(), "a", "b")
	require.ErrorContains(t, download(t, srv, dest), "escapes destination")
	_, err := os.Stat(filepath.Join(dest, "..", "escaped.yaml"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestDownloadAppleTarballNoMatchingFiles(t *testing.T) {
	srv := serveTarball(t, makeTarball(t, map[string]string{"docs/schema.md": "x"}))
	require.ErrorContains(t, download(t, srv, t.TempDir()), "has no YAML files")
}

func TestDownloadAppleTarballNotGzip(t *testing.T) {
	srv := serveTarball(t, []byte("<html>not a tarball</html>"))
	require.ErrorContains(t, download(t, srv, t.TempDir()), "gzip")
}

func TestDownloadAppleTarballHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	defer srv.Close()
	require.ErrorContains(t, download(t, srv, t.TempDir()), "unexpected status 404")
}

func TestRunAppleWithoutUpdate(t *testing.T) {
	pinned := &profilevalidation.AppleVersion{Release: "Release-v27.0", Commit: testSHA}
	// A nil client panics on use, proving these paths never touch the network.
	run := func(dir string, pin *profilevalidation.AppleVersion) (profilevalidation.AppleVersion, error) {
		return runApple(t.Context(), slog.New(slog.DiscardHandler), nil, "", "", dir, pin, false)
	}
	withMarker := func(commit string) string {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, commitMarker), []byte(commit+"\n"), 0o644))
		return dir
	}

	_, err := run(withMarker(testSHA), nil)
	require.ErrorContains(t, err, "no Apple commit pinned in the output; run with -update")

	_, err = run(t.TempDir(), pinned)
	require.ErrorContains(t, err, "no Apple download in")

	_, err = run(withMarker("1111111111111111111111111111111111111111"), pinned)
	require.ErrorContains(t, err, "holds Apple commit 1111111111111111111111111111111111111111 but the output pins "+testSHA)

	got, err := run(withMarker(testSHA), pinned)
	require.NoError(t, err)
	assert.Equal(t, *pinned, got)
}

func TestRunAppleUpdate(t *testing.T) {
	// End to end against fake GitHub: resolve, download, and the marker written last.
	var tarball []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/apple/device-management/commits/release":
			_, _ = w.Write([]byte(`{"sha": "` + testSHA + `", "commit": {"message": "Release-v27.0"}}`))
		case "/apple/device-management/tar.gz/" + testSHA:
			_, _ = w.Write(tarball)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	tarball = makeTarball(t, map[string]string{"mdm/profiles/a.yaml": "x"})

	dir := filepath.Join(t.TempDir(), "apple")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stale.yaml"), []byte("old"), 0o644))

	got, err := runApple(t.Context(), slog.New(slog.DiscardHandler), fleethttp.NewClient(), srv.URL, srv.URL, dir,
		&profilevalidation.AppleVersion{Release: "Release-v26.4", Commit: "1111111111111111111111111111111111111111"}, true)
	require.NoError(t, err)
	assert.Equal(t, profilevalidation.AppleVersion{Release: "Release-v27.0", Commit: testSHA}, got)
	assert.ElementsMatch(t, []string{commitMarker, "mdm/profiles/a.yaml"}, listFiles(t, dir), "previous download is cleared")
	marker, err := os.ReadFile(filepath.Join(dir, commitMarker))
	require.NoError(t, err)
	assert.Equal(t, testSHA+"\n", string(marker))
}
