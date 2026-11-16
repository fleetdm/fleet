// Package reservations reads and writes slot reservations: a JSON file of
// slots, each naming the worktree that holds it, written under an exclusive
// flock on a sibling .lock file.
package reservations

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

// Reservation is one slot's entry. CreatedAt is the reservation's identity:
// a new reservation for the same worktree gets a new value.
type Reservation struct {
	Worktree         string
	PreviousWorktree *string
	Ports            settings.ServerPorts
	ComposeProject   string
	CreatedAt        string
}

// entry is the on-disk shape, in the file's key order.
type entry struct {
	Worktree         string               `json:"worktree"`
	PreviousWorktree *string              `json:"previous_worktree"`
	Ports            settings.ServerPorts `json:"ports"`
	ComposeProject   string               `json:"compose_project"`
	// Always false here; kept so every entry has the format's full shape.
	Bound bool `json:"bound"`
	// Float Unix seconds; the raw text is enough for identity.
	CreatedAt json.RawMessage `json:"created_at"`
}

func (e entry) reservation() Reservation {
	return Reservation{
		Worktree:         e.Worktree,
		PreviousWorktree: e.PreviousWorktree,
		Ports:            e.Ports,
		ComposeProject:   e.ComposeProject,
		CreatedAt:        string(e.CreatedAt),
	}
}

// Read returns the reservations keyed by slot id. A missing file yields an
// empty map; a malformed one is an error so callers refuse.
func Read(path string) (map[string]Reservation, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]Reservation{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading reservations: %w", err)
	}
	var f struct {
		Slots map[string]entry `json:"slots"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parsing reservations %s: %w", path, err)
	}
	out := make(map[string]Reservation, len(f.Slots))
	for id, e := range f.Slots {
		out[id] = e.reservation()
	}
	return out, nil
}

// ErrHeld means another worktree holds the slot's reservation.
var ErrHeld = errors.New("slot is reserved by another worktree")

// ErrChanged means the reservation to release is gone or was replaced.
var ErrChanged = errors.New("reservation changed")

// OtherSlotError means the worktree already holds Slot. One worktree gets one
// slot: volumes named after the worktree would otherwise back two projects.
type OtherSlotError struct{ Slot string }

func (e *OtherSlotError) Error() string { return "worktree already holds " + e.Slot }

// canonical resolves symlinks so two spellings of one worktree compare equal.
func canonical(p string) string {
	p = filepath.Clean(p)
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// Reserve records r for slot, stamped now, unless the slot is reserved. An
// existing reservation for r.Worktree is returned as is (created false); one
// for another worktree returns ErrHeld with that reservation. A worktree that
// already holds another slot gets *OtherSlotError.
func Reserve(path, slot string, r Reservation) (res Reservation, created bool, err error) {
	want := canonical(r.Worktree)
	err = update(path, func(slots *object) (bool, error) {
		if raw, ok := slots.vals[slot]; ok {
			var e entry
			if err := json.Unmarshal(raw, &e); err != nil {
				return false, fmt.Errorf("parsing reservation %s: %w", slot, err)
			}
			res = e.reservation()
			if canonical(e.Worktree) != want {
				return false, ErrHeld
			}
			return false, nil
		}
		for _, id := range slots.keys {
			var e entry
			if err := json.Unmarshal(slots.vals[id], &e); err != nil {
				return false, fmt.Errorf("parsing reservation %s: %w", id, err)
			}
			if canonical(e.Worktree) == want {
				return false, &OtherSlotError{Slot: id}
			}
		}
		e := entry{
			Worktree: r.Worktree, PreviousWorktree: r.PreviousWorktree, Ports: r.Ports,
			ComposeProject: r.ComposeProject, CreatedAt: timestamp(time.Now()),
		}
		b, err := json.Marshal(e)
		if err != nil {
			return false, err
		}
		slots.set(slot, b)
		res, created = e.reservation(), true
		return true, nil
	})
	if err != nil && !errors.Is(err, ErrHeld) {
		return Reservation{}, false, err
	}
	return res, created, err
}

// Release deletes slot's reservation if it's still worktree's, with the
// identity createdAt the caller checked.
func Release(path, slot, worktree, createdAt string) error {
	return update(path, func(slots *object) (bool, error) {
		raw, ok := slots.vals[slot]
		if !ok {
			return false, ErrChanged
		}
		var e entry
		if err := json.Unmarshal(raw, &e); err != nil {
			return false, fmt.Errorf("parsing reservation %s: %w", slot, err)
		}
		if filepath.Clean(e.Worktree) != filepath.Clean(worktree) || string(e.CreatedAt) != createdAt {
			return false, ErrChanged
		}
		slots.del(slot)
		return true, nil
	})
}

// timestamp is created_at as float Unix seconds, e.g. 1793735621.921309.
func timestamp(t time.Time) json.RawMessage {
	s := strconv.FormatFloat(float64(t.UnixMicro())/1e6, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return json.RawMessage(s)
}

// update runs fn on the file's "slots" object under the file's lock and
// writes the result when fn reports a change. Everything fn doesn't touch is
// written back with the same values.
func update(path string, fn func(slots *object) (bool, error)) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating reservations dir: %w", err)
	}
	// reservations.json locks through reservations.lock beside it.
	lockPath := strings.TrimSuffix(path, ".json") + ".lock"
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("opening %s: %w", lockPath, err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("locking %s: %w", lockPath, err)
	}

	top := &object{vals: map[string]json.RawMessage{}}
	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return fmt.Errorf("reading reservations: %w", err)
	default:
		if err := json.Unmarshal(b, top); err != nil {
			return fmt.Errorf("parsing reservations %s: %w", path, err)
		}
	}
	slots := &object{vals: map[string]json.RawMessage{}}
	if raw, ok := top.vals["slots"]; ok && string(raw) != "null" {
		if err := json.Unmarshal(raw, slots); err != nil {
			return fmt.Errorf("parsing reservations %s: %w", path, err)
		}
	}
	changed, err := fn(slots)
	if err != nil || !changed {
		return err
	}
	sb, err := slots.MarshalJSON()
	if err != nil {
		return err
	}
	top.set("slots", sb)

	// Two-space indent and a trailing newline; values keep their raw bytes.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(top); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".reservations-*.tmp")
	if err != nil {
		return fmt.Errorf("writing reservations: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		return fmt.Errorf("writing reservations: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing reservations: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("writing reservations: %w", err)
	}
	return nil
}

// object is a JSON object that keeps its key order and each value's raw bytes,
// so fields Hangar doesn't know survive a rewrite by value. Keys are decoded
// and re-encoded: an escaped key keeps its meaning, not its spelling.
type object struct {
	keys []string
	vals map[string]json.RawMessage
}

func (o *object) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return errors.New("expected a JSON object")
	}
	o.keys, o.vals = nil, map[string]json.RawMessage{}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return err
		}
		o.set(t.(string), v)
	}
	_, err := dec.Token()
	return err
}

func (o *object) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		b.Write(kb)
		b.WriteByte(':')
		b.Write(o.vals[k])
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

func (o *object) set(k string, v json.RawMessage) {
	if _, ok := o.vals[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.vals[k] = v
}

func (o *object) del(k string) {
	delete(o.vals, k)
	for i, key := range o.keys {
		if key == k {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			return
		}
	}
}
