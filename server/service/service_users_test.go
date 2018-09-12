package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kolide/fleet/server/config"
	"github.com/kolide/fleet/server/contexts/viewer"
	"github.com/kolide/fleet/server/datastore/inmem"
	"github.com/kolide/fleet/server/kolide"

	"github.com/WatchBeam/clock"
	"github.com/kolide/fleet/server/mock"
	pkg_errors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticatedUser(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)
	createTestUsers(t, ds)
	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)
	admin1, err := ds.User("admin1")
	assert.Nil(t, err)
	admin1Session, err := ds.NewSession(&kolide.Session{
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
	user := &kolide.User{
		ID:      3,
		Admin:   false,
		Email:   "foo@bar.com",
		Enabled: true,
	}
	user.SetPassword("password", 10, 10)
	ms := new(mock.Store)
	ms.PendingEmailChangeFunc = func(id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(id uint) (*kolide.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func() (*kolide.AppConfig, error) {
		config := &kolide.AppConfig{
			SMTPPort:               1025,
			SMTPConfigured:         true,
			SMTPServer:             "127.0.0.1",
			SMTPSenderAddress:      "xxx@kolide.co",
			SMTPAuthenticationType: kolide.AuthTypeNone,
		}
		return config, nil
	}
	ms.SaveUserFunc = func(u *kolide.User) error {
		// verify this isn't changed yet
		assert.Equal(t, "foo@bar.com", u.Email)
		// verify is changed per bug 1123
		assert.Equal(t, "minion", u.Position)
		return nil
	}
	svc, err := newTestService(ms, nil)
	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := kolide.UserPayload{
		Email:    stringPtr("zip@zap.com"),
		Password: stringPtr("password"),
		Position: stringPtr("minion"),
	}
	_, err = svc.ModifyUser(ctx, 3, payload)
	require.Nil(t, err)
	assert.True(t, ms.PendingEmailChangeFuncInvoked)
	assert.True(t, ms.SaveUserFuncInvoked)

}

func TestModifyUserCannotUpdateAdminEnabled(t *testing.T) {
	// The modify user function should not be able to update the admin or
	// enabled status of a user. These should only be updated explicitly
	// through the ChangeUserAdmin and ChangeUserEnabled functions.
	user := &kolide.User{
		ID:      3,
		Admin:   false,
		Email:   "foo@bar.com",
		Enabled: true,
	}
	user.SetPassword("password", 10, 10)
	ms := new(mock.Store)
	ms.UserByIDFunc = func(id uint) (*kolide.User, error) {
		return user, nil
	}
	ms.SaveUserFunc = func(u *kolide.User) error {
		assert.Equal(t, false, u.Admin, "should not be able to update admin status!")
		assert.Equal(t, true, u.Enabled, "should not be able to update enabled status!")
		return nil
	}
	svc, err := newTestService(ms, nil)
	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := kolide.UserPayload{
		Admin:   boolPtr(true),
		Enabled: boolPtr(false),
	}
	_, err = svc.ModifyUser(ctx, 3, payload)
	require.Nil(t, err)
	assert.True(t, ms.SaveUserFuncInvoked)

}

func TestModifyUserEmailNoPassword(t *testing.T) {
	user := &kolide.User{
		ID:      3,
		Admin:   true,
		Email:   "foo@bar.com",
		Enabled: true,
	}
	user.SetPassword("password", 10, 10)
	ms := new(mock.Store)
	ms.PendingEmailChangeFunc = func(id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(id uint) (*kolide.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func() (*kolide.AppConfig, error) {
		config := &kolide.AppConfig{
			SMTPPort:               1025,
			SMTPConfigured:         true,
			SMTPServer:             "127.0.0.1",
			SMTPSenderAddress:      "xxx@kolide.co",
			SMTPAuthenticationType: kolide.AuthTypeNone,
		}
		return config, nil
	}
	ms.SaveUserFunc = func(u *kolide.User) error {
		return nil
	}
	svc, err := newTestService(ms, nil)
	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := kolide.UserPayload{
		Email: stringPtr("zip@zap.com"),
		// NO PASSWORD
		//	Password: stringPtr("password"),
	}
	_, err = svc.ModifyUser(ctx, 3, payload)
	require.NotNil(t, err)
	invalid, ok := err.(*invalidArgumentError)
	require.True(t, ok)
	require.Len(t, *invalid, 1)
	assert.Equal(t, "cannot be empty if email is changed", (*invalid)[0].reason)
	assert.False(t, ms.PendingEmailChangeFuncInvoked)
	assert.False(t, ms.SaveUserFuncInvoked)

}

func TestModifyAdminUserEmailNoPassword(t *testing.T) {
	user := &kolide.User{
		ID:      3,
		Admin:   true,
		Email:   "foo@bar.com",
		Enabled: true,
	}
	user.SetPassword("password", 10, 10)
	ms := new(mock.Store)
	ms.PendingEmailChangeFunc = func(id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(id uint) (*kolide.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func() (*kolide.AppConfig, error) {
		config := &kolide.AppConfig{
			SMTPPort:               1025,
			SMTPConfigured:         true,
			SMTPServer:             "127.0.0.1",
			SMTPSenderAddress:      "xxx@kolide.co",
			SMTPAuthenticationType: kolide.AuthTypeNone,
		}
		return config, nil
	}
	ms.SaveUserFunc = func(u *kolide.User) error {
		return nil
	}
	svc, err := newTestService(ms, nil)
	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := kolide.UserPayload{
		Email: stringPtr("zip@zap.com"),
		// NO PASSWORD
		//	Password: stringPtr("password"),
	}
	_, err = svc.ModifyUser(ctx, 3, payload)
	require.NotNil(t, err)
	invalid, ok := err.(*invalidArgumentError)
	require.True(t, ok)
	require.Len(t, *invalid, 1)
	assert.Equal(t, "cannot be empty if email is changed", (*invalid)[0].reason)
	assert.False(t, ms.PendingEmailChangeFuncInvoked)
	assert.False(t, ms.SaveUserFuncInvoked)

}

func TestModifyAdminUserEmailPassword(t *testing.T) {
	user := &kolide.User{
		ID:      3,
		Admin:   true,
		Email:   "foo@bar.com",
		Enabled: true,
	}
	user.SetPassword("password", 10, 10)
	ms := new(mock.Store)
	ms.PendingEmailChangeFunc = func(id uint, em, tk string) error {
		return nil
	}
	ms.UserByIDFunc = func(id uint) (*kolide.User, error) {
		return user, nil
	}
	ms.AppConfigFunc = func() (*kolide.AppConfig, error) {
		config := &kolide.AppConfig{
			SMTPPort:               1025,
			SMTPConfigured:         true,
			SMTPServer:             "127.0.0.1",
			SMTPSenderAddress:      "xxx@kolide.co",
			SMTPAuthenticationType: kolide.AuthTypeNone,
		}
		return config, nil
	}
	ms.SaveUserFunc = func(u *kolide.User) error {
		return nil
	}
	svc, err := newTestService(ms, nil)
	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	payload := kolide.UserPayload{
		Email:    stringPtr("zip@zap.com"),
		Password: stringPtr("password"),
	}
	_, err = svc.ModifyUser(ctx, 3, payload)
	require.Nil(t, err)
	assert.True(t, ms.PendingEmailChangeFuncInvoked)
	assert.True(t, ms.SaveUserFuncInvoked)

}

func TestRequestPasswordReset(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)
	createTestAppConfig(t, ds)

	createTestUsers(t, ds)
	admin1, err := ds.User("admin1")
	assert.Nil(t, err)
	user1, err := ds.User("user1")
	assert.Nil(t, err)
	var defaultEmailFn = func(e kolide.Email) error {
		return nil
	}
	var errEmailFn = func(e kolide.Email) error {
		return errors.New("test err")
	}
	svc := service{
		ds:     ds,
		config: config.TestConfig(),
	}

	var requestPasswordResetTests = []struct {
		email   string
		emailFn func(e kolide.Email) error
		wantErr error
		user    *kolide.User
		vc      *viewer.Viewer
	}{
		{
			email:   admin1.Email,
			emailFn: defaultEmailFn,
			user:    admin1,
			vc:      &viewer.Viewer{User: admin1},
		},
		{
			email:   admin1.Email,
			emailFn: defaultEmailFn,
			user:    admin1,
			vc:      nil,
		},
		{
			email:   user1.Email,
			emailFn: defaultEmailFn,
			user:    user1,
			vc:      &viewer.Viewer{User: admin1},
		},
		{
			email:   admin1.Email,
			emailFn: errEmailFn,
			user:    user1,
			vc:      nil,
			wantErr: errors.New("test err"),
		},
	}

	for _, tt := range requestPasswordResetTests {
		t.Run("", func(t *testing.T) {
			ctx := context.Background()
			if tt.vc != nil {
				ctx = viewer.NewContext(ctx, *tt.vc)
			}
			mailer := &mockMailService{SendEmailFn: tt.emailFn}
			svc.mailService = mailer
			serviceErr := svc.RequestPasswordReset(ctx, tt.email)
			assert.Equal(t, tt.wantErr, serviceErr)
			assert.True(t, mailer.Invoked, "email should be sent if vc is not admin")
			if serviceErr == nil {
				req, err := ds.FindPassswordResetsByUserID(tt.user.ID)
				assert.Nil(t, err)
				assert.NotEmpty(t, req, "user should have at least one password reset request")
			}
		})
	}
}

func TestCreateUser(t *testing.T) {
	ds, _ := inmem.New(config.TestConfig())
	svc, _ := newTestService(ds, nil)
	invites := setupInvites(t, ds, []string{"admin2@example.com"})
	ctx := context.Background()

	var createUserTests = []struct {
		Username           *string
		Password           *string
		Email              *string
		NeedsPasswordReset *bool
		Admin              *bool
		InviteToken        *string
		wantErr            error
	}{
		{
			Username:    stringPtr("admin2"),
			Password:    stringPtr("foobarbaz1234!"),
			InviteToken: &invites["admin2@example.com"].Token,
			wantErr:     &invalidArgumentError{invalidArgument{name: "email", reason: "missing required argument"}},
		},
		{
			Username: stringPtr("admin2"),
			Password: stringPtr("foobarbaz1234!"),
			Email:    stringPtr("admin2@example.com"),
			wantErr:  &invalidArgumentError{invalidArgument{name: "invite_token", reason: "missing required argument"}},
		},
		{
			Username:           stringPtr("admin2"),
			Password:           stringPtr("foobarbaz1234!"),
			Email:              stringPtr("admin2@example.com"),
			NeedsPasswordReset: boolPtr(true),
			Admin:              boolPtr(false),
			InviteToken:        &invites["admin2@example.com"].Token,
		},
		{ // should return ErrNotFound because the invite is deleted
			// after a user signs up
			Username:           stringPtr("admin2"),
			Password:           stringPtr("foobarbaz1234!"),
			Email:              stringPtr("admin2@example.com"),
			NeedsPasswordReset: boolPtr(true),
			Admin:              boolPtr(false),
			InviteToken:        &invites["admin2@example.com"].Token,
			wantErr:            errors.New("Invite with token admin2@example.com was not found in the datastore"),
		},
		{
			Username:           stringPtr("admin3"),
			Password:           stringPtr("foobarbaz1234!"),
			Email:              &invites["expired"].Email,
			NeedsPasswordReset: boolPtr(true),
			Admin:              boolPtr(false),
			InviteToken:        &invites["expired"].Token,
			wantErr:            &invalidArgumentError{{name: "invite_token", reason: "Invite token has expired."}},
		},
		{
			Username:           stringPtr("@admin2"),
			Password:           stringPtr("foobarbaz1234!"),
			Email:              stringPtr("admin2@example.com"),
			NeedsPasswordReset: boolPtr(true),
			Admin:              boolPtr(false),
			InviteToken:        &invites["admin2@example.com"].Token,
			wantErr:            &invalidArgumentError{invalidArgument{name: "username", reason: "'@' character not allowed in usernames"}},
		},
	}

	for _, tt := range createUserTests {
		t.Run("", func(t *testing.T) {
			payload := kolide.UserPayload{
				Username:    tt.Username,
				Password:    tt.Password,
				Email:       tt.Email,
				Admin:       tt.Admin,
				InviteToken: tt.InviteToken,
			}
			user, err := svc.NewUser(ctx, payload)
			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr.Error(), err.Error())
			}
			if err != nil {
				// skip rest of the test if error is not nil
				return
			}

			assert.NotZero(t, user.ID)

			err = user.ValidatePassword(*tt.Password)
			assert.Nil(t, err)

			err = user.ValidatePassword("different_password")
			assert.NotNil(t, err)

			assert.Equal(t, user.Admin, *tt.Admin)
		})

	}
}

func setupInvites(t *testing.T, ds kolide.Datastore, emails []string) map[string]*kolide.Invite {
	invites := make(map[string]*kolide.Invite)
	users := createTestUsers(t, ds)
	mockClock := clock.NewMockClock()
	for _, e := range emails {
		invite, err := ds.NewInvite(&kolide.Invite{
			InvitedBy: users["admin1"].ID,
			Token:     e,
			Email:     e,
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: mockClock.Now(),
				},
			},
		})
		require.Nil(t, err)
		invites[e] = invite
	}
	// add an expired invitation
	invite, err := ds.NewInvite(&kolide.Invite{
		InvitedBy: users["admin1"].ID,
		Token:     "expired",
		Email:     "expiredinvite@gmail.com",
		UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
			CreateTimestamp: kolide.CreateTimestamp{
				CreatedAt: mockClock.Now().AddDate(-1, 0, 0),
			},
		},
	})
	require.Nil(t, err)
	invites["expired"] = invite
	return invites
}

func TestChangePassword(t *testing.T) {
	ds, _ := inmem.New(config.TestConfig())
	svc, _ := newTestService(ds, nil)
	users := createTestUsers(t, ds)
	var passwordChangeTests = []struct {
		user        kolide.User
		oldPassword string
		newPassword string
		anyErr      bool
		wantErr     error
	}{
		{ // all good
			user:        users["admin1"],
			oldPassword: "foobarbaz1234!",
			newPassword: "12345cat!",
		},
		{ // prevent password reuse
			user:        users["admin1"],
			oldPassword: "12345cat!",
			newPassword: "foobarbaz1234!",
			wantErr:     &invalidArgumentError{invalidArgument{name: "new_password", reason: "cannot reuse old password"}},
		},
		{ // all good
			user:        users["user1"],
			oldPassword: "foobarbaz1234!",
			newPassword: "newpassa1234!",
		},
		{ // bad old password
			user:        users["user1"],
			oldPassword: "wrong_password",
			newPassword: "12345cat!",
			anyErr:      true,
		},
		{ // missing old password
			newPassword: "123cataaa!",
			wantErr:     &invalidArgumentError{invalidArgument{name: "old_password", reason: "cannot be empty"}},
		},
		{ // missing new password
			oldPassword: "abcd",
			wantErr: &invalidArgumentError{
				{name: "new_password", reason: "cannot be empty"},
				{name: "new_password", reason: "password does not meet validation requirements"},
			},
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
				require.Equal(t, tt.wantErr, pkg_errors.Cause(err))
			} else {
				require.Nil(t, err)
			}

			if err != nil {
				return
			}

			// Attempt login after successful change
			_, _, err = svc.Login(context.Background(), tt.user.Username, tt.newPassword)
			require.Nil(t, err, "should be able to login with new password")
		})
	}
}

func TestResetPassword(t *testing.T) {
	ds, _ := inmem.New(config.TestConfig())
	svc, _ := newTestService(ds, nil)
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
			wantErr:     &invalidArgumentError{invalidArgument{name: "new_password", reason: "cannot reuse old password"}},
		},
		{ // bad token
			token:       "dcbaz",
			newPassword: "123cat!",
			wantErr:     errors.New("PasswordResetRequest was not found in the datastore"),
		},
		{ // missing token
			newPassword: "123cat!",
			wantErr:     &invalidArgumentError{invalidArgument{name: "token", reason: "cannot be empty field"}},
		},
		{ // missing password
			token: "abcd",
			wantErr: &invalidArgumentError{
				{name: "new_password", reason: "cannot be empty field"},
				{name: "new_password", reason: "password does not meet validation requirements"},
			},
		},
	}

	for _, tt := range passwordResetTests {
		t.Run("", func(t *testing.T) {
			ctx := context.Background()
			request := &kolide.PasswordResetRequest{
				UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
					CreateTimestamp: kolide.CreateTimestamp{
						CreatedAt: time.Now(),
					},
					UpdateTimestamp: kolide.UpdateTimestamp{
						UpdatedAt: time.Now(),
					},
				},
				ExpiresAt: time.Now().Add(time.Hour * 24),
				UserID:    1,
				Token:     "abcd",
			}
			_, err := ds.NewPasswordResetRequest(request)
			assert.Nil(t, err)

			serr := svc.ResetPassword(ctx, tt.token, tt.newPassword)
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr.Error(), pkg_errors.Cause(serr).Error())
			} else {
				assert.Nil(t, serr)
			}
		})
	}
}

func TestRequirePasswordReset(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)
	svc, err := newTestService(ds, nil)
	require.Nil(t, err)

	createTestUsers(t, ds)

	for _, tt := range testUsers {
		t.Run(tt.Username, func(t *testing.T) {
			user, err := ds.User(tt.Username)
			require.Nil(t, err)

			var sessions []*kolide.Session
			ctx := context.Background()

			// Log user in
			if tt.Enabled {
				_, _, err = svc.Login(ctx, tt.Username, tt.PlaintextPassword)
				require.Nil(t, err, "login unsuccessful")
				sessions, err = svc.GetInfoAboutSessionsForUser(ctx, user.ID)
				require.Nil(t, err)
				require.Len(t, sessions, 1, "user should have one session")
			}

			// Reset and verify sessions destroyed
			retUser, err := svc.RequirePasswordReset(ctx, user.ID, true)
			require.Nil(t, err)
			assert.True(t, retUser.AdminForcedPasswordReset)
			checkUser, err := ds.User(tt.Username)
			require.Nil(t, err)
			assert.True(t, checkUser.AdminForcedPasswordReset)
			sessions, err = svc.GetInfoAboutSessionsForUser(ctx, user.ID)
			require.Nil(t, err)
			require.Len(t, sessions, 0, "sessions should be destroyed")

			// try undo
			retUser, err = svc.RequirePasswordReset(ctx, user.ID, false)
			require.Nil(t, err)
			assert.False(t, retUser.AdminForcedPasswordReset)
			checkUser, err = ds.User(tt.Username)
			require.Nil(t, err)
			assert.False(t, checkUser.AdminForcedPasswordReset)

		})
	}
}

func TestPerformRequiredPasswordReset(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)
	svc, err := newTestService(ds, nil)
	require.Nil(t, err)

	createTestUsers(t, ds)

	for _, tt := range testUsers {
		t.Run(tt.Username, func(t *testing.T) {
			if !tt.Enabled {
				return
			}

			user, err := ds.User(tt.Username)
			require.Nil(t, err)

			ctx := context.Background()

			_, err = svc.RequirePasswordReset(ctx, user.ID, true)
			require.Nil(t, err)

			// should error when not logged in
			_, err = svc.PerformRequiredPasswordReset(ctx, "new_pass")
			require.NotNil(t, err)

			session, err := ds.NewSession(&kolide.Session{
				UserID: user.ID,
			})
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: user, Session: session})

			// should error when reset not required
			_, err = svc.RequirePasswordReset(ctx, user.ID, false)
			require.Nil(t, err)
			_, err = svc.PerformRequiredPasswordReset(ctx, "new_pass")
			require.NotNil(t, err)

			_, err = svc.RequirePasswordReset(ctx, user.ID, true)
			require.Nil(t, err)

			// should error when using same password
			_, err = svc.PerformRequiredPasswordReset(ctx, tt.PlaintextPassword)
			require.NotNil(t, err)

			// should succeed with good new password
			u, err := svc.PerformRequiredPasswordReset(ctx, "new_pass")
			require.Nil(t, err)
			assert.False(t, u.AdminForcedPasswordReset)

			ctx = context.Background()

			// Now user should be able to login with new password
			u, _, err = svc.Login(ctx, tt.Username, "new_pass")
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
