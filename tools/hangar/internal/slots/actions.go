package slots

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/db"
	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

// Actions a job can run.
const (
	ActionBuild     = "build"
	ActionDockerUp  = "docker-up"
	ActionPrepareDB = "prepare-db"
	ActionServe     = "serve"
	ActionStop      = "stop"
	ActionBackup    = "backup"
)

// Runner is the slice of the process manager a Sequencer drives.
type Runner interface {
	Start(id string, a processes.StartArgs) (processes.Invocation, error)
	// Wait returns the invocation's own exit code, -1 when it was signaled
	// or stopped from outside.
	Wait(ctx context.Context, inv processes.Invocation) (exitCode int, err error)
	Stop(id, owner string) error
	ComposeDown(id, cwd, project, owner string) error
}

// ErrBackupExists means the backup's final name is taken.
var ErrBackupExists = errors.New("backup already exists")

// Tunables, shortened by tests.
var (
	pollInterval = 500 * time.Millisecond
	portFreeWait = 5 * time.Second
	serveTimeout = 10 * time.Minute
	stopWait     = 30 * time.Second
)

// Result is what a finished action reports beyond its steps.
type Result struct {
	BackupPath string
}

// Sequencer runs one action's steps on a slot as one owner.
type Sequencer struct {
	Runner     Runner
	Probe      Probe
	Slot       settings.ServerProfile
	Owner      string
	BackupsDir string
	// Before runs before every step; an error aborts the action.
	Before func() error
	// Running lists the slot's running or stopping process ids.
	Running func() []string
	// OnStep observes each step; exitCode is nil when it starts.
	OnStep func(id string, exitCode *int)
	Now    func() time.Time
}

// Run performs action. backupName is the validated final file name for
// ActionBackup and ignored otherwise.
func (s *Sequencer) Run(ctx context.Context, action, backupName string) (Result, error) {
	if s.Slot.WorktreePath == nil || *s.Slot.WorktreePath == "" {
		return Result{}, errors.New("slot has no worktree")
	}
	switch action {
	case ActionBuild:
		for _, st := range BuildChain(s.Slot) {
			if err := s.step(ctx, st.ID, processes.StartArgs{Label: st.Label, Program: st.Program, Args: st.Args}); err != nil {
				return Result{}, err
			}
		}
		return Result{}, nil
	case ActionDockerUp:
		return Result{}, s.dockerUp(ctx)
	case ActionPrepareDB:
		return s.prepareDB(ctx)
	case ActionServe:
		return Result{}, s.serve(ctx)
	case ActionStop:
		return Result{}, s.stop(ctx)
	case ActionBackup:
		path, err := s.backup(ctx, backupName, nil)
		return Result{BackupPath: path}, err
	}
	return Result{}, fmt.Errorf("unknown action %q", action)
}

func (s *Sequencer) worktree() string { return *s.Slot.WorktreePath }

func (s *Sequencer) report(id string, code *int) {
	if s.OnStep != nil {
		s.OnStep(id, code)
	}
}

func (s *Sequencer) before() error {
	if s.Before == nil {
		return nil
	}
	return s.Before()
}

func (s *Sequencer) start(ctx context.Context, id string, a processes.StartArgs) (processes.Invocation, error) {
	if err := s.before(); err != nil {
		return processes.Invocation{}, err
	}
	a.Cwd, a.Owner = s.worktree(), s.Owner
	if a.LogChannel == "" {
		a.LogChannel = AgentChannel(s.Slot.ID)
	}
	if err := ctx.Err(); err != nil {
		return processes.Invocation{}, err
	}
	inv, err := s.Runner.Start(id, a)
	if err != nil {
		return processes.Invocation{}, fmt.Errorf("starting %s: %w", id, err)
	}
	s.report(id, nil)
	return inv, nil
}

// step runs one process to completion. Only its own exit code 0 succeeds.
func (s *Sequencer) step(ctx context.Context, id string, a processes.StartArgs) error {
	inv, err := s.start(ctx, id, a)
	if err != nil {
		return err
	}
	code, err := s.Runner.Wait(ctx, inv)
	if err != nil {
		if ctx.Err() != nil {
			code = s.stopAndWait(inv)
			s.report(id, &code)
			return ctx.Err()
		}
		return fmt.Errorf("waiting for %s: %w", id, err)
	}
	s.report(id, &code)
	if code != 0 {
		return fmt.Errorf("%s exited with code %d", id, code)
	}
	return nil
}

// stopAndWait stops a step and waits for it, so nothing the job spawned
// outlives it.
func (s *Sequencer) stopAndWait(inv processes.Invocation) int {
	if err := s.Runner.Stop(inv.ID, s.Owner); err != nil {
		slog.Warn("agent job: stopping step", "id", inv.ID, "err", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), stopWait)
	defer cancel()
	code, err := s.Runner.Wait(ctx, inv)
	if err != nil {
		slog.Warn("agent job: step outlived stop", "id", inv.ID, "err", err)
		return -1
	}
	return code
}

func (s *Sequencer) dockerUp(ctx context.Context) error {
	id := ProcID(s.Slot.ID, "docker-compose-up")
	if err := s.before(); err != nil {
		return err
	}
	if err := s.Runner.ComposeDown(id, s.worktree(), s.Slot.ComposeProject, s.Owner); err != nil {
		return fmt.Errorf("compose down: %w", err)
	}
	return s.step(ctx, id, processes.StartArgs{
		Label: "docker compose up -d", Program: "docker",
		Args: DockerUpArgs(s.Slot), Env: DockerEnv(s.Slot),
	})
}

func (s *Sequencer) prepareDB(ctx context.Context) (Result, error) {
	if err := CheckMount(ctx, s.Probe, s.worktree(), s.Slot.ComposeProject); err != nil {
		return Result{}, err
	}
	n, err := s.Probe.FleetTableCount(ctx, MySQLContainer(s.Slot.ComposeProject))
	if err != nil {
		return Result{}, fmt.Errorf("counting fleet tables: %w", err)
	}
	var res Result
	if n > 0 {
		name := fmt.Sprintf("checkpoint-%s-%s.sql.gz", s.Slot.ID, s.now().UTC().Format("20060102T150405Z"))
		note := "checkpoint before prepare-db"
		path, err := s.backup(ctx, name, &note)
		if err != nil {
			return Result{}, fmt.Errorf("checkpoint: %w", err)
		}
		res.BackupPath = path
	}
	return res, s.step(ctx, ProcID(s.Slot.ID, "fleet-prepare-db"), processes.StartArgs{
		Label: "fleet prepare db --dev", Program: "./build/fleet", Args: PrepareDBArgs(s.Slot),
	})
}

func (s *Sequencer) serve(ctx context.Context) error {
	if err := CheckMount(ctx, s.Probe, s.worktree(), s.Slot.ComposeProject); err != nil {
		return err
	}
	id := ProcID(s.Slot.ID, "fleet-serve")
	if err := s.before(); err != nil {
		return err
	}
	if err := s.Runner.Stop(id, s.Owner); err != nil {
		return fmt.Errorf("stopping running serve: %w", err)
	}
	// Free port first, so the readiness check below can only see our serve.
	port := s.Slot.Ports.Server
	for deadline := time.Now().Add(portFreeWait); s.Probe.Listening(port); {
		if time.Now().After(deadline) {
			return fmt.Errorf("port %d still has a listener", port)
		}
		if err := sleep(ctx, pollInterval); err != nil {
			return err
		}
	}

	inv, err := s.start(ctx, id, processes.StartArgs{
		Label: ServeLabel(s.Slot), Program: "./build/fleet",
		Args: ServeArgs(s.Slot), Env: ServeEnv(s.Slot), LogChannel: ServeChannel(s.Slot.ID),
	})
	if err != nil {
		return err
	}
	waitCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exited := make(chan int, 1)
	go func() {
		code, err := s.Runner.Wait(waitCtx, inv)
		if err != nil {
			if waitCtx.Err() != nil {
				return
			}
			code = -1
		}
		exited <- code
	}()
	exitErr := func(code int) error {
		s.report(id, &code)
		return fmt.Errorf("%s exited with code %d before listening on %d", id, code, port)
	}
	tick := time.NewTicker(pollInterval)
	defer tick.Stop()
	deadline := time.After(serveTimeout)
	for {
		select {
		case code := <-exited:
			return exitErr(code)
		case <-ctx.Done():
			cancel()
			code := s.stopAndWait(inv)
			s.report(id, &code)
			return ctx.Err()
		case <-deadline:
			cancel()
			code := s.stopAndWait(inv)
			s.report(id, &code)
			return fmt.Errorf("%s not listening on %d after %s", id, port, serveTimeout)
		case <-tick.C:
			if !s.Probe.Listening(port) {
				continue
			}
			// Ready only if this invocation is still the one running.
			select {
			case code := <-exited:
				return exitErr(code)
			default:
				return nil
			}
		}
	}
}

func (s *Sequencer) stop(ctx context.Context) error {
	if err := s.before(); err != nil {
		return err
	}
	ids := s.Running()
	for _, id := range ids {
		if err := s.Runner.Stop(id, s.Owner); err != nil {
			return fmt.Errorf("stopping %s: %w", id, err)
		}
	}
	for deadline := time.Now().Add(stopWait); ; {
		left := slices.DeleteFunc(s.Running(), func(id string) bool { return !slices.Contains(ids, id) })
		if len(left) == 0 {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("still running after stop: %v", left)
		}
		if err := sleep(ctx, pollInterval); err != nil {
			return err
		}
	}
	if err := s.before(); err != nil {
		return err
	}
	if err := s.Runner.ComposeDown(ProcID(s.Slot.ID, "docker-compose-up"), s.worktree(), s.Slot.ComposeProject, s.Owner); err != nil {
		return fmt.Errorf("compose down: %w", err)
	}
	return nil
}

// backup dumps to an exclusive temp file, verifies the gzip stream to EOF,
// then links it to name, which fails rather than replace an existing file.
// A sidecar failure keeps the published dump.
func (s *Sequencer) backup(ctx context.Context, name string, note *string) (string, error) {
	dir, err := db.EnsureDir(s.BackupsDir)
	if err != nil {
		return "", err
	}
	final := filepath.Join(dir, name)
	if _, err := os.Lstat(final); err == nil {
		return "", fmt.Errorf("%w: %s", ErrBackupExists, final)
	}
	tmp, err := os.CreateTemp(dir, "."+name+".tmp-*")
	if err != nil {
		return "", fmt.Errorf("creating backup temp file: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	if err := s.step(ctx, ProcID(s.Slot.ID, "db-backup"), processes.StartArgs{
		Label: "db-backup " + name, Program: "bash", Args: []string{"-c", DBBackupCommand(s.Slot, tmpPath)},
	}); err != nil {
		return "", err
	}
	if err := verifyGzip(tmpPath); err != nil {
		return "", err
	}
	if err := os.Link(tmpPath, final); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%w: %s", ErrBackupExists, final)
		}
		return "", fmt.Errorf("publishing backup: %w", err)
	}
	if err := db.SaveBackupMeta(final, s.Slot.Branch, note, uint64(s.now().UnixMilli())); err != nil {
		slog.Warn("agent job: backup sidecar", "path", final, "err", err)
	}
	return final, nil
}

func verifyGzip(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("backup isn't gzip: %w", err)
	}
	if _, err := io.Copy(io.Discard, zr); err != nil {
		return fmt.Errorf("backup gzip stream: %w", err)
	}
	return zr.Close()
}

func (s *Sequencer) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
