package sso

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionStore(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		store := NewSessionStore(pool)

		sessionID, err := generateSessionID()
		require.NoError(t, err)

		// Create session that lives for 1 second.
		err = store.create(sessionID, "requestID123", "https://originalurl.com", "some metadata", 1, SSORequestData{HostUUID: "host-uuid-123"})
		require.NoError(t, err)

		sess, err := store.get(sessionID)
		require.NoError(t, err)
		require.NotNil(t, sess)
		assert.Equal(t, "requestID123", sess.RequestID)
		assert.Equal(t, "https://originalurl.com", sess.OriginalURL)
		assert.Equal(t, "some metadata", sess.Metadata)
		assert.Equal(t, "host-uuid-123", sess.RequestData.HostUUID)

		// Wait a little bit more than one second, session should no longer be present.
		time.Sleep(1100 * time.Millisecond)
		sess, err = store.get(sessionID)
		var authRequiredError *fleet.AuthRequiredError
		assert.ErrorAs(t, err, &authRequiredError)
		// The SSO callbacks tell an expired session apart from other failures
		// with this, so that they can explain the timeout to the end user.
		require.ErrorIs(t, err, ErrSessionNotFound)
		assert.Nil(t, sess)

		// Create another session for 1 second
		sessionID2, err := generateSessionID()
		require.NoError(t, err)
		err = store.create(sessionID2, "requestID456", "https://originalurl.com", "some metadata", 1, SSORequestData{})
		require.NoError(t, err)

		// Forcefully expire it
		err = store.expire(sessionID2)
		require.NoError(t, err)

		// It is not present anymore
		sess, err = store.get(sessionID2)
		assert.ErrorAs(t, err, &authRequiredError)
		assert.Nil(t, sess)

		// Expire a session that does not exist is fine
		sessionID3, err := generateSessionID()
		require.NoError(t, err)
		err = store.expire(sessionID3)
		require.NoError(t, err)
	}

	t.Run("standalone", func(t *testing.T) {
		p := redistest.SetupRedis(t, sessionKeyPrefix, false, false, false)
		runTest(t, p)
	})

	t.Run("cluster", func(t *testing.T) {
		p := redistest.SetupRedis(t, sessionKeyPrefix, true, false, false)
		runTest(t, p)
	})
}

// The store only resolves canonical server-generated session IDs, and only
// to keys inside its own namespace.
func TestSessionStoreKeyNamespace(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		store := NewSessionStore(pool)

		// A complete JSON object followed by unrelated bytes.
		sessionJSON := `{"request_id":"id123","metadata":"some metadata","original_url":"/"}` + "\x00trailing bytes"

		// Values at keys outside the session namespace, including one that
		// holds well-formed session JSON. One connection per key: in cluster
		// mode a connection binds to the node of the first key it touches.
		for _, key := range []string{"sso:unrelated:{1}", "sso:someotherkey"} {
			conn := pool.Get()
			_, err := conn.Do("SET", key, sessionJSON, "EX", 60)
			conn.Close()
			require.NoError(t, err)
		}

		var authRequiredError *fleet.AuthRequiredError
		for _, sessionID := range []string{
			"sso:unrelated:{1}",
			"sso:someotherkey",
			"",
			"short",
			"sessionID123",
			// 32 characters but not canonical base64.
			"................................",
			// Valid base64 of the wrong length.
			"c2hvcnQ=",
			// URL-safe alphabet, right length.
			"abcdefghijklmnopqrstuvwxyz-_ABCD",
			// Whitespace is ignored by base64 decoding, but any whitespace
			// leaves fewer than 32 significant characters, so these can
			// never decode to the required 24 bytes. The round-trip
			// re-encode in sessionKey is a backstop on top of that.
			"MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY\n",
			"MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3\n",
		} {
			sess, err := store.get(sessionID)
			require.ErrorAs(t, err, &authRequiredError, "session ID %q", sessionID)
			assert.Nil(t, sess, "session ID %q", sessionID)

			sess, err = store.Fullfill(sessionID)
			require.Error(t, err, "session ID %q", sessionID)
			assert.Nil(t, sess, "session ID %q", sessionID)

			require.NoError(t, store.expire(sessionID))
			require.Error(t, store.create(sessionID, "requestID", "/", "metadata", 60, SSORequestData{}))
		}

		// The values outside the namespace are untouched.
		for _, key := range []string{"sso:unrelated:{1}", "sso:someotherkey"} {
			conn := pool.Get()
			val, err := redigo.String(conn.Do("GET", key))
			conn.Close()
			require.NoError(t, err)
			// Byte equality on purpose (not JSONEq): the value must be
			// exactly as written.
			require.Equal(t, sessionJSON, val) //nolint:testifylint // not JSON equivalence
		}

		// A valid session ID round-trips through Fullfill exactly once.
		sessionID, err := generateSessionID()
		require.NoError(t, err)
		require.NoError(t, store.create(sessionID, "requestID789", "/dashboard", "some metadata", 60, SSORequestData{}))
		sess, err := store.Fullfill(sessionID)
		require.NoError(t, err)
		require.Equal(t, "requestID789", sess.RequestID)
		_, err = store.Fullfill(sessionID)
		require.Error(t, err)
	}

	t.Run("standalone", func(t *testing.T) {
		// "sso:" and not sessionKeyPrefix: this test creates keys adjacent to
		// the session namespace that also need cleanup.
		p := redistest.SetupRedis(t, "sso:", false, false, false)
		runTest(t, p)
	})

	t.Run("cluster", func(t *testing.T) {
		p := redistest.SetupRedis(t, "sso:", true, false, false)
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
