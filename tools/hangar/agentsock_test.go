package main

import (
	"context"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// shortDir keeps socket paths under sun_path's limit; t.TempDir is too deep
// on macOS.
func shortDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", "hs")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

func TestAgentSocket(t *testing.T) {
	data := shortDir(t)
	sock := filepath.Join(data, "agent", "hangar.sock")
	// A stale regular file from a crashed run is replaced.
	if err := os.MkdirAll(filepath.Dir(sock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sock, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	stop, err := startAgentSocket(data, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	for path, want := range map[string]fs.FileMode{filepath.Dir(sock): 0o700, sock: 0o600} {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != want {
			t.Errorf("%s mode = %o, want %o", path, fi.Mode().Perm(), want)
		}
	}
	if fi, _ := os.Stat(sock); fi.Mode()&fs.ModeSocket == 0 {
		t.Fatalf("%s isn't a socket", sock)
	}

	client := http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}}
	resp, err := client.Get("http://hangar/v1/slots")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if b, _ := io.ReadAll(resp.Body); string(b) != "ok" {
		t.Fatalf("body = %q", b)
	}
}

func TestAgentSocketPathTooLong(t *testing.T) {
	data := filepath.Join(shortDir(t), strings.Repeat("d", 100))
	if _, err := startAgentSocket(data, http.NotFoundHandler()); err == nil || !strings.Contains(err.Error(), "agent API disabled") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(data, "agent")); err == nil {
		t.Fatal("created the agent dir for a disabled socket")
	}
}
