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
// on-disk log writers. Concurrency: stateMu guards procs/pids/lastArgs;
// storeMu guards logStore; writersMu guards logWriters. They're never held
// nested, so there's no lock-ordering deadlock.
type Manager struct {
	logDir  string
	dataDir string
	emit    Emitter

	stateMu  sync.Mutex
	procs    map[string]*ProcInfo
	pids     map[string]int
	lastArgs map[string]StartArgs
	// lifecycles[id].ctx is cancelled when that process has fully terminated
	// (after finalize + pid removal). Stop/Restart select on it to return as
	// soon as the process exits instead of always blocking a fixed grace.
	lifecycles map[string]*procLifecycle

	storeMu  sync.Mutex
	logStore map[string]*ring

	writersMu  sync.Mutex
	logWriters map[string]*channelWriter
}

// procLifecycle ties a managed process to a context that's cancelled on its
// termination.
type procLifecycle struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// New builds a Manager. logDir/dataDir are where channel logs and
// running.json live; emit delivers proc:log / proc:state to the frontend
// (may be nil in tests that don't assert on events).
func New(logDir, dataDir string, emit Emitter) *Manager {
	return &Manager{
		logDir: logDir, dataDir: dataDir, emit: emit,
		procs: map[string]*ProcInfo{}, pids: map[string]int{}, lastArgs: map[string]StartArgs{},
		lifecycles: map[string]*procLifecycle{},
		logStore:   map[string]*ring{}, logWriters: map[string]*channelWriter{},
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

// procDone returns the termination context for id (cancelled when the
// process has fully terminated), or nil if id isn't tracked.
func (m *Manager) procDone(id string) context.Context {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	if lc := m.lifecycles[id]; lc != nil {
		return lc.ctx
	}
	return nil
}

// finishLifecycle cancels and removes id's termination context, signaling
// any Stop/Restart waiters that the process is gone. Idempotent.
func (m *Manager) finishLifecycle(id string) {
	m.stateMu.Lock()
	lc := m.lifecycles[id]
	delete(m.lifecycles, id)
	m.stateMu.Unlock()
	if lc != nil {
		lc.cancel()
	}
}

func (m *Manager) collectPidRecords() []PidRecord {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	var recs []PidRecord
	for id, pid := range m.pids {
		a, ok := m.lastArgs[id]
		if !ok {
			continue
		}
		recs = append(recs, PidRecord{ID: id, PID: pid, Program: a.Program, Args: a.Args})
	}
	return recs
}

func (m *Manager) persistPids() { writePidFile(m.dataDir, m.collectPidRecords()) }

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
// running. The child gets its own process group (so we can signal the whole
// group) and the login-shell PATH plus any caller env (empty keys dropped).
func (m *Manager) Start(id string, a StartArgs) error {
	m.stateMu.Lock()
	_, running := m.pids[id]
	m.stateMu.Unlock()
	if running {
		return fmt.Errorf("process %s is already running", id)
	}

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
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to spawn %s: %w", a.Program, err)
	}
	pid := cmd.Process.Pid

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
	lc := &procLifecycle{}
	lc.ctx, lc.cancel = context.WithCancel(context.Background())

	m.stateMu.Lock()
	m.procs[id] = info
	m.pids[id] = pid
	m.lastArgs[id] = a
	m.lifecycles[id] = lc
	m.stateMu.Unlock()
	m.persistPids()
	m.emitState(id, "running", nil, nil)

	var wg sync.WaitGroup
	wg.Add(2)
	m.spawnLogReader(id, a.LogChannel, "stdout", stdout, &wg)
	m.spawnLogReader(id, a.LogChannel, "stderr", stderr, &wg)

	go m.waitAndFinalize(id, cmd, &wg)
	return nil
}

// waitAndFinalize waits for the readers to drain the pipes (required before
// cmd.Wait when using StdoutPipe), reaps the process, then records the final
// state and emits proc:state.
func (m *Manager) waitAndFinalize(id string, cmd *exec.Cmd, wg *sync.WaitGroup) {
	wg.Wait()
	waitErr := cmd.Wait()
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

	m.stateMu.Lock()
	wasUserStop := m.procs[id] != nil && m.procs[id].State == "stopping"
	m.stateMu.Unlock()
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

	m.stateMu.Lock()
	if info := m.procs[id]; info != nil {
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
	delete(m.pids, id)
	m.stateMu.Unlock()
	m.persistPids()
	m.emitState(id, final, exitCode, exitSignal)
	// Signal Stop/Restart waiters that this process is fully gone (state
	// recorded, pid removed). Done last so ctx.Done ⟺ finalized.
	m.finishLifecycle(id)
}

// signalStop SIGTERMs a process (its group), escalating to SIGKILL after
// 800ms. docker-compose processes route to a `compose down` instead.
func (m *Manager) signalStop(id string) error {
	m.stateMu.Lock()
	info := m.procs[id]
	var cmdStr, label string
	if info != nil {
		cmdStr, label = info.Command, info.Label
	}
	pid, hasPid := m.pids[id]
	m.stateMu.Unlock()

	if info != nil && (strings.HasPrefix(cmdStr, "docker compose") || strings.HasPrefix(label, "docker compose")) {
		return m.dockerComposeDownFor(id)
	}
	if !hasPid {
		return nil
	}

	m.stateMu.Lock()
	if info := m.procs[id]; info != nil {
		info.State = "stopping"
		info.WasUserStopped = true
	}
	m.stateMu.Unlock()
	m.emitState(id, "stopping", nil, nil)

	signalGroup(pid, syscall.SIGTERM)
	// Wait up to 800ms for graceful exit, but return immediately if the
	// process terminates first (ctx cancelled by waitAndFinalize).
	if done := m.procDone(id); done != nil {
		select {
		case <-done.Done():
			return nil
		case <-time.After(800 * time.Millisecond):
		}
	} else {
		time.Sleep(800 * time.Millisecond)
	}
	m.stateMu.Lock()
	_, stillAlive := m.pids[id]
	m.stateMu.Unlock()
	if stillAlive {
		signalGroup(pid, syscall.SIGKILL)
	}
	return nil
}

// Stop requests a graceful stop of id.
func (m *Manager) Stop(id string) error { return m.signalStop(id) }

// Forget drops a terminated process from tracking (refuses if still running).
func (m *Manager) Forget(id string) error {
	m.stateMu.Lock()
	if _, ok := m.pids[id]; ok {
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
func (m *Manager) Restart(id string) error {
	// Capture the running process's termination ctx before stopping so we can
	// wait for it to fully exit (pid removed) instead of a blind sleep.
	done := m.procDone(id)
	_ = m.signalStop(id)
	if done != nil {
		select {
		case <-done.Done():
		case <-time.After(2 * time.Second): // safety cap
		}
	}
	m.stateMu.Lock()
	a, ok := m.lastArgs[id]
	m.stateMu.Unlock()
	if !ok {
		return fmt.Errorf("no remembered args for %s", id)
	}
	return m.Start(id, a)
}

// ShutdownNow stops every running managed process and tears down docker
// compose (unconditionally, since `up -d` exits and leaves containers). It
// does NOT exit the app — the caller handles that after this returns.
func (m *Manager) ShutdownNow(repoPath string) {
	m.stateMu.Lock()
	var ids []string
	for id, info := range m.procs {
		if info.State == "running" || info.State == "stopping" {
			ids = append(ids, id)
		}
	}
	m.stateMu.Unlock()

	for _, id := range ids {
		_ = m.signalStop(id)
	}
	if repoPath != "" {
		cmd := dockerCmd("compose", "down")
		cmd.Dir = repoPath
		_ = cmd.Run()
	}
	// Final safety net for anything still alive.
	for _, id := range ids {
		m.stateMu.Lock()
		pid, ok := m.pids[id]
		m.stateMu.Unlock()
		if ok {
			signalGroup(pid, syscall.SIGKILL)
		}
	}
	time.Sleep(150 * time.Millisecond)
}
