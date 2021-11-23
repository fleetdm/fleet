package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

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

func TestAuthenticatedUser(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	createTestUsers(t, ds)
	svc := newTestService(ds, nil, nil)
	admin1, err := ds.UserByEmail(context.Background(), "admin1@example.com")
	assert.Nil(t, err)
	admin1Session, err := ds.NewSession(context.Background(), &fleet.Session{
		UserID: admin1.ID,
		Key:    "admin1",
	})
	assert.Nil(t, err)

	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin1, Session: admin1Session})
	user, err := svc.AuthenticatedUser(ctx)
	assert.Nil(t, err)
	assert.Equal(t, user, admin1)
}

func TestModifyUserEmail(t *testing.T) {
	user := &fleet.User{
		ID:    3,
		Email: "foo@bar.com",
	}
	user.SetPassword("password", 10, 10)
	ms := new(mock.Store)
	ms.PendingEmailChangeFunc = func(ctx context.Context, id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		config := &fleet.AppConfig{
			SMTPSettings: fleet.SMTPSettings{
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
	svc := newTestService(ms, nil, nil)
	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := fleet.UserPayload{
		Email:    ptr.String("zip@zap.com"),
		Password: ptr.String("password"),
		Position: ptr.String("minion"),
	}
	_, err := svc.ModifyUser(ctx, 3, payload)
	require.Nil(t, err)
	assert.True(t, ms.PendingEmailChangeFuncInvoked)
	assert.True(t, ms.SaveUserFuncInvoked)
}

func TestModifyUserEmailNoPassword(t *testing.T) {
	user := &fleet.User{
		ID:    3,
		Email: "foo@bar.com",
	}
	user.SetPassword("password", 10, 10)
	ms := new(mock.Store)
	ms.PendingEmailChangeFunc = func(ctx context.Context, id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		config := &fleet.AppConfig{
			SMTPSettings: fleet.SMTPSettings{
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
	svc := newTestService(ms, nil, nil)
	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := fleet.UserPayload{
		Email: ptr.String("zip@zap.com"),
		// NO PASSWORD
		//	Password: ptr.String("password"),
	}
	_, err := svc.ModifyUser(ctx, 3, payload)
	require.NotNil(t, err)
	var iae *fleet.InvalidArgumentError
	ok := errors.As(err, &iae)
	require.True(t, ok)
	require.Len(t, *iae, 1)
	assert.False(t, ms.PendingEmailChangeFuncInvoked)
	assert.False(t, ms.SaveUserFuncInvoked)
}

func TestModifyAdminUserEmailNoPassword(t *testing.T) {
	user := &fleet.User{
		ID:    3,
		Email: "foo@bar.com",
	}
	user.SetPassword("password", 10, 10)
	ms := new(mock.Store)
	ms.PendingEmailChangeFunc = func(ctx context.Context, id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		config := &fleet.AppConfig{
			SMTPSettings: fleet.SMTPSettings{
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
	svc := newTestService(ms, nil, nil)
	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := fleet.UserPayload{
		Email: ptr.String("zip@zap.com"),
		// NO PASSWORD
		//	Password: ptr.String("password"),
	}
	_, err := svc.ModifyUser(ctx, 3, payload)
	require.NotNil(t, err)
	var iae *fleet.InvalidArgumentError
	ok := errors.As(err, &iae)
	require.True(t, ok)
	require.Len(t, *iae, 1)
	assert.False(t, ms.PendingEmailChangeFuncInvoked)
	assert.False(t, ms.SaveUserFuncInvoked)
}

func TestModifyAdminUserEmailPassword(t *testing.T) {
	user := &fleet.User{
		ID:    3,
		Email: "foo@bar.com",
	}
	user.SetPassword("password", 10, 10)
	ms := new(mock.Store)
	ms.PendingEmailChangeFunc = func(ctx context.Context, id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		config := &fleet.AppConfig{
			SMTPSettings: fleet.SMTPSettings{
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
	svc := newTestService(ms, nil, nil)
	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := fleet.UserPayload{
		Email:    ptr.String("zip@zap.com"),
		Password: ptr.String("password"),
	}
	_, err := svc.ModifyUser(ctx, 3, payload)
	require.Nil(t, err)
	assert.True(t, ms.PendingEmailChangeFuncInvoked)
	assert.True(t, ms.SaveUserFuncInvoked)
}

func TestChangePassword(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	svc := newTestService(ds, nil, nil)
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
			oldPassword: "foobarbaz1234!",
			newPassword: "12345cat!",
		},
		{ // prevent password reuse
			user:        users["admin1@example.com"],
			oldPassword: "12345cat!",
			newPassword: "foobarbaz1234!",
			wantErr:     fleet.NewInvalidArgumentError("new_password", "cannot reuse old password"),
		},
		{ // all good
			user:        users["user1@example.com"],
			oldPassword: "foobarbaz1234!",
			newPassword: "newpassa1234!",
		},
		{ // bad old password
			user:        users["user1@example.com"],
			oldPassword: "wrong_password",
			newPassword: "12345cat!",
			anyErr:      true,
		},
		{ // missing old password
			newPassword: "123cataaa!",
			wantErr:     fleet.NewInvalidArgumentError("old_password", "Old password cannot be empty"),
		},
	}

	for _, tt := range passwordChangeTests {
		t.Run("", func(t *testing.T) {
			ctx := context.Background()
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: &tt.user})

			err := svc.ChangePassword(ctx, tt.oldPassword, tt.newPassword)
			if tt.anyErr {
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
			_, _, err = svc.Login(context.Background(), tt.user.Email, tt.newPassword)
			require.Nil(t, err, "should be able to login with new password")
		})
	}
}

func TestResetPassword(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	svc := newTestService(ds, nil, nil)
	createTestUsers(t, ds)
	passwordResetTests := []struct {
		token       string
		newPassword string
		wantErr     error
	}{
		{ // all good
			token:       "abcd",
			newPassword: "123cat!",
		},
		{ // prevent reuse
			token:       "abcd",
			newPassword: "123cat!",
			wantErr:     fleet.NewInvalidArgumentError("new_password", "cannot reuse old password"),
		},
		{ // bad token
			token:       "dcbaz",
			newPassword: "123cat!",
			wantErr:     sql.ErrNoRows,
		},
		{ // missing token
			newPassword: "123cat!",
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

			serr := svc.ResetPassword(test.UserContext(&fleet.User{ID: 1}), tt.token, tt.newPassword)
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr.Error(), ctxerr.Cause(serr).Error())
			} else {
				assert.Nil(t, serr)
			}
		})
	}
}

func TestRequirePasswordReset(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	svc := newTestService(ds, nil, nil)
	createTestUsers(t, ds)

	for _, tt := range testUsers {
		t.Run(tt.Email, func(t *testing.T) {
			user, err := ds.UserByEmail(context.Background(), tt.Email)
			require.Nil(t, err)

			var sessions []*fleet.Session

			// Log user in
			_, _, err = svc.Login(test.UserContext(test.UserAdmin), tt.Email, tt.PlaintextPassword)
			require.Nil(t, err, "login unsuccessful")
			sessions, err = svc.GetInfoAboutSessionsForUser(test.UserContext(test.UserAdmin), user.ID)
			require.Nil(t, err)
			require.Len(t, sessions, 1, "user should have one session")

			// Reset and verify sessions destroyed
			retUser, err := svc.RequirePasswordReset(test.UserContext(test.UserAdmin), user.ID, true)
			require.Nil(t, err)
			assert.True(t, retUser.AdminForcedPasswordReset)
			checkUser, err := ds.UserByEmail(context.Background(), tt.Email)
			require.Nil(t, err)
			assert.True(t, checkUser.AdminForcedPasswordReset)
			sessions, err = svc.GetInfoAboutSessionsForUser(test.UserContext(test.UserAdmin), user.ID)
			require.Nil(t, err)
			require.Len(t, sessions, 0, "sessions should be destroyed")

			// try undo
			retUser, err = svc.RequirePasswordReset(test.UserContext(test.UserAdmin), user.ID, false)
			require.Nil(t, err)
			assert.False(t, retUser.AdminForcedPasswordReset)
			checkUser, err = ds.UserByEmail(context.Background(), tt.Email)
			require.Nil(t, err)
			assert.False(t, checkUser.AdminForcedPasswordReset)
		})
	}
}

func refreshCtx(t *testing.T, ctx context.Context, user *fleet.User, ds fleet.Datastore, session *fleet.Session) context.Context {
	reloadedUser, err := ds.UserByEmail(ctx, user.Email)
	require.NoError(t, err)

	return viewer.NewContext(ctx, viewer.Viewer{User: reloadedUser, Session: session})
}

func TestPerformRequiredPasswordReset(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	svc := newTestService(ds, nil, nil)

	createTestUsers(t, ds)

	for _, tt := range testUsers {
		t.Run(tt.Email, func(t *testing.T) {
			user, err := ds.UserByEmail(context.Background(), tt.Email)
			require.Nil(t, err)

			ctx := context.Background()

			_, err = svc.RequirePasswordReset(test.UserContext(test.UserAdmin), user.ID, true)
			require.Nil(t, err)

			ctx = refreshCtx(t, ctx, user, ds, nil)

			session, err := ds.NewSession(context.Background(), &fleet.Session{UserID: user.ID})
			require.Nil(t, err)
			ctx = refreshCtx(t, ctx, user, ds, session)

			// should error when reset not required
			_, err = svc.RequirePasswordReset(ctx, user.ID, false)
			require.Nil(t, err)
			ctx = refreshCtx(t, ctx, user, ds, session)
			_, err = svc.PerformRequiredPasswordReset(ctx, "new_pass")
			require.NotNil(t, err)

			_, err = svc.RequirePasswordReset(ctx, user.ID, true)
			require.Nil(t, err)
			ctx = refreshCtx(t, ctx, user, ds, session)

			// should error when using same password
			_, err = svc.PerformRequiredPasswordReset(ctx, tt.PlaintextPassword)
			require.Equal(t, "validation failed: new_password cannot reuse old password", err.Error())

			// should succeed with good new password
			u, err := svc.PerformRequiredPasswordReset(ctx, "new_pass")
			require.Nil(t, err)
			assert.False(t, u.AdminForcedPasswordReset)

			ctx = context.Background()

			// Now user should be able to login with new password
			u, _, err = svc.Login(ctx, tt.Email, "new_pass")
			require.Nil(t, err)
			assert.False(t, u.AdminForcedPasswordReset)
		})
	}
}

func TestUserPasswordRequirements(t *testing.T) {
	passwordTests := []struct {
		password string
		wantErr  bool
	}{
		{
			password: "foobar",
			wantErr:  true,
		},
		{
			password: "foobarbaz",
			wantErr:  true,
		},
		{
			password: "foobarbaz!",
			wantErr:  true,
		},
		{
			password: "foobarbaz!3",
		},
	}

	for _, tt := range passwordTests {
		t.Run(tt.password, func(t *testing.T) {
			err := validatePasswordRequirements(tt.password)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestUserAuth(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

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
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		if id == 999 {
			return &fleet.User{
				ID:    999,
				Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}},
			}, nil
		}
		return &fleet.User{
			ID:         888,
			GlobalRole: ptr.String(fleet.RoleMaintainer),
		}, nil
	}
	ds.SaveUserFunc = func(ctx context.Context, user *fleet.User) error {
		return nil
	}

	testCases := []struct {
		name                  string
		user                  *fleet.User
		shouldFailGlobalWrite bool
		shouldFailTeamWrite   bool
		shouldFailRead        bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
			true,
			true,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
			true,
		},
		{
			"team admin, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			false,
			false,
		},
		{
			"team maintainer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			true,
			false,
		},
		{
			"team observer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
			true,
		},
		{
			"team maintainer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			true,
			true,
			true,
		},
		{
			"team admin, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			true,
			true,
			true,
		},
		{
			"team observer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			true,
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			teams := []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}
			_, err := svc.CreateUser(ctx, fleet.UserPayload{
				Name:     ptr.String("Some Name"),
				Email:    ptr.String("some@email.com"),
				Password: ptr.String("passw0rd."),
				Teams:    &teams,
			})
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			_, err = svc.CreateUser(ctx, fleet.UserPayload{
				Name:       ptr.String("Some Name"),
				Email:      ptr.String("some@email.com"),
				Password:   ptr.String("passw0rd."),
				GlobalRole: ptr.String(fleet.RoleAdmin),
			})
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			_, err = svc.ModifyUser(ctx, 999, fleet.UserPayload{Teams: &teams})
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			_, err = svc.ModifyUser(ctx, 888, fleet.UserPayload{Teams: &teams})
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			_, err = svc.ModifyUser(ctx, 888, fleet.UserPayload{GlobalRole: ptr.String(fleet.RoleMaintainer)})
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)
		})
	}
}

// Test that CreateUser creates a user that will be forced to
// reset its password upon first login (see #2570).
func TestCreateUserForcePasswdReset(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	svc := newTestService(ds, nil, nil)

	// Create admin user.
	admin := &fleet.User{
		Name:       "Fleet Admin",
		Email:      "admin@foo.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	err := admin.SetPassword("p4ssw0rd.", 10, 10)
	require.NoError(t, err)
	admin, err = ds.NewUser(context.Background(), admin)
	require.NoError(t, err)

	// As the admin, create a new user.
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: admin})
	user, err := svc.CreateUser(ctx, fleet.UserPayload{
		Name:       ptr.String("Some Observer"),
		Email:      ptr.String("some-observer@email.com"),
		Password:   ptr.String("passw0rd."),
		GlobalRole: ptr.String(fleet.RoleObserver),
	})
	require.NoError(t, err)

	user, err = ds.UserByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.True(t, user.AdminForcedPasswordReset)
}
