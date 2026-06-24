package shellpath

import (
	"os"
	"path/filepath"
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

func TestLookPathIn(t *testing.T) {
	// Build a fake PATH: an empty dir, then one holding an executable "tool"
	// and a non-executable "data" file and a "sub" directory.
	dirEmpty := t.TempDir()
	dirReal := t.TempDir()
	if err := os.WriteFile(filepath.Join(dirReal, "tool"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirReal, "data"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dirReal, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := dirEmpty + ":" + dirReal

	// Resolves an executable to the first PATH dir that has it.
	if got, err := LookPathIn(path, "tool"); err != nil || got != filepath.Join(dirReal, "tool") {
		t.Errorf("LookPathIn(tool) = %q, %v; want %q", got, err, filepath.Join(dirReal, "tool"))
	}
	// A non-executable file is not a match.
	if _, err := LookPathIn(path, "data"); err == nil {
		t.Error("LookPathIn(data): want error for non-executable file")
	}
	// A directory is not a match.
	if _, err := LookPathIn(path, "sub"); err == nil {
		t.Error("LookPathIn(sub): want error for directory")
	}
	// A name not present anywhere errors.
	if _, err := LookPathIn(path, "absent"); err == nil {
		t.Error("LookPathIn(absent): want not-found error")
	}
	// A name containing a separator is checked directly, not searched.
	abs := filepath.Join(dirReal, "tool")
	if got, err := LookPathIn(dirEmpty, abs); err != nil || got != abs {
		t.Errorf("LookPathIn(abs path) = %q, %v; want %q", got, err, abs)
	}
}

func TestCommandResolvesAndSetsEnv(t *testing.T) {
	// Pin the cached shell PATH to a temp dir holding "tool" so Command
	// resolves to an absolute path (not the bare name) and presets cmd.Env.
	dir := t.TempDir()
	toolPath := filepath.Join(dir, "tool")
	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	orig := cached
	cached = dir
	mu.Unlock()
	t.Cleanup(func() { mu.Lock(); cached = orig; mu.Unlock() })

	cmd := Command("tool", "--flag")
	if cmd.Path != toolPath {
		t.Errorf("Command resolved Path = %q, want %q", cmd.Path, toolPath)
	}
	if len(cmd.Env) == 0 {
		t.Error("Command should preset cmd.Env to the login-shell env")
	}

	// A name not on the shell PATH falls back to the bare name so Start()
	// still produces the standard not-found error.
	if miss := Command("definitely-not-a-real-tool"); miss.Path == "" {
		t.Error("Command(miss) should keep the bare name in Path")
	}
}
