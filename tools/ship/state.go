package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// State files come in two flavors:
//
//   1. Persistent, user-scoped: lives under ~/.config/fleet-ship/. Includes
//      the Fleet server private key (paste-only, never auto-generated) and
//      DB snapshots.
//
//   2. Per-running-session: lives at tools/ship/.state/active.json inside
//      the active worktree, so coding agents can read it at a stable
//      repo-relative path. Written when ship starts a Fleet instance,
//      deleted on clean shutdown.

// PrivateKeyPath is ~/.config/fleet-ship/server_private_key.
func PrivateKeyPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "server_private_key"), nil
}

// LoadPrivateKey reads the Fleet server private key. Returns empty + ok=false
// when the file is missing — that's the trigger for the wizard to ask the
// user to paste theirs from 1Password.
func LoadPrivateKey() (key string, exists bool, err error) {
	path, err := PrivateKeyPath()
	if err != nil {
		return "", false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read %s: %w", path, err)
	}
	return strings.TrimSpace(string(data)), true, nil
}

// SavePrivateKey persists the Fleet server private key with mode 0600 so other
// users on the machine can't read it.
func SavePrivateKey(key string) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	path := filepath.Join(dir, "server_private_key")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(key)+"\n"), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// ActiveSession is what we write to tools/ship/.state/active.json so coding
// agents can `cat` it and use values like fleet_pid + fleet_log with normal
// shell tools.
type ActiveSession struct {
	FleetPID       int       `json:"fleet_pid"`
	FleetLog       string    `json:"fleet_log"`
	BuildLog       string    `json:"build_log"`
	ComposeProject string    `json:"compose_project"`
	MySQLContainer string    `json:"mysql_container"`
	RedisContainer string    `json:"redis_container"`
	MySQLDatabase  string    `json:"mysql_database"`
	Worktree       string    `json:"worktree"`
	Branch         string    `json:"branch"`
	Commit         string    `json:"commit"`
	NgrokURL       string    `json:"ngrok_url"`
	StartedAt      time.Time `json:"started_at"`
}

// activeStateDir is tools/ship/.state, resolved relative to the working
// directory ship was launched from (which is tools/ship/, courtesy of the
// `make ship` Makefile target).
func activeStateDir() string {
	return filepath.Join(".state")
}

// ActiveSessionPath is the path agents read.
func ActiveSessionPath() string {
	return filepath.Join(activeStateDir(), "active.json")
}

// WriteActiveSession serializes the running-session info atomically.
func WriteActiveSession(s ActiveSession) error {
	if err := os.MkdirAll(activeStateDir(), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", activeStateDir(), err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	final := ActiveSessionPath()
	tmp, err := os.CreateTemp(activeStateDir(), "active-*.json")
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
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// ClearActiveSession removes the file. Called during clean shutdown so a
// stale file doesn't mislead the next run or any agents poking at it.
func ClearActiveSession() error {
	err := os.Remove(ActiveSessionPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
