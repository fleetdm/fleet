package sso

import (
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/mna/redisc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPool(t *testing.T) *redisc.Cluster {
	if _, ok := os.LookupEnv("REDIS_TEST"); ok {
		var (
			addr     = "127.0.0.1:6379"
			password = ""
			database = 0
			useTLS   = false
		)

		pool, err := pubsub.NewRedisPool(addr, password, database, useTLS)
		require.NoError(t, err)
		_, err = pool.Get().Do("PING")
		require.Nil(t, err)
		return pool
	}
	return nil
}

func TestSessionStore(t *testing.T) {
	if _, ok := os.LookupEnv("REDIS_TEST"); !ok {
		t.Skip("skipping sso session store tests")
	}
	p := newPool(t)
	require.NotNil(t, p)
	defer p.Close()
	store := NewSessionStore(p)
	require.NotNil(t, store)
	// Create session that lives for 1 second.
	err := store.create("request123", "https://originalurl.com", "some metadata", 1)
	require.Nil(t, err)
	sess, err := store.Get("request123")
	require.Nil(t, err)
	require.NotNil(t, sess)
	assert.Equal(t, "https://originalurl.com", sess.OriginalURL)
	assert.Equal(t, "some metadata", sess.Metadata)
	// Wait a little bit more than one second, session should no longer be present.
	time.Sleep(1100 * time.Millisecond)
	sess, err = store.Get("request123")
	assert.Equal(t, ErrSessionNotFound, err)
	assert.Nil(t, sess)
}
