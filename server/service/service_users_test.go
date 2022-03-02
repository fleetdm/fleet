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
