package apps

import "testing"

// TestAppCollectorScopePrecedence covers the machine-wide-first ordering the
// Windows scan relies on: an app installed both machine-wide and per-user must
// be reported once, with scope "system".
func TestAppCollectorScopePrecedence(t *testing.T) {
	c := newAppCollector()
	if !c.add(appCandidate{MatchTokens: []string{"Cursor"}, Scope: "system", Source: "registry", Version: "1.0"}) {
		t.Fatal("first Cursor candidate was not added")
	}
	if c.add(appCandidate{MatchTokens: []string{"Cursor"}, Scope: "user", Source: "registry", Version: "2.0"}) {
		t.Error("second Cursor candidate was added; the machine-wide one should win")
	}

	got := c.apps()
	if len(got) != 1 {
		t.Fatalf("got %d apps, want 1: %+v", len(got), got)
	}
	if got[0].Scope != "system" || got[0].Version != "1.0" {
		t.Errorf("got scope=%q version=%q, want scope=\"system\" version=\"1.0\"", got[0].Scope, got[0].Version)
	}
}

// TestAppCollectorSharedAcrossSources is the invariant that keeps the uninstall
// and MSIX scans from double-reporting one app: they share a collector, and the
// earlier scan's richer metadata wins.
func TestAppCollectorSharedAcrossSources(t *testing.T) {
	c := newAppCollector()
	c.add(appCandidate{
		MatchTokens: []string{"ChatGPT"},
		Source:      "registry",
		Version:     "1.2024.30",
		Path:        `C:\Users\alice\AppData\Local\Programs\ChatGPT`,
		Scope:       "user",
	})
	// The same app, discovered again as an MSIX package.
	c.add(appCandidate{
		MatchTokens: []string{"OpenAI.ChatGPT-Desktop"},
		Source:      "appx",
		Version:     "1.2024.30.0",
		Path:        `C:\Program Files\WindowsApps\OpenAI.ChatGPT-Desktop_1.2024.30.0_x64__2p2nqsd0c76g0`,
		Scope:       "user",
	})

	got := c.apps()
	if len(got) != 1 {
		t.Fatalf("got %d apps, want 1: %+v", len(got), got)
	}
	if got[0].PlatformSource != "registry" {
		t.Errorf("got source %q, want \"registry\": the uninstall entry is scanned first and carries better metadata", got[0].PlatformSource)
	}
}

// TestAppCollectorImpostorDoesNotMaskRealInstall covers the masking hazard of
// first-match-wins dedup: an unrelated system-wide program whose name merely
// contains a known-app token must not match at all, so it can never claim the
// slot of a genuine per-user install of that app scanned later.
func TestAppCollectorImpostorDoesNotMaskRealInstall(t *testing.T) {
	c := newAppCollector()
	if c.add(appCandidate{
		MatchTokens: []string{"NVIDIA Control Panel"},
		Vendor:      "NVIDIA Corporation",
		Version:     "8.1.969.0",
		Path:        `C:\Program Files\NVIDIA Corporation\Control Panel Client`,
		Scope:       "system",
		Source:      "registry",
	}) {
		t.Error("NVIDIA Control Panel was collected as an AI app")
	}
	if !c.add(appCandidate{
		MatchTokens: []string{"Dia"},
		DisplayName: "Dia",
		Vendor:      "The Browser Company",
		Version:     "1.0.0",
		Path:        `C:\Users\alice\AppData\Local\Programs\Dia`,
		Scope:       "user",
		Source:      "registry",
	}) {
		t.Error("genuine per-user Dia install was not collected")
	}

	got := c.apps()
	if len(got) != 1 {
		t.Fatalf("got %d apps, want 1: %+v", len(got), got)
	}
	if got[0].Name != "dia" || got[0].DisplayName != "Dia" || got[0].Scope != "user" || got[0].Vendor != "The Browser Company" {
		t.Errorf("got %+v, want the genuine user-scoped Dia install", got[0])
	}
}

func TestAppCollectorSkipsUnknown(t *testing.T) {
	c := newAppCollector()
	for _, tokens := range [][]string{
		{"Microsoft.WindowsCalculator"},
		{"7-Zip 23.01"},
		{""},
	} {
		if c.add(appCandidate{MatchTokens: tokens, Source: "registry"}) {
			t.Errorf("add(%q) reported a match against the known-app list", tokens)
		}
	}
	if got := c.apps(); len(got) != 0 {
		t.Errorf("got %d apps, want 0: %+v", len(got), got)
	}
}

// TestAppCollectorWants covers the pre-check the MSIX scan uses to avoid reading
// registry values for packages it would discard.
func TestAppCollectorWants(t *testing.T) {
	c := newAppCollector()

	if !c.wants("OpenAI.ChatGPT-Desktop") {
		t.Error("wants(ChatGPT package) = false before it is collected, want true")
	}
	if c.wants("Microsoft.WindowsCalculator") {
		t.Error("wants(unknown package) = true, want false")
	}

	c.add(appCandidate{MatchTokens: []string{"OpenAI.ChatGPT-Desktop"}, Source: "appx"})
	if c.wants("OpenAI.ChatGPT-Desktop") {
		t.Error("wants(ChatGPT package) = true after it is collected, want false")
	}
	// A different discovery of the same app is also unwanted, not just the
	// identical token.
	if c.wants("ChatGPT") {
		t.Error("wants(ChatGPT display name) = true after the package was collected, want false")
	}
}

func TestAppCollectorPreservesOrderAndFields(t *testing.T) {
	c := newAppCollector()
	c.add(appCandidate{
		MatchTokens: []string{"Ollama"},
		DisplayName: "Ollama",
		Vendor:      "Ollama Inc.",
		Version:     "0.5.7",
		Path:        `C:\Users\alice\AppData\Local\Programs\Ollama`,
		Scope:       "user",
		Source:      "registry",
	})
	c.add(appCandidate{MatchTokens: []string{"Cursor"}, Source: "registry", Scope: "system"})

	got := c.apps()
	if len(got) != 2 {
		t.Fatalf("got %d apps, want 2: %+v", len(got), got)
	}
	if got[0].Name != "ollama" || got[1].Name != "cursor" {
		t.Errorf("got order [%q %q], want [\"ollama\" \"cursor\"]", got[0].Name, got[1].Name)
	}
	want := App{
		Name: "ollama", DisplayName: "Ollama", Vendor: "Ollama Inc.", Version: "0.5.7",
		Path: `C:\Users\alice\AppData\Local\Programs\Ollama`, Scope: "user", PlatformSource: "registry",
	}
	if got[0] != want {
		t.Errorf("got %+v, want %+v", got[0], want)
	}
}
