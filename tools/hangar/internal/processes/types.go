// Package processes is the spawn/log/lifecycle manager for every child
// process Hangar runs (fleet serve, docker compose, osquery-perf, builds,
// ...). Ported from src-tauri/src/processes.rs.
//
// It's decoupled from Wails via the Emitter interface so the engine is
// unit-testable with a fake emitter; the service layer adapts Wails'
// app.Event.Emit to it. Directories (logs, app-data) are injected for the
// same reason.
package processes

// Emitter delivers a backend event to the frontend. Implemented in
// production by a thin Wails adapter; in tests by a recorder.
type Emitter interface {
	Emit(name string, data any)
}

// ProcInfo is the public state of one managed process.
type ProcInfo struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Command     string  `json:"command"`
	Cwd         string  `json:"cwd"`
	State       string  `json:"state"` // idle | running | done | failed | stopping
	StartedAtMS *uint64 `json:"started_at_ms"`
	EndedAtMS   *uint64 `json:"ended_at_ms"`
	ExitCode    *int    `json:"exit_code"`
	// ExitSignal is the terminating signal number on Unix when killed by a
	// signal (nil = exited normally). Lets the UI surface the real cause.
	ExitSignal     *int     `json:"exit_signal"`
	RecentLog      []string `json:"recent_log"`
	WasUserStopped bool     `json:"was_user_stopped"`
}

// LogLine is the per-line "proc:log" event payload.
type LogLine struct {
	ProcID string `json:"proc_id"`
	Stream string `json:"stream"`
	Line   string `json:"line"`
	TsMS   uint64 `json:"ts_ms"`
}

// ProcEvent is the "proc:state" event payload.
type ProcEvent struct {
	ProcID     string `json:"proc_id"`
	State      string `json:"state"`
	ExitCode   *int   `json:"exit_code"`
	ExitSignal *int   `json:"exit_signal"`
}

// LogEntry is one structured, stored log line (in the ring + on disk).
type LogEntry struct {
	TsMS    uint64  `json:"ts_ms"`
	Stream  string  `json:"stream"`
	Level   *string `json:"level"` // debug | info | warn | error | nil
	Message string  `json:"message"`
	Channel string  `json:"channel"`
}

// EnvPair is one KEY=VALUE applied to a spawn (empty keys are dropped).
type EnvPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// StartArgs is everything needed to (re)spawn a managed process.
type StartArgs struct {
	Label      string
	Cwd        string
	Program    string
	Args       []string
	LogChannel string // "" = no structured logging
	Env        []EnvPair
}

// ContainerState is one docker compose service's state.
type ContainerState struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

// DockerStatus summarizes `docker compose ps`.
type DockerStatus struct {
	Running    bool             `json:"running"`
	Containers []ContainerState `json:"containers"`
}

// LogWindow is a filtered slice of the structured log store.
type LogWindow struct {
	Entries       []LogEntry `json:"entries"`
	TotalInWindow int        `json:"total_in_window"`
	WarnCount     int        `json:"warn_count"`
	ErrorCount    int        `json:"error_count"`
}
