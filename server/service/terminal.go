package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// CreateTerminalSession implements fleet.Service.
//
// It verifies the caller has write access to the host, creates an in-memory
// terminal session, and returns the session ID.  The orbit agent learns about
// the session on its next config poll (OrbitConfigNotifications.PendingTerminalSessionIDs)
// and dials back to open a PTY.
func (svc *Service) CreateTerminalSession(ctx context.Context, hostID uint) (string, error) {
	// HostLite returns *fleet.Host with the primary host data (no detail columns).
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get host for terminal session")
	}

	// Terminal is a Fleet Premium feature.
	lic, _ := license.FromContext(ctx)
	if lic == nil || !lic.IsPremium() {
		return "", ctxerr.Wrap(ctx, fleet.ErrMissingLicense)
	}

	// Authorize first: standard host write check (filters unauthenticated
	// callers, observers, etc.) then further restrict to global admins only.
	// Platform validation comes after so that unauthorized callers cannot
	// probe host existence or platform via a cheaper error path.
	if err := svc.authz.Authorize(ctx, host, fleet.ActionWrite); err != nil {
		return "", err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok || vc.User == nil || vc.User.GlobalRole == nil || *vc.User.GlobalRole != fleet.RoleAdmin {
		return "", fleet.NewPermissionError("terminal sessions require global admin role")
	}

	// Reject unsupported platforms only after the caller is verified.
	if !fleet.IsLinux(host.Platform) && host.Platform != "darwin" && host.Platform != "windows" {
		return "", fleet.NewInvalidArgumentError("host_id", fmt.Sprintf("web terminal is not supported on %q hosts", host.Platform))
	}

	// Bind the session to the creating user so that markBrowserClaimed later
	// rejects any other global admin who might have obtained the session UUID.
	sessionID, _ := terminalStore.create(host.ID, host.DisplayName(), vc.User.ID)
	// Orbit is NOT notified here.  The browser WebSocket handler calls
	// terminalNotifyStore.notifyHost after the browser authenticates, so no
	// shell is ever started for a session that no browser has claimed.

	// Activity is recorded in the browser WebSocket handler, after the orbit
	// agent has connected and the shell is truly live. Recording it here
	// (at session creation) would log the event even if orbit never connects.

	return sessionID, nil
}

// pendingTerminalSessionIDsForHost returns session IDs waiting for this host.
// Called from GetOrbitConfig to populate PendingTerminalSessionIDs.
func pendingTerminalSessionIDsForHost(hostID uint) []string {
	return terminalStore.pendingForHost(hostID)
}
