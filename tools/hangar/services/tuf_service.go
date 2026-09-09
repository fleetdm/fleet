package services

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
	"github.com/fleetdm/fleet/tools/hangar/internal/troubleshoot"
	"github.com/fleetdm/fleet/tools/hangar/internal/tuf"
)

// TufService drives a local TUF test repo + fleetd installer generation by
// running tools/tuf/test/main.sh (via the process engine, so its output streams
// to the Logs ring), and offers server-status / kill / delete-assets helpers.
// A separate top-level tab drives it; it's independent of the active server.
type TufService struct {
	pm *processes.Manager
}

// NewTufService wires the service to the shared process manager (the build runs
// as a tracked process so the tab can tail its output live).
func NewTufService(pm *processes.Manager) *TufService {
	return &TufService{pm: pm}
}

// TufServerStatus reports whether the local TUF server answers on :8081.
func (s *TufService) TufServerStatus() tuf.ServerStatus {
	return tuf.ProbeServer(tuf.DefaultPort)
}

// TufAssetsExist reports whether a generated TUF repo is present (so the UI can
// only offer "Delete assets" when there's something to delete).
func (s *TufService) TufAssetsExist() bool {
	repo, err := primaryRepoPath()
	if err != nil {
		return false
	}
	return tuf.AssetsExist(repo)
}

// TufStartBuild runs main.sh with the config's env from the primary repo. The
// run streams to the tuf-build log channel; when it exits the TUF file-server it
// spawned keeps serving (status flips up).
func (s *TufService) TufStartBuild(cfg settings.TufConfig) error {
	if len(cfg.Platforms) == 0 {
		return errors.New("select at least one platform")
	}
	if strings.TrimSpace(cfg.EnrollSecret) == "" {
		return errors.New("enroll secret is required to generate packages")
	}
	repo, err := primaryRepoPath()
	if err != nil {
		return err
	}
	script := tuf.MainScriptPath(repo)
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("TUF test script not found at %s", script)
	}

	// Bail before touching anything if a build is already in flight — otherwise
	// a re-entrant request would rm test_tuf out from under it and only then
	// fail on the duplicate pm.Start.
	if s.isProcRunning(tuf.ProcID) {
		return errors.New("a TUF build is already running")
	}

	// create_repository prompts interactively to remove an existing test_tuf,
	// which would hang a non-interactive build. Clear it up front — the same
	// `rm -rf test_tuf` from the manual workflow, i.e. auto-answering "yes".
	_ = tuf.DeleteAssets(repo)

	// The file-server must be up DURING the build: fleetctl package (gen_pkgs)
	// reaches the TUF --update-url to walk the root chain. Start it before the
	// build (idempotent); it serves files as create_repository writes them.
	if err := s.ensureFileServer(repo); err != nil {
		return fmt.Errorf("start TUF server: %w", err)
	}

	// Clear the previous build's log ring so the panel shows only this run.
	_ = s.pm.ClearLogChannel(tuf.LogChannel)
	return s.pm.Start(tuf.ProcID, processes.StartArgs{
		Label:      "TUF build",
		Cwd:        repo,
		Program:    "bash",
		Args:       []string{script},
		Env:        tuf.EnvFor(cfg),
		LogChannel: tuf.LogChannel,
	})
}

// TufStopBuild cancels a running build.
func (s *TufService) TufStopBuild() error {
	return s.pm.Stop(tuf.ProcID)
}

// TufStartServer runs the TUF file-server for the manual "Start server" button.
// Requires a built repo (the build starts the server itself).
func (s *TufService) TufStartServer() error {
	repo, err := primaryRepoPath()
	if err != nil {
		return err
	}
	if !tuf.AssetsExist(repo) {
		return errors.New("no TUF repo yet — run a build first")
	}
	return s.ensureFileServer(repo)
}

// isProcRunning reports whether the process engine currently tracks id as
// running (or stopping).
func (s *TufService) isProcRunning(id string) bool {
	for _, p := range s.pm.ListProcesses() {
		if p.ID == id && (p.State == "running" || p.State == "stopping") {
			return true
		}
	}
	return false
}

// ensureFileServer starts the managed TUF file-server (go run ./tools/file-server)
// if it isn't already running. Idempotent — a no-op when it's up. The file-server
// tolerates a not-yet-populated repo dir (os.DirFS resolves per request), so it's
// safe to start before create_repository writes the metadata.
func (s *TufService) ensureFileServer(repo string) error {
	if s.isProcRunning(tuf.ServerProcID) {
		return nil
	}
	_ = s.pm.ClearLogChannel(tuf.ServerLogChannel)
	return s.pm.Start(tuf.ServerProcID, processes.StartArgs{
		Label:      "TUF file-server",
		Cwd:        repo,
		Program:    "go",
		Args:       tuf.FileServerArgs(),
		LogChannel: tuf.ServerLogChannel,
	})
}

// TufKillServer stops Hangar's managed file-server and reaps any other process
// bound to the TUF port (e.g. an orphan from a prior run or the CLI).
func (s *TufService) TufKillServer() ([]troubleshoot.KillOutcome, error) {
	_ = s.pm.Stop(tuf.ServerProcID)
	procs, err := troubleshoot.ScanPort(tuf.DefaultPort)
	if err != nil {
		return nil, err
	}
	outcomes := make([]troubleshoot.KillOutcome, 0, len(procs))
	for _, p := range procs {
		outcomes = append(outcomes, troubleshoot.KillPID(p.PID))
	}
	return outcomes, nil
}

// TufDeleteAssets removes the generated TUF repo (<repo>/test_tuf).
func (s *TufService) TufDeleteAssets() error {
	repo, err := primaryRepoPath()
	if err != nil {
		return err
	}
	return tuf.DeleteAssets(repo)
}
