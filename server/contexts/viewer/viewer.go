// Package viewer enables setting and reading the current
// user contexts
package viewer

import (
	"context"
	"strings"

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

// GetErrorAttributes implements ctxerr.ErrorAttributeProvider
func (v *Viewer) GetErrorAttributes() map[string]any {
	vdata := map[string]any{
		"is_logged_in": v.IsLoggedIn(),
	}
	if v.User != nil {
		vdata["sso_enabled"] = v.User.SSOEnabled
	}
	return map[string]any{
		"viewer": vdata,
	}
}

// GetTelemetryAttributes implements ctxerr.TelemetryAttributeProvider
func (v *Viewer) GetTelemetryAttributes() map[string]any {
	if v.User == nil {
		return nil
	}
	return map[string]any{
		"user.id":    v.User.ID,
		"user.email": maskEmail(v.User.Email),
	}
}

// maskEmail anonymizes an email address for telemetry by showing only
// the first character of the local part and the full domain.
// Example: "john.doe@example.com" -> "j***@example.com"
func maskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || len(parts[0]) == 0 {
		return "***"
	}
	return string(parts[0][0]) + "***@" + parts[1]
}
