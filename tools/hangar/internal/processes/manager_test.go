package processes

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type recorder struct {
	mu     sync.Mutex
	events []recordedEvent
}

type recordedEvent struct {
	name string
	data any
}

func (r *recorder) Emit(name string, data any) {
	r.mu.Lock()
	r.events = append(r.events, recordedEvent{name, data})
	r.mu.Unlock()
}

func (r *recorder) states() []ProcEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []ProcEvent
	for _, e := range r.events {
		if e.name == "proc:state" {
			out = append(out, e.data.(ProcEvent))
		}
	}
	return out
}

func (r *recorder) logLines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, e := range r.events {
		if e.name == "proc:log" {
			out = append(out, e.data.(LogLine).Line)
		}
	}
	return out
}

func waitForState(t *testing.T, m *Manager, id, want string, timeout time.Duration) ProcInfo {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, p := range m.ListProcesses() {
			if p.ID == id && p.State == want {
				return p
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("process %s did not reach state %q within %s", id, want, timeout)
	return ProcInfo{}
}

func TestSpawnCaptureAndComplete(t *testing.T) {
	rec := &recorder{}
	m := New(t.TempDir(), t.TempDir(), rec)

	err := m.Start("job", StartArgs{
		Label:      "job",
		Cwd:        t.TempDir(),
		Program:    "sh",
		Args:       []string{"-c", "echo hello; echo oops 1>&2"},
		LogChannel: "job-chan",
	})
	if err != nil {
		t.Fatal(err)
	}

	done := waitForState(t, m, "job", "done", 5*time.Second)
	if done.ExitCode == nil || *done.ExitCode != 0 {
		t.Errorf("exit code = %v, want 0", done.ExitCode)
	}

	// proc:log carried both streams.
	lines := strings.Join(rec.logLines(), "\n")
	if !strings.Contains(lines, "hello") || !strings.Contains(lines, "oops") {
		t.Errorf("proc:log missing stdout/stderr: %q", lines)
	}
	// proc:state went running -> done.
	st := rec.states()
	if len(st) < 2 || st[0].State != "running" || st[len(st)-1].State != "done" {
		t.Errorf("state sequence = %+v", st)
	}

	// Structured store has the lines.
	w := m.ReadLogWindow("job-chan", 0, []string{"debug", "info", "warn", "error"}, nil, nil)
	if w.TotalInWindow != 2 {
		t.Errorf("log window total = %d, want 2", w.TotalInWindow)
	}
	// On-disk log written.
	if _, err := os.Stat(filepath.Join(m.logDir, "job-chan.log")); err != nil {
		t.Errorf("disk log not written: %v", err)
	}
}

func TestStartAlreadyRunning(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	if err := m.Start("s", StartArgs{Program: "sleep", Args: []string{"30"}}); err != nil {
		t.Fatal(err)
	}
	defer m.Stop("s")
	waitForState(t, m, "s", "running", 3*time.Second)
	if err := m.Start("s", StartArgs{Program: "sleep", Args: []string{"30"}}); err == nil {
		t.Error("second Start of same id should error")
	}
}

func TestStopIsGracefulDone(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	if err := m.Start("s", StartArgs{Program: "sleep", Args: []string{"30"}}); err != nil {
		t.Fatal(err)
	}
	waitForState(t, m, "s", "running", 3*time.Second)

	if err := m.Stop("s"); err != nil {
		t.Fatal(err)
	}
	done := waitForState(t, m, "s", "done", 5*time.Second)
	if !done.WasUserStopped {
		t.Error("user-stopped process should have was_user_stopped=true")
	}
	// pid removed from tracking.
	m.stateMu.Lock()
	_, ok := m.live["s"]
	m.stateMu.Unlock()
	if ok {
		t.Error("pid should be cleared after stop")
	}
}

func TestStopReturnsPromptlyOnFastExit(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	if err := m.Start("s", StartArgs{Program: "sleep", Args: []string{"60"}}); err != nil {
		t.Fatal(err)
	}
	waitForState(t, m, "s", "running", 3*time.Second)

	// `sleep` dies on SIGTERM almost immediately. With the context lifecycle,
	// Stop returns as soon as it terminates — not after the fixed 800ms grace.
	start := time.Now()
	if err := m.Stop("s"); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Errorf("Stop took %v; expected prompt return on fast exit (well under the 800ms grace)", elapsed)
	}
}

func TestFailedExitCode(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	if err := m.Start("f", StartArgs{Program: "sh", Args: []string{"-c", "exit 7"}}); err != nil {
		t.Fatal(err)
	}
	p := waitForState(t, m, "f", "failed", 5*time.Second)
	if p.ExitCode == nil || *p.ExitCode != 7 {
		t.Errorf("exit code = %v, want 7", p.ExitCode)
	}
	if !hasLine(p.RecentLog, "[exit: code 7]") {
		t.Errorf("missing synth exit line: %v", p.RecentLog)
	}
}

func TestCrashBySignalIsFailed(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	if err := m.Start("c", StartArgs{Program: "sleep", Args: []string{"30"}}); err != nil {
		t.Fatal(err)
	}
	waitForState(t, m, "c", "running", 3*time.Second)

	// Kill directly (NOT via Stop) so state stays "running" — simulates a crash.
	m.stateMu.Lock()
	pid := m.live["c"].pid
	m.stateMu.Unlock()
	_ = syscall.Kill(pid, syscall.SIGKILL)

	p := waitForState(t, m, "c", "failed", 5*time.Second)
	if p.ExitSignal == nil || *p.ExitSignal != 9 {
		t.Errorf("exit signal = %v, want 9", p.ExitSignal)
	}
	if !hasLine(p.RecentLog, "SIGKILL") {
		t.Errorf("missing signal synth line: %v", p.RecentLog)
	}
}

func TestForgetRefusesRunning(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	m.Start("r", StartArgs{Program: "sleep", Args: []string{"30"}})
	waitForState(t, m, "r", "running", 3*time.Second)
	if err := m.Forget("r"); err == nil {
		t.Error("Forget should refuse a running process")
	}
	m.Stop("r")
	waitForState(t, m, "r", "done", 5*time.Second)
	if err := m.Forget("r"); err != nil {
		t.Errorf("Forget after stop should succeed: %v", err)
	}
	for _, p := range m.ListProcesses() {
		if p.ID == "r" {
			t.Error("forgotten process should not be listed")
		}
	}
}

func TestClearAndSnapshot(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	m.Start("j", StartArgs{Program: "sh", Args: []string{"-c", "echo one; echo two"}, LogChannel: "ch"})
	waitForState(t, m, "j", "done", 5*time.Second)

	// SaveLogSnapshot rejects traversal, accepts a basename.
	if _, err := m.SaveLogSnapshot("../escape.txt", "x"); err == nil {
		t.Error("snapshot should reject path traversal")
	}
	path, err := m.SaveLogSnapshot("snap.txt", "captured")
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != "captured" {
		t.Errorf("snapshot content wrong: %q", b)
	}

	// Clear empties the store.
	if err := m.ClearLogChannel("ch"); err != nil {
		t.Fatal(err)
	}
	w := m.ReadLogWindow("ch", 0, []string{"debug", "info", "warn", "error"}, nil, nil)
	if w.TotalInWindow != 0 {
		t.Errorf("after clear, window should be empty, got %d", w.TotalInWindow)
	}
}

func hasLine(lines []string, substr string) bool {
	for _, l := range lines {
		if strings.Contains(l, substr) {
			return true
		}
	}
	return false
}

func TestConcurrentStartSpawnsOnce(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	const n = 16
	var ready sync.WaitGroup
	ready.Add(n)
	gate := make(chan struct{})
	errs := make(chan error, n)
	for range n {
		go func() {
			ready.Done()
			<-gate
			_, err := m.StartInvocation("same", sleeper(""))
			errs <- err
		}()
	}
	ready.Wait()
	close(gate)
	started := 0
	for range n {
		if <-errs == nil {
			started++
		}
	}
	defer m.Stop("same")
	if started != 1 {
		t.Fatalf("%d concurrent starts of one id succeeded, want 1", started)
	}
	if recs := readPidRecords(m.dataDir); len(recs) != 1 {
		t.Errorf("running.json has %d records, want 1", len(recs))
	}
}

func TestSpawnFailureReleasesID(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	if _, err := m.StartInvocation("x", StartArgs{Program: "/nonexistent/hangar-test-binary"}); err == nil {
		t.Fatal("spawn of a missing binary should fail")
	}
	inv, err := m.StartInvocation("x", StartArgs{Program: "sh", Args: []string{"-c", "exit 3"}})
	if err != nil {
		t.Fatalf("id still reserved after a failed spawn: %v", err)
	}
	if code, err := m.Wait(t.Context(), inv); err != nil || code != 3 {
		t.Errorf("Wait = %d, %v; want 3", code, err)
	}
}

func TestWaitExitCodes(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	ok, err := m.StartInvocation("ok", StartArgs{Program: "sh", Args: []string{"-c", "exit 0"}})
	if err != nil {
		t.Fatal(err)
	}
	if code, err := m.Wait(t.Context(), ok); err != nil || code != 0 {
		t.Errorf("clean exit: Wait = %d, %v; want 0", code, err)
	}

	stopped, err := m.StartInvocation("stopped", StartArgs{Program: "sh", Args: []string{"-c", "trap 'exit 0' TERM; sleep 30 & wait"}})
	if err != nil {
		t.Fatal(err)
	}
	waitForState(t, m, "stopped", "running", 3*time.Second)
	if err := m.Stop("stopped"); err != nil {
		t.Fatal(err)
	}
	if code, err := m.Wait(t.Context(), stopped); err != nil || code != -1 {
		t.Errorf("stopped from outside: Wait = %d, %v; want -1 even though it exited 0", code, err)
	}

	killed, err := m.StartInvocation("killed", sleeper(""))
	if err != nil {
		t.Fatal(err)
	}
	_ = syscall.Kill(killed.inv.pid, syscall.SIGKILL)
	if code, err := m.Wait(t.Context(), killed); err != nil || code != -1 {
		t.Errorf("signaled: Wait = %d, %v; want -1", code, err)
	}
}

func TestStaleFinalizerKeepsNewerInvocation(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	old, err := m.StartInvocation("x", StartArgs{Program: "sh", Args: []string{"-c", "exit 0"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Wait(t.Context(), old); err != nil {
		t.Fatal(err)
	}
	cur, err := m.StartInvocation("x", sleeper(""))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Stop("x")

	m.finish(old.inv, "failed", 9, nil, nil, "[stale]")

	m.stateMu.Lock()
	live, state := m.live["x"], m.procs["x"].State
	m.stateMu.Unlock()
	if live != cur.inv || state != "running" {
		t.Fatalf("stale finalizer touched the newer invocation: live=%v state=%q", live == cur.inv, state)
	}
	if recs := readPidRecords(m.dataDir); len(recs) != 1 || recs[0].PID != cur.inv.pid {
		t.Errorf("running.json = %+v, want the newer pid %d", recs, cur.inv.pid)
	}
}

func TestConcurrentStartAndExitRunningJSON(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	var short []Invocation
	for i := range 20 {
		wg.Go(func() {
			inv, err := m.StartInvocation(fmt.Sprintf("short-%d", i), StartArgs{Program: "sh", Args: []string{"-c", "exit 0"}})
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			short = append(short, inv)
			mu.Unlock()
		})
	}
	long := []string{"long-0", "long-1", "long-2", "long-3"}
	for _, id := range long {
		wg.Go(func() {
			if err := m.Start(id, sleeper("")); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	for _, inv := range short {
		if _, err := m.Wait(t.Context(), inv); err != nil {
			t.Fatal(err)
		}
	}

	var got []string
	for _, r := range readPidRecords(m.dataDir) {
		got = append(got, r.ID)
	}
	slices.Sort(got)
	if !slices.Equal(got, long) {
		t.Errorf("running.json ids = %v, want %v", got, long)
	}

	for _, id := range long {
		m.Stop(id)
		waitForState(t, m, id, "done", 3*time.Second)
	}
	if _, err := os.Stat(pidFilePath(m.dataDir)); !os.IsNotExist(err) {
		t.Errorf("running.json should be gone once nothing runs: %v", err)
	}
}

func TestStopComposeUpWaitsForGroupBeforeDown(t *testing.T) {
	// up leaves a TERM-ignoring child in its group; down records whether
	// that child is still alive when it runs.
	dir := fakeDocker(t, `d=$(dirname "$0")
case " $* " in
*" up "*) (trap '' TERM; exec sleep 30) & echo $! > "$d/child.pid"; wait ;;
*" down "*) if kill -0 "$(cat "$d/child.pid")" 2>/dev/null; then echo alive > "$d/down.txt"; else echo gone > "$d/down.txt"; fi ;;
esac
`)
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	inv, err := m.StartInvocation("s2:docker-compose-up", StartArgs{
		Label: "docker compose up", Cwd: dir, Program: "docker", Args: []string{"compose", "-p", "p2", "up"},
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForFile(t, filepath.Join(dir, "child.pid"))

	if err := m.Stop("s2:docker-compose-up"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "down.txt"))
	if err != nil {
		t.Fatalf("compose down didn't run: %v", err)
	}
	if got := strings.TrimSpace(string(b)); got != "gone" {
		t.Errorf("compose down ran while the up group was %s", got)
	}
	if code, err := m.Wait(t.Context(), inv); err != nil || code != -1 {
		t.Errorf("Wait = %d, %v; want -1 for a stopped up", code, err)
	}
	if p := waitForState(t, m, "s2:docker-compose-up", "done", time.Second); !p.WasUserStopped {
		t.Error("stopped compose row should be was_user_stopped")
	}
}

func TestReadLogWindowScrubbed(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	for _, e := range []LogEntry{
		{TsMS: 50, Message: "before the job", Channel: "agent-s2"},
		{TsMS: 100, Message: "curl -H 'Authorization: Bearer abc.def.ghi' /api", Channel: "agent-s2"},
		{TsMS: 150, Message: "other job", Channel: "agent-s3"},
		{TsMS: 300, Message: "after the job", Channel: "agent-s2"},
	} {
		m.pushToLogStore(e)
	}

	w := m.ReadLogWindowScrubbed("agent-s2", 100, 200, 0)
	if len(w.Entries) != 1 {
		t.Fatalf("entries = %+v, want only the in-window agent-s2 line", w.Entries)
	}
	msg := w.Entries[0].Message
	if strings.Contains(msg, "abc.def.ghi") || !strings.Contains(msg, "Bearer [redacted]") {
		t.Errorf("token not scrubbed: %q", msg)
	}
	if raw := m.ReadLogWindow("agent-s2", 100, []string{"info"}, nil, nil); !strings.Contains(raw.Entries[0].Message, "abc.def.ghi") {
		t.Error("scrubbing mutated the stored entry")
	}
	if w := m.ReadLogWindowScrubbed("agent-s2", 0, 0, 1); len(w.Entries) != 1 || w.Entries[0].Message != "after the job" {
		t.Errorf("untilMS 0 with maxLines 1 = %+v, want the newest line", w.Entries)
	}
}

func TestStopKillsTermIgnoringGrandchild(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	// Once the leader is reaped its pgid may be reused, so a real signal to
	// the group then could hit an unrelated one.
	var mu sync.Mutex
	var late []string
	m.kill = func(pid int, sig syscall.Signal) error {
		if sig != 0 && pid < 0 {
			if out, _ := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(-pid)).Output(); len(strings.TrimSpace(string(out))) == 0 {
				mu.Lock()
				late = append(late, fmt.Sprintf("%v to group %d", sig, -pid))
				mu.Unlock()
			}
		}
		return syscall.Kill(pid, sig)
	}
	// The grandchild ignores TERM and drops the pipes, so the leader can exit
	// and finalize before the KILL escalation.
	inv, err := m.StartInvocation("g", StartArgs{Program: "sh", Args: []string{
		"-c", `(trap '' TERM; exec sleep 30) >/dev/null 2>&1 </dev/null & echo $! > "$1.tmp" && mv "$1.tmp" "$1"; wait`, "sh", pidFile,
	}})
	if err != nil {
		t.Fatal(err)
	}
	waitForFile(t, pidFile)
	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	child, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	t.Cleanup(func() { _ = syscall.Kill(child, syscall.SIGKILL) })

	if err := m.Stop("g"); err != nil {
		t.Fatal(err)
	}
	if code, err := m.Wait(t.Context(), inv); err != nil || code != -1 {
		t.Errorf("Wait = %d, %v; want -1", code, err)
	}
	if syscall.Kill(child, 0) == nil {
		t.Errorf("TERM-ignoring grandchild %d survived Stop and Wait", child)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(late) > 0 {
		t.Errorf("signaled a group after reaping its leader: %v", late)
	}
}

func TestNoGroupSignalAfterReapWithoutExitWait(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	waited := make(chan struct{})
	m.waitExit = func(int) bool { close(waited); return false } // as when kqueue can't be set up
	var late []string
	m.kill = func(pid int, sig syscall.Signal) error {
		if sig != 0 && pid < 0 && !listed(-pid) {
			late = append(late, fmt.Sprintf("%v to group %d", sig, -pid))
		}
		return syscall.Kill(pid, sig)
	}
	// The leader drops its pipes, so the finalizer reaches cmd.Wait while it
	// still runs.
	inv, err := m.StartInvocation("x", StartArgs{Program: "sh", Args: []string{"-c", "exec >/dev/null 2>&1; sleep 0.3"}})
	if err != nil {
		t.Fatal(err)
	}
	<-waited
	time.Sleep(50 * time.Millisecond) // let the finalizer into cmd.Wait

	// Holding stateMu keeps the finalizer from recording anything after the
	// reap, and stands in for a Stop or quit signaling in that window.
	m.stateMu.Lock()
	for deadline := time.Now().Add(3 * time.Second); listed(inv.inv.pid) && time.Now().Before(deadline); {
		time.Sleep(20 * time.Millisecond)
	}
	m.signalLocked(inv.inv, syscall.SIGKILL)
	m.stateMu.Unlock()

	if _, err := m.Wait(t.Context(), inv); err != nil {
		t.Fatal(err)
	}
	if len(late) > 0 {
		t.Errorf("signaled a group after reaping its leader: %v", late)
	}
}

// listed reports whether ps still lists pid, alive or as an unreaped zombie.
func listed(pid int) bool {
	out, _ := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
	return len(strings.TrimSpace(string(out))) > 0
}
