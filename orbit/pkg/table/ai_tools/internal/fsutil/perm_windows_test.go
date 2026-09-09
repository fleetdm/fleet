//go:build windows

package fsutil

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// TestWorldSIDConstants pins the SID strings in perm_acl.go to what Windows
// itself reports for the well-known SIDs they name. They have the same blind
// spot the mask constants do — TestWorldPermFromACEs compares them against
// themselves, so a typo is self-consistent there and would simply stop matching
// any real ACE, silently disabling the whole check. The masks are pinned at
// compile time in perm_acl_windows.go; these need a live SID to compare against.
func TestWorldSIDConstants(t *testing.T) {
	for _, c := range []struct {
		label string
		typ   windows.WELL_KNOWN_SID_TYPE
		want  string
	}{
		{"Everyone", windows.WinWorldSid, sidEveryone},
		{"Authenticated Users", windows.WinAuthenticatedUserSid, sidAuthenticatedUsers},
	} {
		sid, err := windows.CreateWellKnownSid(c.typ)
		if err != nil {
			t.Fatalf("CreateWellKnownSid(%s): %v", c.label, err)
		}
		got := sid.String()
		if got != c.want {
			t.Errorf("%s: Windows reports %s, our constant is %s", c.label, got, c.want)
		}
		if !isWorldSID(got) {
			t.Errorf("%s (%s) not treated as a world SID", c.label, got)
		}
	}
}

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
	if p := Stat(path); !p.Known || !p.WorldReadable || !p.WorldWritable {
		t.Errorf("Everyone:(F): %+v want known, world readable and writable", p)
	}

	// Denying just the two data-write rights (icacls names them WD and AD) leaves
	// the DELETE and WRITE_DAC from the grant above, so Everyone can still replace
	// the file or re-ACL it — it stays world writable. icacls canonicalizes the
	// DACL, putting this deny ahead of the allow, which is the ordering that would
	// mask the risk if a deny settled write wholesale.
	//
	// The rights are named individually on purpose: icacls (W) expands to
	// FILE_GENERIC_WRITE, which carries READ_CONTROL, and denying that to Everyone
	// would stop the scanner from opening the file at all.
	icacls(t, path, "/deny", "*"+sidEveryone+":(WD,AD)")
	if p := Stat(path); !p.Known || !p.WorldWritable {
		t.Errorf("Everyone:(F) with (WD,AD) denied: %+v want known and still world writable", p)
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
