package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionGetters(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	user, err := ds.NewUser(context.Background(), &fleet.User{
		Password:   []byte("supersecret"),
		Email:      "other@bobcom",
		GlobalRole: ptr.String(fleet.RoleObserver),
	})
	require.NoError(t, err)

	session, err := ds.NewSession(&fleet.Session{UserID: user.ID, Key: "somekey"})
	require.NoError(t, err)
	require.NotZero(t, session.ID)

	gotByID, err := ds.SessionByID(session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.Key, gotByID.Key)

	gotByKey, err := ds.SessionByKey(session.Key)
	require.NoError(t, err)
	assert.Equal(t, session.ID, gotByKey.ID)

	newSession, err := ds.NewSession(&fleet.Session{UserID: user.ID, Key: "somekey2"})
	require.NoError(t, err)

	sessions, err := ds.ListSessionsForUser(user.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	require.NoError(t, ds.DestroySession(session))

	prevAccessedAt := newSession.AccessedAt

	require.NoError(t, ds.MarkSessionAccessed(newSession))

	sessions, err = ds.ListSessionsForUser(user.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.NotEqual(t, prevAccessedAt, sessions[0].AccessedAt)

	require.NoError(t, ds.DestroyAllSessionsForUser(user.ID))
}
