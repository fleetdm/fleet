//go:build unix

package homes

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

// TestOwnerUsesStatNotName verifies that attribution comes from the directory's
// real owner, not its name: a directory named "root" but owned by the current
// (non-root) user must not be attributed to root.
func TestOwnerUsesStatNotName(t *testing.T) {
	cur, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	if cur.Username == "root" || cur.Uid == "0" {
		t.Skip("test must run as a non-root user to be meaningful")
	}

	dir := filepath.Join(t.TempDir(), "root") // misleadingly named after another account
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}

	uid, username := owner(dir, fi)
	if uid != cur.Uid {
		t.Errorf("uid = %q, want the real owner %q (must be read from stat, not the name)", uid, cur.Uid)
	}
	if username != cur.Username {
		t.Errorf("username = %q, want the real owner %q (must be resolved from the owner uid, not the name)", username, cur.Username)
	}
}
