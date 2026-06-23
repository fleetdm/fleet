package traymenu

import "testing"

func find(items []Item, id string) (Item, bool) {
	for _, it := range items {
		if it.ID == id {
			return it, true
		}
	}
	return Item{}, false
}

func TestBuildItemsNoRepoIdle(t *testing.T) {
	items := BuildItems(State{})

	if got := BranchLabel(State{}); got != "No repo configured" {
		t.Errorf("branch label = %q", got)
	}
	// Nothing running → Start all present, but disabled (no repo).
	start, ok := find(items, "tray:start-all")
	if !ok {
		t.Fatal("expected tray:start-all when idle")
	}
	if start.Enabled {
		t.Error("Start all should be disabled with no repo")
	}
	if _, ok := find(items, "tray:stop-all"); ok {
		t.Error("Stop all should not be present when idle")
	}
	// Service rows are informational (disabled) and show the off dot.
	serve, _ := find(items, "tray:svc-serve")
	if serve.Enabled {
		t.Error("service rows should be disabled")
	}
	if serve.Label != "⚪  fleet serve" {
		t.Errorf("idle serve label = %q", serve.Label)
	}
}

func TestBuildItemsRepoConfiguredIdle(t *testing.T) {
	b := "main"
	items := BuildItems(State{Branch: &b})
	if BranchLabel(State{Branch: &b}) != "Branch: main" {
		t.Error("branch label should include the branch")
	}
	start, _ := find(items, "tray:start-all")
	if !start.Enabled {
		t.Error("Start all should be enabled once a repo is configured")
	}
}

func TestBuildItemsRunning(t *testing.T) {
	b := "main"
	items := BuildItems(State{Branch: &b, ServeUp: true, DockerUp: true})

	// Running → Stop all replaces Start all.
	if _, ok := find(items, "tray:start-all"); ok {
		t.Error("Start all should be hidden when something is running")
	}
	stop, ok := find(items, "tray:stop-all")
	if !ok || !stop.Enabled {
		t.Errorf("Stop all should be present and enabled: %+v ok=%v", stop, ok)
	}
	// serve up → green dot + port.
	serve, _ := find(items, "tray:svc-serve")
	if serve.Label != "🟢  fleet serve  ·  :8080" {
		t.Errorf("running serve label = %q", serve.Label)
	}
	// docker up → green dot, no extra.
	docker, _ := find(items, "tray:svc-docker")
	if docker.Label != "🟢  docker" {
		t.Errorf("running docker label = %q", docker.Label)
	}
}

func TestMenuShapeStable(t *testing.T) {
	// Always: branch, sep, 4 services, sep, toggle, show, quit = 10 entries.
	if n := len(BuildItems(State{})); n != 10 {
		t.Errorf("menu has %d entries, want 10", n)
	}
	// show + quit always present and enabled.
	items := BuildItems(State{})
	for _, id := range []string{"tray:show", "tray:quit"} {
		it, ok := find(items, id)
		if !ok || !it.Enabled {
			t.Errorf("%s should be present and enabled", id)
		}
	}
}
