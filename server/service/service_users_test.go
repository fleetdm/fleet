package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticatedUser(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	createTestUsers(t, ds)
	svc := newTestService(ds, nil, nil)
	admin1, err := ds.UserByEmail("admin1@example.com")
	assert.Nil(t, err)
	admin1Session, err := ds.NewSession(&fleet.Session{
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
	ms.PendingEmailChangeFunc = func(id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(id uint) (*fleet.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func() (*fleet.AppConfig, error) {
		config := &fleet.AppConfig{
			SMTPPort:               1025,
			SMTPConfigured:         true,
			SMTPServer:             "127.0.0.1",
			SMTPSenderAddress:      "xxx@fleet.co",
			SMTPAuthenticationType: fleet.AuthTypeNone,
		}
		return config, nil
	}
	ms.SaveUserFunc = func(u *fleet.User) error {
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
	ms.PendingEmailChangeFunc = func(id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(id uint) (*fleet.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func() (*fleet.AppConfig, error) {
		config := &fleet.AppConfig{
			SMTPPort:               1025,
			SMTPConfigured:         true,
			SMTPServer:             "127.0.0.1",
			SMTPSenderAddress:      "xxx@fleet.co",
			SMTPAuthenticationType: fleet.AuthTypeNone,
		}
		return config, nil
	}
	ms.SaveUserFunc = func(u *fleet.User) error {
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
	invalid, ok := err.(*fleet.InvalidArgumentError)
	require.True(t, ok)
	require.Len(t, *invalid, 1)
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
	ms.PendingEmailChangeFunc = func(id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(id uint) (*fleet.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func() (*fleet.AppConfig, error) {
		config := &fleet.AppConfig{
			SMTPPort:               1025,
			SMTPConfigured:         true,
			SMTPServer:             "127.0.0.1",
			SMTPSenderAddress:      "xxx@fleet.co",
			SMTPAuthenticationType: fleet.AuthTypeNone,
		}
		return config, nil
	}
	ms.SaveUserFunc = func(u *fleet.User) error {
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
	invalid, ok := err.(*fleet.InvalidArgumentError)
	require.True(t, ok)
	require.Len(t, *invalid, 1)
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
	ms.PendingEmailChangeFunc = func(id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(id uint) (*fleet.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func() (*fleet.AppConfig, error) {
		config := &fleet.AppConfig{
			SMTPPort:               1025,
			SMTPConfigured:         true,
			SMTPServer:             "127.0.0.1",
			SMTPSenderAddress:      "xxx@fleet.co",
			SMTPAuthenticationType: fleet.AuthTypeNone,
		}
		return config, nil
	}
	ms.SaveUserFunc = func(u *fleet.User) error {
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

// func TestRequestPasswordReset(t *testing.T) {
// 	ds, err := inmem.New(config.TestConfig())
// 	require.Nil(t, err)
// 	createTestAppConfig(t, ds)

// 	createTestUsers(t, ds)
// 	admin1, err := ds.User("admin1")
// 	assert.Nil(t, err)
// 	user1, err := ds.User("user1")
// 	assert.Nil(t, err)
// 	var defaultEmailFn = func(e fleet.Email) error {
// 		return nil
// 	}
// 	var errEmailFn = func(e fleet.Email) error {
// 		return errors.New("test err")
// 	}
// 	authz, err := authz.NewAuthorizer()
// 	require.NoError(t, err)
// 	svc := service{
// 		ds:     ds,
// 		config: config.TestConfig(),
// 		authz:  authz,
// 	}

// 	var requestPasswordResetTests = []struct {
// 		email   string
// 		emailFn func(e fleet.Email) error
// 		wantErr error
// 		user    *fleet.User
// 		vc      *viewer.Viewer
// 	}{
// 		{
// 			email:   admin1.Email,
// 			emailFn: defaultEmailFn,
// 			user:    admin1,
// 			vc:      &viewer.Viewer{User: admin1},
// 		},
// 		{
// 			email:   admin1.Email,
// 			emailFn: defaultEmailFn,
// 			user:    admin1,
// 			vc:      nil,
// 		},
// 		{
// 			email:   user1.Email,
// 			emailFn: defaultEmailFn,
// 			user:    user1,
// 			vc:      &viewer.Viewer{User: admin1},
// 		},
// 		{
// 			email:   admin1.Email,
// 			emailFn: errEmailFn,
// 			user:    user1,
// 			vc:      nil,
// 			wantErr: errors.New("test err"),
// 		},
// 	}

// 	for _, tt := range requestPasswordResetTests {
// 		t.Run("", func(t *testing.T) {
// 			ctx := context.Background()
// 			if tt.vc != nil {
// 				ctx = viewer.NewContext(ctx, *tt.vc)
// 			}
// 			mailer := &mockMailService{SendEmailFn: tt.emailFn}
// 			svc.mailService = mailer
// 			serviceErr := svc.RequestPasswordReset(ctx, tt.email)
// 			assert.Equal(t, tt.wantErr, serviceErr)
// 			assert.True(t, mailer.Invoked, "email should be sent if vc is not admin")
// 			if serviceErr == nil {
// 				req, err := ds.FindPassswordResetsByUserID(tt.user.ID)
// 				assert.Nil(t, err)
// 				assert.NotEmpty(t, req, "user should have at least one password reset request")
// 			}
// 		})
// 	}
// }

// func TestCreateUserFromInvite(t *testing.T) {
// 	ds, _ := inmem.New(config.TestConfig())
// 	svc, _ := newTestService(ds, nil, nil)
// 	invites := setupInvites(t, ds, []string{"admin2@example.com", "admin3@example.com"})
// 	ctx := context.Background()

// 	var newUserTests = []struct {
// 		Username           *string
// 		Password           *string
// 		Email              *string
// 		NeedsPasswordReset *bool
// 		InviteToken        *string
// 		wantErr            error
// 	}{
// 		{
// 			Username:    ptr.String("admin2"),
// 			Password:    ptr.String("foobarbaz1234!"),
// 			InviteToken: &invites["admin2@example.com"].Token,
// 			wantErr:     &invalidArgumentError{invalidArgument{name: "email", reason: "missing required argument"}},
// 		},
// 		{
// 			Username: ptr.String("admin2"),
// 			Password: ptr.String("foobarbaz1234!"),
// 			Email:    ptr.String("admin2@example.com"),
// 			wantErr:  &invalidArgumentError{invalidArgument{name: "invite_token", reason: "missing required argument"}},
// 		},
// 		{
// 			Username:           ptr.String("admin2"),
// 			Password:           ptr.String("foobarbaz1234!"),
// 			Email:              ptr.String("admin2@example.com"),
// 			NeedsPasswordReset: ptr.Bool(true),
// 			InviteToken:        &invites["admin2@example.com"].Token,
// 		},
// 		{ // should return ErrNotFound because the invite is deleted
// 			// after a user signs up
// 			Username:           ptr.String("admin2"),
// 			Password:           ptr.String("foobarbaz1234!"),
// 			Email:              ptr.String("admin2@example.com"),
// 			NeedsPasswordReset: ptr.Bool(true),
// 			InviteToken:        &invites["admin2@example.com"].Token,
// 			wantErr:            errors.New("Invite with token admin2@example.com was not found in the datastore"),
// 		},
// 		{
// 			Username:           ptr.String("admin3"),
// 			Password:           ptr.String("foobarbaz1234!"),
// 			Email:              &invites["expired"].Email,
// 			NeedsPasswordReset: ptr.Bool(true),
// 			InviteToken:        &invites["expired"].Token,
// 			wantErr:            &invalidArgumentError{{name: "invite_token", reason: "Invite token has expired."}},
// 		},
// 		{
// 			Username:           ptr.String("admin3@example.com"),
// 			Password:           ptr.String("foobarbaz1234!"),
// 			Email:              ptr.String("admin3@example.com"),
// 			NeedsPasswordReset: ptr.Bool(true),
// 			InviteToken:        &invites["admin3@example.com"].Token,
// 		},
// 	}

// 	for _, tt := range newUserTests {
// 		t.Run("", func(t *testing.T) {
// 			payload := fleet.UserPayload{
// 				Username:    tt.Username,
// 				Password:    tt.Password,
// 				Email:       tt.Email,
// 				InviteToken: tt.InviteToken,
// 			}
// 			user, err := svc.CreateUserFromInvite(ctx, payload)
// 			if tt.wantErr != nil {
// 				require.Error(t, err)
// 				assert.Equal(t, tt.wantErr.Error(), err.Error())
// 				return
// 			}
// 			require.NoError(t, err)
// 			assert.NotZero(t, user.ID)

// 			err = user.ValidatePassword(*tt.Password)
// 			assert.Nil(t, err)

// 			err = user.ValidatePassword("different_password")
// 			assert.NotNil(t, err)
// 		})

// 	}
// }

func setupInvites(t *testing.T, ds fleet.Datastore, emails []string) map[string]*fleet.Invite {
	invites := make(map[string]*fleet.Invite)
	users := createTestUsers(t, ds)
	mockClock := clock.NewMockClock()
	for _, e := range emails {
		invite, err := ds.NewInvite(&fleet.Invite{
			InvitedBy: users["admin1"].ID,
			Token:     e,
			Email:     e,
			UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
				CreateTimestamp: fleet.CreateTimestamp{
					CreatedAt: mockClock.Now(),
				},
			},
		})
		require.Nil(t, err)
		invites[e] = invite
	}
	// add an expired invitation
	invite, err := ds.NewInvite(&fleet.Invite{
		InvitedBy: users["admin1"].ID,
		Token:     "expired",
		Email:     "expiredinvite@gmail.com",
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{
				CreatedAt: mockClock.Now().AddDate(-1, 0, 0),
			},
		},
	})
	require.Nil(t, err)
	invites["expired"] = invite
	return invites
}

func TestChangePassword(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	svc := newTestService(ds, nil, nil)
	users := createTestUsers(t, ds)
	var passwordChangeTests = []struct {
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
				require.Equal(t, tt.wantErr, errors.Cause(err))
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
	defer ds.Close()

	svc := newTestService(ds, nil, nil)
	createTestUsers(t, ds)
	var passwordResetTests = []struct {
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
			_, err := ds.NewPasswordResetRequest(request)
			assert.Nil(t, err)

			serr := svc.ResetPassword(test.UserContext(&fleet.User{ID: 1}), tt.token, tt.newPassword)
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr.Error(), errors.Cause(serr).Error())
			} else {
				assert.Nil(t, serr)
			}
		})
	}
}

func TestRequirePasswordReset(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	svc := newTestService(ds, nil, nil)
	createTestUsers(t, ds)

	for _, tt := range testUsers {
		t.Run(tt.Email, func(t *testing.T) {
			user, err := ds.UserByEmail(tt.Email)
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
			checkUser, err := ds.UserByEmail(tt.Email)
			require.Nil(t, err)
			assert.True(t, checkUser.AdminForcedPasswordReset)
			sessions, err = svc.GetInfoAboutSessionsForUser(test.UserContext(test.UserAdmin), user.ID)
			require.Nil(t, err)
			require.Len(t, sessions, 0, "sessions should be destroyed")

			// try undo
			retUser, err = svc.RequirePasswordReset(test.UserContext(test.UserAdmin), user.ID, false)
			require.Nil(t, err)
			assert.False(t, retUser.AdminForcedPasswordReset)
			checkUser, err = ds.UserByEmail(tt.Email)
			require.Nil(t, err)
			assert.False(t, checkUser.AdminForcedPasswordReset)

		})
	}
}

func refreshCtx(t *testing.T, ctx context.Context, user *fleet.User, ds fleet.Datastore, session *fleet.Session) context.Context {
	reloadedUser, err := ds.UserByEmail(user.Email)
	require.NoError(t, err)

	return viewer.NewContext(ctx, viewer.Viewer{User: reloadedUser, Session: session})
}

func TestPerformRequiredPasswordReset(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	svc := newTestService(ds, nil, nil)

	createTestUsers(t, ds)

	for _, tt := range testUsers {
		t.Run(tt.Email, func(t *testing.T) {
			user, err := ds.UserByEmail(tt.Email)
			require.Nil(t, err)

			ctx := context.Background()

			_, err = svc.RequirePasswordReset(test.UserContext(test.UserAdmin), user.ID, true)
			require.Nil(t, err)

			ctx = refreshCtx(t, ctx, user, ds, nil)

			session, err := ds.NewSession(&fleet.Session{UserID: user.ID})
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
	var passwordTests = []struct {
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
