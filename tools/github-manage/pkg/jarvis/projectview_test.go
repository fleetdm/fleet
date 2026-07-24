package jarvis

import (
	"testing"

	"fleetdm/gm/pkg/ghapi"
)

func TestReplaceProjectView(t *testing.T) {
	// Two projects in the Project View, plus the same issue #200 lingering in a
	// leverage bucket from the last full fetch.
	m := &Model{
		board: Board{Buckets: map[Bucket][]Item{
			BucketPrimary: {
				{Kind: KindProject, Number: 108, Title: "apple"},
				{Kind: KindIssue, Number: 100},
				{Kind: KindProject, Number: 109, Title: "patching"},
				{Kind: KindIssue, Number: 300},
			},
			BucketNeedsYourHands: {{Kind: KindIssue, Number: 200}},
		}},
		statuses: map[int]string{},
		projects: map[int]int{},
	}

	// Refresh project 108: issue #100 stays and newly-assigned #200 appears.
	m.replaceProjectView(projectRefreshedMsg{
		project: 108,
		header:  Item{Kind: KindProject, Number: 108, Title: "apple"},
		issues: []Item{
			{Kind: KindIssue, Number: 100},
			{Kind: KindIssue, Number: 200},
		},
		statuses: map[int]string{200: "In progress"},
		projects: map[int]int{200: 108},
	})

	primary := m.board.Buckets[BucketPrimary]
	var gotNums []int
	for _, it := range primary {
		gotNums = append(gotNums, it.Number)
	}
	// project 108 segment refreshed (108,100,200), project 109 segment untouched (109,300).
	want := []int{108, 100, 200, 109, 300}
	if len(gotNums) != len(want) {
		t.Fatalf("BucketPrimary = %v, want %v", gotNums, want)
	}
	for i := range want {
		if gotNums[i] != want[i] {
			t.Fatalf("BucketPrimary = %v, want %v", gotNums, want)
		}
	}
	// #200 must be removed from the leverage bucket (no longer shown twice).
	if len(m.board.Buckets[BucketNeedsYourHands]) != 0 {
		t.Errorf("expected #200 dropped from NeedsYourHands, got %v", m.board.Buckets[BucketNeedsYourHands])
	}
	if m.statuses[200] != "In progress" || m.projects[200] != 108 {
		t.Errorf("expected #200 status/project merged, got %q/%d", m.statuses[200], m.projects[200])
	}
}

func TestNormalizeStatus(t *testing.T) {
	cases := map[string]string{
		"🥚 Ready":             "ready",
		"🐣 In progress":       "in progress",
		"🐥 Ready for review":  "ready for review",
		"✅ Ready for release": "ready for release",
		"✔️Awaiting QA":       "awaiting qa",
		"Done":                "done",
		"📨 Inbox":             "inbox",
	}
	for in, want := range cases {
		if got := normalizeStatus(in); got != want {
			t.Errorf("normalizeStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStatusExcludedAndReady(t *testing.T) {
	if !statusExcluded("Done") || !statusExcluded("✅ Ready for release") {
		t.Error("Done and Ready for release must be excluded")
	}
	for _, s := range []string{"🥚 Ready", "🐣 In progress", "🐥 Ready for review", "✔️Awaiting QA"} {
		if statusExcluded(s) {
			t.Errorf("%q should not be excluded", s)
		}
	}
	if !statusIsReady("🥚 Ready") {
		t.Error("🥚 Ready should be the Ready column")
	}
	for _, s := range []string{"🐥 Ready for review", "✅ Ready for release"} {
		if statusIsReady(s) {
			t.Errorf("%q must not count as the Ready column", s)
		}
	}
}

func TestResolveProject(t *testing.T) {
	org := []ghapi.OrgProject{
		{Number: 108, Title: "🍎 #g-apple-at-work", URL: "https://github.com/orgs/fleetdm/projects/108"},
		{Number: 109, Title: "❤️‍🩹 #g-auto-patching", URL: "https://github.com/orgs/fleetdm/projects/109"},
	}
	// by name (config uses the bare slug; title carries emoji + #)
	if n, _, url := resolveProject("g-apple-at-work", "fleetdm", org); n != 108 || url == "" {
		t.Errorf("name resolve got %d %q, want 108", n, url)
	}
	// by number
	if n, _, _ := resolveProject("109", "fleetdm", org); n != 109 {
		t.Errorf("numeric resolve got %d, want 109", n)
	}
	// unknown
	if n, _, _ := resolveProject("does-not-exist", "fleetdm", org); n != 0 {
		t.Errorf("unknown should resolve to 0, got %d", n)
	}
}
