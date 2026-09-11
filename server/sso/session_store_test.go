package sso

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionStore(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		store := NewSessionStore(pool)

		// Create session that lives for 1 second.
		err := store.create("sessionID123", "requestID123", "https://originalurl.com", "some metadata", 1, SSORequestData{HostUUID: "host-uuid-123"})
		require.NoError(t, err)

		sess, err := store.get("sessionID123")
		require.NoError(t, err)
		require.NotNil(t, sess)
		assert.Equal(t, "requestID123", sess.RequestID)
		assert.Equal(t, "https://originalurl.com", sess.OriginalURL)
		assert.Equal(t, "some metadata", sess.Metadata)
		assert.Equal(t, "host-uuid-123", sess.RequestData.HostUUID)

		// Wait a little bit more than one second, session should no longer be present.
		time.Sleep(1100 * time.Millisecond)
		sess, err = store.get("sessionID123")
		var authRequiredError *fleet.AuthRequiredError
		assert.ErrorAs(t, err, &authRequiredError)
		// The SSO callbacks tell an expired session apart from other failures
		// with this, so that they can explain the timeout to the end user.
		require.ErrorIs(t, err, ErrSessionNotFound)
		assert.Nil(t, sess)

		// Create another session for 1 second
		err = store.create("sessionID456", "requestID456", "https://originalurl.com", "some metadata", 1, SSORequestData{})
		require.NoError(t, err)

		// Forcefully expire it
		err = store.expire("sessionID456")
		require.NoError(t, err)

		// It is not present anymore
		sess, err = store.get("sessionID456")
		assert.ErrorAs(t, err, &authRequiredError)
		assert.Nil(t, sess)

		// Expire a session that does not exist is fine
		err = store.expire("sessionIDNOSUCH")
		require.NoError(t, err)
	}

	t.Run("standalone", func(t *testing.T) {
		p := redistest.SetupRedis(t, "request", false, false, false)
		runTest(t, p)
	})

	t.Run("cluster", func(t *testing.T) {
		p := redistest.SetupRedis(t, "request", true, false, false)
		runTest(t, p)
	})
}

// A missing session has to answer to two different callers: the authz
// middleware matches on the AuthRequiredError type, and the SSO callbacks match
// on ErrSessionNotFound so they can tell the end user their sign-in timed out.
func TestSessionNotFoundErrorSatisfiesBothCallers(t *testing.T) {
	authRequired := fleet.NewAuthRequiredError("session not found")
	err := &sessionNotFoundError{authRequired: authRequired}

	var asAuthRequired *fleet.AuthRequiredError
	require.ErrorAs(t, err, &asAuthRequired)
	require.ErrorIs(t, err, ErrSessionNotFound)

	// Callers that surface the message must not see it change.
	require.Equal(t, authRequired.Error(), err.Error())
}
