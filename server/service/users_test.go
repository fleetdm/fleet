package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	const (
		teamID      = 1
		otherTeamID = 2
	)
	ds.TeamsSummaryFunc = func(ctx context.Context) ([]*fleet.TeamSummary, error) {
		team1 := &fleet.TeamSummary{ID: teamID}
		team2 := &fleet.TeamSummary{ID: otherTeamID}
		return []*fleet.TeamSummary{team1, team2}, nil
	}
	ds.InviteByTokenFunc = func(ctx context.Context, token string) (*fleet.Invite, error) {
		return &fleet.Invite{
			Email: "some@email.com",
			Token: "ABCD",
			UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
				CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
				UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Now()},
			},
		}, nil
	}
	ds.NewUserFunc = func(ctx context.Context, user *fleet.User) (*fleet.User, error) {
		return &fleet.User{}, nil
	}
	ds.DeleteInviteFunc = func(ctx context.Context, id uint) error {
		return nil
	}
	ds.InviteByEmailFunc = func(ctx context.Context, email string) (*fleet.Invite, error) {
		return nil, errors.New("AA")
	}

	userTeamMaintainerID := uint(999)
	userGlobalMaintainerID := uint(888)
	var self *fleet.User // to be set by tests
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		switch id {
		case userTeamMaintainerID:
			return &fleet.User{
				ID:    userTeamMaintainerID,
				Teams: []fleet.UserTeam{{Team: fleet.Team{ID: teamID}, Role: fleet.RoleMaintainer}},
			}, nil
		case userGlobalMaintainerID:
			return &fleet.User{
				ID:         userGlobalMaintainerID,
				GlobalRole: ptr.String(fleet.RoleMaintainer),
			}, nil
		default:
			return self, nil
		}
	}

	ds.SaveUserFunc = func(ctx context.Context, user *fleet.User) error {
		return nil
	}
	ds.ListUsersFunc = func(ctx context.Context, opts fleet.UserListOptions) ([]*fleet.User, error) {
		return nil, nil
	}
	ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
		return nil
	}
	ds.DestroyAllSessionsForUserFunc = func(ctx context.Context, id uint) error {
		return nil
	}
	ds.ListSessionsForUserFunc = func(ctx context.Context, id uint) ([]*fleet.Session, error) {
		return nil, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}

	testCases := []struct {
		name string
		user *fleet.User

		shouldFailGlobalWrite bool
		shouldFailTeamWrite   bool

		shouldFailWriteRoleGlobalToGlobal    bool
		shouldFailWriteRoleGlobalToTeam      bool
		shouldFailWriteRoleTeamToAnotherTeam bool
		shouldFailWriteRoleTeamToGlobal      bool

		shouldFailWriteRoleOwnDomain bool

		shouldFailGlobalRead bool
		shouldFailTeamRead   bool

		shouldFailGlobalDelete bool
		shouldFailTeamDelete   bool

		shouldFailGlobalPasswordReset bool
		shouldFailTeamPasswordReset   bool

		shouldFailGlobalChangePassword bool
		shouldFailTeamChangePassword   bool

		shouldFailListAll  bool
		shouldFailListTeam bool
	}{
		{
			name:                                 "global admin",
			user:                                 &fleet.User{ID: 1000, GlobalRole: ptr.String(fleet.RoleAdmin)},
			shouldFailGlobalWrite:                false,
			shouldFailTeamWrite:                  false,
			shouldFailWriteRoleGlobalToGlobal:    false,
			shouldFailWriteRoleGlobalToTeam:      false,
			shouldFailWriteRoleTeamToAnotherTeam: false,
			shouldFailWriteRoleTeamToGlobal:      false,
			shouldFailWriteRoleOwnDomain:         false,
			shouldFailGlobalRead:                 false,
			shouldFailTeamRead:                   false,
			shouldFailGlobalDelete:               false,
			shouldFailTeamDelete:                 false,
			shouldFailGlobalPasswordReset:        false,
			shouldFailTeamPasswordReset:          false,
			shouldFailGlobalChangePassword:       false,
			shouldFailTeamChangePassword:         false,
			shouldFailListAll:                    false,
			shouldFailListTeam:                   false,
		},
		{
			name:                                 "global maintainer",
			user:                                 &fleet.User{ID: 1000, GlobalRole: ptr.String(fleet.RoleMaintainer)},
			shouldFailGlobalWrite:                true,
			shouldFailTeamWrite:                  true,
			shouldFailWriteRoleGlobalToGlobal:    true,
			shouldFailWriteRoleGlobalToTeam:      true,
			shouldFailWriteRoleTeamToAnotherTeam: true,
			shouldFailWriteRoleTeamToGlobal:      true,
			shouldFailWriteRoleOwnDomain:         true,
			shouldFailGlobalRead:                 true,
			shouldFailTeamRead:                   true,
			shouldFailGlobalDelete:               true,
			shouldFailTeamDelete:                 true,
			shouldFailGlobalPasswordReset:        true,
			shouldFailTeamPasswordReset:          true,
			shouldFailGlobalChangePassword:       true,
			shouldFailTeamChangePassword:         true,
			shouldFailListAll:                    true,
			shouldFailListTeam:                   true,
		},
		{
			name:                                 "global observer",
			user:                                 &fleet.User{ID: 1000, GlobalRole: ptr.String(fleet.RoleObserver)},
			shouldFailGlobalWrite:                true,
			shouldFailTeamWrite:                  true,
			shouldFailWriteRoleGlobalToGlobal:    true,
			shouldFailWriteRoleGlobalToTeam:      true,
			shouldFailWriteRoleTeamToAnotherTeam: true,
			shouldFailWriteRoleTeamToGlobal:      true,
			shouldFailWriteRoleOwnDomain:         true,
			shouldFailGlobalRead:                 true,
			shouldFailTeamRead:                   true,
			shouldFailGlobalDelete:               true,
			shouldFailTeamDelete:                 true,
			shouldFailGlobalPasswordReset:        true,
			shouldFailTeamPasswordReset:          true,
			shouldFailGlobalChangePassword:       true,
			shouldFailTeamChangePassword:         true,
			shouldFailListAll:                    true,
			shouldFailListTeam:                   true,
		},
		{
			name:                                 "team admin, belongs to team",
			user:                                 &fleet.User{ID: 1000, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: teamID}, Role: fleet.RoleAdmin}}},
			shouldFailGlobalWrite:                true,
			shouldFailTeamWrite:                  false,
			shouldFailWriteRoleGlobalToGlobal:    true,
			shouldFailWriteRoleGlobalToTeam:      true,
			shouldFailWriteRoleTeamToAnotherTeam: true,
			shouldFailWriteRoleTeamToGlobal:      true,
			shouldFailWriteRoleOwnDomain:         false,
			shouldFailGlobalRead:                 true,
			shouldFailTeamRead:                   false,
			shouldFailGlobalDelete:               true,
			shouldFailTeamDelete:                 false,
			shouldFailGlobalPasswordReset:        true,
			shouldFailTeamPasswordReset:          true,
			shouldFailGlobalChangePassword:       true,
			shouldFailTeamChangePassword:         true,
			shouldFailListAll:                    true,
			shouldFailListTeam:                   false,
		},
		{
			name:                                 "team maintainer, belongs to team",
			user:                                 &fleet.User{ID: 1000, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: teamID}, Role: fleet.RoleMaintainer}}},
			shouldFailGlobalWrite:                true,
			shouldFailTeamWrite:                  true,
			shouldFailWriteRoleGlobalToGlobal:    true,
			shouldFailWriteRoleGlobalToTeam:      true,
			shouldFailWriteRoleTeamToAnotherTeam: true,
			shouldFailWriteRoleTeamToGlobal:      true,
			shouldFailWriteRoleOwnDomain:         true,
			shouldFailGlobalRead:                 true,
			shouldFailTeamRead:                   true,
			shouldFailGlobalDelete:               true,
			shouldFailTeamDelete:                 true,
			shouldFailGlobalPasswordReset:        true,
			shouldFailTeamPasswordReset:          true,
			shouldFailGlobalChangePassword:       true,
			shouldFailTeamChangePassword:         true,
			shouldFailListAll:                    true,
			shouldFailListTeam:                   true,
		},
		{
			name:                                 "team observer, belongs to team",
			user:                                 &fleet.User{ID: 1000, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: teamID}, Role: fleet.RoleObserver}}},
			shouldFailGlobalWrite:                true,
			shouldFailTeamWrite:                  true,
			shouldFailWriteRoleGlobalToGlobal:    true,
			shouldFailWriteRoleGlobalToTeam:      true,
			shouldFailWriteRoleTeamToAnotherTeam: true,
			shouldFailWriteRoleTeamToGlobal:      true,
			shouldFailWriteRoleOwnDomain:         true,
			shouldFailGlobalRead:                 true,
			shouldFailTeamRead:                   true,
			shouldFailGlobalDelete:               true,
			shouldFailTeamDelete:                 true,
			shouldFailGlobalPasswordReset:        true,
			shouldFailTeamPasswordReset:          true,
			shouldFailGlobalChangePassword:       true,
			shouldFailTeamChangePassword:         true,
			shouldFailListAll:                    true,
			shouldFailListTeam:                   true,
		},
		{
			name:                                 "team maintainer, DOES NOT belong to team",
			user:                                 &fleet.User{ID: 1000, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: otherTeamID}, Role: fleet.RoleMaintainer}}},
			shouldFailGlobalWrite:                true,
			shouldFailTeamWrite:                  true,
			shouldFailWriteRoleGlobalToGlobal:    true,
			shouldFailWriteRoleGlobalToTeam:      true,
			shouldFailWriteRoleTeamToAnotherTeam: true,
			shouldFailWriteRoleTeamToGlobal:      true,
			shouldFailWriteRoleOwnDomain:         true,
			shouldFailGlobalRead:                 true,
			shouldFailTeamRead:                   true,
			shouldFailGlobalDelete:               true,
			shouldFailTeamDelete:                 true,
			shouldFailGlobalPasswordReset:        true,
			shouldFailTeamPasswordReset:          true,
			shouldFailGlobalChangePassword:       true,
			shouldFailTeamChangePassword:         true,
			shouldFailListAll:                    true,
			shouldFailListTeam:                   true,
		},
		{
			name:                                 "team admin, DOES NOT belong to team",
			user:                                 &fleet.User{ID: 1000, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: otherTeamID}, Role: fleet.RoleAdmin}}},
			shouldFailGlobalWrite:                true,
			shouldFailTeamWrite:                  true,
			shouldFailWriteRoleGlobalToGlobal:    true,
			shouldFailWriteRoleGlobalToTeam:      true,
			shouldFailWriteRoleTeamToAnotherTeam: true,
			shouldFailWriteRoleTeamToGlobal:      true,
			shouldFailWriteRoleOwnDomain:         false, // this is testing changing its own role in the team it belongs to.
			shouldFailGlobalRead:                 true,
			shouldFailTeamRead:                   true,
			shouldFailGlobalDelete:               true,
			shouldFailTeamDelete:                 true,
			shouldFailGlobalPasswordReset:        true,
			shouldFailTeamPasswordReset:          true,
			shouldFailGlobalChangePassword:       true,
			shouldFailTeamChangePassword:         true,
			shouldFailListAll:                    true,
			shouldFailListTeam:                   true,
		},
		{
			name:                                 "team observer, DOES NOT belong to team",
			user:                                 &fleet.User{ID: 1000, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: otherTeamID}, Role: fleet.RoleObserver}}},
			shouldFailGlobalWrite:                true,
			shouldFailTeamWrite:                  true,
			shouldFailWriteRoleGlobalToGlobal:    true,
			shouldFailWriteRoleGlobalToTeam:      true,
			shouldFailWriteRoleTeamToAnotherTeam: true,
			shouldFailWriteRoleTeamToGlobal:      true,
			shouldFailWriteRoleOwnDomain:         true,
			shouldFailGlobalRead:                 true,
			shouldFailTeamRead:                   true,
			shouldFailGlobalDelete:               true,
			shouldFailTeamDelete:                 true,
			shouldFailGlobalPasswordReset:        true,
			shouldFailTeamPasswordReset:          true,
			shouldFailGlobalChangePassword:       true,
			shouldFailTeamChangePassword:         true,
			shouldFailListAll:                    true,
			shouldFailListTeam:                   true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			err := tt.user.SetPassword(test.GoodPassword, 10, 10)
			require.NoError(t, err)

			// To test a user reading/modifying itself.
			u := *tt.user
			self = &u

			// A user can always read itself (read rego action).
			_, err = svc.User(ctx, tt.user.ID)
			require.NoError(t, err)

			// A user can always write itself (write rego action).
			_, err = svc.ModifyUser(ctx, tt.user.ID, fleet.UserPayload{Name: ptr.String("Foo")})
			require.NoError(t, err)

			// A user can always change its own password (change_password rego action).
			_, err = svc.ModifyUser(ctx, tt.user.ID, fleet.UserPayload{Password: ptr.String(test.GoodPassword), NewPassword: ptr.String(test.GoodPassword2)})
			require.NoError(t, err)

			changeRole := func(role string) string {
				switch role {
				case fleet.RoleMaintainer:
					return fleet.RoleAdmin // promote
				case fleet.RoleAdmin:
					return fleet.RoleMaintainer // demote
				case fleet.RoleObserver:
					return fleet.RoleAdmin // promote
				default:
					t.Fatalf("unknown role: %s", role)
					return ""
				}
			}

			// Test a user modifying its own role within its domain (write_role rego action).
			if tt.user.GlobalRole != nil {
				_, err = svc.ModifyUser(ctx, tt.user.ID, fleet.UserPayload{GlobalRole: ptr.String(changeRole(*tt.user.GlobalRole))})
				checkAuthErr(t, tt.shouldFailWriteRoleOwnDomain, err)
			} else { // Team user
				ownTeamDifferentRole := []fleet.UserTeam{
					{
						Team: fleet.Team{ID: tt.user.Teams[0].ID},
						Role: changeRole(tt.user.Teams[0].Role),
					},
				}
				_, err = svc.ModifyUser(ctx, tt.user.ID, fleet.UserPayload{Teams: &ownTeamDifferentRole})
				checkAuthErr(t, tt.shouldFailWriteRoleOwnDomain, err)
			}

			teams := []fleet.UserTeam{{Team: fleet.Team{ID: teamID}, Role: fleet.RoleMaintainer}}
			_, _, err = svc.CreateUser(ctx, fleet.UserPayload{
				Name:     ptr.String("Some Name"),
				Email:    ptr.String("some@email.com"),
				Password: ptr.String(test.GoodPassword),
				Teams:    &teams,
			})
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			_, _, err = svc.CreateUser(ctx, fleet.UserPayload{
				Name:       ptr.String("Some Name"),
				Email:      ptr.String("some@email.com"),
				Password:   ptr.String(test.GoodPassword),
				GlobalRole: ptr.String(fleet.RoleAdmin),
			})
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			_, err = svc.ModifyUser(ctx, userGlobalMaintainerID, fleet.UserPayload{Name: ptr.String("Foo")})
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			_, err = svc.ModifyUser(ctx, userTeamMaintainerID, fleet.UserPayload{Name: ptr.String("Bar")})
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			_, err = svc.ModifyUser(ctx, userGlobalMaintainerID, fleet.UserPayload{GlobalRole: ptr.String(fleet.RoleMaintainer)})
			checkAuthErr(t, tt.shouldFailWriteRoleGlobalToGlobal, err)

			_, err = svc.ModifyUser(ctx, userGlobalMaintainerID, fleet.UserPayload{Teams: &teams})
			checkAuthErr(t, tt.shouldFailWriteRoleGlobalToTeam, err)

			anotherTeams := []fleet.UserTeam{{Team: fleet.Team{ID: otherTeamID}, Role: fleet.RoleMaintainer}}
			_, err = svc.ModifyUser(ctx, userTeamMaintainerID, fleet.UserPayload{Teams: &anotherTeams})
			checkAuthErr(t, tt.shouldFailWriteRoleTeamToAnotherTeam, err)

			_, err = svc.ModifyUser(ctx, userTeamMaintainerID, fleet.UserPayload{GlobalRole: ptr.String(fleet.RoleMaintainer)})
			checkAuthErr(t, tt.shouldFailWriteRoleTeamToGlobal, err)

			_, err = svc.User(ctx, userGlobalMaintainerID)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)

			_, err = svc.User(ctx, userTeamMaintainerID)
			checkAuthErr(t, tt.shouldFailTeamRead, err)

			err = svc.DeleteUser(ctx, userGlobalMaintainerID)
			checkAuthErr(t, tt.shouldFailGlobalDelete, err)

			err = svc.DeleteUser(ctx, userTeamMaintainerID)
			checkAuthErr(t, tt.shouldFailTeamDelete, err)

			_, err = svc.RequirePasswordReset(ctx, userGlobalMaintainerID, false)
			checkAuthErr(t, tt.shouldFailGlobalPasswordReset, err)

			_, err = svc.RequirePasswordReset(ctx, userTeamMaintainerID, false)
			checkAuthErr(t, tt.shouldFailTeamPasswordReset, err)

			_, err = svc.ModifyUser(ctx, userGlobalMaintainerID, fleet.UserPayload{NewPassword: ptr.String(test.GoodPassword2)})
			checkAuthErr(t, tt.shouldFailGlobalChangePassword, err)

			_, err = svc.ModifyUser(ctx, userTeamMaintainerID, fleet.UserPayload{NewPassword: ptr.String(test.GoodPassword2)})
			checkAuthErr(t, tt.shouldFailTeamChangePassword, err)

			_, err = svc.ListUsers(ctx, fleet.UserListOptions{})
			checkAuthErr(t, tt.shouldFailListAll, err)

			_, err = svc.ListUsers(ctx, fleet.UserListOptions{TeamID: teamID})
			checkAuthErr(t, tt.shouldFailListTeam, err)
		})
	}
}

func TestModifyUserEmail(t *testing.T) {
	user := &fleet.User{
		ID:    3,
		Email: "foo@bar.com",
	}
	err := user.SetPassword(test.GoodPassword, 10, 10)
	require.NoError(t, err)
	ms := new(mock.Store)
	ms.PendingEmailChangeFunc = func(ctx context.Context, id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return user, nil
	}
	ms.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		return nil, notFoundErr{}
	}
	ms.InviteByEmailFunc = func(ctx context.Context, email string) (*fleet.Invite, error) {
		return nil, notFoundErr{}
	}
	ms.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		config := &fleet.AppConfig{
			SMTPSettings: &fleet.SMTPSettings{
				SMTPConfigured:         true,
				SMTPAuthenticationType: fleet.AuthTypeNameNone,
				SMTPPort:               1025,
				SMTPServer:             "127.0.0.1",
				SMTPSenderAddress:      "xxx@fleet.co",
			},
		}
		return config, nil
	}
	ms.SaveUserFunc = func(ctx context.Context, u *fleet.User) error {
		// verify this isn't changed yet
		assert.Equal(t, "foo@bar.com", u.Email)
		// verify is changed per bug 1123
		assert.Equal(t, "minion", u.Position)
		return nil
	}
	svc, ctx := newTestService(t, ms, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := fleet.UserPayload{
		Email:    ptr.String("zip@zap.com"),
		Password: ptr.String(test.GoodPassword),
		Position: ptr.String("minion"),
	}
	_, err = svc.ModifyUser(ctx, 3, payload)
	require.Nil(t, err)
	assert.True(t, ms.PendingEmailChangeFuncInvoked)
	assert.True(t, ms.SaveUserFuncInvoked)
}

func TestModifyUserEmailNoPassword(t *testing.T) {
	user := &fleet.User{
		ID:    3,
		Email: "foo@bar.com",
	}
	err := user.SetPassword(test.GoodPassword, 10, 10)
	require.NoError(t, err)
	ms := new(mock.Store)
	ms.PendingEmailChangeFunc = func(ctx context.Context, id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return user, nil
	}
	ms.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		config := &fleet.AppConfig{
			SMTPSettings: &fleet.SMTPSettings{
				SMTPConfigured:         true,
				SMTPAuthenticationType: fleet.AuthTypeNameNone,
				SMTPPort:               1025,
				SMTPServer:             "127.0.0.1",
				SMTPSenderAddress:      "xxx@fleet.co",
			},
		}
		return config, nil
	}
	ms.SaveUserFunc = func(ctx context.Context, u *fleet.User) error {
		return nil
	}
	svc, ctx := newTestService(t, ms, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := fleet.UserPayload{
		Email: ptr.String("zip@zap.com"),
		// NO PASSWORD
	}
	_, err = svc.ModifyUser(ctx, 3, payload)
	require.NotNil(t, err)
	var iae *fleet.InvalidArgumentError
	ok := errors.As(err, &iae)
	require.True(t, ok)
	require.Len(t, iae.Errors, 1)
	assert.False(t, ms.PendingEmailChangeFuncInvoked)
	assert.False(t, ms.SaveUserFuncInvoked)
}

func TestModifyAdminUserEmailNoPassword(t *testing.T) {
	user := &fleet.User{
		ID:    3,
		Email: "foo@bar.com",
	}
	err := user.SetPassword(test.GoodPassword, 10, 10)
	require.NoError(t, err)
	ms := new(mock.Store)
	ms.PendingEmailChangeFunc = func(ctx context.Context, id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return user, nil
	}
	ms.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		config := &fleet.AppConfig{
			SMTPSettings: &fleet.SMTPSettings{
				SMTPConfigured:         true,
				SMTPAuthenticationType: fleet.AuthTypeNameNone,
				SMTPPort:               1025,
				SMTPServer:             "127.0.0.1",
				SMTPSenderAddress:      "xxx@fleet.co",
			},
		}
		return config, nil
	}
	ms.SaveUserFunc = func(ctx context.Context, u *fleet.User) error {
		return nil
	}
	svc, ctx := newTestService(t, ms, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := fleet.UserPayload{
		Email: ptr.String("zip@zap.com"),
		// NO PASSWORD
		// Password: &test.TestGoodPassword,
	}
	_, err = svc.ModifyUser(ctx, 3, payload)
	require.NotNil(t, err)
	var iae *fleet.InvalidArgumentError
	ok := errors.As(err, &iae)
	require.True(t, ok)
	require.Len(t, iae.Errors, 1)
	assert.False(t, ms.PendingEmailChangeFuncInvoked)
	assert.False(t, ms.SaveUserFuncInvoked)
}

func TestModifyAdminUserEmailPassword(t *testing.T) {
	user := &fleet.User{
		ID:    3,
		Email: "foo@bar.com",
	}
	err := user.SetPassword(test.GoodPassword, 10, 10)
	require.NoError(t, err)
	ms := new(mock.Store)
	ms.PendingEmailChangeFunc = func(ctx context.Context, id uint, em, tk string) error {
		return nil
	}
	ms.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		return nil, notFoundErr{}
	}
	ms.InviteByEmailFunc = func(ctx context.Context, email string) (*fleet.Invite, error) {
		return nil, notFoundErr{}
	}
	ms.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		config := &fleet.AppConfig{
			SMTPSettings: &fleet.SMTPSettings{
				SMTPConfigured:         true,
				SMTPAuthenticationType: fleet.AuthTypeNameNone,
				SMTPPort:               1025,
				SMTPServer:             "127.0.0.1",
				SMTPSenderAddress:      "xxx@fleet.co",
			},
		}
		return config, nil
	}
	ms.SaveUserFunc = func(ctx context.Context, u *fleet.User) error {
		return nil
	}
	svc, ctx := newTestService(t, ms, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := fleet.UserPayload{
		Email:    ptr.String("zip@zap.com"),
		Password: ptr.String(test.GoodPassword),
	}
	_, err = svc.ModifyUser(ctx, 3, payload)
	require.Nil(t, err)
	assert.True(t, ms.PendingEmailChangeFuncInvoked)
	assert.True(t, ms.SaveUserFuncInvoked)
}

func TestUsersWithDS(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *mysql.Datastore)
	}{
		{"CreateUserForcePasswdReset", testUsersCreateUserForcePasswdReset},
		{"ChangePassword", testUsersChangePassword},
		{"RequirePasswordReset", testUsersRequirePasswordReset},
		{"UsersCreateUserWithAPIOnly", testUsersCreateUserWithAPIOnly},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// Test that CreateUser creates a user that will be forced to
// reset its password upon first login (see #2570).
func testUsersCreateUserForcePasswdReset(t *testing.T, ds *mysql.Datastore) {
	svc, ctx := newTestService(t, ds, nil, nil)

	// Create admin user.
	admin := &fleet.User{
		Name:       "Fleet Admin",
		Email:      "admin@foo.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	err := admin.SetPassword(test.GoodPassword, 10, 10)
	require.NoError(t, err)
	admin, err = ds.NewUser(ctx, admin)
	require.NoError(t, err)

	// As the admin, create a new user.
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})
	user, sessionKey, err := svc.CreateUser(ctx, fleet.UserPayload{
		Name:       ptr.String("Some Observer"),
		Email:      ptr.String("some-observer@email.com"),
		Password:   ptr.String(test.GoodPassword),
		GlobalRole: ptr.String(fleet.RoleObserver),
	})
	require.NoError(t, err)
	require.Nil(t, sessionKey) // only set when creating API-only users

	user, err = ds.UserByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.True(t, user.AdminForcedPasswordReset)
}

func testUsersChangePassword(t *testing.T, ds *mysql.Datastore) {
	svc, ctx := newTestService(t, ds, nil, nil)
	users := createTestUsers(t, ds)
	passwordChangeTests := []struct {
		user        fleet.User
		oldPassword string
		newPassword string
		anyErr      bool
		wantErr     error
	}{
		{ // all good
			user:        users["admin1@example.com"],
			oldPassword: test.GoodPassword,
			newPassword: test.GoodPassword2,
		},
		{ // prevent password reuse
			user:        users["admin1@example.com"],
			oldPassword: test.GoodPassword2,
			newPassword: test.GoodPassword,
			wantErr:     fleet.NewInvalidArgumentError("new_password", "Cannot reuse old password"),
		},
		{ // all good
			user:        users["user1@example.com"],
			oldPassword: test.GoodPassword,
			newPassword: test.GoodPassword2,
		},
		{ // bad old password
			user:        users["user1@example.com"],
			oldPassword: "wrong_password",
			newPassword: test.GoodPassword2,
			anyErr:      true,
		},
		{ // missing old password
			user:        users["user1@example.com"],
			newPassword: test.GoodPassword2,
			wantErr:     fleet.NewInvalidArgumentError("old_password", "Old password cannot be empty"),
		},
	}

	for _, tt := range passwordChangeTests {
		t.Run("", func(t *testing.T) {
			tt := tt
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: &tt.user})

			err := svc.ChangePassword(ctx, tt.oldPassword, tt.newPassword)
			if tt.anyErr { //nolint:gocritic // ignore ifElseChain
				require.NotNil(t, err)
			} else if tt.wantErr != nil {
				require.Equal(t, tt.wantErr, ctxerr.Cause(err))
			} else {
				require.Nil(t, err)
			}

			if err != nil {
				return
			}

			// Attempt login after successful change
			_, _, err = svc.Login(context.Background(), tt.user.Email, tt.newPassword, false)
			require.Nil(t, err, "should be able to login with new password")
		})
	}
}

func testUsersRequirePasswordReset(t *testing.T, ds *mysql.Datastore) {
	svc, ctx := newTestService(t, ds, nil, nil)
	createTestUsers(t, ds)

	for _, tt := range testUsers {
		t.Run(tt.Email, func(t *testing.T) {
			user, err := ds.UserByEmail(context.Background(), tt.Email)
			require.Nil(t, err)

			var sessions []*fleet.Session

			// Log user in
			_, _, err = svc.Login(test.UserContext(ctx, test.UserAdmin), tt.Email, tt.PlaintextPassword, false)
			require.Nil(t, err, "login unsuccessful")
			sessions, err = svc.GetInfoAboutSessionsForUser(test.UserContext(ctx, test.UserAdmin), user.ID)
			require.Nil(t, err)
			require.Len(t, sessions, 1, "user should have one session")

			// Reset and verify sessions destroyed
			retUser, err := svc.RequirePasswordReset(test.UserContext(ctx, test.UserAdmin), user.ID, true)
			require.Nil(t, err)
			assert.True(t, retUser.AdminForcedPasswordReset)
			checkUser, err := ds.UserByEmail(context.Background(), tt.Email)
			require.Nil(t, err)
			assert.True(t, checkUser.AdminForcedPasswordReset)
			sessions, err = svc.GetInfoAboutSessionsForUser(test.UserContext(ctx, test.UserAdmin), user.ID)
			require.Nil(t, err)
			require.Len(t, sessions, 0, "sessions should be destroyed")

			// try undo
			retUser, err = svc.RequirePasswordReset(test.UserContext(ctx, test.UserAdmin), user.ID, false)
			require.Nil(t, err)
			assert.False(t, retUser.AdminForcedPasswordReset)
			checkUser, err = ds.UserByEmail(context.Background(), tt.Email)
			require.Nil(t, err)
			assert.False(t, checkUser.AdminForcedPasswordReset)
		})
	}
}

func TestPerformRequiredPasswordReset(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	svc, ctx := newTestService(t, ds, nil, nil)

	createTestUsers(t, ds)

	for _, tt := range testUsers {
		t.Run(tt.Email, func(t *testing.T) {
			user, err := ds.UserByEmail(context.Background(), tt.Email)
			require.Nil(t, err)

			_, err = svc.RequirePasswordReset(test.UserContext(ctx, test.UserAdmin), user.ID, true)
			require.Nil(t, err)

			ctx = refreshCtx(t, ctx, user, ds, nil)

			session, err := ds.NewSession(context.Background(), user.ID, 8)
			require.Nil(t, err)
			ctx = refreshCtx(t, ctx, user, ds, session)

			// should error when reset not required
			_, err = svc.RequirePasswordReset(ctx, user.ID, false)
			require.Nil(t, err)
			ctx = refreshCtx(t, ctx, user, ds, session)
			_, err = svc.PerformRequiredPasswordReset(ctx, test.GoodPassword2)
			require.NotNil(t, err)

			_, err = svc.RequirePasswordReset(ctx, user.ID, true)
			require.Nil(t, err)
			ctx = refreshCtx(t, ctx, user, ds, session)

			// should error when using same password
			_, err = svc.PerformRequiredPasswordReset(ctx, tt.PlaintextPassword)
			require.Equal(t, "validation failed: new_password Cannot reuse old password", err.Error())

			// should succeed with good new password
			u, err := svc.PerformRequiredPasswordReset(ctx, test.GoodPassword2)
			require.Nil(t, err)
			assert.False(t, u.AdminForcedPasswordReset)

			ctx = context.Background()

			// Now user should be able to login with new password
			u, _, err = svc.Login(ctx, tt.Email, test.GoodPassword2, false)
			require.Nil(t, err)
			assert.False(t, u.AdminForcedPasswordReset)
		})
	}
}

func TestResetPassword(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	svc, ctx := newTestService(t, ds, nil, nil)
	createTestUsers(t, ds)
	passwordResetTests := []struct {
		token       string
		newPassword string
		wantErr     error
	}{
		{ // all good
			token:       "abcd",
			newPassword: test.GoodPassword2,
		},
		{ // prevent reuse
			token:       "abcd",
			newPassword: test.GoodPassword2,
			wantErr:     fleet.NewInvalidArgumentError("new_password", "Cannot reuse old password"),
		},
		{ // bad token
			token:       "dcbaz",
			newPassword: test.GoodPassword,
			wantErr:     fleet.NewAuthFailedError("invalid password reset token"),
		},
		{ // missing token
			newPassword: test.GoodPassword,
			wantErr:     fleet.NewInvalidArgumentError("token", "Token cannot be empty field"),
		},
	}

	for _, tt := range passwordResetTests {
		t.Run("", func(t *testing.T) {
			request := &fleet.PasswordResetRequest{
				UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
					CreateTimestamp: fleet.CreateTimestamp{
						CreatedAt: time.Now(),
					},
					UpdateTimestamp: fleet.UpdateTimestamp{
						UpdatedAt: time.Now(),
					},
				},
				ExpiresAt: time.Now().Add(time.Hour * 24),
				UserID:    1,
				Token:     "abcd",
			}
			_, err := ds.NewPasswordResetRequest(context.Background(), request)
			assert.Nil(t, err)

			serr := svc.ResetPassword(test.UserContext(ctx, &fleet.User{ID: 1}), tt.token, tt.newPassword)
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr.Error(), ctxerr.Cause(serr).Error())
			} else {
				assert.Nil(t, serr)
			}
		})
	}
}

func refreshCtx(t *testing.T, ctx context.Context, user *fleet.User, ds fleet.Datastore, session *fleet.Session) context.Context {
	reloadedUser, err := ds.UserByEmail(ctx, user.Email)
	require.NoError(t, err)

	return viewer.NewContext(ctx, viewer.Viewer{User: reloadedUser, Session: session})
}

func TestAuthenticatedUser(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	createTestUsers(t, ds)
	svc, ctx := newTestService(t, ds, nil, nil)
	admin1, err := ds.UserByEmail(context.Background(), "admin1@example.com")
	require.NoError(t, err)
	admin1Session, err := ds.NewSession(context.Background(), admin1.ID, 8)
	require.NoError(t, err)

	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin1, Session: admin1Session})
	user, err := svc.AuthenticatedUser(ctx)
	assert.Nil(t, err)
	assert.Equal(t, user, admin1)
}

func TestIsAdminOfTheModifiedTeams(t *testing.T) {
	type teamWithRole struct {
		teamID uint
		role   string
	}
	type roles struct {
		global *string
		teams  []teamWithRole
	}
	for _, tc := range []struct {
		name string
		// actionUserRoles are the roles of the user executing the role change action.
		actionUserRoles roles
		// targetUserOriginalTeams are the original teams the target user belongs to.
		targetUserOriginalTeams []teamWithRole
		// targetUserNewTeams are the new teams the target user will be added to.
		targetUserNewTeams []teamWithRole

		expected bool
	}{
		{
			name: "global-admin-allmighty",
			actionUserRoles: roles{
				global: ptr.String(fleet.RoleAdmin),
			},
			targetUserOriginalTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
			},
			targetUserNewTeams: []teamWithRole{
				{
					teamID: 2,
					role:   fleet.RoleAdmin,
				},
			},
			expected: true,
		},
		{
			name: "global-maintainer-cannot-modify-team-users",
			actionUserRoles: roles{
				global: ptr.String(fleet.RoleMaintainer),
			},
			targetUserOriginalTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
			},
			targetUserNewTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleMaintainer,
				},
			},
			expected: false,
		},
		{
			name: "team-admin-of-original-and-new",
			actionUserRoles: roles{
				teams: []teamWithRole{
					{
						teamID: 1,
						role:   fleet.RoleAdmin,
					},
					{
						teamID: 2,
						role:   fleet.RoleAdmin,
					},
				},
			},
			targetUserOriginalTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
			},
			targetUserNewTeams: []teamWithRole{
				{
					teamID: 2,
					role:   fleet.RoleAdmin,
				},
			},
			expected: true,
		},
		{
			name: "team-admin-of-one-original-and-leave-other-team-unmodified",
			actionUserRoles: roles{
				teams: []teamWithRole{
					{
						teamID: 1,
						role:   fleet.RoleMaintainer,
					},
					{
						teamID: 2,
						role:   fleet.RoleAdmin,
					},
				},
			},
			targetUserOriginalTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleMaintainer,
				},
				{
					teamID: 2,
					role:   fleet.RoleMaintainer,
				},
			},
			targetUserNewTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleMaintainer,
				},
				{
					teamID: 2,
					role:   fleet.RoleAdmin,
				},
			},
			expected: true,
		},
		{
			name: "team-admin-of-original-only",
			actionUserRoles: roles{
				teams: []teamWithRole{
					{
						teamID: 1,
						role:   fleet.RoleAdmin,
					},
					{
						teamID: 2,
						role:   fleet.RoleMaintainer,
					},
				},
			},
			targetUserOriginalTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
			},
			targetUserNewTeams: []teamWithRole{
				{
					teamID: 2,
					role:   fleet.RoleAdmin,
				},
			},
			expected: false,
		},
		{
			name: "team-admin-of-new-only",
			actionUserRoles: roles{
				teams: []teamWithRole{
					{
						teamID: 1,
						role:   fleet.RoleObserver,
					},
					{
						teamID: 2,
						role:   fleet.RoleAdmin,
					},
				},
			},
			targetUserOriginalTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
			},
			targetUserNewTeams: []teamWithRole{
				{
					teamID: 2,
					role:   fleet.RoleAdmin,
				},
			},
			expected: false,
		},
		{
			name: "team-admin-but-new-another-team-observer",
			actionUserRoles: roles{
				teams: []teamWithRole{
					{
						teamID: 1,
						role:   fleet.RoleAdmin,
					},
				},
			},
			targetUserOriginalTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
			},
			targetUserNewTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
				{
					teamID: 2,
					role:   fleet.RoleObserver,
				},
			},
			expected: false,
		},
		{
			name: "team-admin-but-new-another-team-admin",
			actionUserRoles: roles{
				teams: []teamWithRole{
					{
						teamID: 1,
						role:   fleet.RoleAdmin,
					},
				},
			},
			targetUserOriginalTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
			},
			targetUserNewTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
				{
					teamID: 2,
					role:   fleet.RoleAdmin,
				},
			},
			expected: false,
		},
		{
			name: "team-admin-but-original-another-team",
			actionUserRoles: roles{
				teams: []teamWithRole{
					{
						teamID: 1,
						role:   fleet.RoleAdmin,
					},
				},
			},
			targetUserOriginalTeams: []teamWithRole{
				{
					teamID: 2,
					role:   fleet.RoleAdmin,
				},
			},
			targetUserNewTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
			},
			expected: false,
		},
		{
			name: "team-admin-but-change-role-another-team",
			actionUserRoles: roles{
				teams: []teamWithRole{
					{
						teamID: 1,
						role:   fleet.RoleAdmin,
					},
				},
			},
			targetUserOriginalTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
				{
					teamID: 2,
					role:   fleet.RoleAdmin,
				},
			},
			targetUserNewTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
				{
					teamID: 2,
					role:   fleet.RoleMaintainer,
				},
			},
			expected: false,
		},
		{
			name: "team-admin-of-one-original-only",
			actionUserRoles: roles{
				teams: []teamWithRole{
					{
						teamID: 1,
						role:   fleet.RoleMaintainer,
					},
					{
						teamID: 2,
						role:   fleet.RoleAdmin,
					},
				},
			},
			targetUserOriginalTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleMaintainer,
				},
				{
					teamID: 2,
					role:   fleet.RoleMaintainer,
				},
			},
			targetUserNewTeams: []teamWithRole{
				{
					teamID: 1,
					role:   fleet.RoleAdmin,
				},
				{
					teamID: 2,
					role:   fleet.RoleAdmin,
				},
			},
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			userTeamsFn := func(twr []teamWithRole) []fleet.UserTeam {
				var userTeams []fleet.UserTeam
				for _, ot := range twr {
					userTeams = append(userTeams, fleet.UserTeam{
						Team: fleet.Team{ID: ot.teamID},
						Role: ot.role,
					})
				}
				return userTeams
			}

			actionUserTeams := userTeamsFn(tc.actionUserRoles.teams)
			originalUserTeams := userTeamsFn(tc.targetUserOriginalTeams)
			newUserTeams := userTeamsFn(tc.targetUserNewTeams)

			result := isAdminOfTheModifiedTeams(
				&fleet.User{
					GlobalRole: tc.actionUserRoles.global,
					Teams:      actionUserTeams,
				},
				originalUserTeams,
				newUserTeams,
			)
			require.Equal(t, tc.expected, result)
		})
	}
}

// TestAdminAddRoleOtherTeam is an explicit test to check that
// that an admin cannot add itself to another team.
func TestTeamAdminAddRoleOtherTeam(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// adminTeam2 is a team admin of team with ID=2.
	adminTeam2 := &fleet.User{
		ID: 1,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 2},
				Role: fleet.RoleAdmin,
			},
		},
	}

	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		if id != 1 {
			return nil, newNotFoundError()
		}
		return adminTeam2, nil
	}
	ds.SaveUserFunc = func(ctx context.Context, user *fleet.User) error {
		return nil
	}

	ctx = viewer.NewContext(ctx, viewer.Viewer{User: adminTeam2})
	require.NoError(t, adminTeam2.SetPassword("p4ssw0rd.1337", 10, 10))

	// adminTeam2 tries to add itself to team with ID=3 as admin.
	_, err := svc.ModifyUser(ctx, adminTeam2.ID, fleet.UserPayload{
		Teams: &[]fleet.UserTeam{
			{
				Team: fleet.Team{ID: 2},
				Role: fleet.RoleAdmin,
			},
			{
				Team: fleet.Team{ID: 3},
				Role: fleet.RoleAdmin,
			},
		},
	})
	require.Equal(t, (&authz.Forbidden{}).Error(), err.Error())
	require.False(t, ds.SaveUserFuncInvoked)
}

func testUsersCreateUserWithAPIOnly(t *testing.T, ds *mysql.Datastore) {
	svc, ctx := newTestService(t, ds, nil, nil)

	host, err := ds.NewHost(ctx, &fleet.Host{
		UUID:          "uuid-42",
		OsqueryHostID: ptr.String("osquery_host_id-42"),
	})
	require.NoError(t, err)

	// Create admin user.
	admin := &fleet.User{
		Name:       "Fleet Admin",
		Email:      "admin@foo.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	err = admin.SetPassword(test.GoodPassword, 10, 10)
	require.NoError(t, err)
	admin, err = ds.NewUser(ctx, admin)
	require.NoError(t, err)

	// As the admin, create a new API-only user.
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})
	apiOnlyUser, sessionKey, err := svc.CreateUser(ctx, fleet.UserPayload{
		Name:       ptr.String("Some Observer"),
		Email:      ptr.String("some-observer@email.com"),
		Password:   ptr.String(test.GoodPassword),
		GlobalRole: ptr.String(fleet.RoleObserver),
		APIOnly:    ptr.Bool(true),
	})
	require.NoError(t, err)
	require.NotNil(t, sessionKey)
	require.NotEmpty(t, *sessionKey)

	sessions, err := svc.GetInfoAboutSessionsForUser(ctx, apiOnlyUser.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	session := sessions[0]
	require.Equal(t, *sessionKey, session.Key)

	refreshCtx(t, ctx, apiOnlyUser, ds, session)

	hosts, err := svc.ListHosts(ctx, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	require.Equal(t, host.ID, hosts[0].ID)
}
