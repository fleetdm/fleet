package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpand(t *testing.T) {
	home := filepath.FromSlash("/Users/tester")
	cases := []struct {
		in, want string
	}{
		{"~/repositories/fleet", filepath.Join(home, "repositories/fleet")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
		// Only a leading "~/" or bare "~" is special. "~foo" (no slash) and
		// a mid-path tilde are left untouched, matching the Rust shellexpand.
		{"~foo", "~foo"},
		{"/etc/~/x", "/etc/~/x"},
	}
	for _, c := range cases {
		if got := expand(c.in, home); got != c.want {
			t.Errorf("expand(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestUnderHome(t *testing.T) {
	home := "/Users/tester"
	ok := []string{
		"/Users/tester",
		"/Users/tester/Library/Application Support/x",
		"/Users/tester/.fleet/config",
	}
	for _, p := range ok {
		if err := underHome(p, home); err != nil {
			t.Errorf("underHome(%q) = %v, want nil", p, err)
		}
	}

	bad := []string{
		"/etc/passwd",
		"/Users/tester2/x",                 // sibling — must not match as a string prefix
		"/Users/tester/../tester2/secrets", // ".." traversal, even though it textually starts under home
		"relative/path",
	}
	for _, p := range bad {
		if err := underHome(p, home); err == nil {
			t.Errorf("underHome(%q) = nil, want error", p)
		}
	}
}

func TestHasPathPrefix(t *testing.T) {
	cases := []struct {
		p, prefix string
		want      bool
	}{
		{"/a/b/c", "/a/b", true},
		{"/a/b", "/a/b", true},
		{"/a/bc", "/a/b", false},  // component-wise: "bc" != "b"
		{"/a/b2/c", "/a/b", false}, // sibling
		{"/a", "/a/b", false},      // p shorter than prefix
		{"/x/y", "/a/b", false},
	}
	for _, c := range cases {
		if got := HasPathPrefix(c.p, c.prefix); got != c.want {
			t.Errorf("HasPathPrefix(%q, %q) = %v, want %v", c.p, c.prefix, got, c.want)
		}
	}
}

func TestHasExt(t *testing.T) {
	cases := []struct {
		path    string
		allowed []string
		want    bool
	}{
		{"foo.yml", []string{"yml", "yaml"}, true},
		{"foo.YAML", []string{"yml", "yaml"}, true}, // case-insensitive
		{"foo.json", []string{"yml", "yaml"}, false},
		{"noext", []string{"yml"}, false},
		{"archive.sql.gz", []string{"gz"}, true}, // Ext() returns the last component
		{"foo.txt", []string{"yml", "yaml", "log", "json", "txt", "md", "sql", "gz"}, true},
	}
	for _, c := range cases {
		if got := HasExt(c.path, c.allowed...); got != c.want {
			t.Errorf("HasExt(%q, %v) = %v, want %v", c.path, c.allowed, got, c.want)
		}
	}
}

func TestDataDirOverride(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hangar-data")
	t.Setenv("HANGAR_DATA_DIR", root)

	for name, fn := range map[string]func() (string, error){"ConfigDir": ConfigDir, "DataDir": DataDir} {
		got, err := fn()
		if err != nil || got != root {
			t.Errorf("%s() = %q, %v; want %q", name, got, err, root)
		}
	}
	logs, err := LogDir()
	if err != nil || logs != filepath.Join(root, "logs") {
		t.Errorf("LogDir() = %q, %v; want %q", logs, err, filepath.Join(root, "logs"))
	}
	for _, d := range []string{root, logs} {
		info, err := os.Stat(d)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o700 {
			t.Errorf("%s mode = %o, want 700", d, perm)
		}
	}
}

func TestRelativeOverridesRejected(t *testing.T) {
	t.Setenv("HANGAR_DATA_DIR", "relative/data")
	t.Setenv("HANGAR_RESERVATIONS_FILE", "relative/reservations.json")
	for name, fn := range map[string]func() (string, error){
		"ConfigDir": ConfigDir, "DataDir": DataDir, "LogDir": LogDir, "ReservationsFile": ReservationsFile,
	} {
		if got, err := fn(); err == nil {
			t.Errorf("%s() = %q, want error for a relative override", name, got)
		}
	}
}

func TestReservationsFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HANGAR_RESERVATIONS_FILE", "")
	want := filepath.Join(home, ".local", "share", "fleet-workflow", "runtime-reservations.json")
	if got, err := ReservationsFile(); err != nil || got != want {
		t.Errorf("default ReservationsFile() = %q, %v; want %q", got, err, want)
	}

	override := filepath.Join(t.TempDir(), "res.json")
	t.Setenv("HANGAR_RESERVATIONS_FILE", override)
	if got, err := ReservationsFile(); err != nil || got != override {
		t.Errorf("ReservationsFile() = %q, %v; want %q", got, err, override)
	}
}

func TestLockDataDir(t *testing.T) {
	dir := t.TempDir()
	release, err := LockDataDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LockDataDir(dir); err == nil {
		t.Fatal("second LockDataDir on a held dir should fail")
	}
	release()
	release2, err := LockDataDir(dir)
	if err != nil {
		t.Fatalf("LockDataDir after release: %v", err)
	}
	release2()
}
