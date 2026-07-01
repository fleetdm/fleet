// Package services holds the Wails-bound service structs. Each method is
// callable from the frontend; the structs are thin adapters that resolve
// real paths and delegate to the internal/* packages (where the logic and
// its tests live).
package services

import (
	"fmt"

	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

// SettingsService exposes settings, repo probing, ngrok parsing, and the
// sandboxed file helpers. Mirrors the settings/* commands from settings.rs.
type SettingsService struct{}

// GetSettings loads the persisted settings (defaults if none saved yet).
func (s *SettingsService) GetSettings() (settings.Settings, error) {
	dir, err := paths.ConfigDir()
	if err != nil {
		return settings.Settings{}, err
	}
	return settings.Load(dir)
}

// SaveSettings persists the given settings.
func (s *SettingsService) SaveSettings(in settings.Settings) error {
	dir, err := paths.ConfigDir()
	if err != nil {
		return err
	}
	return settings.Save(dir, in)
}

// NewServerProfile returns a fresh server profile for the next free slot
// (canonical-but-offset ports, compose project, color, serve config), leaving
// name/worktree for the caller to fill before saving. Errors if the server cap
// is already reached. The slot is derived from the currently-saved servers so
// the new ports/project/ID never collide.
func (s *SettingsService) NewServerProfile() (settings.ServerProfile, error) {
	dir, err := paths.ConfigDir()
	if err != nil {
		return settings.ServerProfile{}, err
	}
	cur, err := settings.Load(dir)
	if err != nil {
		return settings.ServerProfile{}, err
	}
	p, ok := settings.NextServerProfile(cur.Servers)
	if !ok {
		return settings.ServerProfile{}, fmt.Errorf("server limit reached (max %d)", settings.MaxServers)
	}
	return p, nil
}

// ProbeFleetRepo validates a single path, or (when path is empty) discovers
// Fleet clones under the well-known dev roots.
func (s *SettingsService) ProbeFleetRepo(path string) []settings.RepoProbe {
	if path != "" {
		return []settings.RepoProbe{settings.ProbeOne(path)}
	}
	return settings.DiscoverFleetRepos()
}

// DetectFleetConfig returns the relative serve-config name in the repo root
// (fleet.yml/fleet.yaml), or "" if none.
func (s *SettingsService) DetectFleetConfig(repo string) string {
	return settings.DetectFleetConfig(repo)
}

// ParseNgrokYml summarizes an ngrok.yml (empty path = ngrok's default).
func (s *SettingsService) ParseNgrokYml(path string) settings.NgrokYamlInfo {
	return settings.ParseNgrokYml(path)
}

// ReadTextFile reads a .yml/.yaml file under $HOME.
func (s *SettingsService) ReadTextFile(path string) (string, error) {
	return settings.ReadTextFile(path)
}

// WriteTextFile writes a .yml/.yaml file under $HOME.
func (s *SettingsService) WriteTextFile(path, contents string) error {
	return settings.WriteTextFile(path, contents)
}

// OpenPath opens a dir or allowed file in the system file manager.
func (s *SettingsService) OpenPath(path string, reveal bool) error {
	return settings.OpenPath(path, reveal)
}

// OpenURL opens an http(s) URL in the default browser.
func (s *SettingsService) OpenURL(url string) error {
	return settings.OpenURL(url)
}
