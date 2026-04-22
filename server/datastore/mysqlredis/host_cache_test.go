package mysqlredis

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hostCacheTestCleanupPrefix is the key-prefix passed to redistest.SetupRedis so
// every run cleans up only keys owned by these tests (redistest requires a
// prefix to prevent concurrent tests from clobbering each other's keys).
const hostCacheTestCleanupPrefix = "fleet:hostcache:v1"

func TestHostCacheDisabled(t *testing.T) {
	// Sanity: without WithHostCache, every helper is a no-op. We route through
	// NopRedis so any accidental Redis call would silently succeed with zero
	// values, which the helpers must still handle without surprising the
	// caller.
	ctx := t.Context()
	ds := new(mock.Store)
	wrapped := New(ds, redistest.NopRedis())

	host, result := wrapped.hostCacheGet(ctx, "node-abc")
	require.Nil(t, host)
	require.Equal(t, hostCacheLookupMiss, result)

	// No panics on Put/PutNotFound/Delete helpers when disabled.
	nk := "node-abc"
	wrapped.hostCachePut(ctx, &fleet.Host{ID: 1, NodeKey: &nk})
	wrapped.hostCachePutNotFound(ctx, "node-missing")
	wrapped.hostCacheDeleteByNodeKey(ctx, "node-abc", 1, "update")
	wrapped.hostCacheDeleteByID(ctx, 1, "update")
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
			wrapped.hostCachePut(ctx, stored)

			loaded, result := wrapped.hostCacheGet(ctx, nk)
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

		t.Run("put returned *Host is independent of cache", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			nk := "node-independent"
			wrapped.hostCachePut(ctx, &fleet.Host{ID: 8, NodeKey: &nk, Hostname: "original"})

			loaded, result := wrapped.hostCacheGet(ctx, nk)
			require.Equal(t, hostCacheLookupHit, result)
			loaded.Hostname = "mutated-by-caller"

			// A second read must return the stored value, unaffected by the mutation.
			second, result := wrapped.hostCacheGet(ctx, nk)
			require.Equal(t, hostCacheLookupHit, result)
			assert.Equal(t, "original", second.Hostname)
		})

		t.Run("miss returns hostCacheLookupMiss", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			host, result := wrapped.hostCacheGet(ctx, "node-never-cached")
			assert.Nil(t, host)
			assert.Equal(t, hostCacheLookupMiss, result)
		})

		t.Run("negative cache returns hostCacheLookupNegative", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			wrapped.hostCachePutNotFound(ctx, "node-missing")
			host, result := wrapped.hostCacheGet(ctx, "node-missing")
			assert.Nil(t, host)
			assert.Equal(t, hostCacheLookupNegative, result)
		})

		t.Run("positive cache takes precedence over negative", func(t *testing.T) {
			// A future write could simultaneously populate positive and leave a
			// stale negative; we must not surface notFound in that case.
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			nk := "node-both"
			wrapped.hostCachePutNotFound(ctx, nk)
			wrapped.hostCachePut(ctx, &fleet.Host{ID: 9, NodeKey: &nk, Hostname: "wins"})

			loaded, result := wrapped.hostCacheGet(ctx, nk)
			require.Equal(t, hostCacheLookupHit, result)
			assert.Equal(t, "wins", loaded.Hostname)
		})

		t.Run("delete by node_key clears primary, negative, and index", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			nk := "node-del-nk"
			wrapped.hostCachePut(ctx, &fleet.Host{ID: 10, NodeKey: &nk})
			wrapped.hostCachePutNotFound(ctx, nk)

			wrapped.hostCacheDeleteByNodeKey(ctx, nk, 10, "update")

			_, result := wrapped.hostCacheGet(ctx, nk)
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
			wrapped.hostCachePut(ctx, &fleet.Host{ID: 11, NodeKey: &nk})

			wrapped.hostCacheDeleteByID(ctx, 11, "team")

			_, result := wrapped.hostCacheGet(ctx, nk)
			assert.Equal(t, hostCacheLookupMiss, result)

			conn := redis.ConfigureDoer(pool, pool.Get())
			defer conn.Close()
			exists, err := redigo.Bool(conn.Do("EXISTS", hostCacheIndexByID(11)))
			require.NoError(t, err)
			assert.False(t, exists)
		})

		t.Run("delete by id with no index is a no-op", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			// Nothing cached. Should not error, should not panic.
			wrapped.hostCacheDeleteByID(ctx, 999, "delete")
		})

		t.Run("put with nil node_key is a no-op", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			wrapped.hostCachePut(ctx, &fleet.Host{ID: 12, NodeKey: nil})

			// No primary key should have been created; pick a sentinel key and
			// verify the space is clean.
			conn := redis.ConfigureDoer(pool, pool.Get())
			defer conn.Close()
			exists, err := redigo.Bool(conn.Do("EXISTS", hostCacheIndexByID(12)))
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

// TestHostCacheEntryRoundTrip is a drift-catcher: it populates every field this
// cache covers with a non-zero value and asserts round-trip equivalence through
// JSON. If LoadHostByNodeKey's SELECT list gains a column and the field lands
// on fleet.Host but isn't added to hostCacheEntry + the conversion funcs,
// callers would silently lose that field in the cache path. This test is
// intentionally explicit (not reflect-based) so a failing diff points at the
// missing field.
func TestHostCacheEntryRoundTrip(t *testing.T) {
	nk := "node-rt"
	onk := "orbit-rt"
	oqhid := "osq-rt"
	tz := "UTC"
	teamID := uint(3)
	primaryIPID := uint(99)
	certTrue := true
	until := time.Date(2026, time.April, 22, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, time.April, 21, 12, 0, 0, 0, time.UTC)

	orig := &fleet.Host{
		ID:                          42,
		OsqueryHostID:               &oqhid,
		DetailUpdatedAt:             now,
		NodeKey:                     &nk,
		Hostname:                    "h",
		UUID:                        "u",
		Platform:                    "darwin",
		OsqueryVersion:              "5.0",
		OSVersion:                   "14.0",
		Build:                       "23A344",
		PlatformLike:                "darwin",
		CodeName:                    "sonoma",
		Uptime:                      time.Hour,
		Memory:                      1 << 30,
		CPUType:                     "arm64",
		CPUSubtype:                  "m1",
		CPUBrand:                    "Apple",
		CPUPhysicalCores:            8,
		CPULogicalCores:             8,
		HardwareVendor:              "Apple",
		HardwareModel:               "MacBookPro",
		HardwareVersion:             "v1",
		HardwareSerial:              "SN123",
		ComputerName:                "Laptop",
		PrimaryNetworkInterfaceID:   &primaryIPID,
		DistributedInterval:         10,
		LoggerTLSPeriod:             60,
		ConfigTLSRefresh:            60,
		PrimaryIP:                   "10.0.0.1",
		PrimaryMac:                  "aa:bb:cc:dd:ee:ff",
		LabelUpdatedAt:              now,
		LastEnrolledAt:              now,
		RefetchRequested:            true,
		RefetchCriticalQueriesUntil: &until,
		TeamID:                      &teamID,
		PolicyUpdatedAt:             now,
		PublicIP:                    "1.2.3.4",
		OrbitNodeKey:                &onk,
		LastRestartedAt:             now,
		TimeZone:                    &tz,
		GigsDiskSpaceAvailable:      100.5,
		GigsTotalDiskSpace:          500.0,
		PercentDiskSpaceAvailable:   20.1,
		HasHostIdentityCert:         &certTrue,
	}
	orig.CreatedAt = now
	orig.UpdatedAt = now

	got := hostCacheEntryFromHost(orig).toHost()

	// Scalar fields
	assert.Equal(t, orig.ID, got.ID)
	assert.Equal(t, orig.Hostname, got.Hostname)
	assert.Equal(t, orig.UUID, got.UUID)
	assert.Equal(t, orig.Platform, got.Platform)
	assert.Equal(t, orig.OsqueryVersion, got.OsqueryVersion)
	assert.Equal(t, orig.OSVersion, got.OSVersion)
	assert.Equal(t, orig.Build, got.Build)
	assert.Equal(t, orig.PlatformLike, got.PlatformLike)
	assert.Equal(t, orig.CodeName, got.CodeName)
	assert.Equal(t, orig.Uptime, got.Uptime)
	assert.Equal(t, orig.Memory, got.Memory)
	assert.Equal(t, orig.CPUType, got.CPUType)
	assert.Equal(t, orig.CPUSubtype, got.CPUSubtype)
	assert.Equal(t, orig.CPUBrand, got.CPUBrand)
	assert.Equal(t, orig.CPUPhysicalCores, got.CPUPhysicalCores)
	assert.Equal(t, orig.CPULogicalCores, got.CPULogicalCores)
	assert.Equal(t, orig.HardwareVendor, got.HardwareVendor)
	assert.Equal(t, orig.HardwareModel, got.HardwareModel)
	assert.Equal(t, orig.HardwareVersion, got.HardwareVersion)
	assert.Equal(t, orig.HardwareSerial, got.HardwareSerial)
	assert.Equal(t, orig.ComputerName, got.ComputerName)
	assert.Equal(t, orig.DistributedInterval, got.DistributedInterval)
	assert.Equal(t, orig.LoggerTLSPeriod, got.LoggerTLSPeriod)
	assert.Equal(t, orig.ConfigTLSRefresh, got.ConfigTLSRefresh)
	assert.Equal(t, orig.PrimaryIP, got.PrimaryIP)
	assert.Equal(t, orig.PrimaryMac, got.PrimaryMac)
	assert.Equal(t, orig.PublicIP, got.PublicIP)
	assert.Equal(t, orig.RefetchRequested, got.RefetchRequested)
	assert.InDelta(t, orig.GigsDiskSpaceAvailable, got.GigsDiskSpaceAvailable, 0)
	assert.InDelta(t, orig.GigsTotalDiskSpace, got.GigsTotalDiskSpace, 0)
	assert.InDelta(t, orig.PercentDiskSpaceAvailable, got.PercentDiskSpaceAvailable, 0)
	assert.Equal(t, orig.CreatedAt, got.CreatedAt)
	assert.Equal(t, orig.UpdatedAt, got.UpdatedAt)
	assert.Equal(t, orig.DetailUpdatedAt, got.DetailUpdatedAt)
	assert.Equal(t, orig.LabelUpdatedAt, got.LabelUpdatedAt)
	assert.Equal(t, orig.LastEnrolledAt, got.LastEnrolledAt)
	assert.Equal(t, orig.PolicyUpdatedAt, got.PolicyUpdatedAt)
	assert.Equal(t, orig.LastRestartedAt, got.LastRestartedAt)

	// Pointer fields — the security-critical ones
	require.NotNil(t, got.NodeKey)
	assert.Equal(t, *orig.NodeKey, *got.NodeKey)
	require.NotNil(t, got.OrbitNodeKey)
	assert.Equal(t, *orig.OrbitNodeKey, *got.OrbitNodeKey)
	require.NotNil(t, got.OsqueryHostID)
	assert.Equal(t, *orig.OsqueryHostID, *got.OsqueryHostID)
	require.NotNil(t, got.HasHostIdentityCert)
	assert.Equal(t, *orig.HasHostIdentityCert, *got.HasHostIdentityCert)
	require.NotNil(t, got.TeamID)
	assert.Equal(t, *orig.TeamID, *got.TeamID)
	require.NotNil(t, got.RefetchCriticalQueriesUntil)
	assert.Equal(t, *orig.RefetchCriticalQueriesUntil, *got.RefetchCriticalQueriesUntil)
	require.NotNil(t, got.TimeZone)
	assert.Equal(t, *orig.TimeZone, *got.TimeZone)
	require.NotNil(t, got.PrimaryNetworkInterfaceID)
	assert.Equal(t, *orig.PrimaryNetworkInterfaceID, *got.PrimaryNetworkInterfaceID)

	// Also verify JSON layer specifically, since that's what Redis actually sees.
	viaJSON := new(hostCacheEntry)
	raw, err := json.Marshal(hostCacheEntryFromHost(orig))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, viaJSON))
	assert.Equal(t, hostCacheEntryFromHost(orig), viaJSON)
}

func TestJitteredHostCacheTTL(t *testing.T) {
	d := &Datastore{hostCacheEnabled: true, hostCacheTTL: 30 * time.Second}

	base := float64(d.hostCacheTTL)
	halfJitter := base * hostCacheTTLJitterFraction / 2
	minAllowed := time.Duration(base - halfJitter)
	maxAllowed := time.Duration(base + halfJitter)

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
		assert.GreaterOrEqual(t, got, minAllowed, "jittered TTL below ±10% bound")
		assert.LessOrEqual(t, got, maxAllowed, "jittered TTL above ±10% bound")
	}
	// Sanity: we should see a meaningful spread, not just 1000 identical values.
	assert.Less(t, minSeen, maxSeen, "jitter produced no variance over %d samples", samples)

	// Disabled / zero base returns zero.
	zero := &Datastore{hostCacheEnabled: true, hostCacheTTL: 0}
	assert.Equal(t, time.Duration(0), zero.jitteredHostCacheTTL())
}

// newMockLoadHostStore builds a mock.Store with a default LoadHostByNodeKeyFunc
// that returns a fresh host whose NodeKey matches the queried node_key. Tests
// can override the Func field to simulate errors, NotFound, or latency.
func newMockLoadHostStore() *mock.Store {
	ds := new(mock.Store)
	ds.LoadHostByNodeKeyFunc = func(_ context.Context, nodeKey string) (*fleet.Host, error) {
		nk := nodeKey
		return &fleet.Host{
			ID:       1,
			NodeKey:  &nk,
			Hostname: "h-" + nodeKey,
		}, nil
	}
	return ds
}

func TestLoadHostByNodeKey_CacheDisabled(t *testing.T) {
	ctx := t.Context()
	ds := newMockLoadHostStore()
	wrapped := New(ds, redistest.NopRedis()) // no WithHostCache

	got, err := wrapped.LoadHostByNodeKey(ctx, "nk-a")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, ds.LoadHostByNodeKeyFuncInvoked)

	// Second call also hits the inner datastore — there's no cache.
	ds.LoadHostByNodeKeyFuncInvoked = false
	_, err = wrapped.LoadHostByNodeKey(ctx, "nk-a")
	require.NoError(t, err)
	assert.True(t, ds.LoadHostByNodeKeyFuncInvoked)
}

func TestLoadHostByNodeKey_Override(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		t.Run("cache miss then hit", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ctx := t.Context()
			ds := newMockLoadHostStore()
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			first, err := wrapped.LoadHostByNodeKey(ctx, "nk-miss-hit")
			require.NoError(t, err)
			require.NotNil(t, first)
			require.True(t, ds.LoadHostByNodeKeyFuncInvoked)

			ds.LoadHostByNodeKeyFuncInvoked = false
			second, err := wrapped.LoadHostByNodeKey(ctx, "nk-miss-hit")
			require.NoError(t, err)
			require.NotNil(t, second)
			assert.False(t, ds.LoadHostByNodeKeyFuncInvoked, "second call should be served from cache")
			assert.Equal(t, first.ID, second.ID, "cached value should match the initial DB read")
		})

		t.Run("bypass context always hits DB and skips cache populate", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ctx := ctxdb.BypassHostCache(t.Context(), true)
			ds := newMockLoadHostStore()
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			_, err := wrapped.LoadHostByNodeKey(ctx, "nk-bypass")
			require.NoError(t, err)
			require.True(t, ds.LoadHostByNodeKeyFuncInvoked)

			// Non-bypass follow-up must see an empty cache and hit DB again.
			ds.LoadHostByNodeKeyFuncInvoked = false
			_, err = wrapped.LoadHostByNodeKey(t.Context(), "nk-bypass")
			require.NoError(t, err)
			assert.True(t, ds.LoadHostByNodeKeyFuncInvoked, "bypass must not populate cache")
		})

		t.Run("NotFound populates negative cache", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ctx := t.Context()
			ds := new(mock.Store)
			var callCount atomic.Int32
			ds.LoadHostByNodeKeyFunc = func(_ context.Context, _ string) (*fleet.Host, error) {
				callCount.Add(1)
				return nil, common_mysql.NotFound("Host")
			}
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			_, err := wrapped.LoadHostByNodeKey(ctx, "nk-absent")
			require.Error(t, err)
			assert.True(t, fleet.IsNotFound(err))
			assert.Equal(t, int32(1), callCount.Load())

			// Second call hits the negative cache; inner is not invoked.
			_, err = wrapped.LoadHostByNodeKey(ctx, "nk-absent")
			require.Error(t, err)
			assert.True(t, fleet.IsNotFound(err))
			assert.Equal(t, int32(1), callCount.Load(), "negative cache should have served the second call")
		})

		t.Run("transient errors are not cached", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ctx := t.Context()
			ds := new(mock.Store)
			transient := errors.New("simulated timeout")
			var callCount atomic.Int32
			ds.LoadHostByNodeKeyFunc = func(_ context.Context, _ string) (*fleet.Host, error) {
				callCount.Add(1)
				return nil, transient
			}
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			_, err := wrapped.LoadHostByNodeKey(ctx, "nk-transient")
			require.ErrorIs(t, err, transient)

			_, err = wrapped.LoadHostByNodeKey(ctx, "nk-transient")
			require.ErrorIs(t, err, transient)
			assert.Equal(t, int32(2), callCount.Load(), "transient errors must not poison the cache")
		})

		t.Run("singleflight collapses concurrent misses", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			// Block inner LoadHostByNodeKey until the test releases it, so all
			// goroutines pile up in the singleflight group before it resolves.
			ds := new(mock.Store)
			var callCount atomic.Int32
			release := make(chan struct{})
			ds.LoadHostByNodeKeyFunc = func(_ context.Context, nodeKey string) (*fleet.Host, error) {
				callCount.Add(1)
				<-release
				nk := nodeKey
				return &fleet.Host{ID: 42, NodeKey: &nk}, nil
			}
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			const goroutines = 20
			var wg sync.WaitGroup
			errs := make([]error, goroutines)
			hosts := make([]*fleet.Host, goroutines)
			for i := range goroutines {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					h, err := wrapped.LoadHostByNodeKey(t.Context(), "nk-sf")
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
			// Regression test for the PR review finding: without ctx detach,
			// cancelling the caller whose goroutine happens to be the
			// singleflight leader would cancel the shared DB query and fail
			// every joiner even though their own contexts are alive.
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			ds := new(mock.Store)
			release := make(chan struct{})
			ds.LoadHostByNodeKeyFunc = func(innerCtx context.Context, nodeKey string) (*fleet.Host, error) {
				<-release
				// The inner ctx should not observe the canceling caller's
				// Done signal. If it does, this returns ctx.Err() and the
				// joiners fail.
				if err := innerCtx.Err(); err != nil {
					return nil, err
				}
				nk := nodeKey
				return &fleet.Host{ID: 77, NodeKey: &nk}, nil
			}
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
				_, _ = wrapped.LoadHostByNodeKey(cancellableCtx, "nk-cancel")
				close(leaderDone)
			}()
			time.Sleep(50 * time.Millisecond)

			joinerDone := make(chan struct{})
			go func() {
				joinerHost, joinerErr = wrapped.LoadHostByNodeKey(joinerCtx, "nk-cancel")
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

func TestLoadHostByNodeKey_RedisErrorFallsThrough(t *testing.T) {
	ctx := t.Context()
	ds := newMockLoadHostStore()
	wrapped := New(ds, errPool{}, WithHostCache(30*time.Second))

	// Redis ops all error; the caller must still get a host via the DB path.
	got, err := wrapped.LoadHostByNodeKey(ctx, "nk-redis-down")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, ds.LoadHostByNodeKeyFuncInvoked)

	// Second call also hits DB — cache never populated successfully.
	ds.LoadHostByNodeKeyFuncInvoked = false
	_, err = wrapped.LoadHostByNodeKey(ctx, "nk-redis-down")
	require.NoError(t, err)
	assert.True(t, ds.LoadHostByNodeKeyFuncInvoked, "Redis errors must not prevent DB fallthrough")
}

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
