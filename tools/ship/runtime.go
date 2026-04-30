package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// insecureTLS skips cert verification — Fleet's dev server uses a
// self-signed cert, and we're connecting to localhost.
func insecureTLS() *tls.Config { return &tls.Config{InsecureSkipVerify: true} }

// -----------------------------------------------------------------------------
// Step list types — emitted as tea.Msg values during the start sequence and
// rendered by dashboardModel.
// -----------------------------------------------------------------------------

type stepKind int

const (
	stepDockerUp stepKind = iota
	stepMakeDeps
	stepMakeBuild
	stepGenerateDev
	stepPrepareDB
	stepServe
	stepNgrok
)

func (k stepKind) String() string {
	switch k {
	case stepDockerUp:
		return "start docker services"
	case stepMakeDeps:
		return "install dependencies"
	case stepMakeBuild:
		return "build fleet binary"
	case stepGenerateDev:
		return "start webpack dev mode"
	case stepPrepareDB:
		return "run migrations"
	case stepServe:
		return "start fleet server"
	case stepNgrok:
		return "open public tunnel"
	}
	return "?"
}

type stepStatus int

const (
	stepPending stepStatus = iota
	stepRunning
	stepDone
	stepFailed
	stepSkipped
)

// stepUpdateMsg is the per-step progress update.
type stepUpdateMsg struct {
	Kind    stepKind
	Status  stepStatus
	Detail  string        // "1.2s" when done, error string when failed
	Elapsed time.Duration // populated on done/failed
}

// logLineMsg wraps a logLine for delivery to the TUI as a tea.Msg.
type logLineMsg logLine

// runtimeReadyMsg fires once the start sequence has fully succeeded.
type runtimeReadyMsg struct{ NgrokURL string }

// runtimeFailedMsg fires if any step in the start sequence fails fatally.
type runtimeFailedMsg struct{ Err error }

// -----------------------------------------------------------------------------
// Runtime — owns the supervised processes and runs the start/stop sequences.
// -----------------------------------------------------------------------------

type runtimeOpts struct {
	cfg        Config
	privateKey string
	repoRoot   string
}

type engine struct {
	opts runtimeOpts

	// sink carries tea.Msg values back to the TUI: step updates, log
	// lines (already wrapped in logLineMsg), readiness, and failures.
	sink chan tea.Msg
	// logSink is what the proc supervisor writes raw log lines into;
	// a fan-in goroutine wraps them in logLineMsg and forwards to sink.
	logSink chan logLine

	mu      sync.Mutex
	fleet   *proc
	genDev  *proc
	ngrok   *proc
	started time.Time
}

func newEngine(opts runtimeOpts) *engine {
	e := &engine{
		opts:    opts,
		sink:    make(chan tea.Msg, 256),
		logSink: make(chan logLine, 256),
	}
	// Pump raw log lines from supervised processes into the TUI message
	// stream. This is done in a single goroutine so we get a stable
	// ordering even when multiple processes are emitting at once.
	go func() {
		for line := range e.logSink {
			select {
			case e.sink <- logLineMsg(line):
			default:
			}
		}
	}()
	return e
}

// listen returns a tea.Cmd that pulls the next message from the runtime's
// sink. The TUI re-issues this cmd after each received message to keep the
// stream flowing.
func (r *engine) listen() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-r.sink
		if !ok {
			return nil
		}
		return msg
	}
}

// -----------------------------------------------------------------------------
// Start sequence
// -----------------------------------------------------------------------------

// Start kicks off the full bring-up sequence in a goroutine and returns
// immediately. Progress flows back via the sink channel.
func (r *engine) Start(ctx context.Context) {
	go r.run(ctx)
}

func (r *engine) run(ctx context.Context) {
	type stepFn func(context.Context) error
	steps := []struct {
		kind stepKind
		fn   stepFn
	}{
		{stepDockerUp, r.stepDockerUp},
		{stepMakeDeps, r.stepMakeDeps},
		{stepMakeBuild, r.stepMakeBuild},
		{stepGenerateDev, r.stepGenerateDev},
		{stepPrepareDB, r.stepPrepareDB},
		{stepServe, r.stepServe},
		{stepNgrok, r.stepNgrok},
	}

	// Mark every step pending up front so the dashboard can render the full
	// list with "·" placeholders.
	for _, s := range steps {
		r.emit(stepUpdateMsg{Kind: s.kind, Status: stepPending})
	}

	for _, s := range steps {
		t0 := time.Now()
		r.emit(stepUpdateMsg{Kind: s.kind, Status: stepRunning})
		if err := s.fn(ctx); err != nil {
			r.emit(stepUpdateMsg{
				Kind: s.kind, Status: stepFailed,
				Detail: err.Error(), Elapsed: time.Since(t0),
			})
			r.emit(runtimeFailedMsg{Err: fmt.Errorf("%s: %w", s.kind, err)})
			return
		}
		r.emit(stepUpdateMsg{
			Kind: s.kind, Status: stepDone, Elapsed: time.Since(t0),
		})
	}

	r.mu.Lock()
	r.started = time.Now()
	r.mu.Unlock()

	r.writeActiveSession()

	r.emit(runtimeReadyMsg{NgrokURL: r.opts.cfg.Ngrok.StaticDomain})
}

// writeActiveSession captures the running-session info for coding agents.
// Best-effort: errors are logged into the start pane and otherwise ignored.
func (r *engine) writeActiveSession() {
	r.mu.Lock()
	fleetPID := 0
	if r.fleet != nil && r.fleet.cmd != nil && r.fleet.cmd.Process != nil {
		fleetPID = r.fleet.cmd.Process.Pid
	}
	started := r.started
	r.mu.Unlock()

	branch := readGit("rev-parse", "--abbrev-ref", "HEAD")
	commit := readGit("rev-parse", "--short", "HEAD")

	logsDir := filepath.Join(r.opts.repoRoot, "tools", "ship", ".state", "logs")
	session := ActiveSession{
		FleetPID:      fleetPID,
		FleetLog:      filepath.Join(logsDir, "fleet.log"),
		BuildLog:      filepath.Join(logsDir, "build.log"),
		MySQLDatabase: "fleet",
		Worktree:      r.opts.repoRoot,
		Branch:        branch,
		Commit:        commit,
		NgrokURL:      "https://" + strings.TrimSpace(r.opts.cfg.Ngrok.StaticDomain),
		StartedAt:     started,
	}
	if err := WriteActiveSession(session); err != nil {
		r.emit(logLineMsg{Source: "start", Line: "could not write active.json: " + err.Error()})
	}
}

// readGit runs `git ARGS` in the repo root, trimmed. Empty string on any
// error so we don't crash the start sequence over a missing .git dir.
func readGit(args ...string) string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (r *engine) emit(msg tea.Msg) {
	select {
	case r.sink <- msg:
	default:
		// Drop rather than block — keeps the start goroutine from
		// stalling if the TUI is briefly behind.
	}
}

// -----------------------------------------------------------------------------
// Individual steps
// -----------------------------------------------------------------------------

func (r *engine) stepDockerUp(ctx context.Context) error {
	// "up -d" returns once containers are started; readiness is checked
	// implicitly by `prepare db` later, which retries on connection errors.
	return runOneShot(ctx, "start", r.opts.repoRoot, nil, r.logSink, "docker", "compose", "up", "-d")
}

func (r *engine) stepMakeDeps(ctx context.Context) error {
	return runOneShot(ctx, "start", r.opts.repoRoot, nil, r.logSink, "make", "deps")
}

func (r *engine) stepMakeBuild(ctx context.Context) error {
	return runOneShot(ctx, "start", r.opts.repoRoot, nil, r.logSink, "make")
}

// stepGenerateDev launches `make generate-dev` in the background and waits
// for webpack to print its first "compiled successfully" line — without
// that, `fleet serve --dev` will fail with a "template not found" error.
func (r *engine) stepGenerateDev(ctx context.Context) error {
	logPath := filepath.Join(r.opts.repoRoot, "tools", "ship", ".state", "logs", "webpack.log")

	// Use a parallel channel just for our readiness scan — the proc itself
	// also forwards to the main sink, so the dashboard pane keeps streaming.
	ready := make(chan struct{})
	scanSink := make(chan logLine, 256)

	go func() {
		for line := range scanSink {
			r.emit(logLineMsg(line))
			if strings.Contains(line.Line, "compiled successfully") ||
				strings.Contains(line.Line, "compiled with") {
				select {
				case <-ready:
				default:
					close(ready)
				}
			}
		}
	}()

	p, err := startProc("webpack", r.opts.repoRoot, nil, logPath, scanSink, "make", "generate-dev")
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.genDev = p
	r.mu.Unlock()

	select {
	case <-ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Minute):
		return errors.New("webpack didn't compile within 3 minutes")
	}
}

func (r *engine) stepPrepareDB(ctx context.Context) error {
	binary := filepath.Join(r.opts.repoRoot, "build", "fleet")
	return runOneShot(ctx, "start", r.opts.repoRoot, nil, r.logSink, binary, "prepare", "db", "--dev")
}

// stepServe starts `./build/fleet serve --dev` with the configured
// premium and MDM-key flags. Long-running.
func (r *engine) stepServe(ctx context.Context) error {
	logPath := filepath.Join(r.opts.repoRoot, "tools", "ship", ".state", "logs", "fleet.log")
	binary := filepath.Join(r.opts.repoRoot, "build", "fleet")

	args := []string{"serve", "--dev"}
	if r.opts.cfg.Fleet.Premium {
		args = append(args, "--dev_license")
	}
	if r.opts.privateKey != "" {
		args = append(args, "--fleet_server_private_key="+r.opts.privateKey)
	}

	p, err := startProc("fleet", r.opts.repoRoot, nil, logPath, r.logSink, binary, args...)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.fleet = p
	r.mu.Unlock()

	// Wait for the server to actually answer requests before declaring the
	// step done, so the dashboard transition lines up with reality.
	if err := waitForFleetReady(ctx, r.opts.cfg.Fleet.Port); err != nil {
		// Best-effort cleanup so we don't leave a half-running server.
		_ = p.Stop(ctx, 2*time.Second)
		return err
	}
	return nil
}

func (r *engine) stepNgrok(ctx context.Context) error {
	domain := strings.TrimSpace(r.opts.cfg.Ngrok.StaticDomain)
	if domain == "" {
		return errors.New("ngrok static domain is empty (run `make ship ARGS=--reconfigure`)")
	}
	logPath := filepath.Join(r.opts.repoRoot, "tools", "ship", ".state", "logs", "ngrok.log")

	target := fmt.Sprintf("http://localhost:%d", r.opts.cfg.Fleet.Port)
	p, err := startProc("ngrok", r.opts.repoRoot, nil, logPath, r.logSink,
		"ngrok", "http", "--domain="+domain, "--log=stdout", target)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.ngrok = p
	r.mu.Unlock()
	// ngrok comes up fast; if the user's static domain is misconfigured the
	// process will exit with output we'll see in the log pane.
	return nil
}

// -----------------------------------------------------------------------------
// Shutdown
// -----------------------------------------------------------------------------

// Stop gracefully tears down everything Start brought up, in reverse order.
// Volumes are preserved (`docker compose down`, no `-v`).
func (r *engine) Stop(ctx context.Context) {
	r.mu.Lock()
	ngrok := r.ngrok
	fleet := r.fleet
	genDev := r.genDev
	r.ngrok, r.fleet, r.genDev = nil, nil, nil
	r.mu.Unlock()

	stopCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if ngrok != nil {
		_ = ngrok.Stop(stopCtx, 2*time.Second)
	}
	if fleet != nil {
		_ = fleet.Stop(stopCtx, 5*time.Second)
	}
	if genDev != nil {
		_ = genDev.Stop(stopCtx, 5*time.Second)
	}
	// `docker compose down` without -v keeps named volumes intact, so the
	// PM's MySQL data + simulated host UUIDs survive the shutdown.
	_ = exec.CommandContext(stopCtx, "docker", "compose", "down").Run()
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// waitForFleetReady polls the Fleet server until it responds or ctx expires.
// We poll the version endpoint because it doesn't require auth.
func waitForFleetReady(ctx context.Context, port int) error {
	url := fmt.Sprintf("https://localhost:%d/version", port)
	client := &http.Client{
		Timeout:   2 * time.Second,
		Transport: &http.Transport{TLSClientConfig: insecureTLS()},
	}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return errors.New("fleet server didn't come up within 60s")
}
