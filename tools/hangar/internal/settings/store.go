package settings

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
)

// EventChanged tells the GUI that settings.json changed outside its own save.
const EventChanged = "settings:changed"

// Store is the only writer of settings.json while the app runs. The GUI saves
// whole objects, so a save can carry a stale view of a slot an agent has since
// reassigned. SaveFromGUI keeps reserved slots' identity from the store.
type Store struct {
	mu       sync.Mutex
	dir      string
	emit     func(event string)
	reserved func() (map[string]bool, error)
	heldBy   func(slotID string) (owner string, held bool)
	cur      Settings
}

// NewStore loads settings from dir. reserved reports the slot ids that have a
// live reservation; emit may be nil.
func NewStore(dir string, emit func(event string), reserved func() (map[string]bool, error)) (*Store, error) {
	cur, err := Load(dir)
	if err != nil {
		return nil, err
	}
	return &Store{dir: dir, emit: emit, reserved: reserved, cur: cur}, nil
}

// SetHeldBy installs the lookup for slots held by an agent job.
func (s *Store) SetHeldBy(fn func(slotID string) (owner string, held bool)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.heldBy = fn
}

// Snapshot returns a copy of the current settings.
func (s *Store) Snapshot() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return clone(s.cur)
}

// Update applies an agent change, saves, and emits settings:changed. A failing
// fn leaves the store and the file untouched.
func (s *Store) Update(fn func(*Settings) error) (Settings, error) {
	s.mu.Lock()
	next := clone(s.cur)
	if err := fn(&next); err != nil {
		s.mu.Unlock()
		return Settings{}, err
	}
	migrate(&next)
	if err := Save(s.dir, next); err != nil {
		s.mu.Unlock()
		return Settings{}, err
	}
	s.cur = next
	s.mu.Unlock()
	s.fire()
	return clone(next), nil
}

// SaveFromGUI saves a GUI-supplied object. Each reserved slot keeps its
// current identity group (presence, worktree, branch, ports, compose project,
// enabled, serve config path) and every other field comes from in.
func (s *Store) SaveFromGUI(in Settings) (Settings, error) {
	s.mu.Lock()
	reserved, err := s.readReserved()
	if err != nil {
		s.mu.Unlock()
		return Settings{}, fmt.Errorf("reading reservations: %w", err)
	}
	if err := s.checkHeld(in, reserved); err != nil {
		s.mu.Unlock()
		return Settings{}, err
	}
	next := clone(in)
	kept := keepReserved(&next, s.cur, reserved)
	migrate(&next)
	if err := Save(s.dir, next); err != nil {
		s.mu.Unlock()
		return Settings{}, err
	}
	s.cur = next
	s.mu.Unlock()
	if kept {
		slog.Info("settings save kept reserved slot identity")
		s.fire()
	}
	return clone(next), nil
}

func (s *Store) fire() {
	if s.emit != nil {
		s.emit(EventChanged)
	}
}

func (s *Store) readReserved() (map[string]bool, error) {
	if s.reserved == nil {
		return nil, nil
	}
	return s.reserved()
}

func (s *Store) checkHeld(in Settings, reserved map[string]bool) error {
	if s.heldBy == nil {
		return nil
	}
	for _, cur := range s.cur.Servers {
		if !reserved[cur.ID] {
			continue
		}
		owner, held := s.heldBy(cur.ID)
		if !held {
			continue
		}
		for _, p := range in.Servers {
			if p.ID == cur.ID && (p.Ports != cur.Ports || p.ComposeProject != cur.ComposeProject || p.Enabled != cur.Enabled) {
				return fmt.Errorf("slot %s is held by %s; ports, compose project, and enabled can't change until it finishes", cur.ID, owner)
			}
		}
	}
	return nil
}

// keepReserved overwrites next's reserved profiles with cur's identity group
// and reports whether that changed anything.
func keepReserved(next *Settings, cur Settings, reserved map[string]bool) bool {
	kept := false
	for i, c := range cur.Servers {
		if !reserved[c.ID] {
			continue
		}
		j := -1
		for k, p := range next.Servers {
			if p.ID == c.ID {
				j = k
				break
			}
		}
		if j < 0 {
			at := min(i, len(next.Servers))
			next.Servers = append(next.Servers, ServerProfile{})
			copy(next.Servers[at+1:], next.Servers[at:])
			next.Servers[at] = c
			kept = true
			continue
		}
		p := &next.Servers[j]
		before := *p
		p.WorktreePath = c.WorktreePath
		p.Branch = c.Branch
		p.Ports = c.Ports
		p.ComposeProject = c.ComposeProject
		p.Enabled = c.Enabled
		p.FleetServe.ConfigPath = c.FleetServe.ConfigPath
		if !reflect.DeepEqual(before, *p) {
			kept = true
		}
	}
	return kept
}

// clone deep-copies through JSON so callers never share slices with the store.
func clone(s Settings) Settings {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	var out Settings
	if err := json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}
