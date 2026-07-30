package processes

import (
	"os"
	"path/filepath"
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
	_, ok := m.pids["s"]
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
	pid := m.pids["c"]
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
