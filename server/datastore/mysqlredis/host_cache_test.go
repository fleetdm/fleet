package mysqlredis

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/go-json-experiment/json/v1"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// hostCacheTestCleanupPrefix is the key-prefix passed to redistest.SetupRedis so
// every run cleans up only keys owned by these tests (redistest requires a
// prefix to prevent concurrent tests from clobbering each other's keys).
const hostCacheTestCleanupPrefix = "fleet:hostcache:v1"

// hostCacheFamily parameterizes the load-path tests across both cache families (osquery `LoadHostByNodeKey`
// and orbit `LoadHostByOrbitNodeKey`). Every end-to-end override test runs once per family; field-fidelity is
// covered by TestHostCacheEnvelopeRoundTrip, so these tests focus on cache semantics only.
//
// Embeds the production cacheFamily so tests have direct access to the same key constructors and
// nodeKeyOf accessor the production code uses. The test-only fields (name, load, mock setters, buildHost)
// wire the tests to the public API and the mock.DataStore harness.
type hostCacheFamily struct {
	cacheFamily
	name       string
	sampleKey  string
	load       func(*Datastore, context.Context, string) (*fleet.Host, error)
	setMock    func(*mock.DataStore, func(context.Context, string) (*fleet.Host, error))
	getInvoked func(*mock.DataStore) bool
	setInvoked func(*mock.DataStore, bool)
	// buildHost returns a minimally-populated *fleet.Host whose relevant node_key pointer matches the
	// argument, so the cache put under this family will succeed.
	buildHost func(id uint, key string) *fleet.Host
}

var hostCacheFamilies = []hostCacheFamily{
	{
		cacheFamily: osqueryCacheFamily,
		name:        "osquery",
		sampleKey:   "nk-test",
		load: func(d *Datastore, ctx context.Context, k string) (*fleet.Host, error) {
			return d.LoadHostByNodeKey(ctx, k)
		},
		setMock: func(ds *mock.DataStore, f func(context.Context, string) (*fleet.Host, error)) {
			ds.LoadHostByNodeKeyFunc = f
		},
		getInvoked: func(ds *mock.DataStore) bool { return ds.LoadHostByNodeKeyFuncInvoked },
		setInvoked: func(ds *mock.DataStore, v bool) { ds.LoadHostByNodeKeyFuncInvoked = v },
		buildHost: func(id uint, k string) *fleet.Host {
			kp := k
			return &fleet.Host{ID: id, NodeKey: &kp, Hostname: "h-" + k}
		},
	},
	{
		cacheFamily: orbitCacheFamily,
		name:        "orbit",
		sampleKey:   "onk-test",
		load: func(d *Datastore, ctx context.Context, k string) (*fleet.Host, error) {
			return d.LoadHostByOrbitNodeKey(ctx, k)
		},
		setMock: func(ds *mock.DataStore, f func(context.Context, string) (*fleet.Host, error)) {
			ds.LoadHostByOrbitNodeKeyFunc = f
		},
		getInvoked: func(ds *mock.DataStore) bool { return ds.LoadHostByOrbitNodeKeyFuncInvoked },
		setInvoked: func(ds *mock.DataStore, v bool) { ds.LoadHostByOrbitNodeKeyFuncInvoked = v },
		buildHost: func(id uint, k string) *fleet.Host {
			kp := k
			return &fleet.Host{ID: id, OrbitNodeKey: &kp, Hostname: "h-" + k}
		},
	},
}

func TestHostCacheHelpers(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		ctx := t.Context()
		ds := new(mock.Store)
		wrapped := New(ds, pool, WithHostCache(30*time.Second))

		t.Run("put then get returns the host", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			nk := "node-put-get"
			teamID := uint(42)
			certTrue := true
			stored := &fleet.Host{
				ID:                  7,
				NodeKey:             &nk,
				TeamID:              &teamID,
				Hostname:            "host-7",
				Platform:            "darwin",
				HasHostIdentityCert: &certTrue,
			}
			wrapped.hostCachePutByNodeKey(ctx, stored)

			loaded, result := wrapped.hostCacheGetByNodeKey(ctx, nk)
			require.Equal(t, hostCacheLookupHit, result)
			require.NotNil(t, loaded)
			assert.Equal(t, stored.ID, loaded.ID)
			assert.Equal(t, stored.Hostname, loaded.Hostname)
			assert.Equal(t, stored.Platform, loaded.Platform)
			require.NotNil(t, loaded.TeamID)
			assert.Equal(t, teamID, *loaded.TeamID)
			require.NotNil(t, loaded.HasHostIdentityCert)
			assert.True(t, *loaded.HasHostIdentityCert)
			require.NotNil(t, loaded.NodeKey)
			assert.Equal(t, nk, *loaded.NodeKey)

			// Reverse index should be populated.
			conn := redis.ConfigureDoer(pool, pool.Get())
			defer conn.Close()
			got, err := redigo.String(conn.Do("GET", hostCacheIndexByID(stored.ID)))
			require.NoError(t, err)
			assert.Equal(t, nk, got)
		})

		t.Run("miss returns hostCacheLookupMiss", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			host, result := wrapped.hostCacheGetByNodeKey(ctx, "node-never-cached")
			assert.Nil(t, host)
			assert.Equal(t, hostCacheLookupMiss, result)
		})

		t.Run("negative cache returns hostCacheLookupNegative", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			wrapped.hostCachePutNotFoundByNodeKey(ctx, "node-missing")
			host, result := wrapped.hostCacheGetByNodeKey(ctx, "node-missing")
			assert.Nil(t, host)
			assert.Equal(t, hostCacheLookupNegative, result)
		})

		t.Run("positive cache takes precedence over negative", func(t *testing.T) {
			// A future write could simultaneously populate positive and leave a
			// stale negative; we must not surface notFound in that case.
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			nk := "node-both"
			wrapped.hostCachePutNotFoundByNodeKey(ctx, nk)
			wrapped.hostCachePutByNodeKey(ctx, &fleet.Host{ID: 9, NodeKey: &nk, Hostname: "wins"})

			loaded, result := wrapped.hostCacheGetByNodeKey(ctx, nk)
			require.Equal(t, hostCacheLookupHit, result)
			assert.Equal(t, "wins", loaded.Hostname)
		})

		t.Run("delete by node_key clears primary, negative, and index", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			// Set up the worst-case state: positive cache, negative cache, and reverse index all
			// populated for the same node_key. This combination occurs in production when a probe
			// arrived before enrollment (writing the negative entry) and then enrollment completed
			// (writing positive + index). The delete contract is "clean up all keys for this
			// node_key regardless of which are live," so we populate all three to verify the
			// delete clears them all.
			nk := "node-del-nk"
			wrapped.hostCachePutByNodeKey(ctx, &fleet.Host{ID: 10, NodeKey: &nk})
			wrapped.hostCachePutNotFoundByNodeKey(ctx, nk)

			wrapped.hostCacheDeleteByNodeKey(ctx, nk, 10, "update")

			_, result := wrapped.hostCacheGetByNodeKey(ctx, nk)
			assert.Equal(t, hostCacheLookupMiss, result)

			conn := redis.ConfigureDoer(pool, pool.Get())
			defer conn.Close()
			for _, k := range []string{
				hostCacheKeyByNodeKey(nk),
				hostCacheKeyMiss(nk),
				hostCacheIndexByID(10),
			} {
				exists, err := redigo.Bool(conn.Do("EXISTS", k))
				require.NoError(t, err)
				assert.False(t, exists, "key %s should have been deleted", k)
			}
		})

		t.Run("delete by id resolves node_key via index", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			nk := "node-del-id"
			wrapped.hostCachePutByNodeKey(ctx, &fleet.Host{ID: 11, NodeKey: &nk})

			wrapped.hostCacheDeleteByID(ctx, 11, "team")

			_, result := wrapped.hostCacheGetByNodeKey(ctx, nk)
			assert.Equal(t, hostCacheLookupMiss, result)

			conn := redis.ConfigureDoer(pool, pool.Get())
			defer conn.Close()
			exists, err := redigo.Bool(conn.Do("EXISTS", hostCacheIndexByID(11)))
			require.NoError(t, err)
			assert.False(t, exists)
		})
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, hostCacheTestCleanupPrefix, false, false, false)
		runTest(t, pool)
	})
	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, hostCacheTestCleanupPrefix, true, true, false)
		runTest(t, pool)
	})
}

// TestJitteredHostCacheTTL covers the invariants TestPBT_JitteredHostCacheTTLBounds cannot:
// variance over many draws at a single base, and the zero-base edge case. Bounds across the
// full base-TTL range are covered by the property-based test.
func TestJitteredHostCacheTTL(t *testing.T) {
	// Variance: 1000 draws at one base should produce a spread, not a constant value. (rapid runs one
	// jitter draw per generated input, so it doesn't directly check that any single base produces
	// non-degenerate variance.)
	d := &Datastore{hostCacheEnabled: true, hostCacheTTL: 30 * time.Second}
	const samples = 1000
	var minSeen, maxSeen time.Duration = math.MaxInt64, 0
	for range samples {
		got := d.jitteredHostCacheTTL()
		if got < minSeen {
			minSeen = got
		}
		if got > maxSeen {
			maxSeen = got
		}
	}
	assert.Less(t, minSeen, maxSeen, "jitter produced no variance over %d samples", samples)

	// Zero base returns zero. (PBT only generates positive bases.)
	zero := &Datastore{hostCacheEnabled: true, hostCacheTTL: 0}
	assert.Equal(t, time.Duration(0), zero.jitteredHostCacheTTL())
}

func TestLoadHost_CacheDisabled(t *testing.T) {
	for _, fam := range hostCacheFamilies {
		t.Run(fam.name, func(t *testing.T) {
			ctx := t.Context()
			ds := new(mock.DataStore)
			fam.setMock(ds, func(_ context.Context, k string) (*fleet.Host, error) {
				return fam.buildHost(1, k), nil
			})
			wrapped := New(ds, redistest.NopRedis()) // no WithHostCache

			_, err := fam.load(wrapped, ctx, fam.sampleKey)
			require.NoError(t, err)
			assert.True(t, fam.getInvoked(ds))

			// Second call also hits the inner datastore (there's no cache).
			fam.setInvoked(ds, false)
			_, err = fam.load(wrapped, ctx, fam.sampleKey)
			require.NoError(t, err)
			assert.True(t, fam.getInvoked(ds), "cache disabled: every call must go to DB")
		})
	}
}

func TestLoadHost_Override(t *testing.T) {
	runFamily := func(t *testing.T, fam hostCacheFamily, pool fleet.RedisPool) {
		t.Run("cache miss then hit", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ctx := t.Context()
			ds := new(mock.DataStore)
			fam.setMock(ds, func(_ context.Context, k string) (*fleet.Host, error) {
				return fam.buildHost(1, k), nil
			})
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			first, err := fam.load(wrapped, ctx, fam.sampleKey)
			require.NoError(t, err)
			require.NotNil(t, first)
			require.True(t, fam.getInvoked(ds))

			fam.setInvoked(ds, false)
			second, err := fam.load(wrapped, ctx, fam.sampleKey)
			require.NoError(t, err)
			require.NotNil(t, second)
			assert.False(t, fam.getInvoked(ds), "second call should be served from cache")
			assert.Equal(t, first.ID, second.ID, "cached value should match the initial DB read")
		})

		t.Run("NotFound populates negative cache", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ctx := t.Context()
			ds := new(mock.DataStore)
			var callCount atomic.Int32
			fam.setMock(ds, func(_ context.Context, _ string) (*fleet.Host, error) {
				callCount.Add(1)
				return nil, common_mysql.NotFound("Host")
			})
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			_, err := fam.load(wrapped, ctx, fam.sampleKey+"-absent")
			require.Error(t, err)
			assert.True(t, fleet.IsNotFound(err))
			assert.Equal(t, int32(1), callCount.Load())

			// Second call hits the negative cache; inner is not invoked.
			_, err = fam.load(wrapped, ctx, fam.sampleKey+"-absent")
			require.Error(t, err)
			assert.True(t, fleet.IsNotFound(err))
			assert.Equal(t, int32(1), callCount.Load(), "negative cache should have served the second call")
		})

		t.Run("transient errors are not cached", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ctx := t.Context()
			ds := new(mock.DataStore)
			transient := errors.New("simulated timeout")
			var callCount atomic.Int32
			fam.setMock(ds, func(_ context.Context, _ string) (*fleet.Host, error) {
				callCount.Add(1)
				return nil, transient
			})
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			_, err := fam.load(wrapped, ctx, fam.sampleKey+"-transient")
			require.ErrorIs(t, err, transient)

			_, err = fam.load(wrapped, ctx, fam.sampleKey+"-transient")
			require.ErrorIs(t, err, transient)
			assert.Equal(t, int32(2), callCount.Load(), "transient errors must not poison the cache")
		})

		t.Run("singleflight collapses concurrent misses", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			// Block the inner DB call until the test releases it, so all
			// goroutines pile up in the singleflight group before it resolves.
			ds := new(mock.DataStore)
			var callCount atomic.Int32
			release := make(chan struct{})
			fam.setMock(ds, func(_ context.Context, k string) (*fleet.Host, error) {
				callCount.Add(1)
				<-release
				return fam.buildHost(42, k), nil
			})
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			const goroutines = 20
			var wg sync.WaitGroup
			errs := make([]error, goroutines)
			hosts := make([]*fleet.Host, goroutines)
			for i := range goroutines {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					h, err := fam.load(wrapped, t.Context(), fam.sampleKey+"-sf")
					hosts[i] = h
					errs[i] = err
				}(i)
			}
			// Give goroutines time to enter singleflight before releasing.
			time.Sleep(50 * time.Millisecond)
			close(release)
			wg.Wait()

			assert.Equal(t, int32(1), callCount.Load(), "singleflight should collapse to one DB call")
			for i, err := range errs {
				require.NoError(t, err, "goroutine %d", i)
				require.NotNil(t, hosts[i])
				assert.Equal(t, uint(42), hosts[i].ID)
			}

			// Each caller must receive its own struct so mutation is safe.
			for i := 1; i < goroutines; i++ {
				assert.NotSame(t, hosts[0], hosts[i], "callers must receive independent structs")
			}
		})

		t.Run("canceled caller does not poison the shared DB call", func(t *testing.T) {
			// Without ctx detach, cancelling the caller whose goroutine happens to be the
			// singleflight leader would cancel the shared DB query and fail
			// every joiner even though their own contexts are alive.
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			ds := new(mock.DataStore)
			release := make(chan struct{})
			fam.setMock(ds, func(innerCtx context.Context, k string) (*fleet.Host, error) {
				<-release
				// The inner ctx should not observe the canceling caller's
				// Done signal. If it does, this returns ctx.Err() and the
				// joiners fail.
				if err := innerCtx.Err(); err != nil {
					return nil, err
				}
				return fam.buildHost(77, k), nil
			})
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			cancellableCtx, cancel := context.WithCancel(t.Context())
			joinerCtx := t.Context()
			var joinerHost *fleet.Host
			var joinerErr error

			// Start the CANCELLABLE caller first and give it time to enter
			// singleflight before the joiner races in. This guarantees the
			// cancellable context is the flight leader — otherwise a lucky
			// scheduling could let the healthy joiner become leader and the
			// test would pass for the wrong reason.
			leaderDone := make(chan struct{})
			go func() {
				_, _ = fam.load(wrapped, cancellableCtx, fam.sampleKey+"-cancel")
				close(leaderDone)
			}()
			time.Sleep(50 * time.Millisecond)

			joinerDone := make(chan struct{})
			go func() {
				joinerHost, joinerErr = fam.load(wrapped, joinerCtx, fam.sampleKey+"-cancel")
				close(joinerDone)
			}()

			// Give the joiner a moment to enter Do() and attach to the flight.
			time.Sleep(20 * time.Millisecond)
			cancel()       // cancel the leader while inner DB call is still blocked
			close(release) // let the inner DB call finish
			<-joinerDone
			<-leaderDone

			require.NoError(t, joinerErr, "joiner must not see the leader's cancellation")
			require.NotNil(t, joinerHost)
			assert.Equal(t, uint(77), joinerHost.ID)
		})
	}

	for _, fam := range hostCacheFamilies {
		t.Run(fam.name, func(t *testing.T) {
			t.Run("standalone", func(t *testing.T) {
				pool := redistest.SetupRedis(t, hostCacheTestCleanupPrefix, false, false, false)
				runFamily(t, fam, pool)
			})
			t.Run("cluster", func(t *testing.T) {
				pool := redistest.SetupRedis(t, hostCacheTestCleanupPrefix, true, true, false)
				runFamily(t, fam, pool)
			})
		})
	}
}

func TestLoadHost_RedisErrorFallsThrough(t *testing.T) {
	for _, fam := range hostCacheFamilies {
		t.Run(fam.name, func(t *testing.T) {
			ctx := t.Context()
			ds := new(mock.DataStore)
			fam.setMock(ds, func(_ context.Context, k string) (*fleet.Host, error) {
				return fam.buildHost(1, k), nil
			})
			wrapped := New(ds, errPool{}, WithHostCache(30*time.Second))

			// Redis ops all error; the caller must still get a host via the DB path.
			got, err := fam.load(wrapped, ctx, fam.sampleKey+"-redis-down")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.True(t, fam.getInvoked(ds))

			// Second call also hits DB (cache never populated successfully).
			fam.setInvoked(ds, false)
			_, err = fam.load(wrapped, ctx, fam.sampleKey+"-redis-down")
			require.NoError(t, err)
			assert.True(t, fam.getInvoked(ds), "Redis errors must not prevent DB fallthrough")
		})
	}
}

// TestHostCacheUnifiedInvalidation proves the unified-invalidation design: a
// write that goes through hostCacheDeleteByID clears BOTH cache families for a
// host that has both agents enrolled. This is the cross-family property that
// per-family override tests can't express.
func TestHostCacheUnifiedInvalidation(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
		ctx := t.Context()
		ds := new(mock.DataStore)
		wrapped := New(ds, pool, WithHostCache(30*time.Second))

		nk := "nk-both"
		onk := "onk-both"
		host := &fleet.Host{ID: 99, NodeKey: &nk, OrbitNodeKey: &onk, Hostname: "both"}
		wrapped.hostCachePutByNodeKey(ctx, host)
		wrapped.hostCachePutByOrbitNodeKey(ctx, host)

		// Sanity: both hot.
		_, res := wrapped.hostCacheGetByNodeKey(ctx, nk)
		require.Equal(t, hostCacheLookupHit, res)
		_, res = wrapped.hostCacheGetByOrbitNodeKey(ctx, onk)
		require.Equal(t, hostCacheLookupHit, res)

		wrapped.hostCacheDeleteByID(ctx, 99, "update")

		_, res = wrapped.hostCacheGetByNodeKey(ctx, nk)
		assert.Equal(t, hostCacheLookupMiss, res)
		_, res = wrapped.hostCacheGetByOrbitNodeKey(ctx, onk)
		assert.Equal(t, hostCacheLookupMiss, res)
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, hostCacheTestCleanupPrefix, false, false, false)
		runTest(t, pool)
	})
	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, hostCacheTestCleanupPrefix, true, true, false)
		runTest(t, pool)
	})
}

// errPool is a minimal fleet.RedisPool whose connections always fail. Used to
// prove the cache layer swallows Redis errors and falls through to the DB
// without propagating to the caller.
type errPool struct{}

func (errPool) Get() redigo.Conn                   { return errConn{} }
func (errPool) Close() error                       { return nil }
func (errPool) Stats() map[string]redigo.PoolStats { return nil }
func (errPool) Mode() fleet.RedisMode              { return fleet.RedisStandalone }

type errConn struct{}

var errRedisDown = errors.New("simulated redis down")

func (errConn) Close() error                       { return nil }
func (errConn) Err() error                         { return errRedisDown }
func (errConn) Do(_ string, _ ...any) (any, error) { return nil, errRedisDown }
func (errConn) Send(_ string, _ ...any) error      { return errRedisDown }
func (errConn) Flush() error                       { return errRedisDown }
func (errConn) Receive() (any, error)              { return nil, errRedisDown }

// cleanupHostCacheKeys removes all host-cache keys between subtests so leftover
// state doesn't leak across cases. Uses redis.ScanKeys which walks every node
// in cluster mode (not just the one backing pool.Get()), so subtests running
// against a cluster pool are fully isolated.
func cleanupHostCacheKeys(t *testing.T, pool fleet.RedisPool) {
	t.Helper()

	for _, sub := range []string{":nk:*", ":nk_miss:*", ":id2nk:*", ":onk:*", ":onk_miss:*", ":id2onk:*"} {
		pattern := hostCacheKeyPrefix + sub
		keys, err := redis.ScanKeys(pool, pattern, 100)
		require.NoError(t, err, "scan %q", pattern)
		for _, k := range keys {
			conn := redis.ConfigureDoer(pool, pool.Get())
			_, err := conn.Do("DEL", k)
			conn.Close()
			require.NoError(t, err, "del %q", k)
		}
	}
}

// hostFieldsGen produces *fleet.Host values that exercise every field
// LoadHostByNodeKey or LoadHostByOrbitNodeKey populates from the database.
// Pointer fields randomly choose between nil and a generated value so the
// generator covers both omitempty-skipped and present cases.
//
// Fields the generator deliberately leaves at zero:
//   - NetworkInterfaces and DiskEncryptionKeyEscrowed are tagged json:"-" on
//     fleet.Host and are NOT shadowed by hostCacheEnvelope. They round-trip
//     to zero by design; varying them would make the round-trip test fail
//     for a non-bug reason.
//   - HostSoftware (embedded) is not loaded by the cache's SQL queries; we
//     leave it at its zero value to mirror real behavior.
//
// Time values are always UTC with no monotonic component, since RFC3339Nano
// JSON encoding does not preserve time.Location names or the monotonic clock
// reading. Testing those would conflate JSON's known representation choice
// with cache-specific bugs.
func hostFieldsGen() *rapid.Generator[*fleet.Host] {
	return rapid.Custom(func(t *rapid.T) *fleet.Host {
		// Bound to year-1970 through ~year-2096 to stay safely within RFC3339's
		// 4-digit-year encoding range.
		drawTime := func(label string) time.Time {
			sec := rapid.Int64Range(0, 4_000_000_000).Draw(t, label+"_sec")
			nsec := rapid.Int64Range(0, 999_999_999).Draw(t, label+"_nsec")
			return time.Unix(sec, nsec).UTC()
		}

		drawPtrString := func(label string) *string {
			if !rapid.Bool().Draw(t, label+"_set") {
				return nil
			}
			v := rapid.String().Draw(t, label+"_v")
			return &v
		}

		drawPtrBool := func(label string) *bool {
			if !rapid.Bool().Draw(t, label+"_set") {
				return nil
			}
			v := rapid.Bool().Draw(t, label+"_v")
			return &v
		}

		drawPtrUint := func(label string) *uint {
			if !rapid.Bool().Draw(t, label+"_set") {
				return nil
			}
			v := uint(rapid.Uint64Range(0, 1<<32).Draw(t, label+"_v"))
			return &v
		}

		drawPtrTime := func(label string) *time.Time {
			if !rapid.Bool().Draw(t, label+"_set") {
				return nil
			}
			v := drawTime(label + "_v")
			return &v
		}

		// Bound floats to a realistic disk-space range. Excludes NaN/Inf,
		// which JSON cannot encode at all (would error rather than mismatch).
		boundedFloat := rapid.Float64Range(0, 1_000_000)

		h := &fleet.Host{
			ID:                          uint(rapid.Uint64Range(0, 1<<32).Draw(t, "id")),
			OsqueryHostID:               drawPtrString("osquery_host_id"),
			DetailUpdatedAt:             drawTime("detail_updated"),
			NodeKey:                     drawPtrString("node_key"),
			Hostname:                    rapid.String().Draw(t, "hostname"),
			UUID:                        rapid.String().Draw(t, "uuid"),
			Platform:                    rapid.String().Draw(t, "platform"),
			OsqueryVersion:              rapid.String().Draw(t, "osquery_version"),
			OSVersion:                   rapid.String().Draw(t, "os_version"),
			Build:                       rapid.String().Draw(t, "build"),
			PlatformLike:                rapid.String().Draw(t, "platform_like"),
			CodeName:                    rapid.String().Draw(t, "code_name"),
			Uptime:                      time.Duration(rapid.Int64Range(0, int64(30*24*time.Hour)).Draw(t, "uptime")),
			Memory:                      rapid.Int64Range(0, 1<<40).Draw(t, "memory"),
			CPUType:                     rapid.String().Draw(t, "cpu_type"),
			CPUSubtype:                  rapid.String().Draw(t, "cpu_subtype"),
			CPUBrand:                    rapid.String().Draw(t, "cpu_brand"),
			CPUPhysicalCores:            rapid.IntRange(0, 256).Draw(t, "cpu_physical_cores"),
			CPULogicalCores:             rapid.IntRange(0, 256).Draw(t, "cpu_logical_cores"),
			HardwareVendor:              rapid.String().Draw(t, "hw_vendor"),
			HardwareModel:               rapid.String().Draw(t, "hw_model"),
			HardwareVersion:             rapid.String().Draw(t, "hw_version"),
			HardwareSerial:              rapid.String().Draw(t, "hw_serial"),
			ComputerName:                rapid.String().Draw(t, "computer_name"),
			TimeZone:                    drawPtrString("timezone"),
			PrimaryNetworkInterfaceID:   drawPtrUint("primary_ip_id"),
			PublicIP:                    rapid.String().Draw(t, "public_ip"),
			PrimaryIP:                   rapid.String().Draw(t, "primary_ip"),
			PrimaryMac:                  rapid.String().Draw(t, "primary_mac"),
			DistributedInterval:         uint(rapid.Uint64Range(0, 86400).Draw(t, "distributed_interval")),
			ConfigTLSRefresh:            uint(rapid.Uint64Range(0, 86400).Draw(t, "config_tls_refresh")),
			LoggerTLSPeriod:             uint(rapid.Uint64Range(0, 86400).Draw(t, "logger_tls_period")),
			LabelUpdatedAt:              drawTime("label_updated"),
			LastEnrolledAt:              drawTime("last_enrolled"),
			RefetchRequested:            rapid.Bool().Draw(t, "refetch_requested"),
			RefetchCriticalQueriesUntil: drawPtrTime("refetch_critical"),
			TeamID:                      drawPtrUint("team_id"),
			PolicyUpdatedAt:             drawTime("policy_updated"),
			OrbitNodeKey:                drawPtrString("orbit_node_key"),
			LastRestartedAt:             drawTime("last_restarted"),
			GigsDiskSpaceAvailable:      boundedFloat.Draw(t, "gigs_avail"),
			GigsTotalDiskSpace:          boundedFloat.Draw(t, "gigs_total"),
			PercentDiskSpaceAvailable:   boundedFloat.Draw(t, "pct_avail"),
			HasHostIdentityCert:         drawPtrBool("has_cert"),
			// Orbit-specific fields. LoadHostByOrbitNodeKey populates these;
			// LoadHostByNodeKey leaves them nil. Either is a valid input.
			DEPAssignedToFleet:    drawPtrBool("dep"),
			DiskEncryptionEnabled: drawPtrBool("enc"),
			TeamName:              drawPtrString("team_name"),
			MDM:                   fleet.MDMHostData{EncryptionKeyAvailable: rapid.Bool().Draw(t, "mdm_eka")},
		}
		h.CreatedAt = drawTime("created_at")
		h.UpdatedAt = drawTime("updated_at")
		return h
	})
}

// TestPBT_HostCacheEnvelopeRoundTrip is the property-test version of the
// example-based round-trip test. It is the schema-drift tripwire's stronger
// form: rapid generates millions of host shapes (varying nil/non-nil for
// every pointer, varying time values, varying string contents), and the
// envelope must round-trip every one of them.
//
// Catches regressions that an example test cannot: a new fleet.Host field
// added without omitempty whose JSON shape we did not anticipate, a field
// type change that breaks the JSON encoder for some inputs (e.g. the
// time.Duration trap that v2 surfaced), or a hidden interaction between
// embedded-struct shadowing and pointer nil-versus-empty cases.
func TestPBT_HostCacheEnvelopeRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orig := hostFieldsGen().Draw(t, "host")

		raw, err := json.Marshal(envelopeFromHost(orig))
		require.NoError(t, err, "marshal must not fail for any generated *fleet.Host")

		envelope := new(hostCacheEnvelope)
		require.NoError(t, json.Unmarshal(raw, envelope), "unmarshal must accept any output of marshal")
		got := envelope.toHost()

		ignoreUnexported := cmpopts.IgnoreUnexported(fleet.Host{}, fleet.MDMHostData{})
		if diff := cmp.Diff(orig, got, ignoreUnexported); diff != "" {
			t.Fatalf("round-trip mismatch (-orig +got):\n%s", diff)
		}

		// Belt-and-braces on the four security-critical shadow fields. A
		// cmp.Diff failure could in principle be obscured by reporting a
		// different field; an explicit assertion fails loudly with a clear
		// message when these specifically drop. require.Equal on pointers
		// handles all three cases correctly: both-nil, one-nil, and
		// both-set-with-equal-pointee.
		require.Equal(t, orig.NodeKey, got.NodeKey, "NodeKey shadow field")
		require.Equal(t, orig.OrbitNodeKey, got.OrbitNodeKey, "OrbitNodeKey shadow field")
		require.Equal(t, orig.OsqueryHostID, got.OsqueryHostID, "OsqueryHostID shadow field")
		require.Equal(t, orig.HasHostIdentityCert, got.HasHostIdentityCert, "HasHostIdentityCert shadow field")
	})
}

// TestPBT_JitteredHostCacheTTLBounds asserts the jitter bounds across the
// full legal range of base TTLs, not just the single 30s value the example
// test uses. Every positive base must yield a strictly-positive output
// within ±(hostCacheTTLJitterFraction/2) of the base. Catches potential
// underflow or sign-flip bugs at extreme magnitudes that a single fixed
// base cannot.
func TestPBT_JitteredHostCacheTTLBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseNanos := rapid.Int64Range(int64(time.Nanosecond), int64(time.Hour)).Draw(t, "base")
		ds := &Datastore{hostCacheEnabled: true, hostCacheTTL: time.Duration(baseNanos)}

		got := ds.jitteredHostCacheTTL()

		base := float64(ds.hostCacheTTL)
		halfJitter := base * hostCacheTTLJitterFraction / 2
		minAllowed := time.Duration(base - halfJitter)
		maxAllowed := time.Duration(base + halfJitter)

		require.GreaterOrEqualf(t, got, minAllowed, "below jitter floor for base=%v: got %v", ds.hostCacheTTL, got)
		require.LessOrEqualf(t, got, maxAllowed, "above jitter ceiling for base=%v: got %v", ds.hostCacheTTL, got)
		require.Greaterf(t, got, time.Duration(0), "non-positive jitter result for base=%v", ds.hostCacheTTL)
	})
}
