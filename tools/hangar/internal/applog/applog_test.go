package applog

import (
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// isolate saves fd 2 and the default logger, since Setup replaces both
// process-wide, and restores them when the test ends.
func isolate(t *testing.T) {
	t.Helper()
	saved, err := syscall.Dup(int(os.Stderr.Fd()))
	if err != nil {
		t.Fatalf("dup stderr: %v", err)
	}
	logger := slog.Default()
	t.Cleanup(func() {
		_ = syscall.Dup2(saved, int(os.Stderr.Fd()))
		_ = syscall.Close(saved)
		slog.SetDefault(logger)
	})
}

func readLog(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(PathFor(dir))
	if err != nil {
		t.Fatalf("read app log: %v", err)
	}
	return string(b)
}

func readSessionMarker(t *testing.T, dir string) sessionRecord {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, markerName))
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	var rec sessionRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatalf("unmarshal marker: %v", err)
	}
	return rec
}

// The point of the package: output written to fd 2 by something that isn't
// slog — which is how the Go runtime reports a fatal error, moments before the
// process dies — has to end up in the log file.
func TestSetupCapturesRawStderr(t *testing.T) {
	isolate(t)
	dir := t.TempDir()

	s := Setup(dir)
	defer s.Close("test")

	if _, err := os.Stderr.WriteString("fatal error: invalid pointer found on stack\n"); err != nil {
		t.Fatalf("write to stderr: %v", err)
	}

	body := readLog(t, dir)
	if !strings.Contains(body, "fatal error: invalid pointer found on stack") {
		t.Errorf("raw stderr not captured in the app log:\n%s", body)
	}
	if !strings.Contains(body, "session start") {
		t.Errorf("no session start line:\n%s", body)
	}
}

// The same thing again, but for real: a crash can't be deferred around or
// recovered from, so it's exercised in a subprocess that genuinely dies. The
// dump has to reach the log file — and only the log file, since the stderr it
// inherited is exactly the /dev/null a Finder-launched .app gets.
func TestCrashOutputLandsInTheLogAndNowhereElse(t *testing.T) {
	if dir := os.Getenv("HANGAR_TEST_CRASH_DIR"); dir != "" {
		Setup(dir)
		// A couple of parked goroutines, to show the dump covers all of them
		// and not just the one that died.
		for i := 0; i < 2; i++ {
			go func() { select {} }()
		}
		panic("simulated crash")
	}

	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run", "^"+t.Name()+"$")
	cmd.Env = append(os.Environ(), "HANGAR_TEST_CRASH_DIR="+dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("subprocess was supposed to crash; output:\n%s", out)
	}

	body := readLog(t, dir)
	for _, want := range []string{"session start", "panic: simulated crash", "goroutine "} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in the app log:\n%s", want, body)
		}
	}
	// debug.SetTraceback("all"): every goroutine, not just the panicking one.
	if n := strings.Count(body, "goroutine "); n < 3 {
		t.Errorf("dump covers %d goroutines, want every one of them:\n%s", n, body)
	}
	if strings.Contains(string(out), "panic: simulated crash") {
		t.Errorf("crash also went to the inherited stderr, where a packaged app would lose it:\n%s", out)
	}
}

func TestSessionMarkerRecordsExit(t *testing.T) {
	isolate(t)
	dir := t.TempDir()

	s := Setup(dir)

	// While the session is live the marker carries no exit, which is what
	// makes an abrupt death detectable at the next launch.
	if live := readSessionMarker(t, dir); live.Exit != "" {
		t.Errorf("live marker already has an exit: %q", live.Exit)
	} else if live.PID != os.Getpid() {
		t.Errorf("marker pid = %d, want %d", live.PID, os.Getpid())
	}

	s.Close("user quit")

	closed := readSessionMarker(t, dir)
	if closed.Exit != "user quit" {
		t.Errorf("marker exit = %q, want %q", closed.Exit, "user quit")
	}
	if !strings.Contains(readLog(t, dir), "session end") {
		t.Error("no session end line after Close")
	}
}

func TestCloseIsIdempotentAndKeepsFirstReason(t *testing.T) {
	isolate(t)
	dir := t.TempDir()

	s := Setup(dir)
	s.Close("signal: terminated")
	s.Close("user quit")

	if got := readSessionMarker(t, dir).Exit; got != "signal: terminated" {
		t.Errorf("marker exit = %q, want the first reason", got)
	}
}

// Closing s.stop doesn't unwind a tick that has already been selected, so a
// heartbeat can still write its liveness refresh after Close has recorded the
// real reason. If that refresh won, the next launch would report a clean quit
// as a crash. Run with -race.
func TestConcurrentHeartbeatCannotEraseTheExitReason(t *testing.T) {
	isolate(t)
	dir := t.TempDir()

	s := Setup(dir)

	// Hammer the marker the way the ticker does, across the Close.
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				s.writeMarker("")
			}
		}()
	}
	s.Close("user quit")
	wg.Wait()

	if got := readSessionMarker(t, dir).Exit; got != "user quit" {
		t.Errorf("marker exit = %q after concurrent refreshes, want %q", got, "user quit")
	}
}

func TestSetupReportsCrashedPreviousSession(t *testing.T) {
	isolate(t)
	dir := t.TempDir()

	started := time.Now().Add(-90 * time.Minute)
	writeTestMarker(t, dir, sessionRecord{
		PID:       4242,
		StartedAt: started,
		LastAlive: started.Add(time.Hour),
		// No Exit: the previous run never got to record one.
	})

	s := Setup(dir)
	defer s.Close("test")

	body := readLog(t, dir)
	if !strings.Contains(body, "it crashed or was killed") {
		t.Errorf("crashed previous session not reported:\n%s", body)
	}
	for _, want := range []string{"pid=4242", "ran_for=1h0m0s"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in the report:\n%s", want, body)
		}
	}
}

func TestSetupReportsCleanPreviousSession(t *testing.T) {
	isolate(t)
	dir := t.TempDir()

	started := time.Now().Add(-time.Hour)
	writeTestMarker(t, dir, sessionRecord{
		PID:       4242,
		StartedAt: started,
		LastAlive: started.Add(time.Minute),
		Exit:      "user quit",
	})

	s := Setup(dir)
	defer s.Close("test")

	body := readLog(t, dir)
	if !strings.Contains(body, "previous session exited normally") {
		t.Errorf("clean exit not reported:\n%s", body)
	}
	if strings.Contains(body, "crashed or was killed") {
		t.Errorf("clean exit misreported as a crash:\n%s", body)
	}
}

// A fresh install has no marker, which is not something to warn about.
func TestSetupSaysNothingWithoutAPreviousSession(t *testing.T) {
	isolate(t)
	dir := t.TempDir()

	s := Setup(dir)
	defer s.Close("test")

	body := readLog(t, dir)
	if strings.Contains(body, "previous session") {
		t.Errorf("reported a previous session on a first run:\n%s", body)
	}
}

func TestHeartbeatIncludesRegisteredStats(t *testing.T) {
	isolate(t)
	dir := t.TempDir()

	s := Setup(dir)
	defer s.Close("test")

	s.SetStats(func() []any { return []any{"running_procs", "fleet-serve-s1,ngrok"} })
	s.heartbeat()

	body := readLog(t, dir)
	for _, want := range []string{"heartbeat", "goroutines=", "running_procs=fleet-serve-s1,ngrok"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in the heartbeat:\n%s", want, body)
		}
	}
}

func TestRotate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)

	if err := os.WriteFile(path, []byte("small"), 0o644); err != nil {
		t.Fatal(err)
	}
	rotate(path, 1024)
	if _, err := os.Stat(path + ".1"); !os.IsNotExist(err) {
		t.Error("rotated a file that was under the limit")
	}

	if err := os.WriteFile(path, []byte(strings.Repeat("x", 2048)), 0o644); err != nil {
		t.Fatal(err)
	}
	rotate(path, 1024)
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Errorf("oversized log not rotated: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("original log still in place after rotation")
	}
}

// Logging must never be able to take the app down, so a session that failed to
// open its file — or was never created — still answers every call.
func TestNilSessionIsSafe(t *testing.T) {
	var s *Session
	s.SetStats(func() []any { return nil })
	s.Close("user quit")
}

func TestHandlerOptionsLevel(t *testing.T) {
	for _, tc := range []struct {
		env  string
		want slog.Level
	}{
		{"", slog.LevelInfo},
		{"debug", slog.LevelDebug},
		{" DEBUG ", slog.LevelDebug},
		{"nonsense", slog.LevelInfo},
	} {
		t.Setenv("HANGAR_LOG_LEVEL", tc.env)
		if got := handlerOptions().Level.Level(); got != tc.want {
			t.Errorf("HANGAR_LOG_LEVEL=%q -> %v, want %v", tc.env, got, tc.want)
		}
	}
}

func writeTestMarker(t *testing.T, dir string, rec sessionRecord) {
	t.Helper()
	b, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, markerName), b, 0o644); err != nil {
		t.Fatal(err)
	}
}
