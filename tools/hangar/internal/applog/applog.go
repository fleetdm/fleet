// Package applog is Hangar's own diagnostic log, as opposed to the
// per-channel logs in internal/processes, which capture the output of the
// child processes Hangar spawns. This one records what Hangar itself did:
// startup, the macOS lifecycle events it saw, how it exited — and, most
// importantly, whatever the Go runtime prints on its way out when it doesn't
// exit on purpose. Everything lands in <logDir>/hangar.log, next to the
// channel logs.
//
// # Why file descriptor 2 gets redirected
//
// Pointing slog at a file is not enough. An unrecoverable runtime failure
// ("fatal error: invalid pointer found on stack", a cgo pointer violation, a
// nil map write) prints its message and goroutine dump straight to fd 2 and
// then exits: no deferred function, no recover(), and no slog handler ever
// runs. A packaged .app launched from Finder has fd 2 on /dev/null, so those
// traces have been going in the bin — which is why Hangar vanishing while
// nobody was looking has left no evidence at all. Redirecting fd 2 into the
// log file is the only way to keep them.
//
// That also rules out the tempting variant of piping fd 2 through a reader
// goroutine so the output can be teed to a terminal: a fatal error stops the
// world, so no goroutine of ours is ever scheduled to drain that pipe and the
// trace dies with the process. The write has to reach the file directly.
//
// # Detecting a death nobody witnessed
//
// A crash that takes the whole process out can't log its own conclusion, and
// SIGKILL (macOS jetsam, `kill -9`) doesn't even get to print a trace. So each
// session also keeps a small liveness marker on disk, refreshed on a timer and
// stamped with a reason when the app exits deliberately. Finding a marker with
// no recorded exit at the next launch means the previous session died; its
// last refresh says when.
//
// macOS-only, like the rest of Hangar (see internal/paths).
package applog

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/buildinfo"
)

const (
	// FileName is the app log's basename inside the log directory, alongside
	// the per-channel <channel>.log files.
	FileName = "hangar.log"

	// markerName holds the current session's liveness record. See the package
	// comment: its presence with no recorded exit at the next launch is how a
	// crash is distinguished from a quit.
	markerName = "last-session.json"

	// maxBytes caps hangar.log. Rotation happens only at startup (see rotate):
	// fd 2 points at the open file for the whole session, so it can't be
	// swapped out mid-run without the runtime writing its crash dump into a
	// file nobody will look at. One previous generation is kept as .1,
	// matching the channel logs.
	maxBytes = 8 << 20

	// tickInterval is how often the liveness marker is refreshed. It sets the
	// precision of "when did it die", so it's short; every logEveryTicks-th
	// tick also writes a heartbeat line, which is the durable record once the
	// marker has been overwritten by a later session.
	tickInterval  = 30 * time.Second
	logEveryTicks = 10
)

// PathFor returns the app log's path inside logDir.
func PathFor(logDir string) string { return filepath.Join(logDir, FileName) }

// sessionRecord is the on-disk liveness marker for one run.
type sessionRecord struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
	// LastAlive is refreshed every tickInterval. On a session that died, it's
	// the last moment Hangar is known to have been running.
	LastAlive time.Time `json:"last_alive"`
	// Exit is empty while the session is live and set to a short reason
	// ("user quit", "signal: terminated") on the way out. Still empty at the
	// next launch means the session never got to record one.
	Exit string `json:"exit"`
}

// Session is a live logging session. Its methods are safe on a nil receiver
// and on a session whose log file couldn't be opened, so callers never have to
// guard: logging must not be able to take the app down with it.
type Session struct {
	logPath    string
	markerPath string
	startedAt  time.Time
	file       *os.File

	stopOnce sync.Once
	stop     chan struct{}
	warnOnce sync.Once

	// markerMu serializes marker writes, and exitRecorded latches once a
	// terminal reason has been written. Closing s.stop doesn't stop a tick
	// that has already been selected, so without this a heartbeat landing in
	// the same instant as Close could rewrite the marker with an empty exit
	// after Close recorded the real one — turning a clean quit into a crash
	// report at the next launch, which is the exact signal this file exists
	// to get right.
	markerMu     sync.Mutex
	exitRecorded bool

	mu    sync.Mutex
	stats func() []any
}

// Setup installs the app log in logDir and returns the live session. It never
// fails in a way that should stop the app: if the log file can't be opened it
// leaves logging on stderr, says why, and returns a session whose methods are
// no-ops.
func Setup(logDir string) *Session {
	s := &Session{
		logPath:    PathFor(logDir),
		markerPath: filepath.Join(logDir, markerName),
		startedAt:  time.Now(),
		stop:       make(chan struct{}),
	}

	// Read the previous session's marker before this session overwrites it.
	prev, prevErr := readMarker(s.markerPath)

	rotate(s.logPath, maxBytes)

	// Deliberately unbuffered: a buffered writer would hold the last few lines
	// before a crash — exactly the ones worth having — in a buffer that never
	// gets flushed.
	f, openErr := os.OpenFile(s.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	var redirectErr error
	if openErr == nil {
		s.file = f
		// Keep whatever stderr was before the redirect so `task dev` still
		// prints to the terminal. Runtime crash output can't be teed this way
		// (see the package comment) and goes to the file only.
		w := io.Writer(f)
		if orig, err := dupStderr(); err == nil {
			w = io.MultiWriter(f, orig)
		}
		redirectErr = redirectStderr(f)
		slog.SetDefault(slog.New(slog.NewTextHandler(w, handlerOptions())))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, handlerOptions())))
	}

	// A fatal runtime failure dumps every goroutine's stack only at traceback
	// "all"; the default prints just the offending one, which isn't enough to
	// tell a dead main thread from a genuine deadlock.
	debug.SetTraceback("all")

	// The build stamp goes in the first line of every session: a crash report
	// is only actionable if you know which build produced it.
	slog.Info("session start",
		"hangar", buildinfo.Current().Summary(),
		"pid", os.Getpid(), "log", s.logPath, "go", runtime.Version())
	if openErr != nil {
		slog.Error("app log unavailable, logging to stderr only",
			"path", s.logPath, "err", openErr)
	}
	if redirectErr != nil {
		slog.Warn("could not redirect stderr into the app log; crash output will be lost",
			"err", redirectErr)
	}
	s.reportPrevious(prev, prevErr)

	s.writeMarker("")
	go s.tick()
	s.watchSignals()
	return s
}

// SetStats registers a provider of extra key/value pairs to fold into each
// heartbeat. Hangar uses it to record which managed processes were running, so
// a silent death can be read against what it took down with it.
func (s *Session) SetStats(fn func() []any) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.stats = fn
	s.mu.Unlock()
}

// Close records why this session ended. On macOS the process is terminated
// inside AppKit on a normal quit, so this has to be called from the last Go
// code that runs (the ShouldQuit hook) rather than deferred. Repeat calls are
// ignored: the first reason is the real one.
func (s *Session) Close(reason string) {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stop)
		slog.Info("session end",
			"reason", reason,
			"uptime", time.Since(s.startedAt).Round(time.Second).String())
		s.writeMarker(reason)
		if s.file != nil {
			_ = s.file.Sync()
		}
	})
}

// reportPrevious writes the verdict on the previous session into this one's
// log. This is the line that turns "Hangar seems to close itself sometimes"
// into a timestamp.
func (s *Session) reportPrevious(prev *sessionRecord, err error) {
	switch {
	case errors.Is(err, os.ErrNotExist):
		// First run after an install, or the marker was cleared out.
	case err != nil:
		slog.Warn("could not read the previous session marker",
			"path", s.markerPath, "err", err)
	case prev.Exit != "":
		slog.Info("previous session exited normally",
			"reason", prev.Exit, "pid", prev.PID,
			"ran_for", prev.LastAlive.Sub(prev.StartedAt).Round(time.Second).String())
	default:
		slog.Warn("previous session never recorded an exit: it crashed or was killed",
			"pid", prev.PID,
			"started", prev.StartedAt.Format(time.RFC3339),
			"last_alive", prev.LastAlive.Format(time.RFC3339),
			"ran_for", prev.LastAlive.Sub(prev.StartedAt).Round(time.Second).String(),
			"hint", "any crash output is in this file just above this line")
	}
}

func (s *Session) tick() {
	t := time.NewTicker(tickInterval)
	defer t.Stop()
	for n := 1; ; n++ {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.writeMarker("")
			if n%logEveryTicks == 0 {
				s.heartbeat()
			}
		}
	}
}

// heartbeat is the "still alive at this timestamp" line. The goroutine count
// is here because a Wails main thread that has died while the process lives on
// shows up as goroutines piling up with nothing draining them.
func (s *Session) heartbeat() {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	args := []any{
		"uptime", time.Since(s.startedAt).Round(time.Second).String(),
		"goroutines", runtime.NumGoroutine(),
		"heap_mb", mem.HeapAlloc >> 20,
	}
	s.mu.Lock()
	fn := s.stats
	s.mu.Unlock()
	if fn != nil {
		args = append(args, fn()...)
	}
	slog.Info("heartbeat", args...)
}

// watchSignals annotates the exit when Hangar is signalled. Without it a
// SIGTERM (logout, `killall`, the dev script's teardown) is indistinguishable
// from a crash in the marker. It deliberately doesn't attempt a graceful
// shutdown — that would change how Hangar exits — and re-raises the signal so
// the process still dies exactly as it would have.
func (s *Session) watchSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig, ok := <-ch
		if !ok {
			return
		}
		s.Close("signal: " + sig.String())
		// Stop handling it, then re-raise: the default disposition takes over
		// and the exit status stays the usual 128+signal.
		signal.Stop(ch)
		if sysSig, ok := sig.(syscall.Signal); ok {
			_ = syscall.Kill(os.Getpid(), sysSig)
		}
	}()
}

// writeMarker persists the liveness record. An empty exit is a routine
// liveness refresh; a non-empty one is terminal and can never be walked back
// by a later refresh (see markerMu).
func (s *Session) writeMarker(exit string) {
	if s.markerPath == "" {
		return
	}

	s.markerMu.Lock()
	defer s.markerMu.Unlock()
	if exit == "" && s.exitRecorded {
		return
	}
	if exit != "" {
		s.exitRecorded = true
	}

	rec := sessionRecord{
		PID:       os.Getpid(),
		StartedAt: s.startedAt,
		LastAlive: time.Now(),
		Exit:      exit,
	}
	b, err := json.Marshal(rec)
	if err == nil {
		err = os.WriteFile(s.markerPath, b, 0o644)
	}
	if err != nil {
		// Worth one line, not one per tick.
		s.warnOnce.Do(func() {
			slog.Warn("could not write the session marker; a crash won't be detectable",
				"path", s.markerPath, "err", err)
		})
	}
}

func readMarker(path string) (*sessionRecord, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec sessionRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// rotate moves an oversized log aside, keeping one generation, before it's
// opened. Startup only — see maxBytes.
func rotate(path string, max int64) {
	fi, err := os.Stat(path)
	if err != nil || fi.Size() < max {
		return
	}
	_ = os.Rename(path, path+".1")
}

// redirectStderr points fd 2 at f, so everything the Go runtime prints on a
// fatal error — and anything the webview's native side writes — lands in the
// log instead of /dev/null.
func redirectStderr(f *os.File) error {
	return syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
}

// dupStderr copies the current fd 2 so ordinary log lines can still reach a
// terminal after the redirect.
func dupStderr() (*os.File, error) {
	fd, err := syscall.Dup(int(os.Stderr.Fd()))
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "stderr"), nil
}

// handlerOptions builds the slog options. HANGAR_LOG_LEVEL raises or lowers
// the level for a run ("debug" when asking someone to reproduce something).
func handlerOptions() *slog.HandlerOptions {
	level := slog.LevelInfo
	if err := level.UnmarshalText([]byte(strings.TrimSpace(os.Getenv("HANGAR_LOG_LEVEL")))); err != nil {
		level = slog.LevelInfo
	}
	return &slog.HandlerOptions{Level: level}
}
