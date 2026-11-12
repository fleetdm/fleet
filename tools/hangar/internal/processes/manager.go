package processes

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
)

const logTailCap = 60 // recent_log lines kept per process

// Manager owns every managed child process plus the structured log store and
// on-disk log writers. Concurrency: stateMu guards procs/lastArgs/live/holds/
// dockerOps; storeMu guards logStore; writersMu guards logWriters.
// pidFileMu is taken before stateMu; no other locks are held nested.
type Manager struct {
	logDir  string
	dataDir string
	emit    Emitter

	stateMu   sync.Mutex
	procs     map[string]*ProcInfo
	lastArgs  map[string]StartArgs
	live      map[string]*invocation // only an invocation's own finalizer removes it
	seq       uint64
	holds     map[string]HoldSpec
	dockerOps map[string]int // compose project -> in-flight down/restart count

	pidFileMu sync.Mutex // snapshot+write of running.json as one step

	// Test seams.
	kill     func(pid int, sig syscall.Signal) error
	waitExit func(pid int) bool

	storeMu  sync.Mutex
	logStore map[string]*ring

	writersMu  sync.Mutex
	logWriters map[string]*channelWriter
}

// invocation is one spawn of an id. done closes once its finalizer has
// recorded the final state and dropped it from live.
type invocation struct {
	id       string
	seq      uint64
	args     StartArgs
	pid      int // 0 while spawning
	stopped  bool
	reaped   bool // set before the leader can be reaped; no group signals after
	exitCode int  // valid after done; -1 when signaled, stopped, or never spawned
	done     chan struct{}
}

// Invocation identifies one spawn of a process id, for Wait.
type Invocation struct {
	ID  string
	Seq uint64

	inv *invocation
}

// New builds a Manager. logDir/dataDir are where channel logs and
// running.json live; emit delivers proc:log / proc:state to the frontend
// (may be nil in tests that don't assert on events).
func New(logDir, dataDir string, emit Emitter) *Manager {
	return &Manager{
		logDir: logDir, dataDir: dataDir, emit: emit,
		procs: map[string]*ProcInfo{}, lastArgs: map[string]StartArgs{},
		live: map[string]*invocation{}, holds: map[string]HoldSpec{}, dockerOps: map[string]int{},
		logStore: map[string]*ring{}, logWriters: map[string]*channelWriter{},
		kill: syscall.Kill, waitExit: waitExit,
	}
}

func nowMS() uint64 { return uint64(time.Now().UnixMilli()) }

func (m *Manager) emitLog(ll LogLine) {
	if m.emit != nil {
		m.emit.Emit("proc:log", ll)
	}
}

func (m *Manager) emitState(id, state string, code, sig *int) {
	if m.emit != nil {
		m.emit.Emit("proc:state", ProcEvent{ProcID: id, State: state, ExitCode: code, ExitSignal: sig})
	}
}

// ListProcesses returns a snapshot of all tracked processes.
func (m *Manager) ListProcesses() []ProcInfo {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	out := make([]ProcInfo, 0, len(m.procs))
	for _, info := range m.procs {
		out = append(out, *info)
	}
	return out
}

func (m *Manager) appendRecentLog(id, line string) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	if info := m.procs[id]; info != nil {
		info.RecentLog = append(info.RecentLog, line)
		if len(info.RecentLog) > logTailCap {
			info.RecentLog = info.RecentLog[len(info.RecentLog)-logTailCap:]
		}
	}
}

func (m *Manager) pushToLogStore(e LogEntry) {
	m.storeMu.Lock()
	defer m.storeMu.Unlock()
	r := m.logStore[e.Channel]
	if r == nil {
		r = newRing(logChannelCap)
		m.logStore[e.Channel] = r
	}
	r.push(e)
}

func (m *Manager) writeLogDisk(channel string, e LogEntry) {
	m.writersMu.Lock()
	defer m.writersMu.Unlock()
	cw := m.logWriters[channel]
	if cw == nil {
		var err error
		cw, err = openChannelWriter(logFilePath(m.logDir, channel), logFileMaxBytes)
		if err != nil {
			slog.Warn("open log writer failed", "channel", channel, "err", err)
			return
		}
		m.logWriters[channel] = cw
	}
	cw.write(e.TsMS, e.Stream, e.Message)
}

func (m *Manager) flushLogWriter(channel string) {
	m.writersMu.Lock()
	defer m.writersMu.Unlock()
	if cw := m.logWriters[channel]; cw != nil {
		cw.close()
		delete(m.logWriters, channel)
	}
}

func (m *Manager) collectPidRecords() []PidRecord {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	var recs []PidRecord
	for id, inv := range m.live {
		if inv.pid == 0 {
			continue
		}
		recs = append(recs, PidRecord{ID: id, PID: inv.pid, Program: inv.args.Program, Args: inv.args.Args})
	}
	return recs
}

func (m *Manager) persistPids() {
	m.pidFileMu.Lock()
	defer m.pidFileMu.Unlock()
	writePidFile(m.dataDir, m.collectPidRecords())
}

// spawnLogReader streams one pipe: per line it updates recent_log, the
// structured store, the on-disk log, and emits proc:log. On EOF it flushes
// the channel's writer so the crash tail is durable.
func (m *Manager) spawnLogReader(id, channel, stream string, r io.Reader, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		br := bufio.NewReader(r)
		for {
			line, err := br.ReadString('\n')
			if len(line) > 0 {
				line = strings.TrimSuffix(line, "\n")
				line = strings.TrimSuffix(line, "\r")
				m.handleLine(id, channel, stream, line)
			}
			if err != nil {
				break
			}
		}
		if channel != "" {
			m.flushLogWriter(channel)
		}
	}()
}

func (m *Manager) handleLine(id, channel, stream, line string) {
	m.appendRecentLog(id, line)
	ts := nowMS()
	if channel != "" {
		var level *string
		if l := detectLevel(line); l != "" {
			level = &l
		}
		entry := LogEntry{TsMS: ts, Stream: stream, Level: level, Message: line, Channel: channel}
		m.pushToLogStore(entry)
		m.writeLogDisk(channel, entry)
	}
	m.emitLog(LogLine{ProcID: id, Stream: stream, Line: line, TsMS: ts})
}

// Start spawns a managed process under id. It errors if id is already
// running.
func (m *Manager) Start(id string, a StartArgs) error {
	_, err := m.StartInvocation(id, a)
	return err
}

// StartInvocation spawns a managed process under id and returns the handle
// for Wait. The id is reserved under stateMu before the spawn, so concurrent
// starts spawn once.
func (m *Manager) StartInvocation(id string, a StartArgs) (Invocation, error) {
	m.stateMu.Lock()
	if _, running := m.live[id]; running {
		m.stateMu.Unlock()
		return Invocation{}, fmt.Errorf("process %s is already running", id)
	}
	if err := m.refuseStartLocked(id, a); err != nil {
		m.stateMu.Unlock()
		return Invocation{}, err
	}
	m.seq++
	inv := &invocation{id: id, seq: m.seq, args: a, exitCode: -1, done: make(chan struct{})}
	m.live[id] = inv
	m.stateMu.Unlock()

	cmd, stdout, stderr, err := spawn(a)
	if err != nil {
		m.stateMu.Lock()
		delete(m.live, id)
		m.stateMu.Unlock()
		close(inv.done)
		return Invocation{}, err
	}

	now := nowMS()
	info := &ProcInfo{
		ID:          id,
		Label:       a.Label,
		Command:     strings.TrimSpace(a.Program + " " + strings.Join(a.Args, " ")),
		Cwd:         a.Cwd,
		State:       "running",
		StartedAtMS: &now,
		RecentLog:   []string{},
	}

	m.stateMu.Lock()
	inv.pid = cmd.Process.Pid
	m.procs[id] = info
	m.lastArgs[id] = a
	m.stateMu.Unlock()
	m.persistPids()
	m.emitState(id, "running", nil, nil)

	var wg sync.WaitGroup
	wg.Add(2)
	m.spawnLogReader(id, a.LogChannel, "stdout", stdout, &wg)
	m.spawnLogReader(id, a.LogChannel, "stderr", stderr, &wg)

	go m.waitAndFinalize(inv, cmd, &wg)
	return Invocation{ID: id, Seq: inv.seq, inv: inv}, nil
}

func spawn(a StartArgs) (*exec.Cmd, io.Reader, io.Reader, error) {
	cmd := shellpath.Command(a.Program, a.Args...)
	cmd.Dir = a.Cwd
	extra := map[string]string{}
	for _, p := range a.Env {
		if p.Key != "" {
			extra[p.Key] = p.Value
		}
	}
	// shellpath.Command preset cmd.Env to the login-shell env; layer the
	// caller's env on top.
	cmd.Env = shellpath.MergeEnv(cmd.Env, extra)
	// Own process group so a single kill(-pid) reaches the whole tree.
	// Deliberately NOT setting a Cancel/WaitDelay: we don't want the child
	// killed if the Manager goes away — crash recovery (running.json) handles
	// orphans on the next launch.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to spawn %s: %w", a.Program, err)
	}
	return cmd, stdout, stderr, nil
}

// Wait blocks until inv exits or ctx ends. exitCode is inv's own exit code;
// -1 when it was signaled or stopped from outside.
func (m *Manager) Wait(ctx context.Context, inv Invocation) (exitCode int, err error) {
	if inv.inv == nil {
		return -1, fmt.Errorf("unknown invocation %s#%d", inv.ID, inv.Seq)
	}
	select {
	case <-inv.inv.done:
		m.stateMu.Lock()
		defer m.stateMu.Unlock()
		return inv.inv.exitCode, nil
	case <-ctx.Done():
		return -1, ctx.Err()
	}
}

// waitAndFinalize waits for the readers to drain the pipes (required before
// cmd.Wait when using StdoutPipe), reaps the process, then records the final
// state and emits proc:state.
func (m *Manager) waitAndFinalize(inv *invocation, cmd *exec.Cmd, wg *sync.WaitGroup) {
	wg.Wait()
	exited := m.waitExit(inv.pid)
	m.stateMu.Lock()
	stopped := inv.stopped
	if stopped {
		// The leader is unreaped, so the pgid is still ours: kill what's left
		// of a stopped group, such as a TERM-ignoring child that dropped the
		// pipes.
		m.signalLocked(inv, syscall.SIGKILL)
	}
	// cmd.Wait reaps the leader, after which its pid could name an unrelated
	// group, so no group signal may follow. Without proof that the leader has
	// exited, this also leaves a leader that dropped its pipes unsignalable.
	inv.reaped = true
	m.stateMu.Unlock()
	if !exited {
		slog.Warn("can't watch process exit without reaping it; skipping further signals to its group", "id", inv.id, "pid", inv.pid)
	}
	waitErr := cmd.Wait()
	m.stateMu.Lock()
	wasUserStop := inv.stopped
	m.stateMu.Unlock()
	if stopped {
		waitGroupGone(inv.pid)
	}
	ok := waitErr == nil

	var exitCode, exitSignal *int
	if ps := cmd.ProcessState; ps != nil {
		if ws, isWS := ps.Sys().(syscall.WaitStatus); isWS {
			if ws.Signaled() {
				s := int(ws.Signal())
				exitSignal = &s
			} else {
				c := ws.ExitStatus()
				exitCode = &c
			}
		}
	}

	final := "failed"
	if wasUserStop || ok {
		final = "done"
	}

	// Synthesize a tail line when the process died silently (so "View error"
	// shows the cause instead of the last ordinary log line).
	synth := ""
	switch {
	case exitSignal != nil:
		synth = fmt.Sprintf("[exit: killed by signal %d (%s)]", *exitSignal, signalName(*exitSignal))
	case final == "failed" && exitCode != nil:
		synth = fmt.Sprintf("[exit: code %d]", *exitCode)
	case final == "failed":
		synth = "[exit: process gone, no exit code or signal]"
	}

	code := -1
	if exitCode != nil && !wasUserStop {
		code = *exitCode
	}
	m.finish(inv, final, code, exitCode, exitSignal, synth)
}

// finish records inv's final state on its row, drops it from live, and wakes
// its waiters. It's a no-op unless inv is still id's current invocation, so a
// stale finalizer can't clear a newer spawn's state or pid.
func (m *Manager) finish(inv *invocation, final string, code int, exitCode, exitSignal *int, synth string) {
	m.stateMu.Lock()
	if m.live[inv.id] != inv {
		m.stateMu.Unlock()
		return
	}
	if info := m.procs[inv.id]; info != nil {
		info.State = final
		info.ExitCode = exitCode
		info.ExitSignal = exitSignal
		ended := nowMS()
		info.EndedAtMS = &ended
		if synth != "" {
			info.RecentLog = append(info.RecentLog, synth)
			if len(info.RecentLog) > logTailCap {
				info.RecentLog = info.RecentLog[len(info.RecentLog)-logTailCap:]
			}
		}
	}
	inv.exitCode = code
	delete(m.live, inv.id)
	m.stateMu.Unlock()
	m.persistPids()
	m.emitState(inv.id, final, exitCode, exitSignal)
	// Last, so done ⟺ finalized (state recorded, pid removed).
	close(inv.done)
}

// signalStop SIGTERMs a process group, then SIGKILLs it after 800ms. A compose
// row gets `compose down` once its running up has exited. The hold check and
// the switch to stopping share one lock.
func (m *Manager) signalStop(id, owner string, bypass bool) error {
	m.stateMu.Lock()
	if !bypass {
		if err := m.heldErrLocked(id, owner); err != nil {
			m.stateMu.Unlock()
			return err
		}
	}
	info := m.procs[id]
	compose := info != nil && (strings.HasPrefix(info.Command, "docker compose") || strings.HasPrefix(info.Label, "docker compose"))
	var cwd, project string
	var endOp func()
	if compose {
		cwd, project = info.Cwd, composeProjectFromArgs(m.lastArgs[id].Args)
		// Tracked from the hold check on, so no Hold is admitted on the
		// project before this stop's compose down has run.
		endOp = m.trackDockerOpLocked(project)
	}
	inv := m.live[id]
	if inv != nil && inv.pid != 0 && info != nil {
		inv.stopped = true
		info.State = "stopping"
		info.WasUserStopped = true
	} else {
		inv = nil
	}
	m.stateMu.Unlock()

	if inv != nil {
		m.emitState(id, "stopping", nil, nil)
		m.terminate(inv, compose)
	}
	if compose {
		defer endOp()
		if cwd == "" {
			return nil
		}
		return m.dockerComposeDownFor(id, cwd, project)
	}
	return nil
}

// terminate SIGTERMs inv's group and SIGKILLs it if inv hasn't finalized
// within 800ms. With waitDone it also waits for the finalizer, which kills
// what's left of a stopped group.
func (m *Manager) terminate(inv *invocation, waitDone bool) {
	m.signal(inv, syscall.SIGTERM)
	select {
	case <-inv.done:
	case <-time.After(800 * time.Millisecond):
		m.signal(inv, syscall.SIGKILL)
	}
	if !waitDone {
		return
	}
	select {
	case <-inv.done:
	case <-time.After(5 * time.Second): // a grandchild outside the group holding the pipes
	}
}

// signal sends sig to inv's process group while its leader is unreaped.
func (m *Manager) signal(inv *invocation, sig syscall.Signal) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	m.signalLocked(inv, sig)
}

// signalLocked is signal with stateMu held. After reaping, the leader's pid
// could name an unrelated process group, so it sends nothing.
func (m *Manager) signalLocked(inv *invocation, sig syscall.Signal) {
	if inv.reaped || inv.pid == 0 {
		return
	}
	if err := m.kill(-inv.pid, sig); err != nil {
		_ = m.kill(inv.pid, sig)
	}
}

// waitGroupGone waits up to 2s for pgid to have no members left. It only
// probes with signal 0, which is harmless if the pgid has been reused.
func waitGroupGone(pgid int) {
	deadline := time.Now().Add(2 * time.Second)
	for syscall.Kill(-pgid, 0) != syscall.ESRCH && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
}

// Stop requests a graceful stop of id.
func (m *Manager) Stop(id string) error { return m.StopAs(id, "") }

// StopAs is Stop on behalf of owner; a held slot refuses anyone but its holder.
func (m *Manager) StopAs(id, owner string) error { return m.signalStop(id, owner, false) }

// Forget drops a terminated process from tracking (refuses if still running).
func (m *Manager) Forget(id string) error {
	m.stateMu.Lock()
	if _, ok := m.live[id]; ok {
		m.stateMu.Unlock()
		return fmt.Errorf("process %s is still running", id)
	}
	delete(m.procs, id)
	delete(m.lastArgs, id)
	m.stateMu.Unlock()
	m.storeMu.Lock()
	delete(m.logStore, id)
	m.storeMu.Unlock()
	return nil
}

// Restart stops then respawns id using its remembered args.
func (m *Manager) Restart(id string) error { return m.RestartAs(id, "") }

// RestartAs is Restart on behalf of owner. The respawn carries owner, never
// the owner remembered from the previous spawn.
func (m *Manager) RestartAs(id, owner string) error {
	m.stateMu.Lock()
	inv := m.live[id]
	m.stateMu.Unlock()
	if err := m.signalStop(id, owner, false); err != nil {
		return err
	}
	if inv != nil {
		select {
		case <-inv.done:
		case <-time.After(2 * time.Second): // safety cap
		}
	}
	m.stateMu.Lock()
	a, ok := m.lastArgs[id]
	m.stateMu.Unlock()
	if !ok {
		return fmt.Errorf("no remembered args for %s", id)
	}
	a.Owner = owner
	return m.Start(id, a)
}

// ComposeTarget is one server's docker compose stack to tear down on quit:
// its worktree dir and compose project name.
type ComposeTarget struct {
	Cwd     string `json:"cwd"`
	Project string `json:"project"`
}

// ShutdownNow stops every running managed process and tears down each server's
// docker compose stack (unconditionally, since `up -d` exits and leaves the
// containers running with the compose-up row already "done"). It bypasses slot
// holds and does NOT exit the app — the caller handles that after this returns.
//
// targets is the set of (cwd, project) compose stacks to bring down — one per
// configured server. We also fold in any compose stacks we can derive from the
// processes Hangar spawned this session, so a stack started here is always torn
// down even if the caller's list is incomplete.
func (m *Manager) ShutdownNow(targets []ComposeTarget) {
	m.stateMu.Lock()
	var ids []string
	for id, info := range m.procs {
		if info.State == "running" || info.State == "stopping" {
			ids = append(ids, id)
		}
	}
	tracked := m.composeTargetsLocked()
	m.stateMu.Unlock()

	for _, id := range ids {
		_ = m.signalStop(id, "", true)
	}

	for _, tgt := range dedupeComposeTargets(append(targets, tracked...)) {
		if tgt.Cwd == "" {
			continue
		}
		_, _, _, _ = runCompose(context.Background(), tgt.Cwd, tgt.Project, "down")
	}

	// Final safety net for anything still alive.
	for _, id := range ids {
		m.stateMu.Lock()
		if inv := m.live[id]; inv != nil {
			m.signalLocked(inv, syscall.SIGKILL)
		}
		m.stateMu.Unlock()
	}
	time.Sleep(150 * time.Millisecond)
}

// composeTargetsLocked derives compose (cwd, project) pairs from every
// compose-up process Hangar has spawned this session. Caller holds stateMu.
func (m *Manager) composeTargetsLocked() []ComposeTarget {
	var out []ComposeTarget
	for id, a := range m.lastArgs {
		info := m.procs[id]
		if info == nil || info.Cwd == "" {
			continue
		}
		if !strings.HasPrefix(info.Command, "docker compose") {
			continue
		}
		out = append(out, ComposeTarget{Cwd: info.Cwd, Project: composeProjectFromArgs(a.Args)})
	}
	return out
}

func dedupeComposeTargets(in []ComposeTarget) []ComposeTarget {
	seen := map[ComposeTarget]struct{}{}
	var out []ComposeTarget
	for _, t := range in {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
