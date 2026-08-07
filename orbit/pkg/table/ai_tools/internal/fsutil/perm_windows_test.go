//go:build windows

package fsutil

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"testing"
)

// TestStatPermWindowsACL drives the real DACL reader end to end with icacls, the
// same tool the bug report used to reproduce. worldPermFromACEs covers the
// decision rules; this covers the Win32 plumbing that feeds it.
func TestStatPermWindowsACL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "GEMINI.md")
	if err := os.WriteFile(path, []byte("# instructions\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Replace the inherited ACL with a single ACE granting only the current user,
	// so the baseline is the same whatever the runner's profile ACL looks like.
	// SIDs are used throughout rather than names like "Everyone", which are
	// localized on non-English Windows images.
	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	icacls(t, path, "/inheritance:r", "/grant:r", "*"+u.Uid+":(F)")
	if p := Stat(path); !p.Known || p.WorldReadable || p.WorldWritable {
		t.Errorf("owner-only ACL: %+v want known, not world readable/writable", p)
	}

	icacls(t, path, "/grant", "*"+sidEveryone+":(R)")
	if p := Stat(path); !p.Known || !p.WorldReadable || p.WorldWritable {
		t.Errorf("Everyone:(R): %+v want known, world readable, not world writable", p)
	}

	icacls(t, path, "/grant", "*"+sidEveryone+":(F)")
	if p := Stat(path); !p.Known || !p.WorldWritable {
		t.Errorf("Everyone:(F): %+v want known and world writable", p)
	}
}

func TestStatPermWindowsMissingFile(t *testing.T) {
	if p := Stat(filepath.Join(t.TempDir(), "absent")); p.Known {
		t.Errorf("%+v want Known=false for a path we cannot open", p)
	}
}

func icacls(t *testing.T, path string, args ...string) {
	t.Helper()
	out, err := exec.Command("icacls", append([]string{path}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("icacls %v: %v\n%s", args, err, out)
	}
}
