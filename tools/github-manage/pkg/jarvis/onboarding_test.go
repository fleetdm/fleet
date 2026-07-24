package jarvis

import (
	"path/filepath"
	"testing"

	"fleetdm/gm/pkg/ghapi"
)

func TestFuzzyScore(t *testing.T) {
	cases := []struct {
		query, target string
		wantOK        bool
	}{
		{"", "anything", true},
		{"apple", "🍎 #g-apple-at-work", true},
		{"aaw", "g-apple-at-work", true}, // subsequence a..a..w
		{"applx", "g-apple-at-work", false},
		{"zzz", "g-apple-at-work", false},
		{"gap", "g-apple", true},
	}
	for _, c := range cases {
		_, ok := fuzzyScore(c.query, c.target)
		if ok != c.wantOK {
			t.Errorf("fuzzyScore(%q, %q) ok = %v, want %v", c.query, c.target, ok, c.wantOK)
		}
	}

	// A tighter, earlier match should outscore a scattered one.
	tight, _ := fuzzyScore("apple", "apple-pie")
	loose, _ := fuzzyScore("apple", "a-p-p-l-e-x")
	if tight <= loose {
		t.Errorf("expected tight match (%d) to outscore loose (%d)", tight, loose)
	}
}

func TestProjectHandle(t *testing.T) {
	cases := map[string]string{
		"🍎 #g-apple-at-work": "g-apple-at-work",
		"❤️‍🩹 #g-auto-patching": "g-auto-patching",
		"🚀 Rocket":            "Rocket",
		"Plain Title":         "Plain Title",
		"#no-emoji":           "no-emoji",
	}
	for in, want := range cases {
		if got := projectHandle(in); got != want {
			t.Errorf("projectHandle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestChosenHandlesSortedByNumber(t *testing.T) {
	projects := []ghapi.OrgProject{
		{Number: 109, Title: "❤️‍🩹 #g-auto-patching"},
		{Number: 108, Title: "🍎 #g-apple-at-work"},
		{Number: 70, Title: "📦 #g-software"},
	}
	m := newOnboardModel("fleetdm", projects)
	// Select all three (indices into the source slice).
	m.selected = map[int]struct{}{0: {}, 1: {}, 2: {}}

	got := m.chosenHandles()
	want := []string{"g-software", "g-apple-at-work", "g-auto-patching"} // by number: 70,108,109
	if len(got) != len(want) {
		t.Fatalf("chosenHandles() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("chosenHandles() = %v, want %v", got, want)
		}
	}
}

func TestConfigSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := &Config{PrimaryProjects: []string{"g-apple-at-work", "g-auto-patching"}}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := LoadConfig(path)
	if len(got.PrimaryProjects) != 2 || got.PrimaryProjects[0] != "g-apple-at-work" {
		t.Errorf("round-trip PrimaryProjects = %v", got.PrimaryProjects)
	}
	// Defaults still fill in for unset fields.
	if len(got.CloneBaseDirs) == 0 {
		t.Errorf("expected CloneBaseDirs default to be filled")
	}
}
