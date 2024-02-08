// Package viewer enables setting and reading the current
// user contexts
package viewer

import (
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type key int

const viewerKey key = 0

// NewContext creates a new context with the current user information.
func NewContext(ctx context.Context, v Viewer) context.Context {
	return context.WithValue(ctx, viewerKey, v)
}

// FromContext returns the current user information if present.
func FromContext(ctx context.Context) (Viewer, bool) {
	v, ok := ctx.Value(viewerKey).(Viewer)
	return v, ok
}

// Viewer holds information about the current
// user and the user's session
type Viewer struct {
	User    *fleet.User
	Session *fleet.Session
}

// UserID is a helper that enables quick access to the user ID of the current
// user.
func (v Viewer) UserID() uint {
	if v.User != nil {
		return v.User.ID
	}
	return 0
}

// Email is a helper that enables quick access to the email of the current
// user.
func (v Viewer) Email() string {
	if v.User != nil {
		return v.User.Email
	}
	return ""
}

// FullName is a helper that enables quick access to the full name of the
// current user.
func (v Viewer) FullName() string {
	if v.User != nil {
		return v.User.Name
	}
	return ""
}

// SessionID returns the current user's session ID
func (v Viewer) SessionID() uint {
	if v.Session != nil {
		return v.Session.ID
	}
	return 0
}

// IsUserID returns true if the given user id the same as the user which is
// represented by this ViewerContext
func (v Viewer) IsUserID(id uint) bool {
	return v.UserID() == id
}

// IsLoggedIn determines whether or not the current VC is attached to a user
// account
func (v Viewer) IsLoggedIn() bool {
	if v.Session != nil {
		// Without having access to a service to call GetInfoAboutSession(id),
		// we can't synchronously check the database here.
		if v.Session.ID != 0 {
			return true
		}
	}
	return false
}

// CanPerformActions returns a bool indicating the current user's ability to
// perform the most basic actions on the site
func (v Viewer) CanPerformActions() bool {
	if v.User != nil {
		return v.IsLoggedIn() && !v.User.IsAdminForcedPasswordReset()
	}
	return false
}

// CanPerformPasswordReset returns a bool indicating the current user's
// ability to perform a password reset (in the case they have been required by
// the admin).
func (v Viewer) CanPerformPasswordReset() bool {
	if v.User != nil {
		return v.IsLoggedIn() && v.User.IsAdminForcedPasswordReset()
	}
	return false
}

// UserIsGitOpsOnly checks if the user in the provided context is exclusively
// assigned the GitOps role.
//
// It evaluates both global roles and team-specific
// roles of the user. The function returns true if the user has the GitOps role
// globally or in all team roles. It returns false if the user has no roles,
// non-GitOps roles, or a mix of GitOps and non-GitOps roles.
//
// In case the context does not contain a user or the user data is incomplete,
// it returns an error.
func UserIsGitOpsOnly(ctx context.Context) (bool, error) {
	vc, ok := FromContext(ctx)
	if !ok {
		return false, fleet.ErrNoContext
	}
	if vc.User == nil {
		return false, errors.New("missing user in context")
	}
	if vc.User.GlobalRole != nil {
		return *vc.User.GlobalRole == fleet.RoleGitOps, nil
	}
	if len(vc.User.Teams) == 0 {
		return false, errors.New("user has no roles")
	}
	for _, teamRole := range vc.User.Teams {
		if teamRole.Role != fleet.RoleGitOps {
			return false, nil
		}
	}
	return true, nil
}

// DetermineActionAllowingGitOps decides on an action based on the user's
// GitOps role. If the user is identified as GitOps-only, a predefined GitOps
// action (fleet.ActionWrite) is returned. Otherwise, the function returns the
// action provided as input. If there's an error in determining the user's
// GitOps role, the function returns an error.
//
// This method is useful in scenarios where certain actions are restricted or
// modified based on the user's role, specifically tailored for users with
// GitOps roles.
func DetermineActionAllowingGitOps(ctx context.Context, action string) (string, error) {
	isGitOps, err := UserIsGitOpsOnly(ctx)
	if err != nil {
		return "", fmt.Errorf("checking if user is gitops only: %w", err)
	}

	if isGitOps {
		return fleet.ActionWrite, nil
	}

	return action, nil

}
