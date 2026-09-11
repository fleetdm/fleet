package jarvis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
)

// FocusStore is the set of issue numbers you're actively focused on. jarvis
// auto-adds issues when you Start Work and auto-drops them when they reach
// Awaiting QA / close, but you can also pin/unpin manually.
type FocusStore struct {
	path   string
	Issues []int `json:"issues"`
}

// DefaultFocusPath returns ~/.config/gm/jarvis/focus.json.
func DefaultFocusPath() string {
	return configPath("focus.json")
}

// LoadFocusStore reads the store from disk, returning an empty store if absent.
func LoadFocusStore(path string) (*FocusStore, error) {
	s := &FocusStore{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, s); err != nil {
			return s, err
		}
	}
	return s, nil
}

// Save persists the store to disk, creating parent directories as needed.
func (s *FocusStore) Save() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

// Has reports whether an issue is in the focus set.
func (s *FocusStore) Has(issue int) bool { return slices.Contains(s.Issues, issue) }

// Add pins an issue (no-op if already present).
func (s *FocusStore) Add(issue int) {
	if !s.Has(issue) {
		s.Issues = append(s.Issues, issue)
	}
}

// Remove unpins an issue.
func (s *FocusStore) Remove(issue int) {
	s.Issues = slices.DeleteFunc(s.Issues, func(n int) bool { return n == issue })
}

// Toggle flips an issue's focus state and reports the new state.
func (s *FocusStore) Toggle(issue int) bool {
	if s.Has(issue) {
		s.Remove(issue)
		return false
	}
	s.Add(issue)
	return true
}
