package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessions(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Getters", testSessionsGetters},
		{"MFA", testMFA},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testMFA(t *testing.T, ds *Datastore) {
	user, err := ds.NewUser(context.Background(), &fleet.User{
		Password:   []byte("supersecret"),
		Email:      "me@example.com",
		GlobalRole: ptr.String(fleet.RoleObserver),
	})
	require.NoError(t, err)

	token, err := ds.NewMFAToken(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// invalid token
	session, mfaUser, err := ds.SessionByMFAToken(context.Background(), "notreal", 8)
	require.Error(t, err)
	require.Nil(t, mfaUser)
	require.Nil(t, session)

	// valid token
	session, mfaUser, err = ds.SessionByMFAToken(context.Background(), token, 6)
	require.NoError(t, err)
	require.Equal(t, user.ID, session.UserID)
	require.Equal(t, user.ID, mfaUser.ID)
	require.Len(t, session.Key, 8) // 48 base64-encoded bits

	// used token
	session, mfaUser, err = ds.SessionByMFAToken(context.Background(), token, 8)
	require.Error(t, err)
	require.Nil(t, mfaUser)
	require.Nil(t, session)

	// expired token
	token, err = ds.NewMFAToken(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(
			context.Background(),
			"UPDATE verification_tokens SET created_at = NOW() - INTERVAL ? SECOND - INTERVAL 0.5 SECOND",
			fleet.MFALinkTTL.Seconds(),
		)
		return err
	})
	session, mfaUser, err = ds.SessionByMFAToken(context.Background(), token, 8)
	require.Error(t, err)
	require.Nil(t, mfaUser)
	require.Nil(t, session)
}

func testSessionsGetters(t *testing.T, ds *Datastore) {
	user, err := ds.NewUser(context.Background(), &fleet.User{
		Password:   []byte("supersecret"),
		Email:      "other@bobcom",
		GlobalRole: ptr.String(fleet.RoleObserver),
	})
	require.NoError(t, err)

	session, err := ds.NewSession(context.Background(), user.ID, 8)
	require.NoError(t, err)
	require.NotZero(t, session.ID)

	gotByID, err := ds.SessionByID(context.Background(), session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.Key, gotByID.Key)
	require.NotNil(t, gotByID.APIOnly)
	assert.False(t, *gotByID.APIOnly)

	gotByKey, err := ds.SessionByKey(context.Background(), session.Key)
	require.NoError(t, err)
	assert.Equal(t, session.ID, gotByKey.ID)
	require.NotNil(t, gotByKey.APIOnly)
	assert.False(t, *gotByKey.APIOnly)

	newSession, err := ds.NewSession(context.Background(), user.ID, 8)
	require.NoError(t, err)

	sessions, err := ds.ListSessionsForUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	require.NoError(t, ds.DestroySession(context.Background(), session))

	prevAccessedAt := newSession.AccessedAt

	// Advance ds's mock clock time (used by MarkSessionAccessed).
	mc := ds.clock.(*clock.MockClock)
	mc.AddTime(1 * time.Second)

	require.NoError(t, ds.MarkSessionAccessed(context.Background(), newSession))

	sessions, err = ds.ListSessionsForUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.NotEqual(t, prevAccessedAt, sessions[0].AccessedAt)

	require.NoError(t, ds.DestroyAllSessionsForUser(context.Background(), user.ID))

	// session for a non-existing user
	newSession, err = ds.NewSession(context.Background(), user.ID+1, 8)
	require.NoError(t, err)

	gotByKey, err = ds.SessionByKey(context.Background(), newSession.Key)
	require.NoError(t, err)
	assert.Equal(t, newSession.ID, gotByKey.ID)
	require.Nil(t, gotByKey.APIOnly)

	_, err = ds.SessionByID(context.Background(), newSession.ID)
	require.NoError(t, err)
	assert.Equal(t, newSession.ID, gotByKey.ID)
	require.Nil(t, gotByKey.APIOnly)

	apiUser, err := ds.NewUser(context.Background(), &fleet.User{
		Password:   []byte("supersecret"),
		GlobalRole: ptr.String(fleet.RoleObserver),
		APIOnly:    true,
	})
	require.NoError(t, err)

	// session for an api user
	apiSession, err := ds.NewSession(context.Background(), apiUser.ID, 8)
	require.NoError(t, err)

	gotByKey, err = ds.SessionByKey(context.Background(), apiSession.Key)
	require.NoError(t, err)
	assert.Equal(t, apiSession.ID, gotByKey.ID)
	require.NotNil(t, gotByKey.APIOnly)
	assert.True(t, *gotByKey.APIOnly)

	_, err = ds.SessionByID(context.Background(), apiSession.ID)
	require.NoError(t, err)
	assert.Equal(t, apiSession.ID, gotByKey.ID)
	require.NotNil(t, gotByKey.APIOnly)
	assert.True(t, *gotByKey.APIOnly)
}
