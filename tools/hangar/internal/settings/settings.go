// Package settings holds the persisted Hangar configuration plus the
// repo-probing, ngrok-parsing and sandboxed file helpers that lived in
// src-tauri/src/settings.rs. Settings are stored as JSON in
// <config-dir>/settings.json.
//
// Pure functions take an explicit directory; the service layer resolves
// the real config dir via the paths package.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const fileName = "settings.json"

// ThemePreference is "system" (follow the OS), "light", or "dark".
type ThemePreference string

const (
	ThemeSystem ThemePreference = "system"
	ThemeLight  ThemePreference = "light"
	ThemeDark   ThemePreference = "dark"
)

// EnvVar is one row of the fleet serve environment editor.
type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	// Enabled is a per-row toggle. Defaults true so rows saved before this
	// field existed (and hand-added rows missing the key) stay applied.
	Enabled bool `json:"enabled"`
}

// UnmarshalJSON defaults Enabled to true when the key is absent, matching
// the Rust `#[serde(default = "true_default")]`. Needed because EnvVar
// lives in a slice — pre-populating the parent struct with defaults can't
// reach freshly-created slice elements.
func (e *EnvVar) UnmarshalJSON(b []byte) error {
	type alias EnvVar
	aux := alias{Enabled: true}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	*e = EnvVar(aux)
	return nil
}

// FleetServeConfig holds the user-tunable bits of `fleet serve --dev`.
type FleetServeConfig struct {
	// ConfigPath is passed to --config; nil/empty omits the flag so serve
	// falls back to env vars / built-in defaults.
	ConfigPath *string `json:"config_path"`
	Premium    bool    `json:"premium"`       // --dev_license
	Debug      bool    `json:"debug"`         // --debug
	Logging    bool    `json:"logging_debug"` // --logging_debug
	// Env is a slice (not a map) so the user's row order is preserved.
	Env []EnvVar `json:"env"`
}

// MaxServers caps how many independent local Fleet servers can be configured.
// Each server is its own worktree + ports + compose project + DB, so this is
// a deliberate ceiling on machine load (matches the product decision).
const MaxServers = 3

// ServerPorts holds the host ports one server's stack binds to. Server 1 keeps
// the canonical dev defaults (8080/3306/6379/9000/9001) so existing scripts and
// muscle memory keep working; additional servers use offset blocks so two
// stacks never collide.
type ServerPorts struct {
	Server    uint16 `json:"server"`     // fleet serve --server_address host port
	MySQL     uint16 `json:"mysql"`      // docker mysql host port
	Redis     uint16 `json:"redis"`      // docker redis host port
	S3        uint16 `json:"s3"`         // docker s3 (object store) host port
	S3Console uint16 `json:"s3_console"` // docker s3 console host port
}

// ServerProfile is one independent local Fleet server instance: its own git
// worktree (so it can build/run a different branch), host ports, docker compose
// project, and serve config. Multiple profiles can run in parallel.
type ServerProfile struct {
	ID    string `json:"id"`    // stable, e.g. "s1" (never reused/renumbered)
	Name  string `json:"name"`  // user-facing label, e.g. "main" / "n-1 repro"
	Color string `json:"color"` // accent key for the switcher/status: green|purple|blue
	// WorktreePath is the git worktree this server builds and runs from. For
	// server 1 this is typically the primary clone; others are `git worktree
	// add`-ed trees. nil until the user picks one.
	WorktreePath *string `json:"worktree_path"`
	// Branch is informational (the branch the worktree was created on). The
	// live branch is read from git; checkout happens in the Git tab.
	Branch         *string          `json:"branch"`
	Ports          ServerPorts      `json:"ports"`
	ComposeProject string           `json:"compose_project"` // docker compose -p value
	FleetServe     FleetServeConfig `json:"fleet_serve"`
	Enabled        bool             `json:"enabled"`
}

// DefaultPortsForIndex returns the canonical port block for the i-th server
// (0-based). Server 0 = the standard dev ports; each later server adds a small
// per-slot offset. These are defaults the user can override per profile.
func DefaultPortsForIndex(i int) ServerPorts {
	return ServerPorts{
		Server:    uint16(8080 + i*10),
		MySQL:     uint16(3306 + i*20),
		Redis:     uint16(6379 + i*10),
		S3:        uint16(9000 + i*20),
		S3Console: uint16(9001 + i*10),
	}
}

// DefaultComposeProject returns the docker compose project name for the i-th
// server. Server 0 keeps the bare "fleet" project so its existing containers
// and data carry over from the single-server era; later servers are suffixed.
func DefaultComposeProject(i int) string {
	if i == 0 {
		return "fleet"
	}
	return fmt.Sprintf("fleet-s%d", i+1)
}

// serverColors is the accent palette assigned to servers by slot.
var serverColors = []string{"green", "purple", "blue"}

func defaultColor(i int) string {
	if i >= 0 && i < len(serverColors) {
		return serverColors[i]
	}
	return serverColors[0]
}

// defaultServer builds a fresh profile for the i-th slot with canonical ports,
// compose project, color, and a default serve config. Name/worktree are left
// to the caller.
func defaultServer(i int) ServerProfile {
	return ServerProfile{
		ID:             fmt.Sprintf("s%d", i+1),
		Name:           fmt.Sprintf("server %d", i+1),
		Color:          defaultColor(i),
		Ports:          DefaultPortsForIndex(i),
		ComposeProject: DefaultComposeProject(i),
		FleetServe:     FleetServeConfig{Premium: true, Debug: true, Logging: true, Env: []EnvVar{}},
		Enabled:        true,
	}
}

// NextServerProfile returns a fresh profile for the next free slot, or false if
// the server cap is already reached. The new ID/ports/project are derived from
// the count of existing servers so they never collide with what's configured.
func NextServerProfile(existing []ServerProfile) (ServerProfile, bool) {
	if len(existing) >= MaxServers {
		return ServerProfile{}, false
	}
	i := len(existing)
	// Guard against an ID clash if a middle server was removed and re-added:
	// bump the index until the derived ID is unused.
	for hasServerID(existing, fmt.Sprintf("s%d", i+1)) {
		i++
	}
	srv := defaultServer(i)
	srv.Name = fmt.Sprintf("server %d", len(existing)+1)
	return srv, true
}

func hasServerID(servers []ServerProfile, id string) bool {
	for _, s := range servers {
		if s.ID == id {
			return true
		}
	}
	return false
}

// NgrokConfig configures the optional ngrok tunnel process.
type NgrokConfig struct {
	Enabled        bool     `json:"enabled"`
	YmlPath        *string  `json:"yml_path"`
	DefaultTunnels []string `json:"default_tunnels"`
	StartAll       bool     `json:"start_all"`
}

// PythonConfig configures the optional python http.server process.
type PythonConfig struct {
	Enabled   bool    `json:"enabled"`
	Port      uint16  `json:"port"`
	Directory *string `json:"directory"`
}

// TufConfig is the saved inputs for a local TUF build (drives
// tools/tuf/test/main.sh). Platforms are UI keys (macos|windows|windows-arm64|
// linux|linux-arm64) that expand to SYSTEMS + GENERATE_* env; the URLs are the
// public (ngrok) Fleet/TUF endpoints baked into the generated installers.
type TufConfig struct {
	Platforms    []string `json:"platforms"`
	FleetURL     string   `json:"fleet_url"`
	TufURL       string   `json:"tuf_url"`
	EnrollSecret string   `json:"enroll_secret"`
	FleetDesktop bool     `json:"fleet_desktop"`
	Debug        bool     `json:"debug"`
}

// Settings is the full persisted configuration.
//
// Servers is the multi-server source of truth. The legacy single-server fields
// (RepoPath, FleetServe) are retained so settings written before multi-server
// support still parse; Load() migrates them into Servers[0]. New code reads
// from Servers / ActiveServerID, not the legacy fields.
type Settings struct {
	RepoPath         *string          `json:"repo_path"`
	FleetctlPath     *string          `json:"fleetctl_path"`
	GitopsDir        *string          `json:"gitops_dir"`
	FirstRunComplete bool             `json:"first_run_complete"`
	Ngrok            NgrokConfig      `json:"ngrok"`
	PythonServer     PythonConfig     `json:"python_server"`
	FleetServe       FleetServeConfig `json:"fleet_serve"`
	Theme            ThemePreference  `json:"theme"`
	FavoriteCrons    []string         `json:"favorite_crons"`
	Tuf              TufConfig        `json:"tuf"`

	// Servers is nil in a pre-multi-server file; migrate() backfills it from
	// the legacy fields on Load so callers always see at least one server.
	Servers        []ServerProfile `json:"servers"`
	ActiveServerID string          `json:"active_server_id"`

	// ScepProfiles are saved SCEP server launch configs (see scep.go). The
	// in-repo scepserver binary is shared; profiles differ by depot/port so
	// several CAs can run at once.
	ScepProfiles []ScepProfile `json:"scep_profiles"`
	// ScepDepotsDir overrides where managed CA depots live; empty means the
	// service default under app-data (<app-data>/scep-depots).
	ScepDepotsDir *string `json:"scep_depots_dir"`
}

// Default returns the zero-config Settings, matching Rust's Default impls:
// serve premium/debug/logging on, python port 8000, theme "system",
// everything else off/empty.
func Default() Settings {
	return Settings{
		FirstRunComplete: false,
		Ngrok: NgrokConfig{
			Enabled:        false,
			DefaultTunnels: []string{},
			StartAll:       false,
		},
		PythonServer: PythonConfig{
			Enabled: false,
			Port:    8000,
		},
		FleetServe: FleetServeConfig{
			Premium: true,
			Debug:   true,
			Logging: true,
			Env:     []EnvVar{},
		},
		Theme:         ThemeSystem,
		FavoriteCrons: []string{},
		ScepProfiles:  []ScepProfile{},
		Tuf: TufConfig{
			// Mirror the common my_tuf.sh run: all platforms, desktop + debug on.
			Platforms:    []string{"macos", "windows", "windows-arm64", "linux", "linux-arm64"},
			FleetDesktop: true,
			Debug:        true,
		},
	}
}

// Load reads settings.json from dir. A missing file yields Default(). For
// an existing file, missing fields keep their default values — we unmarshal
// *into* a pre-defaulted struct so serde's per-field defaults are matched
// (e.g. an old file without "premium" still loads premium=true). Either way
// migrate() guarantees at least one server profile before returning.
func Load(dir string) (Settings, error) {
	s := Default()
	b, err := os.ReadFile(filepath.Join(dir, fileName))
	if errors.Is(err, fs.ErrNotExist) {
		migrate(&s)
		return s, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("reading %s: %w", fileName, err)
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return Settings{}, fmt.Errorf("parsing %s: %w", fileName, err)
	}
	migrate(&s)
	return s, nil
}

// migrate brings a freshly-loaded Settings up to the multi-server shape:
//
//   - If Servers is nil (a pre-multi-server file, or a fresh Default()), it
//     synthesizes server 1 from the legacy single-server fields — the user's
//     repo_path becomes server 1's worktree and their fleet_serve config is
//     adopted, so an upgrade is seamless.
//   - It backfills any per-server fields a hand-edited or partial file left
//     blank (ID, ports, compose project, color).
//   - It repairs ActiveServerID if it's empty or points at a removed server.
//
// migrate is idempotent: a file already in the new shape passes through with
// only defensive backfilling.
func migrate(s *Settings) {
	if s.Servers == nil {
		srv := defaultServer(0)
		srv.Name = "main"
		srv.WorktreePath = s.RepoPath // nil for a fresh install, set on upgrade
		srv.FleetServe = s.FleetServe // adopt the user's existing serve config
		s.Servers = []ServerProfile{srv}
	}
	for i := range s.Servers {
		if s.Servers[i].ID == "" {
			s.Servers[i].ID = fmt.Sprintf("s%d", i+1)
		}
		if s.Servers[i].ComposeProject == "" {
			s.Servers[i].ComposeProject = DefaultComposeProject(i)
		}
		if s.Servers[i].Ports == (ServerPorts{}) {
			s.Servers[i].Ports = DefaultPortsForIndex(i)
		}
		if s.Servers[i].Color == "" {
			s.Servers[i].Color = defaultColor(i)
		}
		if s.Servers[i].FleetServe.Env == nil {
			s.Servers[i].FleetServe.Env = []EnvVar{}
		}
	}
	if s.ActiveServerID == "" || !hasServerID(s.Servers, s.ActiveServerID) {
		s.ActiveServerID = s.Servers[0].ID
	}
	// A pre-SCEP file (or fresh Default()) has no profiles; normalize nil to an
	// empty slice so the frontend always gets a JSON array.
	if s.ScepProfiles == nil {
		s.ScepProfiles = []ScepProfile{}
	}
	// Normalize nil TUF platforms to an empty slice for the frontend.
	if s.Tuf.Platforms == nil {
		s.Tuf.Platforms = []string{}
	}
}

// Save writes settings.json to dir (pretty-printed, 2-space indent to match
// the Rust serde_json output).
func Save(dir string, s Settings) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, fileName), b, 0o644)
}
