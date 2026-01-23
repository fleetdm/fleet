package viewer

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// Weird states
	nilViewer       = Viewer{}
	noSessionViewer = Viewer{
		User: &fleet.User{
			ID:   41,
			Name: "No Session",
		},
	}

	// Regular users
	userViewer = Viewer{
		User: &fleet.User{
			ID:   45,
			Name: "Regular User",
		},
		Session: &fleet.Session{
			ID:     4,
			UserID: 45,
		},
	}

	needsPasswordResetUserViewer = Viewer{
		User: &fleet.User{
			ID:                       47,
			Name:                     "Regular User Needs Password Reset",
			AdminForcedPasswordReset: true,
		},
		Session: &fleet.Session{
			ID:     6,
			UserID: 47,
		},
	}

	// Admin users
	adminViewer = Viewer{
		User: &fleet.User{
			ID:   42,
			Name: "The Admin",
		},
		Session: &fleet.Session{
			ID:     1,
			UserID: 42,
		},
	}
	needsPasswordResetAdminViewer = Viewer{
		User: &fleet.User{
			ID:                       44,
			Name:                     "The Admin Requires Password Reset",
			AdminForcedPasswordReset: true,
		},
		Session: &fleet.Session{
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
}

func TestIsLoggedIn(t *testing.T) {
	assert.Equal(t, false, nilViewer.IsLoggedIn())
	assert.Equal(t, false, noSessionViewer.IsLoggedIn())

	assert.Equal(t, true, userViewer.IsLoggedIn())
	assert.Equal(t, true, needsPasswordResetUserViewer.IsLoggedIn())

	assert.Equal(t, true, adminViewer.IsLoggedIn())
	assert.Equal(t, true, needsPasswordResetAdminViewer.IsLoggedIn())
}

func TestCanPerformActions(t *testing.T) {
	assert.Equal(t, false, nilViewer.CanPerformActions())
	assert.Equal(t, false, noSessionViewer.CanPerformActions())

	assert.Equal(t, true, userViewer.CanPerformActions())
	assert.Equal(t, false, needsPasswordResetUserViewer.CanPerformActions())

	assert.Equal(t, true, adminViewer.CanPerformActions())
	assert.Equal(t, false, needsPasswordResetAdminViewer.CanPerformActions())
}

// TODO update these tests

// func TestCanPerformAdminActions(t *testing.T) {
// 	assert.Equal(t, false, nilViewer.CanPerformAdminActions())
// 	assert.Equal(t, false, noSessionViewer.CanPerformAdminActions())

// 	assert.Equal(t, false, userViewer.CanPerformAdminActions())
// 	assert.Equal(t, false, disabledUserViewer.CanPerformAdminActions())
// 	assert.Equal(t, false, needsPasswordResetUserViewer.CanPerformAdminActions())

// 	assert.Equal(t, true, adminViewer.CanPerformAdminActions())
// 	assert.Equal(t, false, disabledAdminViewer.CanPerformAdminActions())
// 	assert.Equal(t, false, needsPasswordResetAdminViewer.CanPerformAdminActions())
// }

// func TestCanPerformReadActionOnUser(t *testing.T) {
// 	assert.Equal(t, false, nilViewer.CanPerformReadActionOnUser(1))
// 	assert.Equal(t, false, noSessionViewer.CanPerformReadActionOnUser(1))

// 	assert.Equal(t, true, userViewer.CanPerformReadActionOnUser(1))
// 	assert.Equal(t, true, userViewer.CanPerformReadActionOnUser(userViewer.User.ID))
// 	assert.Equal(t, false, disabledUserViewer.CanPerformReadActionOnUser(1))
// 	assert.Equal(t, false, disabledUserViewer.CanPerformReadActionOnUser(disabledUserViewer.User.ID))
// 	assert.Equal(t, false, needsPasswordResetUserViewer.CanPerformReadActionOnUser(1))
// 	assert.Equal(t, true, needsPasswordResetUserViewer.CanPerformReadActionOnUser(needsPasswordResetUserViewer.User.ID))

// 	assert.Equal(t, true, adminViewer.CanPerformReadActionOnUser(1))
// 	assert.Equal(t, true, adminViewer.CanPerformReadActionOnUser(adminViewer.User.ID))
// 	assert.Equal(t, false, disabledAdminViewer.CanPerformReadActionOnUser(1))
// 	assert.Equal(t, false, disabledAdminViewer.CanPerformReadActionOnUser(disabledAdminViewer.User.ID))
// 	assert.Equal(t, false, needsPasswordResetAdminViewer.CanPerformReadActionOnUser(1))
// 	assert.Equal(t, true, needsPasswordResetAdminViewer.CanPerformReadActionOnUser(needsPasswordResetAdminViewer.User.ID))
// }

// func TestCanPerformWriteActionOnUser(t *testing.T) {
// 	assert.Equal(t, false, nilViewer.CanPerformWriteActionOnUser(1))
// 	assert.Equal(t, false, noSessionViewer.CanPerformWriteActionOnUser(1))

// 	assert.Equal(t, false, userViewer.CanPerformWriteActionOnUser(1))
// 	assert.Equal(t, true, userViewer.CanPerformWriteActionOnUser(userViewer.User.ID))
// 	assert.Equal(t, false, disabledUserViewer.CanPerformWriteActionOnUser(1))
// 	assert.Equal(t, false, disabledUserViewer.CanPerformWriteActionOnUser(disabledUserViewer.User.ID))
// 	assert.Equal(t, false, needsPasswordResetUserViewer.CanPerformWriteActionOnUser(1))
// 	assert.Equal(t, true, needsPasswordResetUserViewer.CanPerformWriteActionOnUser(needsPasswordResetUserViewer.User.ID))

// 	assert.Equal(t, true, adminViewer.CanPerformWriteActionOnUser(1))
// 	assert.Equal(t, true, adminViewer.CanPerformWriteActionOnUser(adminViewer.User.ID))
// 	assert.Equal(t, false, disabledAdminViewer.CanPerformWriteActionOnUser(1))
// 	assert.Equal(t, false, disabledAdminViewer.CanPerformWriteActionOnUser(disabledAdminViewer.User.ID))
// 	assert.Equal(t, false, needsPasswordResetAdminViewer.CanPerformWriteActionOnUser(1))
// 	assert.Equal(t, true, needsPasswordResetAdminViewer.CanPerformWriteActionOnUser(needsPasswordResetAdminViewer.User.ID))
// }

func TestMaskEmail(t *testing.T) {
	cases := []struct {
		name     string
		email    string
		expected string
	}{
		{"standard email", "john.doe@example.com", "j***@example.com"},
		{"single char local", "j@example.com", "j***@example.com"},
		{"subdomain", "user@mail.example.com", "u***@mail.example.com"},
		{"empty string", "", "***"},
		{"no at sign", "invalid", "***"},
		{"empty local part", "@example.com", "***"},
		{"only at sign", "@", "***"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := maskEmail(tc.email)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNewSystemContext(t *testing.T) {
	ctx := NewSystemContext(t.Context())

	v, ok := FromContext(ctx)
	require.True(t, ok, "viewer should be present in context")
	require.NotNil(t, v.User, "user should be present in viewer")

	// Verify the system user has the expected properties
	assert.Equal(t, fleet.ActivityAutomationAuthor, v.User.Name, "system user name should match ActivityAutomationAuthor")
	require.NotNil(t, v.User.GlobalRole, "system user should have a global role")
	assert.Equal(t, fleet.RoleAdmin, *v.User.GlobalRole, "system user should have admin role")
}
