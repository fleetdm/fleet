package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func makeRepo(t *testing.T, dir, module string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+module+"\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProbe(t *testing.T) {
	base := t.TempDir()

	good := filepath.Join(base, "fleet")
	makeRepo(t, good, "github.com/fleetdm/fleet/v4")
	if pr := probe(good); !pr.Valid {
		t.Errorf("valid clone reported invalid: %+v", pr)
	}

	// Nonexistent path.
	if pr := probe(filepath.Join(base, "nope")); pr.Valid || pr.Reason == nil {
		t.Errorf("missing path should be invalid with reason: %+v", pr)
	}

	// Dir without go.mod.
	noMod := filepath.Join(base, "nomod")
	if err := os.MkdirAll(noMod, 0o755); err != nil {
		t.Fatal(err)
	}
	if pr := probe(noMod); pr.Valid {
		t.Errorf("dir without go.mod should be invalid: %+v", pr)
	}

	// go.mod with the wrong module.
	wrong := filepath.Join(base, "other")
	makeRepo(t, wrong, "github.com/someone/else")
	if pr := probe(wrong); pr.Valid {
		t.Errorf("wrong module should be invalid: %+v", pr)
	}
}

func TestDiscoverFleetRepos(t *testing.T) {
	home := t.TempDir()

	// Depth-1 clone under a dev root.
	root := filepath.Join(home, "repositories")
	makeRepo(t, filepath.Join(root, "fleet"), "github.com/fleetdm/fleet/v4")
	// A non-repo sibling that IS an org folder containing a clone (depth 2).
	makeRepo(t, filepath.Join(root, "github-org", "fleet"), "github.com/fleetdm/fleet/v4")
	// A noise dir that should be ignored.
	if err := os.MkdirAll(filepath.Join(root, "unrelated"), 0o755); err != nil {
		t.Fatal(err)
	}
	// ~/fleet one-off.
	makeRepo(t, filepath.Join(home, "fleet"), "github.com/fleetdm/fleet/v4")

	got := discoverFleetRepos(home, []string{root})

	var paths []string
	for _, r := range got {
		if !r.Valid {
			t.Errorf("discover returned invalid entry: %+v", r)
		}
		paths = append(paths, r.Path)
	}
	if len(got) != 3 {
		t.Fatalf("found %d repos, want 3: %v", len(got), paths)
	}
	// Sorted ascending by path.
	for i := 1; i < len(paths); i++ {
		if paths[i-1] > paths[i] {
			t.Errorf("results not sorted: %v", paths)
		}
	}
}

func TestDetectFleetConfig(t *testing.T) {
	repo := t.TempDir()
	if got := DetectFleetConfig(repo); got != "" {
		t.Errorf("no config should yield empty, got %q", got)
	}

	if err := os.WriteFile(filepath.Join(repo, "fleet.yaml"), []byte("x: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := DetectFleetConfig(repo); got != "fleet.yaml" {
		t.Errorf("got %q, want fleet.yaml", got)
	}

	// fleet.yml takes precedence (checked first).
	if err := os.WriteFile(filepath.Join(repo, "fleet.yml"), []byte("x: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := DetectFleetConfig(repo); got != "fleet.yml" {
		t.Errorf("got %q, want fleet.yml", got)
	}
}
