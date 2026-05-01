package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	stepStopFleet // restart-only: stop the running fleet binary before rebuilding
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
	case stepStopFleet:
		return "stop running fleet"
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

// rebuildStartedMsg fires when a rebuild kicks off (auto from watcher OR
// manual via `r`). The dashboard uses Reason as the "trigger:" row above
// the step list.
type rebuildStartedMsg struct{ Reason string }

// pauseChangedMsg fires when the user toggles `p`. Queued counts the
// number of changed files captured while paused (zero when resuming
// without pending changes).
type pauseChangedMsg struct {
	Paused bool
	Queued int
}

// switchStartedMsg fires when the user picks a different worktree from
// the switcher. Reason summarizes for the dashboard's "trigger:" row;
// FromName / ToName are the friendly worktree names.
type switchStartedMsg struct {
	Reason   string
	FromName string
	ToName   string
}

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

	// File watcher started after the first runtimeReadyMsg. Auto-rebuild
	// triggers flow through HandleTrigger.
	watch *watcher

	// restartMu serializes Start vs Restart vs another Restart. Acquired
	// for the duration of any sequence run so two rebuilds can't race
	// over the fleet binary or the prepare-db step.
	restartMu sync.Mutex

	// pause state. paused gates auto-rebuild; while true, watcher events
	// accumulate in queued and the dashboard header shows ⏸ paused.
	pauseMu sync.Mutex
	paused  bool
	queued  map[string]struct{} // files seen while paused, deduplicated

	// deferred holds triggers that arrived while a rebuild was in
	// flight. After the rebuild finishes, anything in here fires one
	// follow-up rebuild so changes from the in-flight period don't get
	// dropped.
	deferredMu      sync.Mutex
	deferredFiles   map[string]struct{}
	deferredPending bool // true even if files is empty (e.g. r-press while building)
}

func newEngine(opts runtimeOpts) *engine {
	e := &engine{
		opts:    opts,
		sink:    make(chan tea.Msg, 256),
		logSink: make(chan logLine, 256),
		queued:  map[string]struct{}{},
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

// seqStep is one element of a stepped sequence.
type seqStep struct {
	kind stepKind
	fn   func(context.Context) error
}

// runSequence runs a list of steps in order, emitting per-step pending /
// running / done / failed messages, and finishes by writing active.json
// and emitting runtimeReadyMsg. Used by both initial Start and Restart.
// The caller is expected to hold restartMu.
func (r *engine) runSequence(ctx context.Context, steps []seqStep) {
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
	r.startWatcherOnce()
	r.emit(runtimeReadyMsg{NgrokURL: r.opts.cfg.Ngrok.StaticDomain})
}

// startWatcherOnce spins up the file watcher the first time the engine
// reaches running state. Subsequent rebuilds also call this; the function
// is a no-op when a watcher is already running.
func (r *engine) startWatcherOnce() {
	r.mu.Lock()
	already := r.watch != nil
	r.mu.Unlock()
	if already {
		return
	}
	w, err := newWatcher(r.opts.repoRoot)
	if err != nil {
		// Watcher failure shouldn't block bring-up. Surface it as a
		// log line so the user knows auto-rebuild won't kick in.
		r.emit(logLineMsg{Source: "start", Line: "watcher failed: " + err.Error()})
		return
	}
	r.mu.Lock()
	r.watch = w
	r.mu.Unlock()

	out := w.Start(context.Background())
	go func() {
		for trig := range out {
			r.HandleTrigger(trig.Reason, trig.Files)
		}
	}()
}

// run is the initial bring-up sequence.
//
// Order matters: `make generate-dev` writes server/bindata/generated.go,
// which cmd/fleet imports — so build comes after generate-dev, not before.
func (r *engine) run(ctx context.Context) {
	r.restartMu.Lock()
	defer r.restartMu.Unlock()

	r.runSequence(ctx, []seqStep{
		{stepDockerUp, r.stepDockerUp},
		{stepMakeDeps, r.stepMakeDeps},
		{stepGenerateDev, r.stepGenerateDev},
		{stepMakeBuild, r.stepMakeBuild},
		{stepPrepareDB, r.stepPrepareDB},
		{stepServe, r.stepServe},
		{stepNgrok, r.stepNgrok},
	})
}

// HandleTrigger is called by the TUI when an auto-rebuild trigger arrives
// (file watcher event or `r` keypress). reason is what the dashboard shows
// on the "trigger:" row. files is the deduplicated list of paths (empty
// when triggered manually with r).
//
// Behavior:
//   - paused → accumulate files in r.queued, no rebuild
//   - rebuild already in flight → accumulate files in r.deferred*, fired
//     automatically as one follow-up rebuild after the current one ends
//   - otherwise → spawn a goroutine that owns restartMu for the duration
func (r *engine) HandleTrigger(reason string, files []string) {
	r.pauseMu.Lock()
	if r.paused {
		for _, f := range files {
			r.queued[f] = struct{}{}
		}
		queued := len(r.queued)
		r.pauseMu.Unlock()
		r.emit(pauseChangedMsg{Paused: true, Queued: queued})
		return
	}
	r.pauseMu.Unlock()

	if !r.restartMu.TryLock() {
		// A rebuild is already running. Queue this trigger so the
		// in-flight rebuild's deferred-drain picks it up.
		r.appendDeferred(files)
		return
	}

	go func() {
		defer r.restartMu.Unlock()
		r.runRestartLocked(context.Background(), reason)

		// After the rebuild, fire a single follow-up if any triggers
		// arrived while we were running.
		if d := r.takeDeferred(); d != nil {
			go r.HandleTrigger(d.reason, d.files)
		}
	}()
}

// deferredTrigger is what takeDeferred hands back: the merged file set
// from any HandleTrigger calls that lost the TryLock race during a
// rebuild, plus the buildReason()-derived label for the dashboard.
type deferredTrigger struct {
	files  []string
	reason string
}

// appendDeferred records that a trigger arrived during a rebuild. files
// may be empty for an r-press during a rebuild — pending stays true so a
// follow-up still fires.
func (r *engine) appendDeferred(files []string) {
	r.deferredMu.Lock()
	defer r.deferredMu.Unlock()
	if r.deferredFiles == nil {
		r.deferredFiles = map[string]struct{}{}
	}
	for _, f := range files {
		r.deferredFiles[f] = struct{}{}
	}
	r.deferredPending = true
}

// takeDeferred drains the deferred set and returns it (nil if empty).
func (r *engine) takeDeferred() *deferredTrigger {
	r.deferredMu.Lock()
	defer r.deferredMu.Unlock()
	if !r.deferredPending {
		return nil
	}
	files := make([]string, 0, len(r.deferredFiles))
	for f := range r.deferredFiles {
		files = append(files, f)
	}
	sort.Strings(files)
	r.deferredFiles = nil
	r.deferredPending = false

	reason := buildReason(files)
	if len(files) == 0 {
		// A bare r-press (or pause-resume) during a build queued a
		// pending bit but no files; surface something readable.
		reason = "follow-up rebuild"
	}
	return &deferredTrigger{files: files, reason: reason}
}

// TogglePause flips the paused flag. When transitioning from paused →
// active with queued changes, fires one rebuild whose reason summarizes
// the queue.
func (r *engine) TogglePause() {
	r.pauseMu.Lock()
	r.paused = !r.paused
	now := r.paused
	queuedFiles := make([]string, 0, len(r.queued))
	for f := range r.queued {
		queuedFiles = append(queuedFiles, f)
	}
	if !now {
		r.queued = map[string]struct{}{}
	}
	r.pauseMu.Unlock()

	r.emit(pauseChangedMsg{Paused: now, Queued: len(queuedFiles)})
	if !now && len(queuedFiles) > 0 {
		// Resume fires through HandleTrigger so it picks up the same
		// "in flight? defer" handling as any other trigger.
		go r.HandleTrigger(buildReason(queuedFiles), queuedFiles)
	}
}

// runRestartLocked executes the hot-rebuild sequence: stop fleet, rebuild,
// re-run prepare db, restart fleet. Webpack and ngrok keep running across
// the restart. PR 4's snapshot/restore logic will plug in between
// stepStopFleet and stepPrepareDB.
//
// Caller must already hold restartMu — HandleTrigger acquires it before
// spawning the goroutine that calls this.
func (r *engine) runRestartLocked(ctx context.Context, reason string) {
	r.emit(rebuildStartedMsg{Reason: reason})
	r.runSequence(ctx, []seqStep{
		{stepStopFleet, r.stepStopFleet},
		{stepMakeBuild, r.stepMakeBuild},
		{stepPrepareDB, r.stepPrepareDB},
		{stepServe, r.stepServe},
	})
}

// SwitchTo handles a worktree change. Stops fleet + generate-dev + watcher
// (all of which were tied to the old worktree's filesystem), clears the
// old worktree's active.json, swaps the engine's repoRoot, and runs a
// switch-specific bring-up at the new path: make deps → make generate-dev
// → make → prepare db → start fleet. Docker compose and ngrok keep
// running across the switch — same volumes, same public URL.
func (r *engine) SwitchTo(newRepoRoot, newName string) {
	if !r.restartMu.TryLock() {
		// A rebuild or another switch is in flight; refuse silently
		// and let the user retry. This is rare and switches aren't
		// safe to defer the way rebuild triggers are.
		return
	}
	oldRoot := r.opts.repoRoot
	oldName := DefaultWorktreeName(oldRoot)

	go func() {
		defer r.restartMu.Unlock()
		r.runSwitchLocked(context.Background(), oldRoot, oldName, newRepoRoot, newName)
	}()
}

func (r *engine) runSwitchLocked(ctx context.Context, oldRoot, oldName, newRoot, newName string) {
	r.emit(switchStartedMsg{
		Reason:   "switching to " + newName,
		FromName: oldName,
		ToName:   newName,
	})

	// Tear down everything tied to the OLD worktree's filesystem.
	r.mu.Lock()
	fleet := r.fleet
	genDev := r.genDev
	watch := r.watch
	r.fleet, r.genDev, r.watch = nil, nil, nil
	r.mu.Unlock()

	if watch != nil {
		watch.Stop()
	}
	if genDev != nil {
		_ = genDev.Stop(ctx, 5*time.Second)
	}
	if fleet != nil {
		_ = fleet.Stop(ctx, 5*time.Second)
	}

	_ = ClearActiveSession(oldRoot)

	// Swap the engine's view of "where I'm running" before any step
	// function runs (they read r.opts.repoRoot for `cmd.Dir`).
	r.mu.Lock()
	r.opts.repoRoot = newRoot
	r.mu.Unlock()

	// Switch-specific sequence: docker compose stays up, ngrok keeps
	// forwarding to localhost:8080, only the per-worktree pieces rerun.
	r.runSequence(ctx, []seqStep{
		{stepMakeDeps, r.stepMakeDeps},
		{stepGenerateDev, r.stepGenerateDev},
		{stepMakeBuild, r.stepMakeBuild},
		{stepPrepareDB, r.stepPrepareDB},
		{stepServe, r.stepServe},
	})
}

// stepStopFleet shuts down the running fleet binary so stepServe can
// re-launch it. No-op if the fleet proc isn't running (first-time edge
// case).
func (r *engine) stepStopFleet(ctx context.Context) error {
	r.mu.Lock()
	fleet := r.fleet
	r.fleet = nil
	r.mu.Unlock()
	if fleet == nil {
		return nil
	}
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return fleet.Stop(stopCtx, 5*time.Second)
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
	if err := WriteActiveSession(r.opts.repoRoot, session); err != nil {
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

// composeProject is hardcoded so every worktree of this Fleet repo
// shares the same MySQL/Redis containers and (more importantly) volumes.
// "fleet" specifically (rather than "fleet-ship") matches the default
// project name an engineer would get running `docker compose up` from
// ~/projects/fleet — which means tools/backup_db/{backup,restore}.sh
// (which hardcode `--network fleet_default`) work against ship's
// containers without modification, and PR 4 can reuse them directly.
const composeProject = "fleet"

func (r *engine) stepDockerUp(ctx context.Context) error {
	// "up -d" returns once containers are started; readiness is checked
	// implicitly by `prepare db` later, which retries on connection errors.
	return runOneShot(ctx, "start", r.opts.repoRoot, nil, r.logSink,
		"docker", "compose", "-p", composeProject, "up", "-d")
}

func (r *engine) stepMakeDeps(ctx context.Context) error {
	return runOneShot(ctx, "start", r.opts.repoRoot, nil, r.logSink, "make", "deps")
}

func (r *engine) stepMakeBuild(ctx context.Context) error {
	return runOneShot(ctx, "start", r.opts.repoRoot, nil, r.logSink, "make")
}

// stepGenerateDev launches `make generate-dev` in the background and waits
// for it to produce server/bindata/generated.go — that's the file cmd/fleet
// imports, and webpack's "compiled successfully" output is a misleading
// signal because it fires before go-bindata runs.
func (r *engine) stepGenerateDev(ctx context.Context) error {
	logPath := filepath.Join(r.opts.repoRoot, "tools", "ship", ".state", "logs", "webpack.log")
	bindata := filepath.Join(r.opts.repoRoot, "server", "bindata", "generated.go")

	p, err := startProc("webpack", r.opts.repoRoot, nil, logPath, r.logSink, "make", "generate-dev")
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.genDev = p
	r.mu.Unlock()

	// Poll for the bindata file to appear with non-empty content. Webpack's
	// initial build runs first, then go-bindata writes this file, then
	// webpack switches to --watch. Once the file is here, `make` can build.
	deadline := time.Now().Add(5 * time.Minute)
	for {
		if info, err := os.Stat(bindata); err == nil && info.Size() > 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			return errors.New("make generate-dev didn't produce server/bindata/generated.go within 5 minutes")
		}
	}
}

func (r *engine) stepPrepareDB(ctx context.Context) error {
	binary := filepath.Join(r.opts.repoRoot, "build", "fleet")
	return runOneShot(ctx, "start", r.opts.repoRoot, nil, r.logSink, binary, "prepare", "db", "--dev")
}

// stepServe starts `./build/fleet serve --dev` with the configured
// premium toggle. The Fleet server private key goes through the
// FLEET_SERVER_PRIVATE_KEY env var rather than a CLI flag so the secret
// doesn't end up in `ps aux`. Long-running.
func (r *engine) stepServe(ctx context.Context) error {
	logPath := filepath.Join(r.opts.repoRoot, "tools", "ship", ".state", "logs", "fleet.log")
	binary := filepath.Join(r.opts.repoRoot, "build", "fleet")

	args := []string{
		"serve", "--dev",
		// Don't refuse to start when the shared dev DB has migration
		// rows from a different worktree's branch. Cross-worktree
		// migration drift is the expected price of the shared-DB
		// design — Fleet logs a warning and serves anyway.
		"--upgrades_allow_missing_migrations",
	}
	if r.opts.cfg.Fleet.Premium {
		args = append(args, "--dev_license")
	}

	var env []string
	if r.opts.privateKey != "" {
		env = append(env, "FLEET_SERVER_PRIVATE_KEY="+r.opts.privateKey)
	}

	p, err := startProc("fleet", r.opts.repoRoot, env, logPath, r.logSink, binary, args...)
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

	// Matches the command shape that's known to work: forward to the HTTPS
	// upstream, point at the user's static domain via --url. ngrok handles
	// the localhost self-signed cert automatically here.
	target := fmt.Sprintf("https://localhost:%d", r.opts.cfg.Fleet.Port)
	p, err := startProc("ngrok", r.opts.repoRoot, nil, logPath, r.logSink,
		"ngrok", "http", target, "--url="+domain)
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
	watch := r.watch
	r.ngrok, r.fleet, r.genDev, r.watch = nil, nil, nil, nil
	r.mu.Unlock()

	if watch != nil {
		watch.Stop()
	}

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
	_ = exec.CommandContext(stopCtx, "docker", "compose", "-p", composeProject, "down").Run()
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
