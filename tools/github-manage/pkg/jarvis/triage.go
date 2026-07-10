package jarvis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TriageStatus is the local disposition a user has given a work item.
type TriageStatus string

const (
	StatusSnoozed   TriageStatus = "snoozed"
	StatusDismissed TriageStatus = "dismissed"
	StatusDone      TriageStatus = "done"
)

// TriageEntry records why an item is hidden and the condition under which it resurfaces.
type TriageEntry struct {
	Status      TriageStatus `json:"status"`
	SnoozeUntil time.Time    `json:"snooze_until,omitempty"`
	// ItemUpdated is the item's UpdatedAt when it was dismissed/marked done; if the
	// item is updated after this, it resurfaces (someone acted on it).
	ItemUpdated time.Time `json:"item_updated,omitempty"`
	SetAt       time.Time `json:"set_at"`
}

// TriageStore is a local, JSON-backed record of snoozed/dismissed/done items. It
// is the one piece of real state jarvis owns — GitHub can't tell us when we're
// "done" with something, so the user curates that here.
type TriageStore struct {
	path    string
	Entries map[string]TriageEntry
}

// DefaultTriagePath returns ~/.config/gm/jarvis/triage.json.
func DefaultTriagePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "triage.json"
	}
	return filepath.Join(home, ".config", "gm", "jarvis", "triage.json")
}

// LoadTriageStore reads the store from disk, returning an empty store if absent.
func LoadTriageStore(path string) (*TriageStore, error) {
	s := &TriageStore{path: path, Entries: map[string]TriageEntry{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s.Entries); err != nil {
			return s, err
		}
	}
	return s, nil
}

// Save persists the store to disk, creating parent directories as needed.
func (s *TriageStore) Save() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.Entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *TriageStore) set(key string, e TriageEntry) {
	if s.Entries == nil {
		s.Entries = map[string]TriageEntry{}
	}
	e.SetAt = time.Now()
	s.Entries[key] = e
}

// Snooze hides an item until the given time (or until it's updated).
func (s *TriageStore) Snooze(key string, until, itemUpdated time.Time) {
	s.set(key, TriageEntry{Status: StatusSnoozed, SnoozeUntil: until, ItemUpdated: itemUpdated})
}

// Dismiss hides an item until it changes.
func (s *TriageStore) Dismiss(key string, itemUpdated time.Time) {
	s.set(key, TriageEntry{Status: StatusDismissed, ItemUpdated: itemUpdated})
}

// Done marks an item finished; it resurfaces only if updated afterward.
func (s *TriageStore) Done(key string, itemUpdated time.Time) {
	s.set(key, TriageEntry{Status: StatusDone, ItemUpdated: itemUpdated})
}

// Clear removes any triage state for a key (un-snooze / un-dismiss).
func (s *TriageStore) Clear(key string) { delete(s.Entries, key) }

// Visible reports whether an item should currently be shown.
func (s *TriageStore) Visible(key string, itemUpdated, now time.Time) bool {
	e, ok := s.Entries[key]
	if !ok {
		return true
	}
	switch e.Status {
	case StatusSnoozed:
		if now.After(e.SnoozeUntil) {
			return true // snooze expired
		}
		if !itemUpdated.IsZero() && itemUpdated.After(e.SetAt) {
			return true // activity since you snoozed
		}
		return false
	case StatusDismissed, StatusDone:
		if !itemUpdated.IsZero() && itemUpdated.After(e.ItemUpdated) {
			return true // changed since you cleared it
		}
		return false
	}
	return true
}

// Label returns a short descriptor for a triaged key (for the show-hidden view).
func (s *TriageStore) Label(key string) string {
	e, ok := s.Entries[key]
	if !ok {
		return ""
	}
	switch e.Status {
	case StatusSnoozed:
		return fmt.Sprintf("snoozed until %s", e.SnoozeUntil.Format("Jan 2 15:04"))
	case StatusDismissed:
		return "dismissed"
	case StatusDone:
		return "done"
	}
	return ""
}
