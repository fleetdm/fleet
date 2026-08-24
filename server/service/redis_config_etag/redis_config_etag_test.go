package redis_config_etag

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
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
		testSharedGetSetAndGenerationInvalidation,
		testSharedWriteFence,
		testSharedMalformedRecord,
		testHostGetSetAndValidation,
		testHostQuarantine,
		testHostGenerationInvalidation,
		testLegacyPacksFlag,
		testLabelScopesState,
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
		pool:           pool,
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		fenceTTL:       DefaultFenceTTL,
		etagTTL:        DefaultETagTTL,
		hostETagMinTTL: DefaultHostETagMinTTL,
		hostETagMaxTTL: DefaultHostETagMaxTTL,
		quarantineTTL:  DefaultHostQuarantineTTL,
		version:        "test",
		testPrefix:     t.Name() + ":",
	}
}

// disarmFence deletes the deployment fence key directly, simulating fence
// expiry without sleeping through a real TTL.
func disarmFence(t *testing.T, s *Store) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	_, err := conn.Do("DEL", s.fenceKey())
	require.NoError(t, err)
}

// disarmQuarantine deletes a host's publish quarantine key directly,
// simulating quarantine expiry without sleeping.
func disarmQuarantine(t *testing.T, s *Store, hostID uint) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	_, err := conn.Do("DEL", s.hostQuarantineKey(hostID))
	require.NoError(t, err)
}

func testSharedGetSetAndGenerationInvalidation(t *testing.T, s *Store) {
	ctx := context.Background()

	// empty store: miss
	etag, ok, err := s.GetETagIfCurrent(ctx, "global", "darwin")
	require.NoError(t, err)
	require.False(t, ok)
	require.Empty(t, etag)

	// store an etag with no fence armed
	stored, err := s.SetIfNoFence(ctx, "global", "darwin", `"abc"`)
	require.NoError(t, err)
	require.True(t, stored)

	etag, ok, err = s.GetETagIfCurrent(ctx, "global", "darwin")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `"abc"`, etag)

	// different scope/platform: independent keys
	_, ok, err = s.GetETagIfCurrent(ctx, "global", "windows")
	require.NoError(t, err)
	require.False(t, ok)
	_, ok, err = s.GetETagIfCurrent(ctx, "team:1", "darwin")
	require.NoError(t, err)
	require.False(t, ok)

	// Invalidate bumps the generation: the stored record must immediately
	// read as a miss, even though the key still physically exists. This is
	// the read half of the invalidation.
	require.NoError(t, s.Invalidate(ctx))
	_, ok, err = s.GetETagIfCurrent(ctx, "global", "darwin")
	require.NoError(t, err)
	require.False(t, ok)

	// After the fence is gone, a fresh write under the new generation is
	// valid again.
	disarmFence(t, s)
	stored, err = s.SetIfNoFence(ctx, "global", "darwin", `"def"`)
	require.NoError(t, err)
	require.True(t, stored)
	etag, ok, err = s.GetETagIfCurrent(ctx, "global", "darwin")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `"def"`, etag)
}

func testSharedWriteFence(t *testing.T, s *Store) {
	ctx := context.Background()

	// Arm the fence via Invalidate: writes must be suppressed. This is the
	// poison-prevention property — a build using stale in-memory cache data
	// must not be able to persist its ETag after an invalidation.
	require.NoError(t, s.Invalidate(ctx))

	stored, err := s.SetIfNoFence(ctx, "team:5", "linux", `"stale"`)
	require.NoError(t, err)
	require.False(t, stored, "write must be suppressed while the fence is armed")

	_, ok, err := s.GetETagIfCurrent(ctx, "team:5", "linux")
	require.NoError(t, err)
	require.False(t, ok, "suppressed write must not be readable")

	// Both orderings of (write, invalidate) end with no valid stale record:
	// write-then-invalidate is covered in the generation test (gen bump
	// kills the record); this covers invalidate-then-write (fence
	// suppresses). Once the fence expires, writes settle again.
	disarmFence(t, s)
	stored, err = s.SetIfNoFence(ctx, "team:5", "linux", `"fresh"`)
	require.NoError(t, err)
	require.True(t, stored)
	etag, ok, err := s.GetETagIfCurrent(ctx, "team:5", "linux")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `"fresh"`, etag)
}

func testSharedMalformedRecord(t *testing.T, s *Store) {
	ctx := context.Background()

	// A record that doesn't parse as "<gen>|<etag>" must be a miss, never a
	// match — GetETagIfCurrent can only fail toward a full config build.
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	_, err := conn.Do("SET", s.etagKey("global", "darwin"), "garbage-no-separator")
	require.NoError(t, err)

	_, ok, err := s.GetETagIfCurrent(ctx, "global", "darwin")
	require.NoError(t, err)
	require.False(t, ok)
}

func testHostGetSetAndValidation(t *testing.T, s *Store) {
	ctx := context.Background()
	const hostID = uint(42)

	// empty: miss
	_, ok, err := s.GetHostETagIfCurrent(ctx, hostID, "team:7", "darwin")
	require.NoError(t, err)
	require.False(t, ok)

	stored, err := s.SetHostIfNoFence(ctx, hostID, "team:7", "darwin", `"h42"`)
	require.NoError(t, err)
	require.True(t, stored)

	etag, ok, err := s.GetHostETagIfCurrent(ctx, hostID, "team:7", "darwin")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `"h42"`, etag)

	// the record has a TTL within the jittered backstop bounds
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	ttl, err := redigoInt(conn.Do("TTL", s.hostETagKey(hostID)))
	require.NoError(t, err)
	require.GreaterOrEqual(t, ttl, int(DefaultHostETagMinTTL.Seconds())-1,
		"host record TTL must be within the jittered backstop bounds")
	require.LessOrEqual(t, ttl, int(DefaultHostETagMaxTTL.Seconds()),
		"host record TTL must be within the jittered backstop bounds")

	// isolation: another host never reads this record
	_, ok, err = s.GetHostETagIfCurrent(ctx, 43, "team:7", "darwin")
	require.NoError(t, err)
	require.False(t, ok, "a host record must never be readable by another host")

	// team transfer: stored scope no longer matches → miss, no cleanup needed
	_, ok, err = s.GetHostETagIfCurrent(ctx, hostID, "team:8", "darwin")
	require.NoError(t, err)
	require.False(t, ok, "a team transfer must read as a cache miss")

	// platform change: stored platform no longer matches → miss
	_, ok, err = s.GetHostETagIfCurrent(ctx, hostID, "team:7", "windows")
	require.NoError(t, err)
	require.False(t, ok, "a platform change must read as a cache miss")

	// deployment write fence suppresses per-host publication too
	require.NoError(t, s.Invalidate(ctx))
	stored, err = s.SetHostIfNoFence(ctx, hostID, "team:7", "darwin", `"h42b"`)
	require.NoError(t, err)
	require.False(t, stored, "the deployment fence must suppress per-host publication")
	disarmFence(t, s)
}

func testHostQuarantine(t *testing.T, s *Store) {
	ctx := context.Background()
	const hostID = uint(99)

	// warm record, then invalidate: record gone AND quarantine armed
	stored, err := s.SetHostIfNoFence(ctx, hostID, "global", "darwin", `"old"`)
	require.NoError(t, err)
	require.True(t, stored)

	require.NoError(t, s.InvalidateHost(ctx, hostID))
	_, ok, err := s.GetHostETagIfCurrent(ctx, hostID, "global", "darwin")
	require.NoError(t, err)
	require.False(t, ok, "InvalidateHost must delete the record")

	// A straddling build (read pre-change membership, publish after the DEL)
	// cannot persist while the quarantine is armed.
	stored, err = s.SetHostIfNoFence(ctx, hostID, "global", "darwin", `"stale"`)
	require.NoError(t, err)
	require.False(t, stored, "publication must be suppressed while the quarantine is armed")
	_, ok, err = s.GetHostETagIfCurrent(ctx, hostID, "global", "darwin")
	require.NoError(t, err)
	require.False(t, ok, "suppressed publication must not be readable")

	// quarantine suppression never affects other hosts
	stored, err = s.SetHostIfNoFence(ctx, 100, "global", "darwin", `"other"`)
	require.NoError(t, err)
	require.True(t, stored, "the quarantine is per-host and must not affect other hosts")

	// the quarantine key carries the configured TTL
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	ttl, err := redigoInt(conn.Do("TTL", s.hostQuarantineKey(hostID)))
	require.NoError(t, err)
	require.Positive(t, ttl)
	require.LessOrEqual(t, ttl, int(DefaultHostQuarantineTTL.Seconds()))

	// once the quarantine expires, publication resumes
	disarmQuarantine(t, s, hostID)
	stored, err = s.SetHostIfNoFence(ctx, hostID, "global", "darwin", `"fresh"`)
	require.NoError(t, err)
	require.True(t, stored)
	etag, ok, err := s.GetHostETagIfCurrent(ctx, hostID, "global", "darwin")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `"fresh"`, etag)
}

func testHostGenerationInvalidation(t *testing.T, s *Store) {
	ctx := context.Background()
	const hostID = uint(7)

	stored, err := s.SetHostIfNoFence(ctx, hostID, "global", "linux", `"gen0"`)
	require.NoError(t, err)
	require.True(t, stored)

	// a deployment-wide invalidation (config/report mutation) makes per-host
	// records stale through the embedded generation
	require.NoError(t, s.Invalidate(ctx))
	_, ok, err := s.GetHostETagIfCurrent(ctx, hostID, "global", "linux")
	require.NoError(t, err)
	require.False(t, ok, "deployment invalidation must invalidate per-host records")
	disarmFence(t, s)
}

func testLegacyPacksFlag(t *testing.T, s *Store) {
	ctx := context.Background()

	loaderCalls := 0
	loader := func(v bool, err error) func(context.Context) (bool, error) {
		return func(context.Context) (bool, error) {
			loaderCalls++
			return v, err
		}
	}

	// miss → loader runs and the answer is cached
	present, err := s.LegacyPacksPresent(ctx, loader(false, nil))
	require.NoError(t, err)
	require.False(t, present)
	require.Equal(t, 1, loaderCalls)

	// cached → loader must NOT run again
	present, err = s.LegacyPacksPresent(ctx, loader(true, nil))
	require.NoError(t, err)
	require.False(t, present, "cached answer wins until reset/expiry")
	require.Equal(t, 1, loaderCalls)

	// reset → loader runs again, new answer cached
	require.NoError(t, s.ResetLegacyPacksFlag(ctx))
	present, err = s.LegacyPacksPresent(ctx, loader(true, nil))
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, 2, loaderCalls)

	// loader error → reported as present (fail toward bypassing the fast
	// path) and NOT cached
	require.NoError(t, s.ResetLegacyPacksFlag(ctx))
	present, err = s.LegacyPacksPresent(ctx, loader(false, errors.New("db down")))
	require.Error(t, err)
	require.True(t, present, "errors must count as present")
}

func testLabelScopesState(t *testing.T, s *Store) {
	ctx := context.Background()

	loaderCalls := 0
	loader := func(v fleet.ConfigETagLabelScopes, err error) func(context.Context) (fleet.ConfigETagLabelScopes, error) {
		return func(context.Context) (fleet.ConfigETagLabelScopes, error) {
			loaderCalls++
			return v, err
		}
	}

	// miss → loader runs and the answer is cached
	scopes, err := s.LabelScopes(ctx, loader(fleet.ConfigETagLabelScopes{TeamIDs: []uint{3}}, nil))
	require.NoError(t, err)
	require.False(t, scopes.Global)
	require.Equal(t, []uint{3}, scopes.TeamIDs)
	require.Equal(t, 1, loaderCalls)

	// cached → loader must NOT run again
	scopes, err = s.LabelScopes(ctx, loader(fleet.ConfigETagLabelScopes{Global: true}, nil))
	require.NoError(t, err)
	require.False(t, scopes.Global, "cached answer wins until reset/expiry")
	require.Equal(t, []uint{3}, scopes.TeamIDs)
	require.Equal(t, 1, loaderCalls)

	// reset → loader runs again, new answer cached
	require.NoError(t, s.ResetLabelScopes(ctx))
	scopes, err = s.LabelScopes(ctx, loader(fleet.ConfigETagLabelScopes{Global: true}, nil))
	require.NoError(t, err)
	require.True(t, scopes.Global)
	require.Equal(t, 2, loaderCalls)

	// loader error → error out (caller must treat as unknown → bypass)
	require.NoError(t, s.ResetLabelScopes(ctx))
	_, err = s.LabelScopes(ctx, loader(fleet.ConfigETagLabelScopes{}, errors.New("db down")))
	require.Error(t, err)

	// the cached state key carries the bounded TTL
	require.NoError(t, s.ResetLabelScopes(ctx))
	_, err = s.LabelScopes(ctx, loader(fleet.ConfigETagLabelScopes{Global: true}, nil))
	require.NoError(t, err)
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	ttl, terr := redigoInt(conn.Do("TTL", s.scopeModesKey()))
	require.NoError(t, terr)
	require.Positive(t, ttl)
	require.LessOrEqual(t, ttl, int(gateStateTTL.Seconds()))
}

// redigoInt converts a redigo reply to int, for TTL assertions.
func redigoInt(reply any, err error) (int, error) {
	if err != nil {
		return 0, err
	}
	n, ok := reply.(int64)
	if !ok {
		return 0, errors.New("unexpected reply type")
	}
	return int(n), nil
}

// TestTTLDerivations pins the documented TTL derivations so a change to any
// of them (or to the cache TTLs they depend on) forces a human to re-read
// the reasoning in the package docs.
func TestTTLDerivations(t *testing.T) {
	// packConfigCache TTL (1m) + max relevant cached_mysql TTL (1m) + slack (1m).
	// See the DefaultFenceTTL comment before touching this.
	require.Equal(t, 3*time.Minute, DefaultFenceTTL)
	require.GreaterOrEqual(t, DefaultETagTTL, 10*DefaultFenceTTL,
		"shared backstop TTL should dwarf the fence window; see DefaultETagTTL docs before changing")
	// The quarantine is derived from replica lag + in-flight duration —
	// deliberately NOT the fence's 180s (see DefaultHostQuarantineTTL docs).
	require.Equal(t, 30*time.Second, DefaultHostQuarantineTTL)
	require.Less(t, DefaultHostQuarantineTTL, DefaultFenceTTL)
	// Host record TTLs are jittered around ~1h and must stay within the
	// bounded-staleness contract.
	require.Less(t, DefaultHostETagMinTTL, DefaultHostETagMaxTTL,
		"host record TTLs must be jittered")
	require.GreaterOrEqual(t, DefaultHostETagMinTTL, 30*time.Minute)
	require.LessOrEqual(t, DefaultHostETagMaxTTL, 2*time.Hour)
}

// TestGateLoaderLeaderElection: one Redis miss must NOT become one database
// query per concurrent config request. Exactly one concurrent caller (the
// elected leader) runs the loader; losers return
// fleet.ErrConfigETagGateLoading immediately — they never wait and never
// query — and the state-write therefore happens at most once per miss
// window per container.
func TestGateLoaderLeaderElection(t *testing.T) {
	pool := redistest.SetupRedis(t, t.Name(), false, false, true)
	s := newStoreForTest(t, pool)
	ctx := context.Background()

	var loaderCalls atomic.Int64
	slowLoader := func(context.Context) (bool, error) {
		loaderCalls.Add(1)
		time.Sleep(100 * time.Millisecond) // hold the miss window open
		return false, nil
	}

	const workers = 50
	var wg sync.WaitGroup
	var contended, answered, failed atomic.Int64
	for range workers {
		wg.Go(func() {
			present, err := s.LegacyPacksPresent(ctx, slowLoader)
			switch {
			case errors.Is(err, fleet.ErrConfigETagGateLoading):
				// losers must read as "present" (bypass direction)
				if present {
					contended.Add(1)
				} else {
					failed.Add(1)
				}
			case err != nil:
				failed.Add(1)
			default:
				answered.Add(1)
			}
		})
	}
	wg.Wait()

	require.Equal(t, int64(1), loaderCalls.Load(),
		"exactly one concurrent caller may run the database loader")
	require.Equal(t, int64(0), failed.Load(), "no caller may observe a real error")
	require.Positive(t, contended.Load(), "losers must be told the load is in flight")
	require.Equal(t, int64(workers), contended.Load()+answered.Load())

	// After the window closes, everyone reads the cached answer with no
	// further loader executions.
	present, err := s.LegacyPacksPresent(ctx, slowLoader)
	require.NoError(t, err)
	require.False(t, present)
	require.Equal(t, int64(1), loaderCalls.Load())

	// Same election protocol for the label-scope gate.
	var scopeLoads atomic.Int64
	slowScopes := func(context.Context) (fleet.ConfigETagLabelScopes, error) {
		scopeLoads.Add(1)
		time.Sleep(100 * time.Millisecond)
		return fleet.ConfigETagLabelScopes{Global: true}, nil
	}
	var scopeContended, scopeFailed atomic.Int64
	for range workers {
		wg.Go(func() {
			_, err := s.LabelScopes(ctx, slowScopes)
			switch {
			case errors.Is(err, fleet.ErrConfigETagGateLoading):
				scopeContended.Add(1)
			case err != nil:
				scopeFailed.Add(1)
			}
		})
	}
	wg.Wait()
	require.Equal(t, int64(1), scopeLoads.Load())
	require.Equal(t, int64(0), scopeFailed.Load(), "no caller may observe a real error")
	require.Positive(t, scopeContended.Load())
}
