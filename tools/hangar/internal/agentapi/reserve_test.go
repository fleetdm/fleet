package agentapi

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/reservations"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

func TestReserveLifecycle(t *testing.T) {
	e := newEnv(t)
	e.writeRes(`{"slots": {}}`)
	code, out := e.act("s2", map[string]any{"action": "reserve"})
	if code != 200 {
		t.Fatalf("reserve: %d %v", code, out)
	}
	res, err := reservations.Read(e.resPath)
	if err != nil || res["s2"].Worktree != e.task || res["s2"].PreviousWorktree == nil ||
		*res["s2"].PreviousWorktree != e.prev || res["s2"].Ports != s2Ports() || res["s2"].ComposeProject != "hangar-test-s2" {
		t.Fatalf("reservation = %+v, %v", res["s2"], err)
	}

	e.assignTask()
	_, j := e.act("s2", map[string]any{"action": "stop"})
	if got := e.waitJob(j["id"].(string)); got["state"] != stateSucceeded {
		t.Fatalf("stop = %v", got)
	}
	if code, out := e.act("s2", map[string]any{"action": "assign", "target": e.prev}); code != 200 {
		t.Fatalf("assign back: %d %v", code, out)
	}
	if code, out := e.act("s2", map[string]any{"action": "release"}); code != 200 {
		t.Fatalf("release: %d %v", code, out)
	}
	if res, _ := reservations.Read(e.resPath); len(res) != 0 {
		t.Fatalf("reservations after release = %v", res)
	}
	if n := e.slotEvents.Load(); n != 2 {
		t.Fatalf("agent:slots emitted %d times, want 2", n)
	}
	if _, held := e.mgr.HeldBy("s2"); held {
		t.Fatal("release left the slot held")
	}
}

func TestReserveGuards(t *testing.T) {
	t.Run("s1", func(t *testing.T) {
		e := newEnv(t)
		code, out := e.act("s1", map[string]any{"action": "reserve"})
		expect(t, code, out, 403, "s1 is never available")
	})
	t.Run("port shared with s1", func(t *testing.T) {
		e := newEnv(t)
		e.writeRes(`{"slots": {}}`)
		e.store.edit(func(p *settings.ServerProfile) { p.Ports.MySQL = 13306 })
		code, out := e.act("s2", map[string]any{"action": "reserve"})
		expect(t, code, out, 403, "port 13306 collides with s1")
	})
	t.Run("s1 worktree", func(t *testing.T) {
		e := newEnv(t)
		e.writeRes(`{"slots": {}}`)
		code, out := e.act("s2", map[string]any{"action": "reserve", "worktree": e.s1Worktr})
		expect(t, code, out, 403, "s1's worktree")
	})
	t.Run("not a git worktree", func(t *testing.T) {
		e := newEnv(t)
		e.writeRes(`{"slots": {}}`)
		code, out := e.act("s2", map[string]any{"action": "reserve", "worktree": filepath.Join(e.task, "missing")})
		expect(t, code, out, 403, "isn't a git worktree")
	})
	t.Run("held by another worktree", func(t *testing.T) {
		e := newEnv(t)
		code, out := e.act("s2", map[string]any{"action": "reserve", "worktree": e.prev})
		expect(t, code, out, 409, "reserved by "+e.task)
	})
	t.Run("same worktree is idempotent", func(t *testing.T) {
		e := newEnv(t)
		code, out := e.act("s2", map[string]any{"action": "reserve"})
		if code != 200 {
			t.Fatalf("reserve: %d %v", code, out)
		}
		if res, _ := reservations.Read(e.resPath); res["s2"].CreatedAt != "1.5" {
			t.Fatalf("reservation replaced: %+v", res["s2"])
		}
		if n := e.slotEvents.Load(); n != 0 {
			t.Fatalf("agent:slots emitted %d times for a no-op", n)
		}
	})
}

func TestReleaseGuards(t *testing.T) {
	t.Run("not restored", func(t *testing.T) {
		e := newEnv(t)
		e.assignTask()
		code, out := e.act("s2", map[string]any{"action": "release"})
		expect(t, code, out, 409, "assign it back first")
	})
	t.Run("running container", func(t *testing.T) {
		e := newEnv(t)
		e.probe.running = true
		code, out := e.act("s2", map[string]any{"action": "release"})
		expect(t, code, out, 409, "running hangar-test-s2 containers")
	})
	t.Run("running slot process", func(t *testing.T) {
		e := newEnv(t)
		e.mgr.setProc("s2:fleet-serve", "running", "./build/fleet serve")
		code, out := e.act("s2", map[string]any{"action": "release"})
		expect(t, code, out, 409, "s2:fleet-serve is running")
	})
	t.Run("another worktree's reservation", func(t *testing.T) {
		e := newEnv(t)
		code, out := e.act("s2", map[string]any{"action": "release", "worktree": e.prev})
		expect(t, code, out, 403, "reserved by")
	})
}

// A shared file with ASCII keys comes back byte for byte after an API reserve
// and release.
func TestReserveReleaseKeepsSharedFields(t *testing.T) {
	e := newEnv(t)
	shared := fmt.Sprintf(`{
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
        "perf-1"
      ],
      "generation": "0123456789ab"
    }
  },
  "ui_owner": %q
}
`, e.prev)
	e.writeRes(shared)
	if code, out := e.act("s2", map[string]any{"action": "reserve"}); code != 200 {
		t.Fatalf("reserve: %d %v", code, out)
	}
	if code, out := e.act("s2", map[string]any{"action": "release"}); code != 200 {
		t.Fatalf("release: %d %v", code, out)
	}
	if b, _ := os.ReadFile(e.resPath); string(b) != shared {
		t.Fatalf("file changed:\n%s", b)
	}
}

func TestReserveOneSlotPerWorktree(t *testing.T) {
	e := newEnv(t)
	e.store.mu.Lock()
	e.store.s.Servers = append(e.store.s.Servers, settings.ServerProfile{ID: "s3", Name: "other", Enabled: true,
		Ports:          settings.ServerPorts{Server: 8390, MySQL: 3626, Redis: 6689, S3: 9320, S3Console: 9311},
		ComposeProject: "hangar-test-s3"})
	e.store.mu.Unlock()
	code, out := e.act("s3", map[string]any{"action": "reserve"})
	expect(t, code, out, 409, "already holds s2")
	if res, _ := reservations.Read(e.resPath); len(res) != 1 {
		t.Fatalf("reservations = %v", res)
	}
}

func TestReserveRefusesUnrestorableAssignment(t *testing.T) {
	for name, current := range map[string]func(e *env) string{
		"s1 primary checkout": func(e *env) string { return e.primary },
		"s1 worktree":         func(e *env) string { return e.s1Worktr },
	} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			e.writeRes(`{"slots": {}}`)
			cur := current(e)
			e.store.edit(func(p *settings.ServerProfile) { p.WorktreePath = &cur })
			code, out := e.act("s2", map[string]any{"action": "reserve"})
			expect(t, code, out, 409, "can't be restored by the API")
			if res, _ := reservations.Read(e.resPath); len(res) != 0 {
				t.Fatalf("reservations = %v", res)
			}
		})
	}
}

// guiSave mimics SaveFromGUI changing s2's worktree: under the store lock it
// reads reservations and keeps a reserved slot's worktree.
func (f *fakeStore) guiSave(resPath, worktree string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	res, err := reservations.Read(resPath)
	if err != nil {
		return err
	}
	if _, ok := res["s2"]; ok {
		return nil
	}
	for i := range f.s.Servers {
		if f.s.Servers[i].ID == "s2" {
			f.s.Servers[i].WorktreePath = &worktree
		}
	}
	return nil
}

// A GUI save racing reserve must land wholly before or after it, so
// previous_worktree always matches the slot's saved worktree.
func TestReserveSerializesWithGUISave(t *testing.T) {
	e := newEnv(t)
	e.writeRes(`{"slots": {}}`)
	newPrev := filepath.Join(filepath.Dir(e.task), "prev-new")
	if err := os.Mkdir(newPrev, 0o700); err != nil {
		t.Fatal(err)
	}
	inGit, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	pc := e.api.opts.PrimaryCheckout
	e.api.opts.PrimaryCheckout = func(dir string) (string, error) {
		once.Do(func() { close(inGit); <-resume })
		return pc(dir)
	}
	reserved := make(chan int, 1)
	go func() {
		body := strings.NewReader(fmt.Sprintf(`{"action": "reserve", "worktree": %q}`, e.task))
		resp, err := http.Post(e.srv.URL+"/v1/slots/s2/actions", "application/json", body)
		if err != nil {
			reserved <- 0
			return
		}
		resp.Body.Close()
		reserved <- resp.StatusCode
	}()
	<-inGit
	saved := make(chan error, 1)
	go func() { saved <- e.store.guiSave(e.resPath, newPrev) }()
	select {
	case err := <-saved:
		saved <- err
	case <-time.After(100 * time.Millisecond):
	}
	close(resume)
	if code := <-reserved; code != 200 {
		t.Fatalf("reserve: %d", code)
	}
	if err := <-saved; err != nil {
		t.Fatal(err)
	}
	res, _ := reservations.Read(e.resPath)
	s2, _ := findSlot(e.store.Snapshot(), "s2")
	if prev := res["s2"].PreviousWorktree; prev == nil || s2.WorktreePath == nil || *prev != *s2.WorktreePath {
		t.Fatalf("previous_worktree %v, slot on %v", describe(prev), describe(s2.WorktreePath))
	}
}
