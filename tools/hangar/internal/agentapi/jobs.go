package agentapi

import (
	"context"
	"sync"

	"github.com/fleetdm/fleet/tools/hangar/internal/slots"
)

const (
	stateRunning     = "running"
	stateSucceeded   = "succeeded"
	stateFailed      = "failed"
	stateInterrupted = "interrupted"
)

type job struct {
	ID         string
	Slot       string
	Action     string
	LogChannel string
	cancel     context.CancelFunc

	mu         sync.Mutex
	State      string
	Steps      []stepView
	BackupPath string
	Error      string
	StartedMS  uint64
	EndedMS    *uint64
}

type stepView struct {
	ID       string `json:"id"`
	ExitCode *int   `json:"exit_code"`
}

type jobView struct {
	ID         string     `json:"id"`
	Slot       string     `json:"slot"`
	Action     string     `json:"action"`
	State      string     `json:"state"`
	LogChannel string     `json:"log_channel"`
	Processes  []stepView `json:"processes"`
	BackupPath string     `json:"backup_path,omitempty"`
	Error      string     `json:"error,omitempty"`
	StartedMS  uint64     `json:"started_at_ms"`
	EndedMS    *uint64    `json:"ended_at_ms"`
}

// step records a step starting (code nil) or ending.
func (j *job) step(id string, code *int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if code == nil {
		j.Steps = append(j.Steps, stepView{ID: id})
		return
	}
	for i := len(j.Steps) - 1; i >= 0; i-- {
		if j.Steps[i].ID == id {
			c := *code
			j.Steps[i].ExitCode = &c
			return
		}
	}
}

func (j *job) finish(res slots.Result, err error, cancelled bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.BackupPath = res.BackupPath
	switch {
	case err == nil:
		j.State = stateSucceeded
	case cancelled:
		j.State, j.Error = stateInterrupted, err.Error()
	default:
		j.State, j.Error = stateFailed, err.Error()
	}
	ended := nowMS()
	j.EndedMS = &ended
}

// jobRunner gives a job's compose down the job's ctx, so Close can cancel it,
// when the manager takes one.
type jobRunner struct {
	Manager
	ctx context.Context
}

func (r jobRunner) ComposeDown(id, cwd, project, owner string) error {
	if m, ok := r.Manager.(interface {
		ComposeDownContext(ctx context.Context, id, cwd, project, owner string) error
	}); ok {
		return m.ComposeDownContext(r.ctx, id, cwd, project, owner)
	}
	return r.Manager.ComposeDown(id, cwd, project, owner)
}

func (j *job) view() jobView {
	j.mu.Lock()
	defer j.mu.Unlock()
	return jobView{
		ID: j.ID, Slot: j.Slot, Action: j.Action, State: j.State, LogChannel: j.LogChannel,
		Processes: append([]stepView{}, j.Steps...), BackupPath: j.BackupPath, Error: j.Error,
		StartedMS: j.StartedMS, EndedMS: j.EndedMS,
	}
}
