package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/fleetctl"
	"github.com/fleetdm/fleet/tools/hangar/internal/installers"
	"github.com/fleetdm/fleet/tools/hangar/internal/netinfo"
	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
	"github.com/fleetdm/fleet/tools/hangar/internal/tuf"
)

// InstallersService builds fleetd installers for a fleetctl context's Fleet
// server (or a typed-in URL and secret), enrolling into a fleet picked from a
// scan of that server. One build runs at a time, as a tracked process.
type InstallersService struct {
	pm *processes.Manager

	// starting is set while StartBuild is between its checks and pm.Start (a
	// fleetctl --version can take a moment), so a double-click can't run two.
	startingMu sync.Mutex
	starting   bool
	// manifests tracks the previous build's manifest update, which runs just
	// after its process exits. The next build waits for it: the update claims
	// files replaced since its build started, so it would claim the next build's.
	manifests sync.WaitGroup
}

// NewInstallersService wires the service to the shared process manager.
func NewInstallersService(pm *processes.Manager) *InstallersService {
	return &InstallersService{pm: pm}
}

// InstallersContextInfo is what the tab needs for a context (or a manual
// build) before any scan: where its packages go and which fleetctl builds them.
type InstallersContextInfo struct {
	Address        string `json:"address"`
	OutputDir      string `json:"output_dir"`
	OutputErr      string `json:"output_err"`
	Fleetctl       string `json:"fleetctl"`
	FleetctlExists bool   `json:"fleetctl_exists"`
	// FleetctlServer names the Hangar server whose fleetctl builds this
	// context's packages; "" means Hangar's default fleetctl.
	FleetctlServer string `json:"fleetctl_server"`
}

// InstallersLocalTUF says whether a local TUF build can be targeted, and the
// channels its repo publishes.
type InstallersLocalTUF struct {
	Available bool                `json:"available"`
	Reason    string              `json:"reason"`
	URL       string              `json:"url"`
	Channels  installers.Channels `json:"channels"`
}

// InstallersOfficialChannels are the channels on updates.fleetdm.com.
func (s *InstallersService) InstallersOfficialChannels() installers.Channels {
	return installers.OfficialChannels()
}

// InstallersContextInfo resolves a context's output folder and fleetctl. A
// context pointing at one of Hangar's servers builds with that server's
// fleetctl; anything else (a remote server, a manual build) uses the fleetctl
// the rest of Hangar uses.
func (s *InstallersService) InstallersContextInfo(contextName string, manual bool) (InstallersContextInfo, error) {
	cur, err := loadSettings()
	if err != nil {
		return InstallersContextInfo{}, err
	}
	if manual {
		contextName = ""
	}
	var info InstallersContextInfo
	if contextName != "" {
		if ctxInfo, err := fleetctl.ReadContext(fleetctl.DefaultConfigPath()); err == nil {
			for _, c := range ctxInfo.Contexts {
				if c.Name == contextName && c.Address != nil {
					info.Address = *c.Address
				}
			}
		}
	}
	if dir, err := installers.OutputDir(fileServerDir(cur), primaryRepo(cur), contextName); err != nil {
		info.OutputErr = err.Error()
	} else {
		info.OutputDir = dir
	}
	if srv := installers.ServerForAddress(cur.Servers, info.Address); srv != nil && srv.WorktreePath != nil && *srv.WorktreePath != "" {
		info.Fleetctl = filepath.Join(*srv.WorktreePath, "build", "fleetctl")
		info.FleetctlServer = srv.Name
	} else {
		settingsPath := ""
		if cur.FleetctlPath != nil {
			settingsPath = *cur.FleetctlPath
		}
		info.Fleetctl = fleetctl.ResolveBinary(primaryRepo(cur), settingsPath).Path
	}
	if info.Fleetctl != "" {
		_, statErr := os.Stat(info.Fleetctl)
		info.FleetctlExists = statErr == nil
	}
	return info, nil
}

// InstallersPackagesRoot is the packages folder every context's folder sits
// in (what sharing serves).
func (s *InstallersService) InstallersPackagesRoot() (string, error) {
	cur, err := loadSettings()
	if err != nil {
		return "", err
	}
	return installers.OutputDir(fileServerDir(cur), primaryRepo(cur), "")
}

func fileServerDir(cur settings.Settings) string {
	if cur.PythonServer.Directory == nil {
		return ""
	}
	return *cur.PythonServer.Directory
}

func primaryRepo(cur settings.Settings) string {
	if len(cur.Servers) == 0 || cur.Servers[0].WorktreePath == nil {
		return ""
	}
	return *cur.Servers[0].WorktreePath
}

// InstallersLocalTUF reports whether Hangar's local TUF repo (server 1's
// test_tuf, served at the TUF tab's URL) can be targeted, and its channels.
func (s *InstallersService) InstallersLocalTUF() InstallersLocalTUF {
	out := InstallersLocalTUF{Channels: installers.Channels{Orbit: []string{}, Desktop: []string{}, Osqueryd: []string{}}}
	cur, err := loadSettings()
	if err != nil {
		out.Reason = err.Error()
		return out
	}
	repo, err := primaryRepoPath()
	if err != nil {
		out.Reason = err.Error()
		return out
	}
	b, err := os.ReadFile(filepath.Join(repo, tuf.AssetsDir, "repository", "targets.json"))
	if err != nil {
		out.Reason = "no local TUF repo yet — build one in the TUF tab"
		return out
	}
	ch, err := installers.TUFChannels(b)
	if err != nil {
		out.Reason = err.Error()
		return out
	}
	out.Channels = ch
	out.URL = strings.TrimSpace(cur.Tuf.TufURL)
	if out.URL == "" {
		out.Reason = "set the TUF URL in the TUF tab"
		return out
	}
	out.Available = true
	return out
}

// InstallersScan reads a server's public URL, license and fleets (with which
// ones have an enroll secret) through the named fleetctl context.
func (s *InstallersService) InstallersScan(contextName string) (installers.ScanResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := scanContext(ctx, contextName)
	if err != nil {
		return installers.ScanResult{}, err
	}
	return res, nil
}

func scanContext(ctx context.Context, contextName string) (installers.ScanResult, error) {
	if strings.TrimSpace(contextName) == "" {
		return installers.ScanResult{}, errors.New("pick a fleetctl context")
	}
	creds, err := fleetctl.ReadCredentials(fleetctl.DefaultConfigPath(), contextName)
	if err != nil {
		return installers.ScanResult{}, err
	}
	client, err := installers.NewClient(creds)
	if err != nil {
		return installers.ScanResult{}, fmt.Errorf("context %s: %w", contextName, err)
	}
	res, err := client.Scan(ctx)
	if err != nil {
		return installers.ScanResult{}, fmt.Errorf("scan via context %s: %w", contextName, err)
	}
	res.Context = contextName
	return res, nil
}

// InstallersFleetSecret returns the enroll secret of fleet teamID (nil =
// Unassigned) through a fleetctl context, to fill the card's secret field.
func (s *InstallersService) InstallersFleetSecret(contextName string, teamID *uint) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	scan, err := scanContext(ctx, contextName)
	if err != nil {
		return "", err
	}
	secret, _, err := installers.SecretFor(scan, teamID)
	return secret, err
}

// InstallersCheckExtraFlags validates extra fleetctl flags as typed: "" when
// they're usable, otherwise why not.
func (s *InstallersService) InstallersCheckExtraFlags(flags string) string {
	if _, err := installers.ParseExtraFlags(flags); err != nil {
		return err.Error()
	}
	return ""
}

// InstallersStartBuild builds opts' platforms into the context's packages
// folder, with exactly the Fleet URL and enroll secret in target. Files are
// written by fleetctl and recorded in that folder's manifest.json once the
// build ends.
func (s *InstallersService) InstallersStartBuild(contextName string, manual bool, target settings.InstallerTarget, opts settings.InstallerOptions) error {
	if len(opts.Platforms) == 0 {
		return errors.New("select at least one platform")
	}
	if err := installers.ValidateConfig(opts, target); err != nil {
		return err
	}
	s.startingMu.Lock()
	if s.starting || s.isProcRunning(installers.ProcID) {
		s.startingMu.Unlock()
		return errors.New("a build is already running")
	}
	s.starting = true
	s.startingMu.Unlock()
	defer func() {
		s.startingMu.Lock()
		s.starting = false
		s.startingMu.Unlock()
	}()

	if manual {
		contextName = ""
	}
	info, err := s.InstallersContextInfo(contextName, manual)
	if err != nil {
		return err
	}
	if info.OutputErr != "" {
		return errors.New(info.OutputErr)
	}
	if !info.FleetctlExists {
		return fmt.Errorf("fleetctl not built (%s) — build it from the Server tab", info.Fleetctl)
	}
	in := installers.BuildInputs{FleetURL: strings.TrimSpace(target.FleetURL), EnrollSecret: strings.TrimSpace(target.EnrollSecret)}
	if in.FleetURL == "" {
		return errors.New("enter a Fleet URL")
	}
	if in.EnrollSecret == "" {
		return errors.New("enter an enroll secret (or pick a fleet to fill it in)")
	}
	if in.ExtraFlags, err = installers.ParseExtraFlags(opts.ExtraFlags); err != nil {
		return err
	}
	teamName := target.TeamName
	if teamName == "" {
		teamName = "manual enroll secret"
	}
	if opts.UpdateSource == settings.UpdateSourceLocalTUF {
		local := s.InstallersLocalTUF()
		if !local.Available {
			return fmt.Errorf("local TUF unavailable: %s", local.Reason)
		}
		repo, err := primaryRepoPath()
		if err != nil {
			return err
		}
		roots, err := fleetctl.RunCapture(info.Fleetctl, "", []string{"updates", "roots", "--path", filepath.Join(repo, tuf.AssetsDir)}, nil, "", 30_000)
		if err != nil {
			return fmt.Errorf("read local TUF root keys: %w", err)
		}
		if roots.ExitCode == nil || *roots.ExitCode != 0 {
			return fmt.Errorf("read local TUF root keys: %s", strings.TrimSpace(roots.Stderr))
		}
		// Compact in case fleetctl ever pretty-prints: args must be single-line.
		var compact bytes.Buffer
		if err := json.Compact(&compact, []byte(strings.TrimSpace(roots.Stdout))); err != nil {
			return fmt.Errorf("read local TUF root keys: %w", err)
		}
		in.UpdateURL = local.URL
		in.UpdateRoots = compact.String()
	}
	args := installers.CommonArgs(opts, in)
	if err := installers.ValidateArgs(args); err != nil {
		return err
	}

	if err := os.MkdirAll(info.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", info.OutputDir, err)
	}
	dataDir, err := paths.DataDir()
	if err != nil {
		return err
	}
	// Replaced atomically: bash reads a script as it runs, so rewriting it in
	// place could corrupt a build that's just finishing.
	script := filepath.Join(dataDir, "installers-build.sh")
	if err := installers.WriteFileAtomic(script, []byte(installers.BuildScript), 0o755); err != nil {
		return fmt.Errorf("write build script: %w", err)
	}

	jobs := installers.Jobs(installers.FleetSlug(target), info.OutputDir, opts)
	entry := installers.ManifestEntry{
		BuiltAt:         time.Now().UTC().Format(time.RFC3339),
		Context:         contextName,
		FleetURL:        in.FleetURL,
		Team:            teamName,
		FleetDesktop:    opts.FleetDesktop,
		EnableScripts:   opts.EnableScripts,
		Debug:           opts.Debug,
		UpdateURL:       in.UpdateURL,
		OrbitChannel:    channelOrStable(opts.OrbitChannel),
		DesktopChannel:  channelOrStable(opts.DesktopChannel),
		OsquerydChannel: channelOrStable(opts.OsquerydChannel),
		FleetctlVersion: fleetctlVersion(info.Fleetctl),
		ExtraFlags:      strings.Join(in.ExtraFlags, " "),
	}
	if entry.UpdateURL == "" {
		entry.UpdateURL = "https://updates.fleetdm.com"
	}

	// The tab tells which context a build is for by this label.
	label := "fleetd packages (manual)"
	if contextName != "" {
		label = "fleetd packages (" + contextName + ")"
	}
	s.manifests.Wait()
	_ = s.pm.ClearLogChannel(installers.LogChannel)
	// Tells this build's outputs from files an earlier build left in place.
	before := installers.Snapshot(jobs)
	if err := s.pm.Start(installers.ProcID, processes.StartArgs{
		Label:   label,
		Cwd:     info.OutputDir,
		Program: "bash",
		Args:    []string{script},
		Env: []processes.EnvPair{
			{Key: "HANGAR_FLEETCTL", Value: info.Fleetctl},
			{Key: "HANGAR_JOBS", Value: installers.EncodeJobs(jobs)},
			{Key: "HANGAR_COMMON_ARGS", Value: strings.Join(args, "\n")},
		},
		LogChannel: installers.LogChannel,
	}); err != nil {
		return err
	}
	done := s.pm.Done(installers.ProcID)
	s.manifests.Add(1)
	go func() {
		defer s.manifests.Done()
		<-done
		if _, err := installers.UpdateManifest(info.OutputDir, jobs, before, entry); err != nil {
			slog.Warn("installers: update manifest", "dir", info.OutputDir, "err", err)
		}
	}()
	return nil
}

// InstallersPlannedFiles maps each platform key to the file name a build for
// target with opts would produce, so the tab shows names without
// re-implementing them.
func (s *InstallersService) InstallersPlannedFiles(target settings.InstallerTarget, opts settings.InstallerOptions) map[string]string {
	out := make(map[string]string, len(installers.Platforms))
	fleet := installers.FleetSlug(target)
	for _, p := range installers.Platforms {
		out[p.Key] = installers.FileName(fleet, p, opts)
	}
	return out
}

// InstallersStopBuild cancels the running build.
func (s *InstallersService) InstallersStopBuild() error {
	return s.pm.Stop(installers.ProcID)
}

// InstallersListOutputs lists the installers in a context's packages folder.
func (s *InstallersService) InstallersListOutputs(contextName string, manual bool) ([]installers.OutputFile, error) {
	info, err := s.InstallersContextInfo(contextName, manual)
	if err != nil {
		return nil, err
	}
	if info.OutputErr != "" {
		return []installers.OutputFile{}, nil
	}
	return installers.ListOutputs(info.OutputDir)
}

func (s *InstallersService) isProcRunning(id string) bool {
	for _, p := range s.pm.ListProcesses() {
		if p.ID == id && (p.State == "running" || p.State == "stopping") {
			return true
		}
	}
	return false
}

func channelOrStable(ch string) string {
	if ch == "" {
		return "stable"
	}
	return ch
}

// fleetctlVersion is the first line of `fleetctl --version`, or "" if it
// can't be read.
func fleetctlVersion(bin string) string {
	run, err := fleetctl.RunCapture(bin, "", []string{"--version"}, nil, "", 10_000)
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(strings.TrimSpace(run.Stdout), "\n")
	return line
}

func loadSettings() (settings.Settings, error) {
	dir, err := paths.ConfigDir()
	if err != nil {
		return settings.Settings{}, err
	}
	return settings.Load(dir)
}

// pythonProcID is the frontend's id for python http.server.
const pythonProcID = "python-server"

// InstallersShareURL is where python http.server serves the packages folder,
// or "" when it isn't running with the folder inside its directory. Sharing is
// python http.server's job: the Installers tab starts and stops the same
// process as the Server tab, with the same folder and port.
func (s *InstallersService) InstallersShareURL() string {
	root, err := s.InstallersPackagesRoot()
	if err != nil {
		return ""
	}
	return s.pythonServedURL(root, netinfo.LanIP())
}

// pythonServedURL is where a running python http.server serves root, or "".
func (s *InstallersService) pythonServedURL(root, lanIP string) string {
	for _, p := range s.pm.ListProcesses() {
		if p.ID == pythonProcID && (p.State == "running" || p.State == "stopping") {
			return installers.PythonServedURL(p.Command, p.Cwd, root, lanIP)
		}
	}
	return ""
}

// InstallersSharedFiles lists the installers in the packages folder, with URLs
// and install commands while python http.server serves it.
func (s *InstallersService) InstallersSharedFiles() ([]installers.SharedFile, error) {
	root, err := s.InstallersPackagesRoot()
	if err != nil {
		return []installers.SharedFile{}, nil
	}
	return installers.ListShared(root, s.pythonServedURL(root, netinfo.LanIP()))
}
