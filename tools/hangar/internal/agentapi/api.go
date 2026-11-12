// Package agentapi serves the agent HTTP API: slot listing, slot actions as
// jobs, and job state and scrubbed logs. An agent holding a fleet-workflow
// reservation for a slot can assign its worktree and run the slot lifecycle
// without the GUI. Guards keep agents off s1 and off slots they don't hold.
// They stop cooperating agents' mistakes; they aren't a security boundary.
package agentapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/db"
	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
	"github.com/fleetdm/fleet/tools/hangar/internal/reservations"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
	"github.com/fleetdm/fleet/tools/hangar/internal/slots"
)

// SettingsStore is the settings.Store surface the API uses.
type SettingsStore interface {
	Snapshot() settings.Settings
	Update(fn func(*settings.Settings) error) (settings.Settings, error)
}

// Holder claims slots for one job or assign at a time.
type Holder interface {
	Hold(h processes.HoldSpec) error
	Release(slot, owner string)
	HeldBy(slot string) (owner string, held bool)
}

// Manager is the process manager surface the API drives.
type Manager interface {
	slots.Runner
	Holder
	ListProcesses() []processes.ProcInfo
	ReadLogWindowScrubbed(source string, sinceMS, untilMS uint64, maxLines int) processes.LogWindow
}

// Options wires the API. Probe, PrimaryCheckout, and BackupsDir default to
// Docker, git, and Hangar's central backups dir.
type Options struct {
	Store            SettingsStore
	Manager          Manager
	ReservationsPath string
	// PrimaryCheckout returns the main checkout of the git worktree at dir,
	// erroring unless dir is a worktree's top level.
	PrimaryCheckout func(dir string) (string, error)
	Probe           slots.Probe
	BackupsDir      func(slotID string) (string, error)
}

const (
	maxLogLines  = 2000
	maxBodyBytes = 64 << 10
)

// API is the agent API. Jobs live in memory.
type API struct {
	opts  Options
	epoch int64

	mu     sync.Mutex
	closed bool
	seq    uint64
	jobs   map[string]*job
	wg     sync.WaitGroup
}

func New(opts Options) *API {
	if opts.Probe == nil {
		opts.Probe = slots.DockerProbe{}
	}
	if opts.PrimaryCheckout == nil {
		opts.PrimaryCheckout = gitPrimaryCheckout
	}
	if opts.BackupsDir == nil {
		opts.BackupsDir = db.ServerBackupsDir
	}
	return &API{opts: opts, epoch: time.Now().UnixMilli(), jobs: map[string]*job{}}
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1", a.index)
	mux.HandleFunc("GET /v1/slots", a.listSlots)
	mux.HandleFunc("POST /v1/slots/{id}/actions", a.postAction)
	mux.HandleFunc("GET /v1/jobs/{id}", a.getJob)
	mux.HandleFunc("GET /v1/jobs/{id}/log", a.getJobLog)
	return mux
}

// closeWait bounds Close so a step that ignores cancellation can't block quit.
var closeWait = 15 * time.Second

// Close stops admission, cancels running jobs, and waits up to closeWait for
// each to stop its current step and record "interrupted".
func (a *API) Close() {
	a.mu.Lock()
	a.closed = true
	for _, j := range a.jobs {
		j.cancel()
	}
	a.mu.Unlock()
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(closeWait):
		slog.Warn("agent api: jobs still running at close; continuing teardown", "waited", closeWait)
	}
}

// SlotJob is the job holding a slot.
type SlotJob struct {
	ID     string `json:"id"`
	Action string `json:"action"`
}

// SlotUsage is one slot's reservation and holding job, for the GUI badge.
type SlotUsage struct {
	ReservedBy *string  `json:"reserved_by"`
	Job        *SlotJob `json:"job"`
}

// SlotUsage reports every configured slot's reservation and holder.
func (a *API) SlotUsage() map[string]SlotUsage {
	all, err := reservations.Read(a.opts.ReservationsPath)
	if err != nil {
		slog.Warn("agent api: reading reservations for slot usage", "err", err)
	}
	out := map[string]SlotUsage{}
	for _, s := range a.opts.Store.Snapshot().Servers {
		var u SlotUsage
		if r, ok := all[s.ID]; ok {
			u.ReservedBy = &r.Worktree
		}
		if owner, held := a.opts.Manager.HeldBy(s.ID); held {
			u.Job = &SlotJob{ID: owner, Action: a.ownerAction(owner)}
		}
		out[s.ID] = u
	}
	return out
}

func (a *API) ownerAction(owner string) string {
	if strings.HasPrefix(owner, "assign:") {
		return "assign"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if j := a.jobs[owner]; j != nil {
		return j.Action
	}
	return ""
}

type procView struct {
	ID       string `json:"id"`
	State    string `json:"state"`
	ExitCode *int   `json:"exit_code"`
}

type slotView struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	Worktree       *string              `json:"worktree"`
	Ports          settings.ServerPorts `json:"ports"`
	ComposeProject string               `json:"compose_project"`
	Enabled        bool                 `json:"enabled"`
	ReservedBy     *string              `json:"reserved_by"`
	HeldBy         *string              `json:"held_by"`
	Processes      []procView           `json:"processes"`
}

func (a *API) listSlots(w http.ResponseWriter, _ *http.Request) {
	all, err := reservations.Read(a.opts.ReservationsPath)
	if err != nil {
		writeErr(w, refuse(http.StatusInternalServerError, "%v", err))
		return
	}
	procs := a.opts.Manager.ListProcesses()
	out := []slotView{}
	for _, s := range a.opts.Store.Snapshot().Servers {
		v := slotView{ID: s.ID, Name: s.Name, Worktree: s.WorktreePath, Ports: s.Ports,
			ComposeProject: s.ComposeProject, Enabled: s.Enabled, Processes: []procView{}}
		if r, ok := all[s.ID]; ok {
			v.ReservedBy = &r.Worktree
		}
		if owner, held := a.opts.Manager.HeldBy(s.ID); held {
			v.HeldBy = &owner
		}
		for _, p := range procs {
			if strings.HasPrefix(p.ID, s.ID+":") {
				v.Processes = append(v.Processes, procView{ID: p.ID, State: p.State, ExitCode: p.ExitCode})
			}
		}
		out = append(out, v)
	}
	writeJSON(w, http.StatusOK, map[string]any{"slots": out})
}

type actionRequest struct {
	Action   string `json:"action"`
	Worktree string `json:"worktree"`
	// Target is the assign destination; JSON null restores "no worktree".
	Target json.RawMessage `json:"target"`
	Name   string          `json:"name"`
}

func (a *API) postAction(w http.ResponseWriter, r *http.Request) {
	var req actionRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, refuse(http.StatusBadRequest, "decoding body: %v", err))
		return
	}
	if !filepath.IsAbs(req.Worktree) {
		writeErr(w, refuse(http.StatusBadRequest, "worktree must be an absolute path"))
		return
	}
	slotID := r.PathValue("id")
	switch req.Action {
	case "assign":
		slot, err := a.assign(r.Context(), slotID, req)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"slot": map[string]any{"id": slot.ID, "worktree": slot.WorktreePath}})
	case slots.ActionBuild, slots.ActionDockerUp, slots.ActionPrepareDB, slots.ActionServe, slots.ActionStop, slots.ActionBackup:
		j, err := a.startJob(slotID, req)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, j.view())
	default:
		writeErr(w, refuse(http.StatusBadRequest, "unknown action %q", req.Action))
	}
}

// admit counts an operation against Close, or refuses once closed.
func (a *API) admit() (uint64, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return 0, refuse(http.StatusServiceUnavailable, "Hangar is shutting down")
	}
	a.seq++
	a.wg.Add(1)
	return a.seq, nil
}

func (a *API) assign(ctx context.Context, slotID string, req actionRequest) (settings.ServerProfile, error) {
	if len(req.Target) == 0 {
		return settings.ServerProfile{}, refuse(http.StatusBadRequest, "assign needs target (a worktree path or null)")
	}
	var target *string
	if err := json.Unmarshal(req.Target, &target); err != nil {
		return settings.ServerProfile{}, refuse(http.StatusBadRequest, "target: %v", err)
	}
	adm, err := a.check(slotID, req.Worktree, false, "")
	if err != nil {
		return settings.ServerProfile{}, err
	}
	if target, err = a.checkTarget(a.opts.Store.Snapshot(), adm, target); err != nil {
		return settings.ServerProfile{}, err
	}

	n, err := a.admit()
	if err != nil {
		return settings.ServerProfile{}, err
	}
	defer a.wg.Done()
	owner := fmt.Sprintf("assign:%d", n)
	if err := a.opts.Manager.Hold(processes.HoldSpec{
		Slot: slotID, Owner: owner, Project: adm.slot.ComposeProject, ServerPort: adm.slot.Ports.Server,
	}); err != nil {
		return settings.ServerProfile{}, refuse(http.StatusConflict, "%v", err)
	}
	defer a.opts.Manager.Release(slotID, owner)
	if err := a.quiet(ctx, adm.slot); err != nil {
		return settings.ServerProfile{}, err
	}
	// Re-check under the hold: nothing can start on the slot now.
	if _, err := a.check(slotID, req.Worktree, false, adm.res.CreatedAt); err != nil {
		return settings.ServerProfile{}, err
	}

	var config *string
	if target != nil {
		if c := settings.DetectFleetConfig(*target); c != "" {
			config = &c
		}
	}
	next, err := a.opts.Store.Update(func(s *settings.Settings) error {
		for i := range s.Servers {
			if s.Servers[i].ID == slotID {
				s.Servers[i].WorktreePath = target
				s.Servers[i].FleetServe.ConfigPath = config
				return nil
			}
		}
		return fmt.Errorf("slot %s vanished", slotID)
	})
	if err != nil {
		return settings.ServerProfile{}, refuse(http.StatusInternalServerError, "saving settings: %v", err)
	}
	slot, _ := findSlot(next, slotID)
	slog.Info("agent api: assigned slot", "slot", slotID, "worktree", string(req.Target))
	return slot, nil
}

func (a *API) startJob(slotID string, req actionRequest) (*job, error) {
	adm, err := a.check(slotID, req.Worktree, true, "")
	if err != nil {
		return nil, err
	}
	dir, err := a.opts.BackupsDir(slotID)
	if err != nil {
		return nil, refuse(http.StatusInternalServerError, "backups dir: %v", err)
	}
	var backupName string
	if req.Action == slots.ActionBackup {
		// CheckBackupNameInDir also refuses these; CodeQL only recognizes this form as a path sanitizer.
		if strings.Contains(req.Name, "..") || strings.ContainsAny(req.Name, `/\`) || filepath.Base(req.Name) != req.Name {
			return nil, refuse(http.StatusBadRequest, "backup name must be a plain file name, without path separators or \"..\"")
		}
		c, err := db.CheckBackupNameInDir(dir, req.Name)
		if err != nil {
			return nil, refuse(http.StatusBadRequest, "%v", err)
		}
		if c.Exists {
			return nil, refuse(http.StatusConflict, "backup %s already exists", c.FinalName)
		}
		backupName = c.FinalName
	}

	spec := processes.HoldSpec{Slot: slotID, Project: adm.slot.ComposeProject, ServerPort: adm.slot.Ports.Server}
	serveID := slots.ProcID(slotID, "fleet-serve")
	switch req.Action {
	case slots.ActionServe:
		spec.Replace = []string{serveID}
	case slots.ActionStop:
		spec.Replace = a.running(slotID)
	case slots.ActionBackup:
		spec.Allow = []string{serveID}
	}

	n, err := a.admit()
	if err != nil {
		return nil, err
	}
	j := &job{
		ID: fmt.Sprintf("job-%x-%d", a.epoch, n), Slot: slotID, Action: req.Action,
		LogChannel: slots.AgentChannel(slotID), State: stateRunning, StartedMS: nowMS(),
	}
	if req.Action == slots.ActionServe {
		j.LogChannel = slots.ServeChannel(slotID)
	}
	spec.Owner = j.ID
	if err := a.opts.Manager.Hold(spec); err != nil {
		a.wg.Done()
		return nil, refuse(http.StatusConflict, "%v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	a.mu.Lock()
	a.jobs[j.ID] = j
	closed := a.closed
	a.mu.Unlock()
	if closed {
		cancel() // Close ran between admit and here; the job records interrupted.
	}

	seq := &slots.Sequencer{
		Runner: jobRunner{a.opts.Manager, ctx}, Probe: a.opts.Probe, Slot: adm.slot, Owner: j.ID, BackupsDir: dir,
		Before: func() error {
			_, err := a.check(slotID, req.Worktree, true, adm.res.CreatedAt)
			return err
		},
		Running: func() []string { return a.running(slotID) },
		OnStep:  j.step,
	}
	go func() {
		defer a.wg.Done()
		defer a.opts.Manager.Release(slotID, j.ID)
		defer cancel()
		res, err := seq.Run(ctx, req.Action, backupName)
		j.finish(res, err, ctx.Err() != nil)
		if err != nil {
			slog.Info("agent api: job ended", "job", j.ID, "action", j.Action, "slot", slotID, "err", err)
		}
	}()
	return j, nil
}

// running lists the slot's running or stopping process ids.
func (a *API) running(slotID string) []string {
	var ids []string
	for _, p := range a.opts.Manager.ListProcesses() {
		if strings.HasPrefix(p.ID, slotID+":") && (p.State == "running" || p.State == "stopping") {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

func (a *API) lookupJob(w http.ResponseWriter, r *http.Request) *job {
	a.mu.Lock()
	j := a.jobs[r.PathValue("id")]
	a.mu.Unlock()
	if j == nil {
		writeErr(w, refuse(http.StatusNotFound, "no job %s", r.PathValue("id")))
	}
	return j
}

func (a *API) getJob(w http.ResponseWriter, r *http.Request) {
	if j := a.lookupJob(w, r); j != nil {
		writeJSON(w, http.StatusOK, j.view())
	}
}

func (a *API) getJobLog(w http.ResponseWriter, r *http.Request) {
	j := a.lookupJob(w, r)
	if j == nil {
		return
	}
	var since uint64
	if s := r.URL.Query().Get("since_ms"); s != "" {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			writeErr(w, refuse(http.StatusBadRequest, "since_ms: %v", err))
			return
		}
		since = v
	}
	v := j.view()
	until := uint64(0)
	if v.EndedMS != nil {
		until = *v.EndedMS
	}
	writeJSON(w, http.StatusOK, a.opts.Manager.ReadLogWindowScrubbed(v.LogChannel, max(since, v.StartedMS), until, maxLogLines))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	var he *httpError
	if errors.As(err, &he) {
		status = he.status
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func nowMS() uint64 { return uint64(time.Now().UnixMilli()) }
