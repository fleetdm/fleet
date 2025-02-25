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
		{"CleanupExpiredPasswordResetRequests", testCleanupExpiredPasswordResetRequests},
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
		require.NoError(t, err)
		assert.Equal(t, tt.userID, req.UserID)

		found, err := ds.FindPasswordResetByToken(context.Background(), r.Token)
		require.NoError(t, err)
		assert.Equal(t, req.ID, found.ID)
		assert.Equal(t, tt.userID, found.UserID)
		assert.Equal(t, tt.token, found.Token)
		assert.WithinDuration(t, tt.expires, found.ExpiresAt, 1*time.Minute)
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
		res, err := ds.writer(ctx).ExecContext(ctx, stmt, req.UserID, req.Token, req.ExpiresAt)
		require.NoError(t, err)

		id, _ := res.LastInsertId()
		req.ID = uint(id) //nolint:gosec // dismiss G115

		found, err := ds.FindPasswordResetByToken(context.Background(), req.Token)

		if tt.shouldFail {
			assert.ErrorIs(t, err, sql.ErrNoRows)
			assert.Nil(t, found)
		} else {
			require.NoError(t, err)
			assert.Equal(t, req.ID, found.ID)
			assert.Equal(t, req.UserID, found.UserID)
			assert.Equal(t, req.Token, found.Token)
			assert.WithinDuration(t, req.ExpiresAt.Truncate(time.Minute), found.ExpiresAt.Truncate(time.Minute), time.Minute)
		}
	}
}

func testCleanupExpiredPasswordResetRequests(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	stmt := `INSERT INTO password_reset_requests ( user_id, token, expires_at)
			VALUES (?,?, DATE_ADD(CURRENT_TIMESTAMP, INTERVAL ? HOUR))`

	_, err := ds.writer(ctx).ExecContext(ctx, stmt, uint(1), "now", 0)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, stmt, uint(1), "tomorrow", 24)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, stmt, uint(1), "yesterday", -24)
	require.NoError(t, err)

	var res1 []fleet.PasswordResetRequest
	err = ds.writer(ctx).SelectContext(ctx, &res1, `SELECT * FROM password_reset_requests`)
	require.NoError(t, err)
	require.Len(t, res1, 3)

	err = ds.CleanupExpiredPasswordResetRequests(ctx)
	require.NoError(t, err)

	var res2 []fleet.PasswordResetRequest
	err = ds.writer(ctx).SelectContext(ctx, &res2, `SELECT * FROM password_reset_requests`)
	require.NoError(t, err)
	require.Len(t, res2, 1)
	require.Equal(t, "tomorrow", res2[0].Token)
}
