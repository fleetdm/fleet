package sso

import (
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPool(t *testing.T, cluster bool) fleet.RedisPool {
	if _, ok := os.LookupEnv("REDIS_TEST"); ok {
		var (
			addr     = "127.0.0.1:"
			password = ""
			database = 0
			useTLS   = false
			port     = "6379"
		)
		if cluster {
			port = "7001"
		}
		addr += port

		pool, err := redis.NewRedisPool(redis.PoolConfig{
			Server:      addr,
			Password:    password,
			Database:    database,
			UseTLS:      useTLS,
			ConnTimeout: 5 * time.Second,
			KeepAlive:   10 * time.Second,
		})
		require.NoError(t, err)
		conn := pool.Get()
		defer conn.Close()
		_, err = conn.Do("PING")
		require.Nil(t, err)
		return pool
	}
	return nil
}

func TestSessionStore(t *testing.T) {
	if _, ok := os.LookupEnv("REDIS_TEST"); !ok {
		t.Skip("skipping sso session store tests")
	}

	runTest := func(t *testing.T, pool fleet.RedisPool) {
		store := NewSessionStore(pool)
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

	t.Run("standalone", func(t *testing.T) {
		p := newPool(t, false)
		require.NotNil(t, p)
		defer p.Close()
		runTest(t, p)
	})

	t.Run("cluster", func(t *testing.T) {
		p := newPool(t, true)
		require.NotNil(t, p)
		defer p.Close()
		runTest(t, p)
	})
}
