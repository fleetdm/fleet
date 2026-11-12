package reservations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

// Shaped like fleet-workflow's file, including fields Hangar ignores.
const fixture = `{
  "slots": {
    "s2": {
      "worktree": "/w/task",
      "previous_worktree": "/w/prev",
      "ports": {"server": 8090, "mysql": 3326, "redis": 6389, "s3": 9020, "s3_console": 9011},
      "compose_project": "fleet-s2",
      "bound": false,
      "created_at": 1762950000.123,
      "perf_ids": ["perf-1"]
    },
    "s3": {
      "worktree": "/w/other",
      "previous_worktree": null,
      "ports": {"server": 8100, "mysql": 3346, "redis": 6399, "s3": 9040, "s3_console": 9021},
      "compose_project": "fleet-s3",
      "created_at": 1762950001.5
    }
  },
  "version": 3
}`

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "runtime-reservations.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadFixture(t *testing.T) {
	got, err := Read(write(t, fixture))
	if err != nil {
		t.Fatal(err)
	}
	s2, ok := got["s2"]
	if !ok {
		t.Fatal("s2 missing")
	}
	if s2.Worktree != "/w/task" || s2.PreviousWorktree == nil || *s2.PreviousWorktree != "/w/prev" {
		t.Fatalf("s2 worktrees: %+v", s2)
	}
	if want := (settings.ServerPorts{Server: 8090, MySQL: 3326, Redis: 6389, S3: 9020, S3Console: 9011}); s2.Ports != want {
		t.Fatalf("s2 ports: %+v", s2.Ports)
	}
	if s2.ComposeProject != "fleet-s2" || s2.CreatedAt != "1762950000.123" {
		t.Fatalf("s2 identity: %+v", s2)
	}
	if got["s3"].PreviousWorktree != nil {
		t.Fatal("s3 previous should be null")
	}
	if _, ok := got["s1"]; ok {
		t.Fatal("s1 should be absent")
	}
}

func TestReadMissingFile(t *testing.T) {
	got, err := Read(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestReadMalformed(t *testing.T) {
	if _, err := Read(write(t, `{"slots": {"s2": `)); err == nil {
		t.Fatal("expected parse error")
	}
	if _, err := Read(write(t, `{"slots": {"s2": {"ports": "8090"}}}`)); err == nil {
		t.Fatal("expected type error")
	}
}
