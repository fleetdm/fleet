package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
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
		{"TokenCaseSensitivity", testPasswordResetTokenCaseSensitivity},
		{"CleanupExpiredPasswordResetRequests", testCleanupExpiredPasswordResetRequests},
		{"ResetIsAtomic", testResetPasswordIsAtomic},
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
		assert.WithinDuration(t, tt.expires, found.ExpiresAt, 1*time.Minute)
	}
}

func testPasswordResetTokenExpiration(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	users := createTestUsers(t, ds)

	expiryTests := []struct {
		token      string
		userID     uint
		expiresAt  time.Time
		shouldFail bool
	}{
		{
			token:      "3XP1r3D70K3N",
			userID:     users[0].ID,
			expiresAt:  time.Now().UTC().Add(-1 * time.Hour),
			shouldFail: true,
		},
		{
			token:      "V411D70K3N",
			userID:     users[1].ID,
			expiresAt:  time.Now().UTC().Add(1 * time.Hour),
			shouldFail: false,
		},
	}

	for _, tt := range expiryTests {
		stmt := `INSERT INTO password_reset_requests ( user_id, token_hash, expires_at)
			VALUES (?, UNHEX(SHA2(?, 256)), ?)`
		res, err := ds.writer(ctx).ExecContext(ctx, stmt, tt.userID, tt.token, tt.expiresAt)
		require.NoError(t, err)

		id, _ := res.LastInsertId()

		found, err := ds.FindPasswordResetByToken(context.Background(), tt.token)

		if tt.shouldFail {
			assert.ErrorIs(t, err, sql.ErrNoRows)
			assert.Nil(t, found)
		} else {
			require.NoError(t, err)
			assert.Equal(t, uint(id), found.ID) //nolint:gosec // dismiss G115
			assert.Equal(t, tt.userID, found.UserID)
			assert.WithinDuration(t, tt.expiresAt.Truncate(time.Minute), found.ExpiresAt.Truncate(time.Minute), time.Minute)
		}
	}
}

func testResetPasswordIsAtomic(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	users := createTestUsers(t, ds)
	require.NotEmpty(t, users)
	user := users[0]

	// userWithPassword returns a fresh copy of the user with the given (hashed) password,
	// mirroring what the service passes in after hashing.
	userWithPassword := func(pw string) *fleet.User {
		u := *user
		require.NoError(t, u.SetPassword(pw, 10, 10))
		return &u
	}

	// An unknown token returns a not-found error and changes nothing.
	require.True(t, fleet.IsNotFound(ds.ResetPassword(ctx, "does-not-exist", userWithPassword("Unknown!Pass123"))))

	// An expired token returns a not-found error.
	_, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO password_reset_requests (user_id, token_hash, expires_at) VALUES (?, UNHEX(SHA2(?, 256)), ?)`,
		user.ID, "expired-token", time.Now().UTC().Add(-time.Hour))
	require.NoError(t, err)
	require.True(t, fleet.IsNotFound(ds.ResetPassword(ctx, "expired-token", userWithPassword("Expired!Pass123"))))

	// A valid token is consumed, the new password is persisted, and the user's active
	// sessions are destroyed.
	const okToken = "valid-token"
	_, err = ds.NewPasswordResetRequest(ctx, &fleet.PasswordResetRequest{UserID: user.ID, Token: okToken})
	require.NoError(t, err)
	_, err = ds.NewSession(ctx, user.ID, 32)
	require.NoError(t, err)
	require.NoError(t, ds.ResetPassword(ctx, okToken, userWithPassword("Valid!Pass1234")))

	saved, err := ds.UserByID(ctx, user.ID)
	require.NoError(t, err)
	require.NoError(t, saved.ValidatePassword("Valid!Pass1234"), "new password should be persisted")
	_, err = ds.FindPasswordResetByToken(ctx, okToken)
	require.ErrorIs(t, err, sql.ErrNoRows)
	sessions, err := ds.ListSessionsForUser(ctx, user.ID)
	require.NoError(t, err)
	require.Empty(t, sessions, "resetting the password should destroy the user's sessions")

	// A single valid token consumed concurrently must be claimed by exactly one caller;
	// every other caller must get a not-found error.
	const raceToken = "single-use-token"
	_, err = ds.NewPasswordResetRequest(ctx, &fleet.PasswordResetRequest{UserID: user.ID, Token: raceToken})
	require.NoError(t, err)

	const n = 10
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u := userWithPassword(fmt.Sprintf("Race!Pass%d1234", i))
			<-start
			errs[i] = ds.ResetPassword(ctx, raceToken, u)
		}(i)
	}
	close(start)
	wg.Wait()

	var claimed int
	for i := range errs {
		if errs[i] == nil {
			claimed++
		} else {
			require.True(t, fleet.IsNotFound(errs[i]), "losers should report not found, got: %v", errs[i])
		}
	}
	require.Equal(t, 1, claimed, "exactly one concurrent caller should consume a single-use token")

	// The token is gone after being consumed.
	_, err = ds.FindPasswordResetByToken(ctx, raceToken)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func testPasswordResetTokenCaseSensitivity(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	users := createTestUsers(t, ds)

	// Token generated by RequestPasswordReset is base64url-encoded, so its
	// alphabet is case-sensitive. Lookups must match it byte-for-byte.
	token := "AbCdEfGhIjKlMnOpQrStUvWx" //nolint:gosec // G101: test token, not a real credential
	_, err := ds.NewPasswordResetRequest(ctx, &fleet.PasswordResetRequest{
		UserID: users[0].ID,
		Token:  token,
	})
	require.NoError(t, err)

	// The exact token matches.
	found, err := ds.FindPasswordResetByToken(ctx, token)
	require.NoError(t, err)
	require.Equal(t, users[0].ID, found.UserID)

	// A case-mutated copy of the token must NOT match.
	for _, mutated := range []string{
		strings.ToLower(token),
		strings.ToUpper(token),
		"aBcDeFgHiJkLmNoPqRsTuVwX", // inverted case
	} {
		found, err := ds.FindPasswordResetByToken(ctx, mutated)
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.Nil(t, found)
	}
}

func testCleanupExpiredPasswordResetRequests(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	stmt := `INSERT INTO password_reset_requests ( user_id, token_hash, expires_at)
			VALUES (?, UNHEX(SHA2(?, 256)), DATE_ADD(CURRENT_TIMESTAMP, INTERVAL ? HOUR))`

	_, err := ds.writer(ctx).ExecContext(ctx, stmt, uint(1), "now", 0)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, stmt, uint(1), "tomorrow", 24)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, stmt, uint(1), "yesterday", -24)
	require.NoError(t, err)

	type resetRow struct {
		ID        uint      `db:"id"`
		ExpiresAt time.Time `db:"expires_at"`
		UserID    uint      `db:"user_id"`
	}

	var res1 []resetRow
	err = ds.writer(ctx).SelectContext(ctx, &res1, `SELECT id, expires_at, user_id FROM password_reset_requests`)
	require.NoError(t, err)
	require.Len(t, res1, 3)

	err = ds.CleanupExpiredPasswordResetRequests(ctx)
	require.NoError(t, err)

	var res2 []resetRow
	err = ds.writer(ctx).SelectContext(ctx, &res2, `SELECT id, expires_at, user_id FROM password_reset_requests`)
	require.NoError(t, err)
	require.Len(t, res2, 1)
}
