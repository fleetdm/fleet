package mysql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordReset(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Requests", testPasswordResetRequests},
		{"TokenExpiration", testPasswordResetTokenExpiration},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testPasswordResetRequests(t *testing.T, ds *Datastore) {
	createTestUsers(t, ds)
	now := time.Now().UTC()
	tomorrow := now.Add(time.Hour * 24)
	passwordResetTests := []struct {
		userID  uint
		expires time.Time
		token   string
	}{
		{userID: 1, expires: tomorrow, token: "abcd"},
	}

	for _, tt := range passwordResetTests {
		r := &fleet.PasswordResetRequest{
			UserID: tt.userID,
			Token:  tt.token,
		}
		req, err := ds.NewPasswordResetRequest(context.Background(), r)
		assert.Nil(t, err)
		assert.Equal(t, tt.userID, req.UserID)

		found, err := ds.FindPasswordResetByToken(context.Background(), r.Token)
		assert.Equal(t, req.ID, found.ID)
		assert.Equal(t, tt.userID, found.UserID)
		assert.Equal(t, tt.token, found.Token)
		assert.Equal(t, tt.expires.Round(time.Minute), found.ExpiresAt.Round(time.Minute))
	}
}

func testPasswordResetTokenExpiration(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	users := createTestUsers(t, ds)

	expiryTests := []struct {
		request    fleet.PasswordResetRequest
		shouldFail bool
	}{
		{
			request: fleet.PasswordResetRequest{
				UserID:    users[0].ID,
				Token:     "3XP1r3D70K3N",
				ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
			},
			shouldFail: true,
		},
		{
			request: fleet.PasswordResetRequest{
				UserID:    users[1].ID,
				Token:     "V411D70K3N",
				ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
			},
			shouldFail: false,
		},
	}

	for _, tt := range expiryTests {
		req := tt.request

		stmt := `INSERT INTO password_reset_requests ( user_id, token, expires_at)
			VALUES (?,?, ?)`
		res, err := ds.writer.ExecContext(ctx, stmt, req.UserID, req.Token, req.ExpiresAt)
		require.NoError(t, err)

		id, _ := res.LastInsertId()
		req.ID = uint(id)

		found, err := ds.FindPasswordResetByToken(context.Background(), req.Token)

		if tt.shouldFail {
			assert.ErrorIs(t, err, sql.ErrNoRows)
			assert.Nil(t, found)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, req.ID, found.ID)
			assert.Equal(t, req.UserID, found.UserID)
			assert.Equal(t, req.Token, found.Token)
			assert.Equal(t, req.ExpiresAt.Round(time.Minute), found.ExpiresAt.Round(time.Minute))
		}
	}
}
