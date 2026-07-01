package paths

import (
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
