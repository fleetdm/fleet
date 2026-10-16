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

// InstallersService builds fleetd installers for each configured server with
// that server's own fleetctl, enrolling into a fleet picked from a scan of the
// server. One build per server runs at a time, as a tracked process.
type InstallersService struct {
	pm *processes.Manager

	// starting holds servers whose StartBuild is between its checks and
	// pm.Start (a scan can take seconds), so a double-click can't run two.
	startingMu sync.Mutex
	starting   map[string]bool

	share installers.Share
}

// NewInstallersService wires the service to the shared process manager.
func NewInstallersService(pm *processes.Manager) *InstallersService {
	return &InstallersService{pm: pm, starting: map[string]bool{}}
}

// InstallersTargetInfo is what the build card needs for a target before any
// scan: where its files go, which fleetctl builds it, and (for a Hangar server)
// the fleetctl context matching its port.
type InstallersTargetInfo struct {
	TargetID       string `json:"target_id"`
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

// InstallersTargetInfo resolves a target's slug, fleetctl binary, output
// folder and matching fleetctl context. A Hangar server builds with its own
// fleetctl; the custom target borrows the fleetctl of the server cfg.Fleetctl
// names (or the first server that has one) and has no matched context.
func (s *InstallersService) InstallersTargetInfo(targetID string, cfg settings.InstallerServerConfig, outputDir string) (InstallersTargetInfo, error) {
	cur, err := loadSettings()
	if err != nil {
		return InstallersTargetInfo{}, err
	}
	info := InstallersTargetInfo{TargetID: targetID}
	var builder *settings.ServerProfile
	if targetID == installers.CustomTarget {
		info.Slug = installers.Slug(cfg.Name)
		if info.Slug == "" {
			info.Slug = installers.CustomTarget
		}
		for i := range cur.Servers {
			srv := &cur.Servers[i]
			if srv.WorktreePath == nil || *srv.WorktreePath == "" {
				continue
			}
			if srv.ID == cfg.Fleetctl || (cfg.Fleetctl == "" && builder == nil) {
				builder = srv
			}
		}
	} else {
		for i := range cur.Servers {
			if cur.Servers[i].ID == targetID {
				builder = &cur.Servers[i]
			}
		}
		if builder == nil {
			return InstallersTargetInfo{}, fmt.Errorf("server %s not found", targetID)
		}
		info.Slug = installers.ServerSlug(*builder)
		if ctxInfo, err := fleetctl.ReadContext(fleetctl.DefaultConfigPath()); err == nil {
			addrs := map[string]string{}
			var order []string
			for _, c := range ctxInfo.Contexts {
				if c.Address != nil {
					addrs[c.Name] = *c.Address
				}
				order = append(order, c.Name)
			}
			info.MatchedContext = installers.MatchContext(addrs, order, builder.Ports.Server)
		}
	}
	info.OutputDir = filepath.Join(resolveOutputRoot(outputDir), info.Slug)
	if builder != nil && builder.WorktreePath != nil && *builder.WorktreePath != "" {
		info.Fleetctl = filepath.Join(*builder.WorktreePath, "build", "fleetctl")
		_, statErr := os.Stat(info.Fleetctl)
		info.FleetctlExists = statErr == nil
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

// InstallersStartBuild builds cfg's platforms for a target into
// <outputDir>/<target>/, with exactly the Fleet URL and enroll secret on the
// card. Files are written by fleetctl and recorded in that folder's
// manifest.json once the build ends.
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
		return errors.New("a build is already running for this target")
	}
	s.starting[serverID] = true
	s.startingMu.Unlock()
	defer func() {
		s.startingMu.Lock()
		delete(s.starting, serverID)
		s.startingMu.Unlock()
	}()

	info, err := s.InstallersTargetInfo(serverID, cfg, outputDir)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(info.OutputDir) {
		return fmt.Errorf("output folder must be an absolute path (got %s)", info.OutputDir)
	}
	if !info.FleetctlExists {
		return fmt.Errorf("fleetctl not built (%s) — build it from the Server tab", info.Fleetctl)
	}
	in := installers.BuildInputs{FleetURL: strings.TrimSpace(cfg.FleetURL), EnrollSecret: strings.TrimSpace(cfg.EnrollSecret)}
	if in.FleetURL == "" {
		return errors.New("enter a Fleet URL (or scan to fill it in)")
	}
	if in.EnrollSecret == "" {
		return errors.New("enter an enroll secret (or scan and pick a fleet to fill it in)")
	}
	if in.ExtraFlags, err = installers.ParseExtraFlags(cfg.ExtraFlags); err != nil {
		return err
	}
	teamName := cfg.TeamName
	if teamName == "" {
		teamName = "manual enroll secret"
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
		ExtraFlags:      strings.Join(in.ExtraFlags, " "),
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
	info, err := s.InstallersTargetInfo(serverID, cfg, "")
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(installers.Platforms))
	for _, p := range installers.Platforms {
		out[p.Key] = installers.FileName(info.Slug, p, cfg)
	}
	return out, nil
}

// InstallersStopBuild cancels serverID's running build.
func (s *InstallersService) InstallersStopBuild(serverID string) error {
	return s.pm.Stop(installers.ProcID(serverID))
}

// InstallersListOutputs lists the installers in a target's output folder.
func (s *InstallersService) InstallersListOutputs(serverID string, cfg settings.InstallerServerConfig, outputDir string) ([]installers.OutputFile, error) {
	info, err := s.InstallersTargetInfo(serverID, cfg, outputDir)
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

// Ports the share must not take even when they're momentarily free: the TUF
// file-server and the dev compose stack's SAML IdP (docker-compose.yml).
var fixedHangarPorts = []uint16{tuf.DefaultPort, 9080}

// InstallersSuggestSharePort is the first free port from 8000 that no
// configured Hangar service uses.
func (s *InstallersService) InstallersSuggestSharePort() (uint16, error) {
	return installers.FreePort(installers.DefaultSharePort, hangarPorts())
}

// hangarPorts are the ports Hangar's services bind when started, so the share
// mustn't take them even while they're stopped.
func hangarPorts() map[uint16]bool {
	reserved := map[uint16]bool{}
	for _, p := range fixedHangarPorts {
		reserved[p] = true
	}
	if cur, err := loadSettings(); err == nil {
		for _, srv := range cur.Servers {
			for _, p := range []uint16{srv.Ports.Server, srv.Ports.MySQL, srv.Ports.Redis, srv.Ports.S3, srv.Ports.S3Console} {
				reserved[p] = true
			}
		}
		for _, sp := range cur.ScepProfiles {
			reserved[sp.Port] = true
		}
		reserved[cur.PythonServer.Port] = true
	}
	return reserved
}

// InstallersShareStart serves the output folder (all servers' subfolders)
// over HTTP on port, or on a suggested free port when port is 0.
func (s *InstallersService) InstallersShareStart(outputDir string, port uint16) (installers.ShareStatus, error) {
	root := resolveOutputRoot(outputDir)
	if !filepath.IsAbs(root) {
		return installers.ShareStatus{}, fmt.Errorf("output folder must be an absolute path (got %s)", root)
	}
	if port != 0 && hangarPorts()[port] {
		return installers.ShareStatus{}, fmt.Errorf("port %d is used by one of Hangar's services — pick another", port)
	}
	if port == 0 {
		p, err := s.InstallersSuggestSharePort()
		if err != nil {
			return installers.ShareStatus{}, err
		}
		port = p
	}
	if err := s.share.Start(root, port); err != nil {
		return installers.ShareStatus{}, err
	}
	slog.Info("installers: sharing", "root", root, "port", port)
	return s.share.Status(netinfo.LanIP()), nil
}

// InstallersShareStop stops sharing.
func (s *InstallersService) InstallersShareStop() error {
	return s.share.Stop()
}

// InstallersShareStatus reports whether the folder is being shared, its URL
// and recent downloads.
func (s *InstallersService) InstallersShareStatus() installers.ShareStatus {
	return s.share.Status(netinfo.LanIP())
}

// InstallersSharedFiles lists the installers in the output folder. While
// sharing they come with URLs and install commands (for the folder being
// served); otherwise just paths and sizes for outputDir.
func (s *InstallersService) InstallersSharedFiles(outputDir string) ([]installers.SharedFile, error) {
	st := s.share.Status(netinfo.LanIP())
	if st.Running {
		return installers.ListShared(st.Root, st.URL)
	}
	return installers.ListShared(resolveOutputRoot(outputDir), "")
}
