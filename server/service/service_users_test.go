package service

import (
	"errors"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/kolide/kolide-ose/server/config"
	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"github.com/kolide/kolide-ose/server/datastore/inmem"
	kolide_errors "github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestAuthenticatedUser(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)
	createTestUsers(t, ds)
	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)
	admin1, err := ds.User("admin1")
	assert.Nil(t, err)

	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin1})
	user, err := svc.AuthenticatedUser(ctx)
	assert.Nil(t, err)
	assert.Equal(t, user, admin1)
}

func TestRequestPasswordReset(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)
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
			t.Parallel()
			tt := tt
			ctx := context.Background()
			if tt.vc != nil {
				ctx = viewer.NewContext(ctx, *tt.vc)
			}
			mailer := &mockMailService{SendEmailFn: tt.emailFn}
			svc.mailService = mailer
			serviceErr := svc.RequestPasswordReset(ctx, tt.email)
			assert.Equal(t, tt.wantErr, serviceErr)
			if tt.vc != nil && tt.vc.IsAdmin() {
				assert.False(t, mailer.Invoked, "email should not be sent if reset requested by admin")
				assert.True(t, tt.user.AdminForcedPasswordReset, "AdminForcedPasswordReset should be true if reset requested by admin")
			} else {
				assert.True(t, mailer.Invoked, "email should be sent if vc is not admin")
				if serviceErr == nil {
					req, err := ds.FindPassswordResetsByUserID(tt.user.ID)
					assert.Nil(t, err)
					assert.NotEmpty(t, req, "user should have at least one password reset request")
				}
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
			Password:    stringPtr("foobar"),
			InviteToken: &invites["admin2@example.com"].Token,
			wantErr:     &invalidArgumentError{invalidArgument{name: "email", reason: "missing required argument"}},
		},
		{
			Username: stringPtr("admin2"),
			Password: stringPtr("foobar"),
			Email:    stringPtr("admin2@example.com"),
			wantErr:  &invalidArgumentError{invalidArgument{name: "invite_token", reason: "missing required argument"}},
		},
		{
			Username:           stringPtr("admin2"),
			Password:           stringPtr("foobar"),
			Email:              stringPtr("admin2@example.com"),
			NeedsPasswordReset: boolPtr(true),
			Admin:              boolPtr(false),
			InviteToken:        &invites["admin2@example.com"].Token,
		},
		{ // should return ErrNotFound because the invite is deleted
			// after a user signs up
			Username:           stringPtr("admin2"),
			Password:           stringPtr("foobar"),
			Email:              stringPtr("admin2@example.com"),
			NeedsPasswordReset: boolPtr(true),
			Admin:              boolPtr(false),
			InviteToken:        &invites["admin2@example.com"].Token,
			wantErr:            kolide_errors.ErrNotFound,
		},
		{
			Username:           stringPtr("admin3"),
			Password:           stringPtr("foobar"),
			Email:              &invites["expired"].Email,
			NeedsPasswordReset: boolPtr(true),
			Admin:              boolPtr(false),
			InviteToken:        &invites["expired"].Token,
			wantErr:            errors.New("expired invite token"),
		},
		{
			Username:           stringPtr("@admin2"),
			Password:           stringPtr("foobar"),
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
				Username:                 tt.Username,
				Password:                 tt.Password,
				Email:                    tt.Email,
				Admin:                    tt.Admin,
				InviteToken:              tt.InviteToken,
				AdminForcedPasswordReset: tt.NeedsPasswordReset,
			}
			user, err := svc.NewUser(ctx, payload)
			require.Equal(t, tt.wantErr, err)
			if err != nil {
				// skip rest of the test if error is not nil
				return
			}

			assert.NotZero(t, user.ID)

			err = user.ValidatePassword(*tt.Password)
			assert.Nil(t, err)

			err = user.ValidatePassword("different_password")
			assert.NotNil(t, err)

			assert.Equal(t, user.AdminForcedPasswordReset, *tt.NeedsPasswordReset)
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

func TestChangeUserPassword(t *testing.T) {
	ds, _ := inmem.New(config.TestConfig())
	svc, _ := newTestService(ds, nil)
	createTestUsers(t, ds)
	var passwordChangeTests = []struct {
		token       string
		newPassword string
		wantErr     error
	}{
		{ // all good
			token:       "abcd",
			newPassword: "123cat!",
		},
		{ // bad token
			token:       "dcbaz",
			newPassword: "123cat!",
			wantErr:     kolide_errors.ErrNotFound,
		},
		{ // missing token
			newPassword: "123cat!",
			wantErr:     &invalidArgumentError{invalidArgument{name: "token", reason: "cannot be empty field"}},
		},
		{ // missing password
			token:   "abcd",
			wantErr: &invalidArgumentError{invalidArgument{name: "new_password", reason: "cannot be empty field"}},
		},
	}

	for _, tt := range passwordChangeTests {
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
			assert.Equal(t, tt.wantErr, serr)
		})
	}
}
