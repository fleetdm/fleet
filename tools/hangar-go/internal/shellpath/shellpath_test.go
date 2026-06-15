package shellpath

import (
	"strings"
	"testing"
)

func TestLastNonEmptyLine(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// Banner line before the PATH (no trailing newline after PATH).
		{"Restored session: Thu May 28\n/opt/homebrew/bin:/usr/bin", "/opt/homebrew/bin:/usr/bin"},
		{"/usr/bin:/bin", "/usr/bin:/bin"},
		{"  /usr/bin:/bin  ", "/usr/bin:/bin"}, // trimmed
		{"line1\n\nline2\n\n", "line2"},
		{"", ""},
		{"\n\n", ""},
	}
	for _, c := range cases {
		if got := lastNonEmptyLine(c.in); got != c.want {
			t.Errorf("lastNonEmptyLine(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestAugmentInherited(t *testing.T) {
	home := "/Users/tester"
	got := augmentInherited(home, "/usr/bin:/bin")
	parts := strings.Split(got, ":")

	// Homebrew bin must come first (prepended dirs win over inherited).
	if parts[0] != "/opt/homebrew/bin" {
		t.Errorf("first entry = %q, want /opt/homebrew/bin", parts[0])
	}
	// Home-derived tool dirs are present.
	for _, want := range []string{"/Users/tester/go/bin", "/Users/tester/.local/bin", "/Users/tester/.fleetctl"} {
		if !strings.Contains(got, want) {
			t.Errorf("augmented PATH missing %q: %s", want, got)
		}
	}
	// Inherited entries are appended.
	if !strings.Contains(got, "/usr/bin") || !strings.Contains(got, "/bin") {
		t.Errorf("inherited entries dropped: %s", got)
	}
}

func TestAugmentInheritedDedup(t *testing.T) {
	// An inherited entry that duplicates a prepended one must not appear twice.
	got := augmentInherited("/Users/tester", "/opt/homebrew/bin:/usr/bin:/usr/bin")
	if n := strings.Count(got, "/opt/homebrew/bin"); n != 1 {
		t.Errorf("/opt/homebrew/bin appears %d times, want 1: %s", n, got)
	}
	if n := strings.Count(got, ":/usr/bin"); n != 1 {
		t.Errorf("/usr/bin appears %d times, want 1: %s", n, got)
	}
}

func TestMergeEnv(t *testing.T) {
	base := []string{"PATH=/bin", "HOME=/h", "FOO=old"}
	got := MergeEnv(base, map[string]string{"FOO": "new", "BAR": "added"})

	m := map[string]string{}
	for _, e := range got {
		if i := indexByte(e, '='); i >= 0 {
			m[e[:i]] = e[i+1:]
		}
	}
	if m["FOO"] != "new" {
		t.Errorf("FOO = %q, want new (replaced in place)", m["FOO"])
	}
	if m["BAR"] != "added" {
		t.Errorf("BAR = %q, want added", m["BAR"])
	}
	if m["PATH"] != "/bin" || m["HOME"] != "/h" {
		t.Errorf("untouched vars changed: %v", m)
	}
	// FOO must not be duplicated.
	count := 0
	for _, e := range got {
		if len(e) >= 4 && e[:4] == "FOO=" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("FOO appears %d times, want 1", count)
	}

	// Empty extra returns base unchanged.
	if out := MergeEnv(base, nil); len(out) != len(base) {
		t.Errorf("MergeEnv with nil extra changed length: %v", out)
	}
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func TestEnvWith(t *testing.T) {
	// Overrides an existing PATH in place, keeps other vars.
	base := []string{"HOME=/Users/tester", "PATH=/old/bin", "FOO=bar"}
	got := envWith(base, "/new/bin")
	var pathCount int
	var sawHome, sawFoo bool
	for _, e := range got {
		switch {
		case e == "PATH=/new/bin":
			pathCount++
		case e == "PATH=/old/bin":
			t.Error("old PATH should be replaced")
		case e == "HOME=/Users/tester":
			sawHome = true
		case e == "FOO=bar":
			sawFoo = true
		}
	}
	if pathCount != 1 {
		t.Errorf("want exactly one PATH entry, got %d in %v", pathCount, got)
	}
	if !sawHome || !sawFoo {
		t.Errorf("non-PATH vars dropped: %v", got)
	}

	// Appends PATH when the base has none.
	got = envWith([]string{"HOME=/x"}, "/new/bin")
	found := false
	for _, e := range got {
		if e == "PATH=/new/bin" {
			found = true
		}
	}
	if !found {
		t.Errorf("PATH not appended: %v", got)
	}
}

func TestAugmentInheritedEmptyHome(t *testing.T) {
	// No home dir: skip the ~-derived entries (don't emit relative-path
	// artifacts like "go/bin" from joining onto ""), still return the
	// fixed dirs. Note /usr/local/go/bin is a FIXED dir and stays.
	got := augmentInherited("", "/usr/bin")
	for _, p := range strings.Split(got, ":") {
		if p == "go/bin" || p == ".local/bin" || p == ".fleetctl" {
			t.Errorf("empty home produced relative entry %q: %s", p, got)
		}
	}
	if !strings.HasPrefix(got, "/opt/homebrew/bin") {
		t.Errorf("want homebrew prefix, got %s", got)
	}
}
