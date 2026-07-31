package redis_config_etag

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestRedisConfigETag(t *testing.T) {
	for _, f := range []func(*testing.T, *Store){
		testGetSetAndGenerationInvalidation,
		testWriteFence,
		testMalformedRecord,
		testBlockedFlag,
	} {
		t.Run(test.FunctionName(f), func(t *testing.T) {
			t.Run("standalone", func(t *testing.T) {
				s := setupStore(t, false, false)
				f(t, s)
			})
			t.Run("cluster", func(t *testing.T) {
				s := setupStore(t, true, true)
				f(t, s)
			})
		})
	}
}

func setupStore(t testing.TB, cluster, redir bool) *Store {
	pool := redistest.SetupRedis(t, t.Name(), cluster, redir, true)
	return newStoreForTest(t, pool)
}

func newStoreForTest(t testing.TB, pool fleet.RedisPool) *Store {
	return &Store{
		pool:       pool,
		fenceTTL:   DefaultFenceTTL,
		etagTTL:    DefaultETagTTL,
		version:    "test",
		testPrefix: t.Name() + ":",
	}
}

// disarmFence deletes the fence key directly, simulating fence expiry without
// sleeping through a real TTL.
func disarmFence(t *testing.T, s *Store) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	_, err := conn.Do("DEL", s.testPrefix+fenceKey)
	require.NoError(t, err)
}

func testGetSetAndGenerationInvalidation(t *testing.T, s *Store) {
	ctx := context.Background()

	// empty store: miss
	etag, ok, err := s.GetValid(ctx, "global", "darwin")
	require.NoError(t, err)
	require.False(t, ok)
	require.Empty(t, etag)

	// store an etag with no fence armed
	stored, err := s.SetIfNoFence(ctx, "global", "darwin", `"abc"`)
	require.NoError(t, err)
	require.True(t, stored)

	etag, ok, err = s.GetValid(ctx, "global", "darwin")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `"abc"`, etag)

	// different scope/platform: independent keys
	_, ok, err = s.GetValid(ctx, "global", "windows")
	require.NoError(t, err)
	require.False(t, ok)
	_, ok, err = s.GetValid(ctx, "team:1", "darwin")
	require.NoError(t, err)
	require.False(t, ok)

	// Invalidate bumps the generation: the stored record must immediately
	// read as a miss, even though the key still physically exists. This is
	// the read half of the invalidation.
	require.NoError(t, s.Invalidate(ctx))
	_, ok, err = s.GetValid(ctx, "global", "darwin")
	require.NoError(t, err)
	require.False(t, ok)

	// After the fence is gone, a fresh write under the new generation is
	// valid again.
	disarmFence(t, s)
	stored, err = s.SetIfNoFence(ctx, "global", "darwin", `"def"`)
	require.NoError(t, err)
	require.True(t, stored)
	etag, ok, err = s.GetValid(ctx, "global", "darwin")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `"def"`, etag)
}

func testWriteFence(t *testing.T, s *Store) {
	ctx := context.Background()

	// Arm the fence via Invalidate: writes must be suppressed. This is the
	// poison-prevention property — a build using stale in-memory cache data
	// must not be able to persist its ETag after an invalidation.
	require.NoError(t, s.Invalidate(ctx))

	stored, err := s.SetIfNoFence(ctx, "team:5", "linux", `"stale"`)
	require.NoError(t, err)
	require.False(t, stored, "write must be suppressed while the fence is armed")

	_, ok, err := s.GetValid(ctx, "team:5", "linux")
	require.NoError(t, err)
	require.False(t, ok, "suppressed write must not be readable")

	// Both orderings of (write, invalidate) end with no valid stale record:
	// write-then-invalidate is covered in testGetSetAndGenerationInvalidation
	// (gen bump kills the record); this covers invalidate-then-write (fence
	// suppresses). Once the fence expires, writes settle again.
	disarmFence(t, s)
	stored, err = s.SetIfNoFence(ctx, "team:5", "linux", `"fresh"`)
	require.NoError(t, err)
	require.True(t, stored)
	etag, ok, err := s.GetValid(ctx, "team:5", "linux")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `"fresh"`, etag)
}

func testMalformedRecord(t *testing.T, s *Store) {
	ctx := context.Background()

	// A record that doesn't parse as "<gen>|<etag>" must be a miss, never a
	// match — GetValid can only fail toward a full config build.
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	_, err := conn.Do("SET", s.etagKey("global", "darwin"), "garbage-no-separator")
	require.NoError(t, err)

	_, ok, err := s.GetValid(ctx, "global", "darwin")
	require.NoError(t, err)
	require.False(t, ok)
}

func testBlockedFlag(t *testing.T, s *Store) {
	ctx := context.Background()

	loaderCalls := 0
	loader := func(v bool, err error) func(context.Context) (bool, error) {
		return func(context.Context) (bool, error) {
			loaderCalls++
			return v, err
		}
	}

	// miss → loader runs and the answer is cached
	blocked, err := s.ShortCircuitBlocked(ctx, loader(false, nil))
	require.NoError(t, err)
	require.False(t, blocked)
	require.Equal(t, 1, loaderCalls)

	// cached → loader must NOT run again
	blocked, err = s.ShortCircuitBlocked(ctx, loader(true, nil))
	require.NoError(t, err)
	require.False(t, blocked, "cached answer wins until reset/expiry")
	require.Equal(t, 1, loaderCalls)

	// reset → loader runs again, new answer cached
	require.NoError(t, s.ResetShortCircuitBlockedFlag(ctx))
	blocked, err = s.ShortCircuitBlocked(ctx, loader(true, nil))
	require.NoError(t, err)
	require.True(t, blocked)
	require.Equal(t, 2, loaderCalls)

	// loader error → reported as blocked (fail toward bypassing the fast
	// path) and NOT cached
	require.NoError(t, s.ResetShortCircuitBlockedFlag(ctx))
	blocked, err = s.ShortCircuitBlocked(ctx, loader(false, errors.New("db down")))
	require.Error(t, err)
	require.True(t, blocked, "errors must count as blocked")
}

// TestFenceTTLDerivation pins the documented derivation of DefaultFenceTTL so
// a change to it (or to the cache TTLs it depends on) forces a human to
// re-read the reasoning in the package docs.
func TestFenceTTLDerivation(t *testing.T) {
	// packConfigCache TTL (1m) + max relevant cached_mysql TTL (1m) + slack (1m).
	// See the DefaultFenceTTL comment before touching this.
	require.Equal(t, 3*time.Minute, DefaultFenceTTL)
	require.GreaterOrEqual(t, DefaultETagTTL, 10*DefaultFenceTTL,
		"backstop TTL should dwarf the fence window; see DefaultETagTTL docs before changing")
}
