package reservations

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

// Shaped like a shared reservations file, including fields Hangar ignores.
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
	p := filepath.Join(t.TempDir(), "reservations.json")
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

// sharedFile is a two-space-indented file from another writer, with ASCII
// keys Hangar doesn't know at both levels, so it round-trips byte for byte.
const sharedFile = `{
  "slots": {
    "s3": {
      "worktree": "/w/other",
      "previous_worktree": null,
      "ports": {
        "server": 8100,
        "mysql": 3346,
        "redis": 6399,
        "s3": 9040,
        "s3_console": 9021
      },
      "compose_project": "fleet-s3",
      "bound": true,
      "created_at": 1793735621.921309,
      "perf_ids": [
        "perf-1",
        "perf-2"
      ],
      "generation": "0123456789ab",
      "note": "caf\u00e9 & <tag>",
      "empty": {},
      "list": []
    }
  },
  "ui_owner": "/w/gui",
  "version": 3
}
`

func same(a, b Reservation) bool {
	pa, pb := a.PreviousWorktree, b.PreviousWorktree
	a.PreviousWorktree, b.PreviousWorktree = nil, nil
	return a == b && (pa == nil) == (pb == nil) && (pa == nil || *pa == *pb)
}

func s2(worktree string) Reservation {
	prev := "/w/prev"
	return Reservation{Worktree: worktree, PreviousWorktree: &prev, ComposeProject: "fleet-s2",
		Ports: settings.ServerPorts{Server: 8090, MySQL: 3326, Redis: 6389, S3: 9020, S3Console: 9011}}
}

func TestReserveReleasePreservesSharedFile(t *testing.T) {
	path := write(t, sharedFile)
	res, created, err := Reserve(path, "s2", s2("/w/task"))
	if err != nil || !created {
		t.Fatalf("reserve: %v, created %v", err, created)
	}
	if !strings.Contains(res.CreatedAt, ".") {
		t.Fatalf("created_at %q isn't float seconds", res.CreatedAt)
	}
	b, _ := os.ReadFile(path)
	// The new entry has every known field, in the file's key order.
	want := `    "s2": {
      "worktree": "/w/task",
      "previous_worktree": "/w/prev",
      "ports": {
        "server": 8090,
        "mysql": 3326,
        "redis": 6389,
        "s3": 9020,
        "s3_console": 9011
      },
      "compose_project": "fleet-s2",
      "bound": false,
      "created_at": ` + res.CreatedAt + `
    }`
	if !strings.Contains(string(b), want) {
		t.Fatalf("new entry not in the file's shape:\n%s", b)
	}
	got, err := Read(path)
	if err != nil || !same(got["s2"], res) || got["s3"].CreatedAt != "1793735621.921309" {
		t.Fatalf("read back: %+v, %v", got, err)
	}

	if err := Release(path, "s2", "/w/task", res.CreatedAt); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != sharedFile {
		t.Fatalf("file changed across reserve and release:\n%s", b)
	}
	fi, err := os.Stat(path)
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v, %v", fi.Mode().Perm(), err)
	}
}

func TestReserveIdempotentAndHeld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "reservations.json")
	first, created, err := Reserve(path, "s2", s2("/w/task"))
	if err != nil || !created {
		t.Fatalf("reserve: %v, created %v", err, created)
	}
	if fi, err := os.Stat(filepath.Dir(path)); err != nil || fi.Mode().Perm() != 0o700 {
		t.Fatalf("parent dir: %v, %v", fi.Mode().Perm(), err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "reservations.lock")); err != nil {
		t.Fatalf("lock file: %v", err)
	}
	again, created, err := Reserve(path, "s2", s2("/w/task/"))
	if err != nil || created || !same(again, first) {
		t.Fatalf("same worktree: %+v, created %v, %v", again, created, err)
	}
	held, created, err := Reserve(path, "s2", s2("/w/other"))
	if !errors.Is(err, ErrHeld) || created || held.Worktree != "/w/task" {
		t.Fatalf("other worktree: %+v, created %v, %v", held, created, err)
	}
}

func TestReleaseNeedsSameReservation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservations.json")
	res, _, err := Reserve(path, "s2", s2("/w/task"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ worktree, createdAt string }{
		{"/w/other", res.CreatedAt}, {"/w/task", "1.0"},
	} {
		if err := Release(path, "s2", c.worktree, c.createdAt); !errors.Is(err, ErrChanged) {
			t.Fatalf("release %+v: %v", c, err)
		}
	}
	if err := Release(path, "s3", "/w/task", res.CreatedAt); !errors.Is(err, ErrChanged) {
		t.Fatalf("release missing slot: %v", err)
	}
}

func TestConcurrentReserveHasOneWinner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservations.json")
	var wg sync.WaitGroup
	var winners atomic.Int32
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, created, err := Reserve(path, "s2", s2(fmt.Sprintf("/w/task-%d", i)))
			if created {
				winners.Add(1)
			} else if !errors.Is(err, ErrHeld) {
				t.Errorf("reserve %d: %v", i, err)
			}
		}()
	}
	wg.Wait()
	if n := winners.Load(); n != 1 {
		t.Fatalf("%d winners", n)
	}
}

func TestReserveOneSlotPerWorktree(t *testing.T) {
	dir := t.TempDir()
	task := filepath.Join(dir, "task")
	if err := os.Mkdir(task, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(task, link); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "reservations.json")
	if _, _, err := Reserve(path, "s3", s2(task)); err != nil {
		t.Fatal(err)
	}
	_, created, err := Reserve(path, "s2", s2(link+"/"))
	var other *OtherSlotError
	if created || !errors.As(err, &other) || other.Slot != "s3" {
		t.Fatalf("second slot: created %v, %v", created, err)
	}
	if res, _ := Read(path); len(res) != 1 {
		t.Fatalf("reservations = %v", res)
	}
	if _, created, err := Reserve(path, "s3", s2(link)); err != nil || created {
		t.Fatalf("same slot via symlink: created %v, %v", created, err)
	}
}

// Unknown fields survive by value: their value bytes stay raw, but keys are
// re-encoded, so an escaped key keeps its meaning, not its spelling.
func TestUnknownFieldsKeepValueBytesNotKeySpelling(t *testing.T) {
	path := write(t, `{"slots": {}, "café": 1.50, "x": {"y": 1e3}}`)
	res, _, err := Reserve(path, "s2", s2("/w/task"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Release(path, "s2", "/w/task", res.CreatedAt); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	for _, want := range []string{`"café": 1.50`, `"y": 1e3`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("missing %s in:\n%s", want, b)
		}
	}
}
