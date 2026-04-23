package mysqlredis

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newOrbitMockStore builds a mock.DataStore (not mock.Store — see the osquery
// counterpart's comment) whose LoadHostByOrbitNodeKeyFunc returns a host with
// the supplied orbit node key and orbit-specific fields populated, matching
// what the real LoadHostByOrbitNodeKey returns.
func newOrbitMockStore() *mock.DataStore {
	ds := new(mock.DataStore)
	ds.LoadHostByOrbitNodeKeyFunc = func(_ context.Context, onk string) (*fleet.Host, error) {
		nk := onk
		depTrue := true
		encTrue := true
		team := "team-loadtest"
		return &fleet.Host{
			ID:                    1,
			NodeKey:               &nk,
			OrbitNodeKey:          &onk,
			Hostname:              "h-" + onk,
			Platform:              "windows",
			DEPAssignedToFleet:    &depTrue,
			DiskEncryptionEnabled: &encTrue,
			TeamName:              &team,
			MDM: fleet.MDMHostData{
				EncryptionKeyAvailable: true,
			},
		}, nil
	}
	return ds
}

func TestLoadHostByOrbitNodeKey_CacheDisabled(t *testing.T) {
	ctx := t.Context()
	ds := newOrbitMockStore()
	wrapped := New(ds, redistest.NopRedis()) // no WithHostCache

	got, err := wrapped.LoadHostByOrbitNodeKey(ctx, "onk-a")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, ds.LoadHostByOrbitNodeKeyFuncInvoked)

	ds.LoadHostByOrbitNodeKeyFuncInvoked = false
	_, err = wrapped.LoadHostByOrbitNodeKey(ctx, "onk-a")
	require.NoError(t, err)
	assert.True(t, ds.LoadHostByOrbitNodeKeyFuncInvoked, "cache disabled: every call must go to DB")
}

func TestLoadHostByOrbitNodeKey_Override(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		t.Run("cache miss then hit preserves orbit-specific fields", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ctx := t.Context()
			ds := newOrbitMockStore()
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			first, err := wrapped.LoadHostByOrbitNodeKey(ctx, "onk-miss-hit")
			require.NoError(t, err)
			require.NotNil(t, first)
			require.True(t, ds.LoadHostByOrbitNodeKeyFuncInvoked)

			ds.LoadHostByOrbitNodeKeyFuncInvoked = false
			second, err := wrapped.LoadHostByOrbitNodeKey(ctx, "onk-miss-hit")
			require.NoError(t, err)
			require.NotNil(t, second)
			assert.False(t, ds.LoadHostByOrbitNodeKeyFuncInvoked, "second call should be served from cache")

			// Orbit-specific fields must round-trip (these are why we have a
			// dedicated orbit entry type vs. sharing hostCacheEntry).
			require.NotNil(t, second.DEPAssignedToFleet)
			assert.True(t, *second.DEPAssignedToFleet)
			require.NotNil(t, second.DiskEncryptionEnabled)
			assert.True(t, *second.DiskEncryptionEnabled)
			require.NotNil(t, second.TeamName)
			assert.Equal(t, "team-loadtest", *second.TeamName)
			assert.True(t, second.MDM.EncryptionKeyAvailable)
		})

		t.Run("NotFound populates negative cache", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ctx := t.Context()
			ds := new(mock.DataStore)
			var callCount atomic.Int32
			ds.LoadHostByOrbitNodeKeyFunc = func(_ context.Context, _ string) (*fleet.Host, error) {
				callCount.Add(1)
				return nil, common_mysql.NotFound("Host")
			}
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			_, err := wrapped.LoadHostByOrbitNodeKey(ctx, "onk-absent")
			require.Error(t, err)
			assert.True(t, fleet.IsNotFound(err))
			assert.Equal(t, int32(1), callCount.Load())

			_, err = wrapped.LoadHostByOrbitNodeKey(ctx, "onk-absent")
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
			ds.LoadHostByOrbitNodeKeyFunc = func(_ context.Context, _ string) (*fleet.Host, error) {
				callCount.Add(1)
				return nil, transient
			}
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			_, err := wrapped.LoadHostByOrbitNodeKey(ctx, "onk-transient")
			require.ErrorIs(t, err, transient)
			_, err = wrapped.LoadHostByOrbitNodeKey(ctx, "onk-transient")
			require.ErrorIs(t, err, transient)
			assert.Equal(t, int32(2), callCount.Load(), "transient errors must not poison the cache")
		})

		t.Run("singleflight collapses concurrent misses", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			ds := new(mock.DataStore)
			var callCount atomic.Int32
			release := make(chan struct{})
			ds.LoadHostByOrbitNodeKeyFunc = func(_ context.Context, onk string) (*fleet.Host, error) {
				callCount.Add(1)
				<-release
				k := onk
				return &fleet.Host{ID: 42, OrbitNodeKey: &k}, nil
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
					h, err := wrapped.LoadHostByOrbitNodeKey(t.Context(), "onk-sf")
					hosts[i] = h
					errs[i] = err
				}(i)
			}
			time.Sleep(50 * time.Millisecond)
			close(release)
			wg.Wait()

			assert.Equal(t, int32(1), callCount.Load(), "singleflight should collapse to one DB call")
			for i := 1; i < goroutines; i++ {
				assert.NotSame(t, hosts[0], hosts[i], "callers must receive independent structs")
			}
		})

		t.Run("canceled caller does not poison the shared DB call", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })

			ds := new(mock.DataStore)
			release := make(chan struct{})
			ds.LoadHostByOrbitNodeKeyFunc = func(innerCtx context.Context, onk string) (*fleet.Host, error) {
				<-release
				if err := innerCtx.Err(); err != nil {
					return nil, err
				}
				k := onk
				return &fleet.Host{ID: 77, OrbitNodeKey: &k}, nil
			}
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			cancellableCtx, cancel := context.WithCancel(t.Context())
			var joinerHost *fleet.Host
			var joinerErr error

			// Start the CANCELLABLE caller first so it's guaranteed to be the
			// flight leader; otherwise the healthy joiner could accidentally
			// become leader and the test would pass without exercising the
			// cancelled-leader case.
			leaderDone := make(chan struct{})
			go func() {
				_, _ = wrapped.LoadHostByOrbitNodeKey(cancellableCtx, "onk-cancel")
				close(leaderDone)
			}()
			time.Sleep(50 * time.Millisecond)

			joinerDone := make(chan struct{})
			go func() {
				joinerHost, joinerErr = wrapped.LoadHostByOrbitNodeKey(t.Context(), "onk-cancel")
				close(joinerDone)
			}()
			time.Sleep(20 * time.Millisecond)
			cancel()
			close(release)
			<-joinerDone
			<-leaderDone

			require.NoError(t, joinerErr, "joiner must not see the leader's cancellation")
			require.NotNil(t, joinerHost)
			assert.Equal(t, uint(77), joinerHost.ID)
		})

		t.Run("invalidation by host ID clears both osquery AND orbit entries", func(t *testing.T) {
			// Proves the unified-invalidation design: a write that goes
			// through hostCacheDeleteByID clears both cache families for
			// hosts that have both agents enrolled.
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ctx := t.Context()
			ds := new(mock.DataStore)
			wrapped := New(ds, pool, WithHostCache(30*time.Second))

			nk := "nk-both"
			onk := "onk-both"
			host := &fleet.Host{ID: 99, NodeKey: &nk, OrbitNodeKey: &onk, Hostname: "both"}
			wrapped.hostCachePut(ctx, host)
			wrapped.hostCachePutByOrbit(ctx, host)

			// Sanity: both hot.
			_, res := wrapped.hostCacheGet(ctx, nk)
			require.Equal(t, hostCacheLookupHit, res)
			_, res = wrapped.hostCacheGetByOrbitNodeKey(ctx, onk)
			require.Equal(t, hostCacheLookupHit, res)

			wrapped.hostCacheDeleteByID(ctx, 99, "update")

			_, res = wrapped.hostCacheGet(ctx, nk)
			assert.Equal(t, hostCacheLookupMiss, res)
			_, res = wrapped.hostCacheGetByOrbitNodeKey(ctx, onk)
			assert.Equal(t, hostCacheLookupMiss, res)
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

// TestOrbitHostCacheEntryRoundTrip is the orbit counterpart of
// TestHostCacheEntryRoundTrip: every field LoadHostByOrbitNodeKey returns
// is populated with a non-zero value, round-tripped through JSON, and
// asserted equal on the way back.
func TestOrbitHostCacheEntryRoundTrip(t *testing.T) {
	nk := "node-rt-orbit"
	onk := "orbit-rt"
	oqhid := "osq-rt"
	teamID := uint(3)
	primaryIPID := uint(99)
	certTrue := true
	depTrue := true
	encEnabledTrue := true
	teamName := "team-rt"
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
		DEPAssignedToFleet:          &depTrue,
		DiskEncryptionEnabled:       &encEnabledTrue,
		TeamName:                    &teamName,
		HasHostIdentityCert:         &certTrue,
		MDM:                         fleet.MDMHostData{EncryptionKeyAvailable: true},
	}
	orig.CreatedAt = now
	orig.UpdatedAt = now

	got := orbitHostCacheEntryFromHost(orig).toHost()

	// Orbit-specific fields
	require.NotNil(t, got.DEPAssignedToFleet)
	assert.True(t, *got.DEPAssignedToFleet)
	require.NotNil(t, got.DiskEncryptionEnabled)
	assert.True(t, *got.DiskEncryptionEnabled)
	require.NotNil(t, got.TeamName)
	assert.Equal(t, teamName, *got.TeamName)
	assert.True(t, got.MDM.EncryptionKeyAvailable)
	require.NotNil(t, got.HasHostIdentityCert)
	assert.True(t, *got.HasHostIdentityCert)

	// Auth-critical json:"-" fields
	require.NotNil(t, got.NodeKey)
	assert.Equal(t, nk, *got.NodeKey)
	require.NotNil(t, got.OrbitNodeKey)
	assert.Equal(t, onk, *got.OrbitNodeKey)
	require.NotNil(t, got.OsqueryHostID)
	assert.Equal(t, oqhid, *got.OsqueryHostID)

	// Spot-check a representative scalar to catch drift if the field set
	// diverges silently.
	assert.Equal(t, uint(42), got.ID)
	assert.Equal(t, "h", got.Hostname)
	assert.Equal(t, "darwin", got.Platform)
	assert.Equal(t, now, got.DetailUpdatedAt)
	assert.Equal(t, now, got.CreatedAt)
	require.NotNil(t, got.TeamID)
	assert.Equal(t, teamID, *got.TeamID)

	// Also verify JSON layer specifically, since that's what Redis sees.
	raw, err := json.Marshal(orbitHostCacheEntryFromHost(orig))
	require.NoError(t, err)
	viaJSON := new(orbitHostCacheEntry)
	require.NoError(t, json.Unmarshal(raw, viaJSON))
	assert.Equal(t, orbitHostCacheEntryFromHost(orig), viaJSON)
}

func TestLoadHostByOrbitNodeKey_RedisErrorFallsThrough(t *testing.T) {
	ctx := t.Context()
	ds := newOrbitMockStore()
	wrapped := New(ds, errPool{}, WithHostCache(30*time.Second))

	got, err := wrapped.LoadHostByOrbitNodeKey(ctx, "onk-redis-down")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, ds.LoadHostByOrbitNodeKeyFuncInvoked)

	ds.LoadHostByOrbitNodeKeyFuncInvoked = false
	_, err = wrapped.LoadHostByOrbitNodeKey(ctx, "onk-redis-down")
	require.NoError(t, err)
	assert.True(t, ds.LoadHostByOrbitNodeKeyFuncInvoked, "Redis errors must not prevent DB fallthrough")
}
