// Package viewer enables setting and reading the current
// user contexts
package viewer

import (
	"context"

	"github.com/kolide/fleet/server/kolide"
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
	User    *kolide.User
	Session *kolide.Session
}

// UserID is a helper that enables quick access to the user ID of the current
// user.
func (v Viewer) UserID() uint {
	if v.User != nil {
		return v.User.ID
	}
	return 0
}

// Username is a helper that enables quick access to the username of the current
// user.
func (v Viewer) Username() string {
	if v.User != nil {
		return v.User.Username
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
	if v.User != nil {
		if !v.User.Enabled {
			return false
		}
	}
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
		return v.IsLoggedIn() && !v.User.AdminForcedPasswordReset
	}
	return false
}

// CanPerformAdminActions indicates whether or not the current user can perform
// administrative actions.
func (v Viewer) CanPerformAdminActions() bool {
	if v.User != nil {
		return v.CanPerformActions() && v.User.Admin
	}
	return false
}

// CanPerformReadActionOnUser returns a bool indicating the current user's
// ability to perform read actions on the given user
func (v Viewer) CanPerformReadActionOnUser(uid uint) bool {
	if v.User != nil {
		return v.CanPerformActions() || (v.IsLoggedIn() && v.IsUserID(uid))
	}
	return false
}

// CanPerformWriteActionOnUser returns a bool indicating the current user's
// ability to perform write actions on the given user
func (v Viewer) CanPerformWriteActionOnUser(uid uint) bool {
	if v.User != nil {
		return (v.IsLoggedIn() && v.IsUserID(uid)) || v.CanPerformAdminActions()
	}
	return false
}

// CanPerformPasswordReset returns a bool indicating the current user's
// ability to perform a password reset (in the case they have been required by
// the admin).
func (v Viewer) CanPerformPasswordReset() bool {
	if v.User != nil {
		return v.IsLoggedIn() && v.User.AdminForcedPasswordReset
	}
	return false
}
