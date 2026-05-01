package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is Fleet ship's persistent, user-level configuration. It lives in
// ~/.config/fleet-ship/config.yaml so it's shared across worktrees of the
// same Fleet repo (and across multiple repo clones, for that matter).
//
// Per-running-session state (PID, log paths, ngrok URL) is in state.go and
// lives inside the active worktree under tools/ship/.state/active.json.
type Config struct {
	Ngrok NgrokConfig `yaml:"ngrok"`
	Fleet FleetConfig `yaml:"fleet"`

	// Worktrees registered with ship — populated by auto-register on
	// launch and the new-worktree form. The launching worktree gets
	// silently added if not already present. Removed entries are
	// pruned against `git worktree list --porcelain` on each launch.
	Worktrees []WorktreeEntry `yaml:"worktrees,omitempty"`

	// ActiveWorktree is the Name of the worktree last used by ship.
	// Persisted so a relaunch defaults to the same one.
	ActiveWorktree string `yaml:"active_worktree,omitempty"`
}

// WorktreeEntry is one row in the worktree registry.
type WorktreeEntry struct {
	// Name is what ship calls this worktree in the UI. Defaults to the
	// directory basename (e.g. "fleet", "fleet-vpp-fix"); editable in
	// PR 3+ if we want.
	Name string `yaml:"name"`
	// Path is the absolute filesystem path to the worktree directory.
	Path string `yaml:"path"`
	// Branch is the branch the worktree currently has checked out.
	// Stored at registration time; refreshed by the prune sweep.
	Branch string `yaml:"branch,omitempty"`
}

type NgrokConfig struct {
	// StaticDomain is the ngrok static domain (e.g. "fleet-pm-jane.ngrok-free.app").
	// PMs get one for free at https://dashboard.ngrok.com/domains.
	StaticDomain string `yaml:"static_domain"`
}

type FleetConfig struct {
	// Premium controls whether `fleet serve` is started with --dev_license.
	// Defaults to true — most PMs test premium features.
	Premium bool `yaml:"premium"`
	// Port the local Fleet server binds to. Defaults to 8080.
	Port int `yaml:"port"`
}

// defaultConfig is what we hand back when no config file exists yet.
func defaultConfig() Config {
	return Config{
		Fleet: FleetConfig{Premium: true, Port: 8080},
	}
}

// ConfigDir is ~/.config/fleet-ship.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".config", "fleet-ship"), nil
}

// ConfigPath is ~/.config/fleet-ship/config.yaml.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// LoadConfig reads the config file. Returns defaults + ok=false when the file
// doesn't exist yet (first-run case).
func LoadConfig() (cfg Config, exists bool, err error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultConfig(), false, nil
	}
	if err != nil {
		return Config{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	cfg = defaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, true, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, true, nil
}

// SaveConfig writes the config atomically (write to temp file then rename) so
// a crashed process can't leave a half-written file.
func SaveConfig(cfg Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	final := filepath.Join(dir, "config.yaml")
	tmp, err := os.CreateTemp(dir, "config-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpName, final); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename %s to %s: %w", tmpName, final, err)
	}
	return nil
}
