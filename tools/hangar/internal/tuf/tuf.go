// Package tuf drives a local TUF test repository for QA: it builds the env for
// tools/tuf/test/main.sh from a platform selection (which generates fleetd
// installers pointing at a local/ngrok TUF server), probes whether the TUF
// server is up, and cleans up the generated assets. The heavy lifting stays in
// the existing shell scripts — this package just orchestrates them.
//
// Pure helpers (EnvFor, platform mapping) take explicit inputs so they're
// unit-testable; the service layer resolves the repo path and process manager.
package tuf

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

const (
	// DefaultPort is where run_server.sh serves the TUF repository.
	DefaultPort uint16 = 8081
	// AssetsDir is the TUF repo dir (TUF_PATH), relative to the Fleet repo root.
	AssetsDir = "test_tuf"
	// mainScript is the orchestrator, relative to the Fleet repo root.
	mainScript = "tools/tuf/test/main.sh"
	// ProcID / LogChannel namespace the build run in the process engine.
	ProcID     = "tuf:build"
	LogChannel = "tuf-build"
	// ServerProcID / ServerLogChannel namespace the TUF file-server, which
	// Hangar runs itself (main.sh is run with SKIP_SERVER=1) so the build
	// process exits cleanly and the server has a proper managed lifecycle.
	ServerProcID     = "tuf:server"
	ServerLogChannel = "tuf-server"
)

// Platform maps a UI platform key to its create_repository SYSTEMS name and the
// gen_pkgs GENERATE_* flags it implies.
type Platform struct {
	Key      string
	System   string
	Generate []string
}

// Platforms is the fixed set of selectable platforms (matches lib/tuf.ts).
var Platforms = []Platform{
	{Key: "macos", System: "macos", Generate: []string{"GENERATE_PKG"}},
	{Key: "windows", System: "windows", Generate: []string{"GENERATE_MSI"}},
	{Key: "windows-arm64", System: "windows-arm64", Generate: []string{"GENERATE_MSI_ARM64"}},
	{Key: "linux", System: "linux", Generate: []string{"GENERATE_DEB", "GENERATE_RPM"}},
	{Key: "linux-arm64", System: "linux-arm64", Generate: []string{"GENERATE_DEB_ARM64", "GENERATE_RPM_ARM64"}},
}

func platformByKey(key string) (Platform, bool) {
	for _, p := range Platforms {
		if p.Key == key {
			return p, true
		}
	}
	return Platform{}, false
}

// urlTypes are the package-type prefixes whose *_FLEET_URL / *_TUF_URL vars
// gen_pkgs reads (arm64 variants reuse the base type's URL). We set them all to
// the same URLs — harmless for types not being generated — mirroring my_tuf.sh.
var urlTypes = []string{"PKG", "DEB", "RPM", "MSI", "PKG_TAR_ZST"}

// EnvFor builds the environment for main.sh from a config. Pure; unit-tested.
// Unknown platform keys are ignored.
func EnvFor(cfg settings.TufConfig) []processes.EnvPair {
	var systems []string
	var generates []processes.EnvPair
	for _, key := range cfg.Platforms {
		p, ok := platformByKey(key)
		if !ok {
			continue
		}
		systems = append(systems, p.System)
		for _, g := range p.Generate {
			generates = append(generates, processes.EnvPair{Key: g, Value: "1"})
		}
	}

	env := []processes.EnvPair{{Key: "SYSTEMS", Value: strings.Join(systems, " ")}}
	env = append(env, generates...)
	for _, t := range urlTypes {
		env = append(env,
			processes.EnvPair{Key: t + "_FLEET_URL", Value: cfg.FleetURL},
			processes.EnvPair{Key: t + "_TUF_URL", Value: cfg.TufURL},
		)
	}
	env = append(env, processes.EnvPair{Key: "ENROLL_SECRET", Value: cfg.EnrollSecret})
	if cfg.FleetDesktop {
		env = append(env, processes.EnvPair{Key: "FLEET_DESKTOP", Value: "1"})
	}
	if cfg.Debug {
		env = append(env, processes.EnvPair{Key: "DEBUG", Value: "1"})
	}
	// Hangar runs the file-server itself (see FileServerArgs), so main.sh must
	// not spawn its own — otherwise that backgrounded child inherits the build's
	// stdout/stderr pipe and the build process never reaches "done".
	env = append(env, processes.EnvPair{Key: "SKIP_SERVER", Value: "1"})
	return env
}

// FileServerArgs are the `go` args to serve the generated TUF repo (what
// run_server.sh would have run). Runs from the Fleet repo root.
func FileServerArgs() []string {
	return []string{"run", "./tools/file-server", strconv.Itoa(int(DefaultPort)), AssetsDir + "/repository"}
}

// MainScriptPath is the absolute path to main.sh within the Fleet repo.
func MainScriptPath(repoPath string) string {
	return filepath.Join(repoPath, mainScript)
}

// ServerStatus is the TUF file-server's reachability.
type ServerStatus struct {
	Up  bool   `json:"up"`
	URL string `json:"url"`
}

// ProbeServer reports whether the TUF server answers /root.json on the port
// (the same readiness check run_server.sh uses).
func ProbeServer(port uint16) ServerStatus {
	base := fmt.Sprintf("http://localhost:%d", port)
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(base + "/root.json") //nolint:noctx // short local readiness probe
	if err != nil {
		return ServerStatus{Up: false}
	}
	defer resp.Body.Close()
	return ServerStatus{Up: resp.StatusCode == http.StatusOK, URL: base}
}

// DeleteAssets removes the TUF repo dir (<repo>/test_tuf). No-op if absent.
func DeleteAssets(repoPath string) error {
	if strings.TrimSpace(repoPath) == "" {
		return errors.New("no primary repo configured")
	}
	return os.RemoveAll(filepath.Join(repoPath, AssetsDir))
}

// AssetsExist reports whether the generated TUF repo dir (<repo>/test_tuf) is
// present, so the UI can hide "Delete assets" when there's nothing to delete.
func AssetsExist(repoPath string) bool {
	if strings.TrimSpace(repoPath) == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(repoPath, AssetsDir))
	return err == nil && info.IsDir()
}
