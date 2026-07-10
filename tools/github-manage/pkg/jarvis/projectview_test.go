package jarvis

import (
	"testing"

	"fleetdm/gm/pkg/ghapi"
)

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
