// Package reservations reads fleet-workflow's runtime-reservations.json.
// Hangar never writes it: the helper owns reservations, and Hangar uses them
// to authorize agent mutations of a slot.
package reservations

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

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

type entry struct {
	Worktree         string               `json:"worktree"`
	PreviousWorktree *string              `json:"previous_worktree"`
	Ports            settings.ServerPorts `json:"ports"`
	ComposeProject   string               `json:"compose_project"`
	// The helper writes a float timestamp; the raw text is enough for identity.
	CreatedAt json.RawMessage `json:"created_at"`
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
		out[id] = Reservation{
			Worktree:         e.Worktree,
			PreviousWorktree: e.PreviousWorktree,
			Ports:            e.Ports,
			ComposeProject:   e.ComposeProject,
			CreatedAt:        string(e.CreatedAt),
		}
	}
	return out, nil
}
