package agentapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

type fakeStore struct {
	mu sync.Mutex
	s  settings.Settings
}

func (f *fakeStore) Snapshot() settings.Settings {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := f.s
	c.Servers = slices.Clone(f.s.Servers)
	return c
}

func (f *fakeStore) Update(fn func(*settings.Settings) error) (settings.Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	next := f.s
	next.Servers = slices.Clone(f.s.Servers)
	if err := fn(&next); err != nil {
		return settings.Settings{}, err
	}
	f.s = next
	return next, nil
}

func (f *fakeStore) edit(fn func(*settings.ServerProfile)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.s.Servers {
		if f.s.Servers[i].ID == "s2" {
			fn(&f.s.Servers[i])
		}
	}
}

type fakeInv struct {
	done chan struct{}
	code int
}

// fakeManager follows the processes contract: holds refuse foreign
// processes, Docker ops, and perf runs on the slot's port, and refuse
// non-holder starts. Steps exit with codes[id] unless listed in block.
type fakeManager struct {
	mu       sync.Mutex
	procs    map[string]*processes.ProcInfo
	invs     map[uint64]*fakeInv
	live     map[string]uint64
	seq      uint64
	holds    map[string]processes.HoldSpec
	dockerOp map[string]bool
	block    map[string]bool
	codes    map[string]int
	calls    []string
	logs     []processes.LogEntry
}

func newFakeManager() *fakeManager {
	return &fakeManager{procs: map[string]*processes.ProcInfo{}, invs: map[uint64]*fakeInv{}, live: map[string]uint64{},
		holds: map[string]processes.HoldSpec{}, dockerOp: map[string]bool{}, block: map[string]bool{}, codes: map[string]int{}}
}

func slotOf(id string) string { s, _, _ := strings.Cut(id, ":"); return s }

func (m *fakeManager) setProc(id, state, command string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.procs[id] = &processes.ProcInfo{ID: id, State: state, Command: command, RecentLog: []string{"recent " + command}}
}

func (m *fakeManager) Start(id string, a processes.StartArgs) (processes.Invocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h, ok := m.holds[slotOf(id)]; ok && h.Owner != a.Owner {
		return processes.Invocation{}, fmt.Errorf("%s is held by %s", slotOf(id), h.Owner)
	}
	m.calls = append(m.calls, "start "+id)
	m.seq++
	inv := &fakeInv{done: make(chan struct{})}
	m.invs[m.seq] = inv
	m.procs[id] = &processes.ProcInfo{ID: id, State: "running", Command: a.Program}
	m.logs = append(m.logs, processes.LogEntry{TsMS: nowMS(), Channel: a.LogChannel, Message: "start " + id})
	if m.block[id] {
		m.live[id] = m.seq
	} else {
		m.exitLocked(m.seq, id, m.codes[id])
	}
	return processes.Invocation{ID: id, Seq: m.seq}, nil
}

func (m *fakeManager) exitLocked(seq uint64, id string, code int) {
	delete(m.live, id)
	m.invs[seq].code = code
	m.procs[id].State = "done"
	close(m.invs[seq].done)
}

func (m *fakeManager) exit(id string, code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if seq, ok := m.live[id]; ok {
		m.exitLocked(seq, id, code)
	}
}

func (m *fakeManager) Wait(ctx context.Context, inv processes.Invocation) (int, error) {
	m.mu.Lock()
	i := m.invs[inv.Seq]
	m.mu.Unlock()
	select {
	case <-i.done:
		return i.code, nil
	case <-ctx.Done():
		return -1, ctx.Err()
	}
}

func (m *fakeManager) Stop(id, owner string) error {
	m.mu.Lock()
	m.calls = append(m.calls, "stop "+id)
	if p := m.procs[id]; p != nil {
		p.State = "done"
	}
	m.mu.Unlock()
	m.exit(id, -1)
	return nil
}

func (m *fakeManager) ComposeDown(id, cwd, project, owner string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, "down "+project)
	return nil
}

func (m *fakeManager) Hold(h processes.HoldSpec) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cur, ok := m.holds[h.Slot]; ok {
		return fmt.Errorf("%s is held by %s", h.Slot, cur.Owner)
	}
	if m.dockerOp[h.Project] {
		return fmt.Errorf("docker operation in flight on %s", h.Project)
	}
	for id, p := range m.procs {
		if p.State != "running" && p.State != "stopping" {
			continue
		}
		if slotOf(id) == h.Slot && !slices.Contains(h.Replace, id) && !slices.Contains(h.Allow, id) {
			return fmt.Errorf("%s is running", id)
		}
		if strings.HasPrefix(id, "perf-") && strings.Contains(p.Command, fmt.Sprintf(":%d", h.ServerPort)) {
			return fmt.Errorf("%s targets %s", id, h.Slot)
		}
	}
	m.holds[h.Slot] = h
	return nil
}

func (m *fakeManager) Release(slot, owner string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.holds[slot].Owner == owner {
		delete(m.holds, slot)
	}
}

func (m *fakeManager) HeldBy(slot string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.holds[slot]
	return h.Owner, ok
}

func (m *fakeManager) ListProcesses() []processes.ProcInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []processes.ProcInfo
	for _, p := range m.procs {
		out = append(out, *p)
	}
	return out
}

func (m *fakeManager) ReadLogWindowScrubbed(source string, sinceMS, untilMS uint64, _ int) processes.LogWindow {
	m.mu.Lock()
	defer m.mu.Unlock()
	w := processes.LogWindow{Entries: []processes.LogEntry{}}
	for _, e := range m.logs {
		if e.Channel == source && e.TsMS >= sinceMS && (untilMS == 0 || e.TsMS <= untilMS) {
			w.Entries = append(w.Entries, e)
		}
	}
	return w
}

// started waits until id has been started.
func (m *fakeManager) started(t *testing.T, id string) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		if slices.Contains(m.Calls(), "start "+id) {
			return
		}
	}
	t.Fatalf("%s never started", id)
}

func (m *fakeManager) Calls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.calls)
}

type fakeProbe struct {
	mu         sync.Mutex
	running    bool
	runningErr error
}

func (p *fakeProbe) DockerInspect(context.Context, string) ([]byte, error) {
	return nil, errors.New("no docker in tests")
}
func (p *fakeProbe) FleetTableCount(context.Context, string) (int, error) { return 0, nil }
func (p *fakeProbe) ProjectRunning(context.Context, string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running, p.runningErr
}
func (p *fakeProbe) Listening(uint16) bool { return false }

type env struct {
	t        *testing.T
	api      *API
	srv      *httptest.Server
	store    *fakeStore
	mgr      *fakeManager
	probe    *fakeProbe
	resPath  string
	task     string // T, the reserving worktree
	prev     string // P, the previous assignment
	primary  string
	backups  string
	s1Worktr string
}

func s2Ports() settings.ServerPorts {
	return settings.ServerPorts{Server: 8380, MySQL: 3606, Redis: 6679, S3: 9300, S3Console: 9301}
}

func newEnv(t *testing.T) *env {
	t.Helper()
	root := t.TempDir()
	e := &env{t: t, mgr: newFakeManager(), probe: &fakeProbe{},
		task: filepath.Join(root, "task"), prev: filepath.Join(root, "prev"), primary: filepath.Join(root, "fleet"),
		s1Worktr: filepath.Join(root, "s1"), backups: filepath.Join(root, "backups"), resPath: filepath.Join(root, "reservations.json")}
	for _, d := range []string{e.task, e.prev, e.primary, e.s1Worktr} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	e.store = &fakeStore{s: settings.Settings{Servers: []settings.ServerProfile{
		{ID: "s1", Name: "main", WorktreePath: &e.s1Worktr, Ports: settings.ServerPorts{Server: 18080, MySQL: 13306, Redis: 16379, S3: 19000, S3Console: 19001},
			ComposeProject: "hangar-test-s1", Enabled: true},
		{ID: "s2", Name: "task", WorktreePath: &e.prev, Ports: s2Ports(), ComposeProject: "hangar-test-s2", Enabled: true,
			FleetServe: settings.FleetServeConfig{Env: []settings.EnvVar{{Key: "FLEET_SECRET", Value: "SECRETVALUE", Enabled: true}}}},
	}}}
	e.writeRes(fmt.Sprintf(`{"slots": {"s2": {"worktree": %q, "previous_worktree": %q, "ports": %s, "compose_project": "hangar-test-s2", "created_at": 1.5}}}`,
		e.task, e.prev, portsJSON(s2Ports())))
	e.api = New(Options{
		Store: e.store, Manager: e.mgr, ReservationsPath: e.resPath, Probe: e.probe,
		PrimaryCheckout: func(dir string) (string, error) {
			if _, err := os.Stat(dir); err != nil {
				return "", err
			}
			return e.primary, nil
		},
		BackupsDir: func(string) (string, error) { return e.backups, nil },
	})
	e.srv = httptest.NewServer(e.api.Handler())
	t.Cleanup(func() {
		e.api.Close()
		e.srv.Close()
	})
	return e
}

func portsJSON(p settings.ServerPorts) string {
	b, _ := json.Marshal(p)
	return string(b)
}

// writeRes replaces the fixture atomically, like the helper, so a job's
// concurrent read never sees a truncated file.
func (e *env) writeRes(body string) {
	e.t.Helper()
	tmp := e.resPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o600); err != nil {
		e.t.Fatal(err)
	}
	if err := os.Rename(tmp, e.resPath); err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) do(method, path string, body any) (int, map[string]any) {
	e.t.Helper()
	var r *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	} else {
		r = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, r)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func (e *env) act(slot string, body map[string]any) (int, map[string]any) {
	e.t.Helper()
	if _, ok := body["worktree"]; !ok {
		body["worktree"] = e.task
	}
	return e.do("POST", "/v1/slots/"+slot+"/actions", body)
}

func (e *env) assignTask() {
	e.t.Helper()
	if code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task}); code != 200 {
		e.t.Fatalf("assign: %d %v", code, out)
	}
}

func (e *env) waitJob(id string) map[string]any {
	e.t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(2 * time.Millisecond) {
		_, j := e.do("GET", "/v1/jobs/"+id, nil)
		if j["state"] != stateRunning {
			return j
		}
	}
	e.t.Fatalf("job %s still running", id)
	return nil
}

func expect(t *testing.T, code int, out map[string]any, wantCode int, wantMsg string) {
	t.Helper()
	msg, _ := out["error"].(string)
	if code != wantCode || !strings.Contains(msg, wantMsg) {
		t.Fatalf("got %d %q, want %d containing %q", code, msg, wantCode, wantMsg)
	}
}

func TestS1Guards(t *testing.T) {
	t.Run("s1 id", func(t *testing.T) {
		e := newEnv(t)
		code, out := e.act("s1", map[string]any{"action": "build"})
		expect(t, code, out, 403, "s1 is never available")
	})
	t.Run("single port collides with s1", func(t *testing.T) {
		e := newEnv(t)
		ports := s2Ports()
		ports.Redis = 16379
		e.store.edit(func(p *settings.ServerProfile) { p.Ports = ports })
		e.writeRes(fmt.Sprintf(`{"slots": {"s2": {"worktree": %q, "previous_worktree": %q, "ports": %s, "compose_project": "hangar-test-s2"}}}`, e.task, e.prev, portsJSON(ports)))
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task})
		expect(t, code, out, 403, "port 16379 collides with s1")
	})
	t.Run("canonical default port", func(t *testing.T) {
		e := newEnv(t)
		e.store.edit(func(p *settings.ServerProfile) { p.Ports.S3Console = 9001 })
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task})
		expect(t, code, out, 403, "port 9001 collides with s1")
	})
	for _, project := range []string{"hangar-test-s1", "fleet"} {
		t.Run("project "+project, func(t *testing.T) {
			e := newEnv(t)
			e.store.edit(func(p *settings.ServerProfile) { p.ComposeProject = project })
			code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task})
			expect(t, code, out, 403, "collides with s1")
		})
	}
	t.Run("s1 worktree", func(t *testing.T) {
		e := newEnv(t)
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task, "worktree": e.s1Worktr})
		expect(t, code, out, 403, "s1's worktree")
	})
	t.Run("primary checkout", func(t *testing.T) {
		e := newEnv(t)
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task, "worktree": e.primary})
		expect(t, code, out, 403, "primary checkout")
	})
}

// Only s1's own worktree and its repo's primary checkout are off limits; a
// standalone clone is its own primary checkout and stays usable.
func TestS1RepoGuards(t *testing.T) {
	setup := func(t *testing.T, prev func(e *env) string) *env {
		e := newEnv(t)
		clone := filepath.Join(filepath.Dir(e.task), "clone")
		if err := os.Mkdir(clone, 0o700); err != nil {
			t.Fatal(err)
		}
		e.api.opts.PrimaryCheckout = func(dir string) (string, error) {
			if _, err := os.Stat(dir); err != nil {
				return "", err
			}
			if samePath(dir, clone) {
				return clone, nil
			}
			return e.primary, nil
		}
		if prev == nil {
			prev = func(*env) string { return clone }
		}
		e.writeRes(fmt.Sprintf(`{"slots": {"s2": {"worktree": %q, "previous_worktree": %q, "ports": %s, "compose_project": "hangar-test-s2"}}}`,
			e.task, prev(e), portsJSON(s2Ports())))
		e.assignTask()
		return e
	}
	t.Run("standalone clone restores", func(t *testing.T) {
		e := setup(t, nil)
		clone := filepath.Join(filepath.Dir(e.task), "clone")
		if code, out := e.act("s2", map[string]any{"action": "assign", "target": clone}); code != 200 {
			t.Fatalf("restore: %d %v", code, out)
		}
	})
	t.Run("s1 worktree refused", func(t *testing.T) {
		e := setup(t, func(e *env) string { return e.s1Worktr })
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.s1Worktr})
		expect(t, code, out, 403, "s1's worktree")
	})
	t.Run("s1 primary checkout refused", func(t *testing.T) {
		e := setup(t, func(e *env) string { return e.primary })
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.primary})
		expect(t, code, out, 403, "primary checkout")
	})
}

func TestReservationGuards(t *testing.T) {
	t.Run("worktree mismatch", func(t *testing.T) {
		e := newEnv(t)
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.prev, "worktree": e.prev})
		expect(t, code, out, 403, "reserved by")
	})
	t.Run("profile drift", func(t *testing.T) {
		e := newEnv(t)
		e.store.edit(func(p *settings.ServerProfile) { p.Ports.Server = 8390 })
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task})
		expect(t, code, out, 403, "no longer match")
	})
	t.Run("no reservation", func(t *testing.T) {
		e := newEnv(t)
		e.writeRes(`{"slots": {}}`)
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task})
		expect(t, code, out, 403, "no fleet-workflow reservation")
	})
	t.Run("malformed reservations", func(t *testing.T) {
		e := newEnv(t)
		e.writeRes(`{"slots": `)
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task})
		expect(t, code, out, 409, "parsing reservations")
	})
	t.Run("lifecycle needs the assignment", func(t *testing.T) {
		e := newEnv(t)
		code, out := e.act("s2", map[string]any{"action": "build"})
		expect(t, code, out, 403, "assign it first")
	})
}

func TestAssign(t *testing.T) {
	t.Run("saves the reservation's spelling", func(t *testing.T) {
		e := newEnv(t)
		if code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task + "/"}); code != 200 {
			t.Fatalf("assign: %d %v", code, out)
		}
		if s2, _ := findSlot(e.store.Snapshot(), "s2"); *s2.WorktreePath != e.task {
			t.Fatalf("worktree = %q, want %q", *s2.WorktreePath, e.task)
		}
	})
	t.Run("to the task and back", func(t *testing.T) {
		e := newEnv(t)
		if err := os.WriteFile(filepath.Join(e.task, "fleet.yml"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		e.assignTask()
		s2, _ := findSlot(e.store.Snapshot(), "s2")
		if *s2.WorktreePath != e.task || s2.FleetServe.ConfigPath == nil || *s2.FleetServe.ConfigPath != "fleet.yml" {
			t.Fatalf("s2 = %+v", s2)
		}
		if code, out := e.act("s2", map[string]any{"action": "assign", "target": e.prev}); code != 200 {
			t.Fatalf("restore: %d %v", code, out)
		}
		if s2, _ := findSlot(e.store.Snapshot(), "s2"); *s2.WorktreePath != e.prev {
			t.Fatalf("s2 = %v", *s2.WorktreePath)
		}
		if _, held := e.mgr.HeldBy("s2"); held {
			t.Fatal("assign left the slot held")
		}
	})
	t.Run("target outside the set", func(t *testing.T) {
		e := newEnv(t)
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.primary})
		expect(t, code, out, 403, "target must be")
	})
	t.Run("missing target", func(t *testing.T) {
		e := newEnv(t)
		code, out := e.act("s2", map[string]any{"action": "assign"})
		expect(t, code, out, 400, "needs target")
	})
	t.Run("null target refused when previous is set", func(t *testing.T) {
		e := newEnv(t)
		code, out := e.act("s2", map[string]any{"action": "assign", "target": nil})
		expect(t, code, out, 403, "only when previous_worktree is null")
	})
	t.Run("null target allowed for null previous", func(t *testing.T) {
		e := newEnv(t)
		e.writeRes(fmt.Sprintf(`{"slots": {"s2": {"worktree": %q, "previous_worktree": null, "ports": %s, "compose_project": "hangar-test-s2"}}}`, e.task, portsJSON(s2Ports())))
		e.assignTask()
		if code, out := e.act("s2", map[string]any{"action": "assign", "target": nil}); code != 200 {
			t.Fatalf("restore null: %d %v", code, out)
		}
		if s2, _ := findSlot(e.store.Snapshot(), "s2"); s2.WorktreePath != nil {
			t.Fatalf("worktree = %v", *s2.WorktreePath)
		}
	})
	t.Run("vanished previous", func(t *testing.T) {
		e := newEnv(t)
		if err := os.Remove(e.prev); err != nil {
			t.Fatal(err)
		}
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.prev})
		expect(t, code, out, 409, "runtime-recovery.md")
	})
	t.Run("running slot process", func(t *testing.T) {
		e := newEnv(t)
		e.mgr.setProc("s2:fleet-serve", "running", "./build/fleet serve")
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task})
		expect(t, code, out, 409, "s2:fleet-serve is running")
	})
	t.Run("perf run on the slot port", func(t *testing.T) {
		e := newEnv(t)
		e.mgr.setProc("perf-1", "running", "osquery-perf --server_url https://localhost:8380")
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task})
		expect(t, code, out, 409, "perf-1 targets s2")
	})
	t.Run("running container", func(t *testing.T) {
		e := newEnv(t)
		e.probe.running = true
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task})
		expect(t, code, out, 409, "running hangar-test-s2 containers")
		if _, held := e.mgr.HeldBy("s2"); held {
			t.Fatal("refused assign left the slot held")
		}
	})
	t.Run("failing probe", func(t *testing.T) {
		e := newEnv(t)
		e.probe.runningErr = errors.New("docker daemon down")
		code, out := e.act("s2", map[string]any{"action": "assign", "target": e.task})
		expect(t, code, out, 409, "docker daemon down")
	})
}

func TestJobAdmission(t *testing.T) {
	t.Run("second action on a slot", func(t *testing.T) {
		e := newEnv(t)
		e.assignTask()
		e.mgr.block["s2:make-deps"] = true
		code, j := e.act("s2", map[string]any{"action": "build"})
		if code != 202 {
			t.Fatalf("build: %d %v", code, j)
		}
		code, out := e.act("s2", map[string]any{"action": "docker-up"})
		expect(t, code, out, 409, "held by "+j["id"].(string))
		code, out = e.act("s2", map[string]any{"action": "assign", "target": e.prev})
		expect(t, code, out, 409, "held by")
		e.mgr.started(t, "s2:make-deps")
		e.mgr.exit("s2:make-deps", 0)
		if got := e.waitJob(j["id"].(string)); got["state"] != stateSucceeded {
			t.Fatalf("job = %v", got)
		}
	})
	t.Run("running serve refuses build and prepare-db, admits backup", func(t *testing.T) {
		e := newEnv(t)
		e.assignTask()
		e.mgr.setProc("s2:fleet-serve", "running", "./build/fleet serve")
		for _, action := range []string{"build", "prepare-db"} {
			code, out := e.act("s2", map[string]any{"action": action})
			expect(t, code, out, 409, "s2:fleet-serve is running")
		}
		e.mgr.block["s2:db-backup"] = true
		code, j := e.act("s2", map[string]any{"action": "backup", "name": "mine"})
		if code != 202 {
			t.Fatalf("backup: %d %v", code, j)
		}
		if owner, _ := e.mgr.HeldBy("s2"); owner != j["id"] {
			t.Fatalf("held by %q", owner)
		}
		e.mgr.started(t, "s2:db-backup")
		e.mgr.exit("s2:db-backup", 1)
		if got := e.waitJob(j["id"].(string)); got["state"] != stateFailed {
			t.Fatalf("job = %v", got)
		}
	})
	t.Run("in-flight compose down refuses a job", func(t *testing.T) {
		e := newEnv(t)
		e.assignTask()
		e.mgr.dockerOp["hangar-test-s2"] = true
		code, out := e.act("s2", map[string]any{"action": "docker-up"})
		expect(t, code, out, 409, "docker operation in flight")
	})
	t.Run("existing backup name", func(t *testing.T) {
		e := newEnv(t)
		e.assignTask()
		if err := os.MkdirAll(e.backups, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(e.backups, "mine.sql.gz"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		code, out := e.act("s2", map[string]any{"action": "backup", "name": "mine"})
		expect(t, code, out, 409, "already exists")
		code, out = e.act("s2", map[string]any{"action": "backup", "name": "bad;name"})
		expect(t, code, out, 400, "may only contain")
	})
	t.Run("backup name with a path", func(t *testing.T) {
		e := newEnv(t)
		e.assignTask()
		for _, name := range []string{"a/b", "a..b", `a\b`} {
			code, out := e.act("s2", map[string]any{"action": "backup", "name": name})
			expect(t, code, out, 400, "plain file name")
		}
	})
	t.Run("stop replaces every slot process", func(t *testing.T) {
		e := newEnv(t)
		e.assignTask()
		e.mgr.setProc("s2:fleet-serve", "running", "./build/fleet serve")
		e.mgr.setProc("s2:make-build", "running", "make build")
		e.mgr.setProc("s3:fleet-serve", "running", "./build/fleet serve")
		code, j := e.act("s2", map[string]any{"action": "stop"})
		if code != 202 {
			t.Fatalf("stop: %d %v", code, j)
		}
		e.waitJob(j["id"].(string))
		if got := e.mgr.Calls(); len(got) != 3 || got[2] != "down hangar-test-s2" {
			t.Fatalf("calls = %v", got)
		}
	})
}

func TestReservationChangeAbortsJob(t *testing.T) {
	e := newEnv(t)
	e.assignTask()
	e.mgr.block["s2:make-deps"] = true
	_, j := e.act("s2", map[string]any{"action": "build"})
	e.mgr.started(t, "s2:make-deps")
	e.writeRes(fmt.Sprintf(`{"slots": {"s2": {"worktree": %q, "previous_worktree": %q, "ports": %s, "compose_project": "hangar-test-s2", "created_at": 2.5}}}`,
		e.task, e.prev, portsJSON(s2Ports())))
	e.mgr.exit("s2:make-deps", 0)
	got := e.waitJob(j["id"].(string))
	if got["state"] != stateFailed || !strings.Contains(got["error"].(string), "reservation changed") {
		t.Fatalf("job = %v", got)
	}
	if calls := e.mgr.Calls(); slices.Contains(calls, "start s2:make-generate") {
		t.Fatalf("calls = %v", calls)
	}
}

func TestCloseInterruptsJobs(t *testing.T) {
	e := newEnv(t)
	e.assignTask()
	e.mgr.block["s2:make-deps"] = true
	_, j := e.act("s2", map[string]any{"action": "build"})
	e.mgr.started(t, "s2:make-deps")
	e.api.Close()
	if got := e.mgr.Calls(); !slices.Equal(got, []string{"start s2:make-deps", "stop s2:make-deps"}) {
		t.Fatalf("calls = %v", got)
	}
	_, got := e.do("GET", "/v1/jobs/"+j["id"].(string), nil)
	if got["state"] != stateInterrupted {
		t.Fatalf("job = %v", got)
	}
	if _, held := e.mgr.HeldBy("s2"); held {
		t.Fatal("slot still held")
	}
	code, out := e.act("s2", map[string]any{"action": "build"})
	expect(t, code, out, 503, "shutting down")
}

func TestJobLogsAreBounded(t *testing.T) {
	e := newEnv(t)
	e.assignTask()
	var ids []string
	for range 2 {
		code, j := e.act("s2", map[string]any{"action": "build"})
		if code != 202 {
			t.Fatalf("build: %d %v", code, j)
		}
		e.waitJob(j["id"].(string))
		ids = append(ids, j["id"].(string))
		time.Sleep(3 * time.Millisecond)
	}
	for _, id := range ids {
		_, w := e.do("GET", "/v1/jobs/"+id+"/log?since_ms=0", nil)
		if n := len(w["entries"].([]any)); n != 3 {
			t.Fatalf("job %s has %d lines: %v", id, n, w)
		}
	}
}

func TestListSlotsOmitsSecrets(t *testing.T) {
	e := newEnv(t)
	e.mgr.setProc("s2:fleet-serve", "running", "./build/fleet serve --secret CMDSECRET")
	code, out := e.do("GET", "/v1/slots", nil)
	if code != 200 {
		t.Fatalf("code %d", code)
	}
	b, _ := json.Marshal(out)
	for _, s := range []string{"SECRETVALUE", "FLEET_SECRET", "CMDSECRET", "recent"} {
		if bytes.Contains(b, []byte(s)) {
			t.Fatalf("slots response leaks %q: %s", s, b)
		}
	}
	if !bytes.Contains(b, []byte(`"reserved_by":"`+e.task+`"`)) || !bytes.Contains(b, []byte(`"state":"running"`)) {
		t.Fatalf("slots response = %s", b)
	}
}

func TestSlotUsage(t *testing.T) {
	e := newEnv(t)
	e.assignTask()
	e.mgr.block["s2:make-deps"] = true
	_, j := e.act("s2", map[string]any{"action": "build"})
	e.mgr.started(t, "s2:make-deps")
	u := e.api.SlotUsage()
	if u["s2"].ReservedBy == nil || *u["s2"].ReservedBy != e.task || u["s2"].Job == nil ||
		u["s2"].Job.ID != j["id"] || u["s2"].Job.Action != "build" {
		t.Fatalf("usage = %+v", u["s2"])
	}
	if u["s1"].ReservedBy != nil || u["s1"].Job != nil {
		t.Fatalf("s1 usage = %+v", u["s1"])
	}
	e.mgr.exit("s2:make-deps", 0)
}

func TestGitPrimaryCheckout(t *testing.T) {
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	root := t.TempDir()
	primary, wt := filepath.Join(root, "fleet"), filepath.Join(root, "task")
	if err := os.Mkdir(primary, 0o700); err != nil {
		t.Fatal(err)
	}
	git(primary, "init", "-q")
	git(primary, "commit", "-q", "--allow-empty", "-m", "x")
	git(primary, "worktree", "add", "-q", wt)

	for _, dir := range []string{primary, wt} {
		got, err := gitPrimaryCheckout(dir)
		if err != nil || !samePath(got, primary) {
			t.Fatalf("%s: got %q, %v", dir, got, err)
		}
	}
	sub := filepath.Join(wt, "sub")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := gitPrimaryCheckout(sub); err == nil {
		t.Fatal("subdirectory accepted as a worktree")
	}
	if _, err := gitPrimaryCheckout(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing dir accepted")
	}
}
