package agentapi

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/tools/hangar/internal/reservations"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
)

const recoveryDoc = ".claude/skills/fleet-parallel-task/references/runtime-recovery.md"

// httpError is a refusal with the status the API returns for it.
type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

func refuse(status int, format string, args ...any) error {
	return &httpError{status: status, msg: fmt.Sprintf(format, args...)}
}

// admitted is the slot state an action was checked against.
type admitted struct {
	slot settings.ServerProfile
	res  reservations.Reservation
}

// check runs the guards shared by admission and every job step. assigned
// requires the slot to be on worktree already. A non-empty createdAt must
// still be the reservation's identity.
func (a *API) check(slotID, worktree string, assigned bool, createdAt string) (admitted, error) {
	if slotID == "s1" {
		return admitted{}, refuse(http.StatusForbidden, "s1 is never available to agents")
	}
	snap := a.opts.Store.Snapshot()
	slot, ok := findSlot(snap, slotID)
	if !ok {
		return admitted{}, refuse(http.StatusNotFound, "no slot %s", slotID)
	}
	if err := a.checkS1(snap, slot, worktree); err != nil {
		return admitted{}, err
	}
	all, err := reservations.Read(a.opts.ReservationsPath)
	if err != nil {
		return admitted{}, refuse(http.StatusConflict, "%v", err)
	}
	res, ok := all[slotID]
	switch {
	case !ok:
		return admitted{}, refuse(http.StatusForbidden, "%s has no fleet-workflow reservation", slotID)
	case !samePath(res.Worktree, worktree):
		return admitted{}, refuse(http.StatusForbidden, "%s is reserved by %s", slotID, res.Worktree)
	case createdAt != "" && res.CreatedAt != createdAt:
		return admitted{}, refuse(http.StatusConflict, "%s's reservation changed since the job started", slotID)
	case res.Ports != slot.Ports || res.ComposeProject != slot.ComposeProject:
		return admitted{}, refuse(http.StatusForbidden, "%s's ports or compose project no longer match its reservation", slotID)
	case !slot.Enabled:
		return admitted{}, refuse(http.StatusForbidden, "%s is disabled", slotID)
	case assigned && (slot.WorktreePath == nil || !samePath(*slot.WorktreePath, worktree)):
		return admitted{}, refuse(http.StatusForbidden, "%s isn't assigned to %s; assign it first", slotID, worktree)
	}
	return admitted{slot: slot, res: res}, nil
}

// checkS1 refuses anything s1 uses: its ports and the canonical defaults
// (any single one), its compose project or the default "fleet", and its
// worktree or its repo's primary checkout. Each path must be a git worktree.
func (a *API) checkS1(snap settings.Settings, slot settings.ServerProfile, worktrees ...string) error {
	s1, hasS1 := findSlot(snap, "s1")
	taken := portList(settings.DefaultPortsForIndex(0))
	if hasS1 {
		taken = append(taken, portList(s1.Ports)...)
	}
	for _, p := range portList(slot.Ports) {
		for _, t := range taken {
			if p == t {
				return refuse(http.StatusForbidden, "%s port %d collides with s1", slot.ID, p)
			}
		}
	}
	if slot.ComposeProject == "" || slot.ComposeProject == "fleet" || (hasS1 && slot.ComposeProject == s1.ComposeProject) {
		return refuse(http.StatusForbidden, "%s compose project %q collides with s1", slot.ID, slot.ComposeProject)
	}
	var s1Worktree, s1Primary string
	if hasS1 && s1.WorktreePath != nil {
		s1Worktree = *s1.WorktreePath
		p, err := a.opts.PrimaryCheckout(s1Worktree)
		if err != nil {
			return refuse(http.StatusForbidden, "finding s1's primary checkout: %v", err)
		}
		s1Primary = p
	}
	for _, wt := range worktrees {
		if s1Worktree != "" && samePath(wt, s1Worktree) {
			return refuse(http.StatusForbidden, "%s is s1's worktree", wt)
		}
		if _, err := a.opts.PrimaryCheckout(wt); err != nil {
			return refuse(http.StatusForbidden, "%s isn't a git worktree: %v", wt, err)
		}
		if s1Primary != "" && samePath(wt, s1Primary) {
			return refuse(http.StatusForbidden, "%s is s1's primary checkout", wt)
		}
	}
	return nil
}

// checkTarget validates an assign target against the reservation and returns
// the reservation's own path for it, so the caller's spelling never reaches
// the filesystem or settings. nil restores "no worktree", allowed only when
// the slot had none before.
func (a *API) checkTarget(snap settings.Settings, adm admitted, target *string) (*string, error) {
	res := adm.res
	var canon string
	switch {
	case target == nil:
		if res.PreviousWorktree != nil {
			return nil, refuse(http.StatusForbidden, "target null is allowed only when previous_worktree is null; it's %s", *res.PreviousWorktree)
		}
		return nil, nil
	case samePath(*target, res.Worktree):
		canon = res.Worktree
	case res.PreviousWorktree != nil && samePath(*target, *res.PreviousWorktree):
		canon = *res.PreviousWorktree
		if _, err := os.Stat(canon); errors.Is(err, fs.ErrNotExist) {
			return nil, refuse(http.StatusConflict, "previous_worktree %s no longer exists; recover with the helper's adoption in %s", canon, recoveryDoc)
		}
	default:
		return nil, refuse(http.StatusForbidden, "target must be the reservation's worktree or previous_worktree")
	}
	if err := a.checkS1(snap, adm.slot, canon); err != nil {
		return nil, err
	}
	return &canon, nil
}

// quiet refuses an assign while the slot's containers run. Hold already
// refuses running slot processes and perf runs aimed at the slot.
func (a *API) quiet(ctx context.Context, slot settings.ServerProfile) error {
	running, err := a.opts.Probe.ProjectRunning(ctx, slot.ComposeProject)
	if err != nil {
		return refuse(http.StatusConflict, "checking %s containers: %v", slot.ComposeProject, err)
	}
	if running {
		return refuse(http.StatusConflict, "%s has running %s containers; stop the slot first", slot.ID, slot.ComposeProject)
	}
	return nil
}

func findSlot(s settings.Settings, id string) (settings.ServerProfile, bool) {
	for _, p := range s.Servers {
		if p.ID == id {
			return p, true
		}
	}
	return settings.ServerProfile{}, false
}

func portList(p settings.ServerPorts) []uint16 {
	return []uint16{p.Server, p.MySQL, p.Redis, p.S3, p.S3Console}
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b {
		return true
	}
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && ra == rb
}

// gitPrimaryCheckout returns the main checkout of the worktree at dir, and
// errors unless dir is a worktree's top level.
func gitPrimaryCheckout(dir string) (string, error) {
	cmd := shellpath.Command("git", "-C", dir, "rev-parse", "--path-format=absolute", "--show-toplevel", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		return "", fmt.Errorf("unexpected git rev-parse output %q", out)
	}
	if !samePath(lines[0], dir) {
		return "", fmt.Errorf("top level is %s", lines[0])
	}
	return filepath.Dir(lines[1]), nil
}
