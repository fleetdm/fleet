package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
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
