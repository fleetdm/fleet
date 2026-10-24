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
	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
	"github.com/fleetdm/fleet/tools/hangar/internal/tuf"
)

// InstallersService builds fleetd installers for each configured server with
// that server's own fleetctl, enrolling into a fleet picked from a scan of the
// server. One build per server runs at a time, as a tracked process.
type InstallersService struct {
	pm *processes.Manager

	// starting holds servers whose StartBuild is between its checks and
	// pm.Start (a scan can take seconds), so a double-click can't run two.
	startingMu sync.Mutex
	starting   map[string]bool
}

// NewInstallersService wires the service to the shared process manager.
func NewInstallersService(pm *processes.Manager) *InstallersService {
	return &InstallersService{pm: pm, starting: map[string]bool{}}
}

// InstallersServerInfo is what a server card needs before any scan.
type InstallersServerInfo struct {
	ServerID       string `json:"server_id"`
	Slug           string `json:"slug"`
	MatchedContext string `json:"matched_context"`
	Fleetctl       string `json:"fleetctl"`
	FleetctlExists bool   `json:"fleetctl_exists"`
	OutputDir      string `json:"output_dir"`
}

// InstallersLocalTUF says whether a local TUF build can be targeted, and the
// channels its repo publishes.
type InstallersLocalTUF struct {
	Available bool                `json:"available"`
	Reason    string              `json:"reason"`
	URL       string              `json:"url"`
	Channels  installers.Channels `json:"channels"`
}

// InstallersDefaultOutputDir is ~/Documents/fleetd-installers.
func (s *InstallersService) InstallersDefaultOutputDir() string {
	return defaultInstallersDir()
}

func defaultInstallersDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", "Documents", installers.DefaultDirName)
	}
	return filepath.Join(home, "Documents", installers.DefaultDirName)
}

func resolveOutputRoot(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return defaultInstallersDir()
	}
	return paths.Expand(strings.TrimSpace(dir))
}

// InstallersOfficialChannels are the channels on updates.fleetdm.com.
func (s *InstallersService) InstallersOfficialChannels() installers.Channels {
	return installers.OfficialChannels()
}

// InstallersServerInfo resolves a server's slug, fleetctl binary, output
// folder and the fleetctl context matching its port.
func (s *InstallersService) InstallersServerInfo(serverID, outputDir string) (InstallersServerInfo, error) {
	srv, err := loadServer(serverID)
	if err != nil {
		return InstallersServerInfo{}, err
	}
	slug := installers.ServerSlug(srv)
	info := InstallersServerInfo{
		ServerID:  srv.ID,
		Slug:      slug,
		OutputDir: filepath.Join(resolveOutputRoot(outputDir), slug),
	}
	if srv.WorktreePath != nil && *srv.WorktreePath != "" {
		info.Fleetctl = filepath.Join(*srv.WorktreePath, "build", "fleetctl")
		_, statErr := os.Stat(info.Fleetctl)
		info.FleetctlExists = statErr == nil
	}
	if ctxInfo, err := fleetctl.ReadContext(fleetctl.DefaultConfigPath()); err == nil {
		addrs := map[string]string{}
		var order []string
		for _, c := range ctxInfo.Contexts {
			if c.Address != nil {
				addrs[c.Name] = *c.Address
			}
			order = append(order, c.Name)
		}
		info.MatchedContext = installers.MatchContext(addrs, order, srv.Ports.Server)
	}
	return info, nil
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
		return installers.ScanResult{}, errors.New("pick a fleetctl context for this server")
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

// InstallersStartBuild builds cfg's platforms for serverID into
// <outputDir>/<server>/. The enroll secret for cfg.TeamID is fetched fresh;
// files are written by fleetctl and recorded in that folder's manifest.json
// once the build ends.
func (s *InstallersService) InstallersStartBuild(serverID string, cfg settings.InstallerServerConfig, outputDir string) error {
	if len(cfg.Platforms) == 0 {
		return errors.New("select at least one platform")
	}
	if err := installers.ValidateConfig(cfg); err != nil {
		return err
	}
	procID := installers.ProcID(serverID)
	s.startingMu.Lock()
	if s.starting[serverID] || s.isProcRunning(procID) {
		s.startingMu.Unlock()
		return errors.New("a build is already running for this server")
	}
	s.starting[serverID] = true
	s.startingMu.Unlock()
	defer func() {
		s.startingMu.Lock()
		delete(s.starting, serverID)
		s.startingMu.Unlock()
	}()

	info, err := s.InstallersServerInfo(serverID, outputDir)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(info.OutputDir) {
		return fmt.Errorf("output folder must be an absolute path (got %s)", info.OutputDir)
	}
	if !info.FleetctlExists {
		return fmt.Errorf("fleetctl not built for this server (%s) — build it from the Server tab", info.Fleetctl)
	}
	contextName := cfg.Context
	if contextName == "" {
		contextName = info.MatchedContext
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	scan, err := scanContext(ctx, contextName)
	if err != nil {
		return err
	}
	secret, teamName, err := installers.SecretFor(scan, cfg.TeamID)
	if err != nil {
		return err
	}
	in := installers.BuildInputs{FleetURL: strings.TrimSpace(cfg.FleetURL), EnrollSecret: secret}
	if in.FleetURL == "" {
		in.FleetURL = scan.ServerURL
	}
	if in.FleetURL == "" {
		return errors.New("the server has no server_url set — enter a Fleet URL")
	}
	if cfg.UpdateSource == settings.UpdateSourceLocalTUF {
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
	args := installers.CommonArgs(cfg, in)
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
	// Per server and replaced atomically: bash reads a script as it runs, so
	// rewriting it in place would corrupt another server's running build.
	script := filepath.Join(dataDir, "installers-build-"+serverID+".sh")
	if err := installers.WriteFileAtomic(script, []byte(installers.BuildScript), 0o755); err != nil {
		return fmt.Errorf("write build script: %w", err)
	}

	jobs := installers.Jobs(info.Slug, info.OutputDir, cfg)
	entry := installers.ManifestEntry{
		BuiltAt:         time.Now().UTC().Format(time.RFC3339),
		FleetURL:        in.FleetURL,
		Team:            teamName,
		FleetDesktop:    cfg.FleetDesktop,
		Debug:           cfg.Debug,
		UpdateURL:       in.UpdateURL,
		OrbitChannel:    channelOrStable(cfg.OrbitChannel),
		DesktopChannel:  channelOrStable(cfg.DesktopChannel),
		OsquerydChannel: channelOrStable(cfg.OsquerydChannel),
		FleetctlVersion: fleetctlVersion(info.Fleetctl),
	}
	if entry.UpdateURL == "" {
		entry.UpdateURL = "https://updates.fleetdm.com"
	}

	channel := installers.LogChannel(serverID)
	_ = s.pm.ClearLogChannel(channel)
	// File mtimes are compared against this to tell this build's outputs from
	// files left by an earlier one.
	since := time.Now().Add(-time.Second)
	if err := s.pm.Start(procID, processes.StartArgs{
		Label:   "fleetd installers (" + info.Slug + ")",
		Cwd:     info.OutputDir,
		Program: "bash",
		Args:    []string{script},
		Env: []processes.EnvPair{
			{Key: "HANGAR_FLEETCTL", Value: info.Fleetctl},
			{Key: "HANGAR_JOBS", Value: installers.EncodeJobs(jobs)},
			{Key: "HANGAR_COMMON_ARGS", Value: strings.Join(args, "\n")},
		},
		LogChannel: channel,
	}); err != nil {
		return err
	}
	done := s.pm.Done(procID)
	go func() {
		<-done
		if _, err := installers.UpdateManifest(info.OutputDir, jobs, since, entry); err != nil {
			slog.Warn("installers: update manifest", "dir", info.OutputDir, "err", err)
		}
	}()
	return nil
}

// InstallersPlannedFiles maps each platform key to the file name cfg would
// produce for serverID, so the tab shows names without re-implementing them.
func (s *InstallersService) InstallersPlannedFiles(serverID string, cfg settings.InstallerServerConfig) (map[string]string, error) {
	srv, err := loadServer(serverID)
	if err != nil {
		return nil, err
	}
	slug := installers.ServerSlug(srv)
	out := make(map[string]string, len(installers.Platforms))
	for _, p := range installers.Platforms {
		out[p.Key] = installers.FileName(slug, p, cfg)
	}
	return out, nil
}

// InstallersStopBuild cancels serverID's running build.
func (s *InstallersService) InstallersStopBuild(serverID string) error {
	return s.pm.Stop(installers.ProcID(serverID))
}

// InstallersListOutputs lists the installers in serverID's output folder.
func (s *InstallersService) InstallersListOutputs(serverID, outputDir string) ([]installers.OutputFile, error) {
	info, err := s.InstallersServerInfo(serverID, outputDir)
	if err != nil {
		return nil, err
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

func loadServer(serverID string) (settings.ServerProfile, error) {
	cur, err := loadSettings()
	if err != nil {
		return settings.ServerProfile{}, err
	}
	for _, srv := range cur.Servers {
		if srv.ID == serverID {
			return srv, nil
		}
	}
	return settings.ServerProfile{}, fmt.Errorf("server %s not found", serverID)
}
