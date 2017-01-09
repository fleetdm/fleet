// Package viewer enables setting and reading the current
// user contexts
package viewer

import (
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
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

// IsAdmin indicates whether or not the current user can perform administrative
// actions.
func (v Viewer) IsAdmin() bool {
	if v.User != nil {
		return v.User.Admin && v.User.Enabled
	}
	return false
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
	return "none"
}

// FullName is a helper that enables quick access to the full name of the
// current user.
func (v Viewer) FullName() string {
	if v.User != nil {
		return v.User.Name
	}
	return "none"
}

// SessionID returns the current user's session ID
func (v Viewer) SessionID() uint {
	if v.Session != nil {
		return v.Session.ID
	}
	return 0
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

// IsUserID returns true if the given user id the same as the user which is
// represented by this ViewerContext
func (v Viewer) IsUserID(id uint) bool {
	if v.UserID() == id {
		return true
	}
	return false
}

// CanPerformReadActionsOnUser returns a bool indicating the current user's
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
		// By not requiring v.CanPerformActions() here, we allow the
		// user to update their password when they are in the forced
		// password reset state.
		return v.IsUserID(uid) || v.IsAdmin()
	}
	return false
}
