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

// Settings is the full persisted configuration.
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
	}
}

// Load reads settings.json from dir. A missing file yields Default(). For
// an existing file, missing fields keep their default values — we unmarshal
// *into* a pre-defaulted struct so serde's per-field defaults are matched
// (e.g. an old file without "premium" still loads premium=true).
func Load(dir string) (Settings, error) {
	s := Default()
	b, err := os.ReadFile(filepath.Join(dir, fileName))
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("reading %s: %w", fileName, err)
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return Settings{}, fmt.Errorf("parsing %s: %w", fileName, err)
	}
	return s, nil
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
