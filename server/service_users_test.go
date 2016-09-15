package server

import (
	"errors"
	"strconv"
	"testing"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/config"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestRequestPasswordReset(t *testing.T) {
	ds, err := datastore.New("inmem", "")
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
		wantErr interface{}
		user    *kolide.User
		vc      *viewerContext
	}{
		{
			email:   admin1.Email,
			emailFn: defaultEmailFn,
			user:    admin1,
			vc:      &viewerContext{user: admin1},
		},
		{
			email:   admin1.Email,
			emailFn: defaultEmailFn,
			user:    admin1,
			vc:      emptyVC(),
		},
		{
			email:   user1.Email,
			emailFn: defaultEmailFn,
			user:    user1,
			vc:      &viewerContext{user: admin1},
		},
		{
			email:   admin1.Email,
			emailFn: errEmailFn,
			user:    user1,
			vc:      emptyVC(),
			wantErr: errors.New("test err"),
		},
	}

	for _, tt := range requestPasswordResetTests {
		tt := tt
		t.Run("", func(st *testing.T) {
			st.Parallel()
			ctx := context.Background()
			if tt.vc != nil {
				ctx = context.WithValue(ctx, "viewerContext", tt.vc)
			}
			mailer := &mockMailService{SendEmailFn: tt.emailFn}
			svc.mailService = mailer

			serviceErr := svc.RequestPasswordReset(ctx, tt.email)
			assert.Equal(t, tt.wantErr, serviceErr)
			if tt.vc.IsAdmin() {
				assert.False(st, mailer.Invoked, "email should not be sent if reset requested by admin")
				assert.True(st, tt.user.AdminForcedPasswordReset, "AdminForcedPasswordReset should be true if reset requested by admin")
			} else {
				assert.True(st, mailer.Invoked, "email should be sent if vc is not admin")
				if serviceErr == nil {
					req, err := ds.FindPassswordResetsByUserID(tt.user.ID)
					assert.Nil(st, err)
					assert.NotEmpty(st, req, "user should have at least one password reset request")
				}
			}
		})
	}
}

func TestCreateUser(t *testing.T) {
	ds, _ := datastore.New("inmem", "")
	svc, _ := NewService(ds, kitlog.NewNopLogger(), config.TestConfig(), nil)
	ctx := context.Background()

	var createUserTests = []struct {
		Username           *string
		Password           *string
		Email              *string
		NeedsPasswordReset *bool
		Admin              *bool
		Err                error
	}{
		{
			Username: stringPtr("admin1"),
			Password: stringPtr("foobar"),
			Err:      invalidArgumentError{},
		},
		{
			Username:           stringPtr("admin1"),
			Password:           stringPtr("foobar"),
			Email:              stringPtr("admin1@example.com"),
			NeedsPasswordReset: boolPtr(true),
			Admin:              boolPtr(false),
		},
	}

	for _, tt := range createUserTests {
		payload := kolide.UserPayload{
			Username: tt.Username,
			Password: tt.Password,
			Email:    tt.Email,
			Admin:    tt.Admin,
			AdminForcedPasswordReset: tt.NeedsPasswordReset,
		}
		user, err := svc.NewUser(ctx, payload)
		switch err.(type) {
		case nil:
		case invalidArgumentError:
			continue
		default:
			t.Fatalf("got %q, want %q", err, tt.Err)
		}

		assert.NotZero(t, user.ID)

		err = user.ValidatePassword(*tt.Password)
		assert.Nil(t, err)

		err = user.ValidatePassword("different_password")
		assert.NotNil(t, err)

		assert.Equal(t, user.AdminForcedPasswordReset, *tt.NeedsPasswordReset)
		assert.Equal(t, user.Admin, *tt.Admin)

		// check duplicate creation
		_, err = svc.NewUser(ctx, payload)
		assert.Equal(t, datastore.ErrExists, err)
	}
}

func TestChangeUserPassword(t *testing.T) {
	ds, _ := datastore.New("inmem", "")
	svc, _ := NewService(ds, kitlog.NewNopLogger(), config.TestConfig(), nil)
	createTestUsers(t, ds)
	var passwordChangeTests = []struct {
		token       string
		newPassword string
		wantErr     interface{}
	}{
		{ // all good
			token:       "abcd",
			newPassword: "123cat!",
		},
		{ // bad token
			token:       "dcbaz",
			newPassword: "123cat!",
			wantErr:     datastore.ErrNotFound,
		},
		{ // missing token
			newPassword: "123cat!",
			wantErr:     invalidArgumentError{field: "token", required: true},
		},
		{ // missing password
			token:   "abcd",
			wantErr: invalidArgumentError{field: "password", required: true},
		},
	}

	for i, tt := range passwordChangeTests {
		ctx := context.Background()
		request := &kolide.PasswordResetRequest{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour * 24),
			UserID:    1,
			Token:     "abcd",
		}
		_, err := ds.NewPasswordResetRequest(request)
		assert.Nil(t, err)

		serr := svc.ResetPassword(ctx, tt.token, tt.newPassword)
		assert.Equal(t, tt.wantErr, serr, strconv.Itoa(i))
	}
}

type mockMailService struct {
	SendEmailFn func(e kolide.Email) error
	Invoked     bool
}

func (svc *mockMailService) SendEmail(e kolide.Email) error {
	svc.Invoked = true
	return svc.SendEmailFn(e)
}

var testUsers = map[string]kolide.UserPayload{
	"admin1": {
		Username: stringPtr("admin1"),
		Password: stringPtr("foobar"),
		Email:    stringPtr("admin1@example.com"),
		Admin:    boolPtr(true),
	},
	"user1": {
		Username: stringPtr("user1"),
		Password: stringPtr("foobar"),
		Email:    stringPtr("user1@example.com"),
	},
	"user2": {
		Username: stringPtr("user2"),
		Password: stringPtr("bazfoo"),
		Email:    stringPtr("user2@example.com"),
	},
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
