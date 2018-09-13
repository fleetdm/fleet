package viewer

import (
	"context"
	"testing"

	"github.com/kolide/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
)

var (
	// Weird states
	nilViewer       = Viewer{}
	noSessionViewer = Viewer{
		User: &kolide.User{
			ID:       41,
			Name:     "No Session",
			Username: "nosession",
			Enabled:  true,
		},
	}

	// Regular users
	userViewer = Viewer{
		User: &kolide.User{
			ID:       45,
			Name:     "Regular User",
			Username: "user",
			Admin:    false,
			Enabled:  true,
		},
		Session: &kolide.Session{
			ID:     4,
			UserID: 45,
		},
	}
	disabledUserViewer = Viewer{
		User: &kolide.User{
			ID:       46,
			Name:     "Disabled Regular User",
			Username: "disabled_user",
			Admin:    false,
			Enabled:  false,
		},
		Session: &kolide.Session{
			ID:     5,
			UserID: 46,
		},
	}

	needsPasswordResetUserViewer = Viewer{
		User: &kolide.User{
			ID:                       47,
			Name:                     "Regular User Needs Password Reset",
			Username:                 "reset_user",
			Admin:                    false,
			Enabled:                  true,
			AdminForcedPasswordReset: true,
		},
		Session: &kolide.Session{
			ID:     6,
			UserID: 47,
		},
	}

	// Admin users
	adminViewer = Viewer{
		User: &kolide.User{
			ID:       42,
			Name:     "The Admin",
			Username: "admin",
			Admin:    true,
			Enabled:  true,
		},
		Session: &kolide.Session{
			ID:     1,
			UserID: 42,
		},
	}
	disabledAdminViewer = Viewer{
		User: &kolide.User{
			ID:       43,
			Name:     "The Disabled Admin",
			Username: "disabled_admin",
			Admin:    true,
			Enabled:  false,
		},
		Session: &kolide.Session{
			ID:     2,
			UserID: 43,
		},
	}
	needsPasswordResetAdminViewer = Viewer{
		User: &kolide.User{
			ID:                       44,
			Name:                     "The Admin Requires Password Reset",
			Username:                 "reset_admin",
			Admin:                    true,
			Enabled:                  true,
			AdminForcedPasswordReset: true,
		},
		Session: &kolide.Session{
			ID:     3,
			UserID: 44,
		},
	}
)

func TestContext(t *testing.T) {
	ctx := NewContext(context.Background(), userViewer)
	v, ok := FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, userViewer, v)
}

func TestIsUserID(t *testing.T) {
	assert.True(t, adminViewer.IsUserID(42))
	assert.False(t, adminViewer.IsUserID(7))
	assert.True(t, userViewer.IsUserID(45))
	assert.True(t, disabledUserViewer.IsUserID(46))
}

func TestIsLoggedIn(t *testing.T) {
	assert.Equal(t, false, nilViewer.IsLoggedIn())
	assert.Equal(t, false, noSessionViewer.IsLoggedIn())

	assert.Equal(t, true, userViewer.IsLoggedIn())
	assert.Equal(t, false, disabledUserViewer.IsLoggedIn())
	assert.Equal(t, true, needsPasswordResetUserViewer.IsLoggedIn())

	assert.Equal(t, true, adminViewer.IsLoggedIn())
	assert.Equal(t, false, disabledAdminViewer.IsLoggedIn())
	assert.Equal(t, true, needsPasswordResetAdminViewer.IsLoggedIn())
}

func TestCanPerformActions(t *testing.T) {
	assert.Equal(t, false, nilViewer.CanPerformActions())
	assert.Equal(t, false, noSessionViewer.CanPerformActions())

	assert.Equal(t, true, userViewer.CanPerformActions())
	assert.Equal(t, false, disabledUserViewer.CanPerformActions())
	assert.Equal(t, false, needsPasswordResetUserViewer.CanPerformActions())

	assert.Equal(t, true, adminViewer.CanPerformActions())
	assert.Equal(t, false, disabledAdminViewer.CanPerformActions())
	assert.Equal(t, false, needsPasswordResetAdminViewer.CanPerformActions())
}

func TestCanPerformAdminActions(t *testing.T) {
	assert.Equal(t, false, nilViewer.CanPerformAdminActions())
	assert.Equal(t, false, noSessionViewer.CanPerformAdminActions())

	assert.Equal(t, false, userViewer.CanPerformAdminActions())
	assert.Equal(t, false, disabledUserViewer.CanPerformAdminActions())
	assert.Equal(t, false, needsPasswordResetUserViewer.CanPerformAdminActions())

	assert.Equal(t, true, adminViewer.CanPerformAdminActions())
	assert.Equal(t, false, disabledAdminViewer.CanPerformAdminActions())
	assert.Equal(t, false, needsPasswordResetAdminViewer.CanPerformAdminActions())
}

func TestCanPerformReadActionOnUser(t *testing.T) {
	assert.Equal(t, false, nilViewer.CanPerformReadActionOnUser(1))
	assert.Equal(t, false, noSessionViewer.CanPerformReadActionOnUser(1))

	assert.Equal(t, true, userViewer.CanPerformReadActionOnUser(1))
	assert.Equal(t, true, userViewer.CanPerformReadActionOnUser(userViewer.User.ID))
	assert.Equal(t, false, disabledUserViewer.CanPerformReadActionOnUser(1))
	assert.Equal(t, false, disabledUserViewer.CanPerformReadActionOnUser(disabledUserViewer.User.ID))
	assert.Equal(t, false, needsPasswordResetUserViewer.CanPerformReadActionOnUser(1))
	assert.Equal(t, true, needsPasswordResetUserViewer.CanPerformReadActionOnUser(needsPasswordResetUserViewer.User.ID))

	assert.Equal(t, true, adminViewer.CanPerformReadActionOnUser(1))
	assert.Equal(t, true, adminViewer.CanPerformReadActionOnUser(adminViewer.User.ID))
	assert.Equal(t, false, disabledAdminViewer.CanPerformReadActionOnUser(1))
	assert.Equal(t, false, disabledAdminViewer.CanPerformReadActionOnUser(disabledAdminViewer.User.ID))
	assert.Equal(t, false, needsPasswordResetAdminViewer.CanPerformReadActionOnUser(1))
	assert.Equal(t, true, needsPasswordResetAdminViewer.CanPerformReadActionOnUser(needsPasswordResetAdminViewer.User.ID))
}

func TestCanPerformWriteActionOnUser(t *testing.T) {
	assert.Equal(t, false, nilViewer.CanPerformWriteActionOnUser(1))
	assert.Equal(t, false, noSessionViewer.CanPerformWriteActionOnUser(1))

	assert.Equal(t, false, userViewer.CanPerformWriteActionOnUser(1))
	assert.Equal(t, true, userViewer.CanPerformWriteActionOnUser(userViewer.User.ID))
	assert.Equal(t, false, disabledUserViewer.CanPerformWriteActionOnUser(1))
	assert.Equal(t, false, disabledUserViewer.CanPerformWriteActionOnUser(disabledUserViewer.User.ID))
	assert.Equal(t, false, needsPasswordResetUserViewer.CanPerformWriteActionOnUser(1))
	assert.Equal(t, true, needsPasswordResetUserViewer.CanPerformWriteActionOnUser(needsPasswordResetUserViewer.User.ID))

	assert.Equal(t, true, adminViewer.CanPerformWriteActionOnUser(1))
	assert.Equal(t, true, adminViewer.CanPerformWriteActionOnUser(adminViewer.User.ID))
	assert.Equal(t, false, disabledAdminViewer.CanPerformWriteActionOnUser(1))
	assert.Equal(t, false, disabledAdminViewer.CanPerformWriteActionOnUser(disabledAdminViewer.User.ID))
	assert.Equal(t, false, needsPasswordResetAdminViewer.CanPerformWriteActionOnUser(1))
	assert.Equal(t, true, needsPasswordResetAdminViewer.CanPerformWriteActionOnUser(needsPasswordResetAdminViewer.User.ID))
}

func TestCanPerformPasswordReset(t *testing.T) {
	assert.Equal(t, false, nilViewer.CanPerformPasswordReset())
	assert.Equal(t, false, noSessionViewer.CanPerformPasswordReset())

	assert.Equal(t, false, userViewer.CanPerformPasswordReset())
	assert.Equal(t, false, disabledUserViewer.CanPerformPasswordReset())
	assert.Equal(t, true, needsPasswordResetUserViewer.CanPerformPasswordReset())

	assert.Equal(t, false, adminViewer.CanPerformPasswordReset())
	assert.Equal(t, false, disabledAdminViewer.CanPerformPasswordReset())
	assert.Equal(t, true, needsPasswordResetAdminViewer.CanPerformPasswordReset())

}
