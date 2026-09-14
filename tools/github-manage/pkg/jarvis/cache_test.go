package jarvis

import (
	"path/filepath"
	"testing"

	"fleetdm/gm/pkg/ghapi"
)

func TestCacheRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	res := FetchResult{
		Login: "george",
		Board: Board{Buckets: map[Bucket][]Item{
			BucketQuickWins: {{Kind: KindPR, Number: 55555, Title: "ready", Bucket: BucketQuickWins,
				PR: &ghapi.PullRequest{Number: 55555, HeadRefName: "b"}}},
			BucketNeedsYourHands: {{Kind: KindIssue, Number: 38348, Title: "do it", Bucket: BucketNeedsYourHands}},
		}},
		Statuses:      map[int]string{38348: "In progress"},
		Projects:      map[int]int{38348: 58},
		IssueProjects: map[int][]ProjectRef{38348: {{Number: 58, UpdatedAt: "2026-06-01T00:00:00Z"}}},
	}
	if err := SaveCache(path, res); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}
	got, at, ok := LoadCache(path)
	if !ok {
		t.Fatal("LoadCache returned ok=false")
	}
	if at.IsZero() {
		t.Error("expected a non-zero fetchedAt")
	}
	if got.Login != "george" || got.Statuses[38348] != "In progress" || len(got.IssueProjects[38348]) != 1 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	// The Bucket-keyed board map must survive JSON round-trip.
	if len(got.Board.Buckets[BucketQuickWins]) != 1 || got.Board.Buckets[BucketQuickWins][0].Number != 55555 {
		t.Errorf("board buckets did not round-trip: %+v", got.Board.Buckets)
	}
}

func TestLoadCacheMissing(t *testing.T) {
	if _, _, ok := LoadCache(filepath.Join(t.TempDir(), "nope.json")); ok {
		t.Error("expected ok=false for missing cache")
	}
}

// TestApplyFetchNilMapsSafe guards the panic where a cache predating the
// IssueProjects field (nil maps) made the per-item refresh assign to a nil map.
func TestApplyFetchNilMapsSafe(t *testing.T) {
	tr, _ := LoadTriageStore("")
	lk, _ := LoadLinkStore("")
	fc, _ := LoadFocusStore("")
	m := &Model{
		triage: tr, links: lk, focus: fc,
		statuses: map[int]string{}, projects: map[int]int{}, issueProjects: map[int][]ProjectRef{},
	}
	// A fetch/cache with all-nil enrichment maps (as an old cache would deserialize).
	m.applyFetch(FetchResult{Login: "x", Board: Board{Buckets: map[Bucket][]Item{}}})
	// These assignments panicked ("assignment to entry in nil map") before the fix.
	m.issueProjects[42] = []ProjectRef{{Number: 58}}
	m.statuses[42] = "In progress"
	m.projects[42] = 58
	if m.issueProjects == nil || m.statuses == nil || m.projects == nil {
		t.Fatal("enrichment maps must stay non-nil after applyFetch")
	}
}

func TestIsCompleted(t *testing.T) {
	cases := []struct {
		name string
		it   Item
		want bool
	}{
		{"merged PR", Item{Kind: KindPR, PR: &ghapi.PullRequest{State: "MERGED"}}, true},
		{"closed PR", Item{Kind: KindPR, PR: &ghapi.PullRequest{State: "CLOSED"}}, true},
		{"open PR", Item{Kind: KindPR, PR: &ghapi.PullRequest{State: "OPEN"}}, false},
		{"cherry-pick item (no PR ptr)", Item{Kind: KindPR, Number: 5}, false},
		{"closed issue", Item{Kind: KindIssue, Issue: &ghapi.Issue{State: "CLOSED"}}, true},
		{"open issue", Item{Kind: KindIssue, Issue: &ghapi.Issue{State: "OPEN"}}, false},
		{"session", Item{Kind: KindSession}, false},
	}
	for _, c := range cases {
		if got := isCompleted(c.it); got != c.want {
			t.Errorf("%s: isCompleted = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestMostRecentProject(t *testing.T) {
	m := &Model{issueProjects: map[int][]ProjectRef{
		42: {
			{Number: 58, UpdatedAt: "2026-06-01T00:00:00Z"},
			{Number: 71, UpdatedAt: "2026-06-30T00:00:00Z"}, // most recent
			{Number: 97, UpdatedAt: "2026-05-01T00:00:00Z"},
		},
	}}
	if got := m.mostRecentProject(42); got != 71 {
		t.Errorf("expected most recent project 71, got %d", got)
	}
	if got := m.mostRecentProject(999); got != 0 {
		t.Errorf("unknown issue should give 0, got %d", got)
	}
}
