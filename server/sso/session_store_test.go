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
		err := store.create("sessionID123", "requestID123", "https://originalurl.com", "some metadata", 1)
		require.NoError(t, err)

		sess, err := store.get("sessionID123")
		require.NoError(t, err)
		require.NotNil(t, sess)
		assert.Equal(t, "requestID123", sess.RequestID)
		assert.Equal(t, "https://originalurl.com", sess.OriginalURL)
		assert.Equal(t, "some metadata", sess.Metadata)

		// Wait a little bit more than one second, session should no longer be present.
		time.Sleep(1100 * time.Millisecond)
		sess, err = store.get("sessionID123")
		var authRequiredError *fleet.AuthRequiredError
		assert.ErrorAs(t, err, &authRequiredError)
		assert.Nil(t, sess)

		// Create another session for 1 second
		err = store.create("sessionID456", "requestID456", "https://originalurl.com", "some metadata", 1)
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
