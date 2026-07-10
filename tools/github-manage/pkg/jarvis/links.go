package jarvis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// Link is jarvis's authoritative record of how an issue is being worked: which
// local clone holds the branch, the branch name, the Claude session driving it,
// and the project board that owns its workflow Status. It's written when you
// Start Work through jarvis, and is the primary source for the issue↔PR↔session
// association (GitHub closing-keyword references are only a fallback).
type Link struct {
	ClonePath string `json:"clone_path,omitempty"`
	Branch    string `json:"branch,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Project   int    `json:"project,omitempty"`
}

// LinkStore is a local, JSON-backed map of issue number → Link.
type LinkStore struct {
	path  string
	Links map[string]Link
}

// DefaultLinkPath returns ~/.config/gm/jarvis/links.json.
func DefaultLinkPath() string {
	return configPath("links.json")
}

// LoadLinkStore reads the store from disk, returning an empty store if absent.
func LoadLinkStore(path string) (*LinkStore, error) {
	s := &LinkStore{path: path, Links: map[string]Link{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s.Links); err != nil {
			return s, err
		}
	}
	return s, nil
}

// Save persists the store to disk, creating parent directories as needed.
func (s *LinkStore) Save() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.Links, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

// Get returns the link for an issue, if any.
func (s *LinkStore) Get(issue int) (Link, bool) {
	l, ok := s.Links[strconv.Itoa(issue)]
	return l, ok
}

// Set records (or replaces) the link for an issue.
func (s *LinkStore) Set(issue int, l Link) {
	if s.Links == nil {
		s.Links = map[string]Link{}
	}
	s.Links[strconv.Itoa(issue)] = l
}

// configPath joins a filename under ~/.config/gm/jarvis, falling back to the
// bare filename if the home directory can't be determined.
func configPath(name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return name
	}
	return filepath.Join(home, ".config", "gm", "jarvis", name)
}
