package viewer

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
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

func TestUserIsGitOpsOnly(t *testing.T) {
	for _, tc := range []struct {
		name       string
		user       *fleet.User
		expectedFn func(value bool, err error) bool
	}{
		{
			name: "missing user in context",
			user: nil,
			expectedFn: func(value bool, err error) bool {
				return err != nil && !value
			},
		},
		{
			name: "no roles",
			user: &fleet.User{},
			expectedFn: func(value bool, err error) bool {
				return err != nil && !value
			},
		},
		{
			name: "global gitops",
			user: &fleet.User{
				GlobalRole: ptr.String(fleet.RoleGitOps),
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && value
			},
		},
		{
			name: "global non-gitops",
			user: &fleet.User{
				GlobalRole: ptr.String(fleet.RoleObserver),
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && !value
			},
		},
		{
			name: "team gitops",
			user: &fleet.User{
				GlobalRole: nil,
				Teams: []fleet.UserTeam{
					{
						Team: fleet.Team{ID: 1},
						Role: fleet.RoleGitOps,
					},
				},
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && value
			},
		},
		{
			name: "multiple team gitops",
			user: &fleet.User{
				GlobalRole: nil,
				Teams: []fleet.UserTeam{
					{
						Team: fleet.Team{ID: 1},
						Role: fleet.RoleGitOps,
					},
					{
						Team: fleet.Team{ID: 2},
						Role: fleet.RoleGitOps,
					},
				},
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && value
			},
		},
		{
			name: "multiple teams, not all gitops",
			user: &fleet.User{
				GlobalRole: nil,
				Teams: []fleet.UserTeam{
					{
						Team: fleet.Team{ID: 1},
						Role: fleet.RoleObserver,
					},
					{
						Team: fleet.Team{ID: 2},
						Role: fleet.RoleGitOps,
					},
				},
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && !value
			},
		},
		{
			name: "multiple teams, none gitops",
			user: &fleet.User{
				GlobalRole: nil,
				Teams: []fleet.UserTeam{
					{
						Team: fleet.Team{ID: 1},
						Role: fleet.RoleObserver,
					},
					{
						Team: fleet.Team{ID: 2},
						Role: fleet.RoleMaintainer,
					},
				},
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && !value
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := UserIsGitOpsOnly(NewContext(context.Background(), Viewer{User: tc.user}))
			require.True(t, tc.expectedFn(actual, err))
		})
	}
}

func TestDetermineActionAllowingGitOps(t *testing.T) {
	for _, tc := range []struct {
		name      string
		ctx       context.Context
		action    string
		expected  string
		expectErr bool
	}{
		{
			name: "gitops user, default action",
			ctx: NewContext(context.Background(), Viewer{User: &fleet.User{
				GlobalRole: ptr.String(fleet.RoleGitOps),
			}}),
			action:    "default_action",
			expected:  fleet.ActionWrite,
			expectErr: false,
		},
		{
			name: "non-gitops user, default action",
			ctx: NewContext(context.Background(), Viewer{User: &fleet.User{
				GlobalRole: ptr.String(fleet.RoleObserver),
			}}),
			action:    "default_action",
			expected:  "default_action",
			expectErr: false,
		},
		{
			name:      "no user in context",
			ctx:       NewContext(context.Background(), Viewer{User: nil}),
			action:    "default_action",
			expected:  "",
			expectErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := DetermineActionAllowingGitOps(tc.ctx, tc.action)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, actual)
			}
		})
	}
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
