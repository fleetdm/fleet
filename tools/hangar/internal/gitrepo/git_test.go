package gitrepo

import (
	"strings"
	"testing"
)

func ref(name, sha, subj, author, when, fullref string) string {
	return strings.Join([]string{name, sha, subj, author, when, fullref}, unitSep)
}

func u32p(v uint32) *uint32 { return &v }

func TestParsePorcelain(t *testing.T) {
	// Tracked modification + untracked file.
	mod, clean := parsePorcelain(" M server/foo.go\n?? newfile.txt\n")
	if clean {
		t.Error("tracked change should not be clean")
	}
	if len(mod) != 2 {
		t.Fatalf("got %d changes, want 2", len(mod))
	}
	if mod[0].Status != "M" || mod[0].Path != "server/foo.go" {
		t.Errorf("first change = %+v", mod[0])
	}
	if mod[1].Status != "??" || mod[1].Path != "newfile.txt" {
		t.Errorf("second change = %+v", mod[1])
	}

	// Only untracked → clean.
	_, clean = parsePorcelain("?? a\n?? b\n")
	if !clean {
		t.Error("only-untracked should be clean")
	}
	// Empty → clean.
	if _, clean := parsePorcelain(""); !clean {
		t.Error("empty status should be clean")
	}
}

func TestParseAheadBehind(t *testing.T) {
	cases := []struct {
		raw           string
		ahead, behind uint32
	}{
		{"3\t4", 3, 4},
		{"0\t0", 0, 0},
		{"", 0, 0},
		{"garbage", 0, 0},
		{"5", 5, 0},
	}
	for _, c := range cases {
		a, b := parseAheadBehind(c.raw)
		if a != c.ahead || b != c.behind {
			t.Errorf("parseAheadBehind(%q) = (%d,%d), want (%d,%d)", c.raw, a, b, c.ahead, c.behind)
		}
	}
}

func TestParseLastCommit(t *testing.T) {
	ci := parseLastCommit("abc123" + unitSep + "Fix the thing" + unitSep + "Andrey" + unitSep + "2 hours ago")
	if ci == nil {
		t.Fatal("expected a commit")
	}
	if ci.SHA != "abc123" || ci.Subject != "Fix the thing" || ci.Author != "Andrey" || ci.TimeAgo != "2 hours ago" {
		t.Errorf("parsed = %+v", ci)
	}
	if parseLastCommit("only\x1ftwo") != nil {
		t.Error("wrong field count should yield nil")
	}
}

func TestParseRCMinorKey(t *testing.T) {
	cases := []struct {
		name string
		key  string
		ok   bool
	}{
		{"rc-minor-fleet-v4.86.0", "4.86", true},
		{"rc-patch-fleet-v4.86.3", "4.86", true},
		{"main", "", false},
		{"rc-minor-fleet-vX.Y.Z", "", false},
		{"rc-minor-fleet-v4", "", false},
	}
	for _, c := range cases {
		key, ok := parseRCMinorKey(c.name)
		if ok != c.ok || key != c.key {
			t.Errorf("parseRCMinorKey(%q) = (%q,%v), want (%q,%v)", c.name, key, ok, c.key, c.ok)
		}
	}
}

func TestParseBranchesDedupAndRemote(t *testing.T) {
	raw := strings.Join([]string{
		ref("main", "a", "s", "me", "1d", "refs/heads/main"),
		ref("origin/main", "a", "s", "me", "1d", "refs/remotes/origin/main"), // dup of main
		ref("origin/feature", "d", "x", "me", "2d", "refs/remotes/origin/feature"),
		ref("origin/HEAD", "x", "x", "x", "x", "refs/remotes/origin/HEAD"), // skipped
	}, "\n")

	got := parseBranches(raw, "main", "", false, nil)
	if len(got) != 2 {
		t.Fatalf("got %d branches, want 2: %+v", len(got), got)
	}
	if got[0].Name != "main" || !got[0].IsCurrent || !got[0].IsLocal {
		t.Errorf("main entry wrong: %+v", got[0])
	}
	if got[1].Name != "feature" || !got[1].IsRemote || got[1].IsLocal {
		t.Errorf("feature entry wrong: %+v", got[1])
	}
}

func TestParseBranchesNonRCLimit(t *testing.T) {
	var lines []string
	for _, n := range []string{"a", "b", "c", "d"} {
		lines = append(lines, ref(n, "x", "s", "me", "1d", "refs/heads/"+n))
	}
	got := parseBranches(strings.Join(lines, "\n"), "a", "", false, u32p(2))
	if len(got) != 2 {
		t.Fatalf("limit not applied: got %d, want 2", len(got))
	}
}

func TestParseBranchesRCGrouping(t *testing.T) {
	raw := strings.Join([]string{
		ref("rc-minor-fleet-v4.88.0", "a", "s", "me", "1d", "refs/heads/rc-minor-fleet-v4.88.0"),
		ref("rc-patch-fleet-v4.88.1", "a", "s", "me", "2d", "refs/heads/rc-patch-fleet-v4.88.1"),
		ref("rc-minor-fleet-v4.87.0", "a", "s", "me", "3d", "refs/heads/rc-minor-fleet-v4.87.0"),
		ref("rc-minor-fleet-v4.86.0", "a", "s", "me", "4d", "refs/heads/rc-minor-fleet-v4.86.0"), // current
		ref("rc-minor-fleet-v4.85.0", "a", "s", "me", "5d", "refs/heads/rc-minor-fleet-v4.85.0"),
	}, "\n")

	got := parseBranches(raw, "rc-minor-fleet-v4.86.0", "", true, u32p(2))

	names := map[string]bool{}
	for _, b := range got {
		names[b.Name] = true
	}
	// Keep 2 most-recent minor lines (4.88, 4.87) incl. the patch on 4.88,
	// plus the current branch (4.86). Drop 4.85.
	for _, want := range []string{"rc-minor-fleet-v4.88.0", "rc-patch-fleet-v4.88.1", "rc-minor-fleet-v4.87.0", "rc-minor-fleet-v4.86.0"} {
		if !names[want] {
			t.Errorf("expected %q in RC result, got %v", want, names)
		}
	}
	if names["rc-minor-fleet-v4.85.0"] {
		t.Error("4.85 should have been dropped (beyond N=2 minor lines, not current)")
	}
}

func TestParseBranchesQuery(t *testing.T) {
	// A stale QA branch sits LAST (oldest committerdate), so a recency cap
	// of 1 would normally drop it — a name search must still surface it.
	raw := strings.Join([]string{
		ref("main", "a", "s", "me", "1d", "refs/heads/main"),
		ref("origin/feature-foo", "b", "s", "me", "2d", "refs/remotes/origin/feature-foo"),
		ref("qa-q7x9v2m", "c", "s", "me", "300d", "refs/heads/qa-q7x9v2m"),
	}, "\n")

	// Case-insensitive substring match across the full set, recency cap of 1
	// notwithstanding.
	got := parseBranches(raw, "main", "QA-", false, u32p(1))
	if len(got) != 1 || got[0].Name != "qa-q7x9v2m" {
		t.Fatalf("query match wrong: %+v", got)
	}

	// Limit caps the matches.
	got = parseBranches(raw, "main", "e", false, u32p(1)) // matches main, feature-foo
	if len(got) != 1 {
		t.Fatalf("query limit not applied: got %d, want 1", len(got))
	}

	// RC grouping is bypassed under a query: an old minor line beyond the
	// N-most-recent window still matches.
	rcRaw := strings.Join([]string{
		ref("rc-minor-fleet-v4.88.0", "a", "s", "me", "1d", "refs/heads/rc-minor-fleet-v4.88.0"),
		ref("rc-minor-fleet-v4.50.0", "a", "s", "me", "400d", "refs/heads/rc-minor-fleet-v4.50.0"),
	}, "\n")
	got = parseBranches(rcRaw, "", "4.50", true, u32p(1))
	if len(got) != 1 || got[0].Name != "rc-minor-fleet-v4.50.0" {
		t.Fatalf("RC query should bypass grouping: %+v", got)
	}
}
