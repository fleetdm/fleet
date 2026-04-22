package mysqlredis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// primeCachedHost populates the cache with a fleet.Host for nk/id and asserts
// the cache is hot before returning. Tests use this to set up a known-good
// state, then invoke a write method and verify the cache is cleared.
func primeCachedHost(t *testing.T, d *Datastore, id uint, nk string) {
	t.Helper()
	ctx := t.Context()
	d.hostCachePut(ctx, &fleet.Host{ID: id, NodeKey: &nk, Hostname: "primed-" + nk})
	_, result := d.hostCacheGet(ctx, nk)
	require.Equal(t, hostCacheLookupHit, result, "primeCachedHost failed to prime %q", nk)
}

func requireCacheMiss(t *testing.T, d *Datastore, nk string) {
	t.Helper()
	_, result := d.hostCacheGet(t.Context(), nk)
	require.Equal(t, hostCacheLookupMiss, result, "expected miss for %q, got %v", nk, result)
}

func TestWritePathInvalidation(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		ctx := t.Context()

		t.Run("UpdateHost invalidates by node_key", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			ds.UpdateHostFunc = func(_ context.Context, _ *fleet.Host) error { return nil }
			d := New(ds, pool, WithHostCache(30*time.Second))

			nk := "nk-update"
			primeCachedHost(t, d, 1, nk)
			require.NoError(t, d.UpdateHost(ctx, &fleet.Host{ID: 1, NodeKey: &nk}))
			require.True(t, ds.UpdateHostFuncInvoked)
			requireCacheMiss(t, d, nk)
		})

		t.Run("SerialUpdateHost invalidates by node_key", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			ds.SerialUpdateHostFunc = func(_ context.Context, _ *fleet.Host) error { return nil }
			d := New(ds, pool, WithHostCache(30*time.Second))

			nk := "nk-serial"
			primeCachedHost(t, d, 2, nk)
			require.NoError(t, d.SerialUpdateHost(ctx, &fleet.Host{ID: 2, NodeKey: &nk}))
			require.True(t, ds.SerialUpdateHostFuncInvoked)
			requireCacheMiss(t, d, nk)
		})

		t.Run("UpdateHostOsqueryIntervals invalidates by id", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			ds.UpdateHostOsqueryIntervalsFunc = func(_ context.Context, _ uint, _ fleet.HostOsqueryIntervals) error { return nil }
			d := New(ds, pool, WithHostCache(30*time.Second))

			nk := "nk-intervals"
			primeCachedHost(t, d, 3, nk)
			require.NoError(t, d.UpdateHostOsqueryIntervals(ctx, 3, fleet.HostOsqueryIntervals{}))
			require.True(t, ds.UpdateHostOsqueryIntervalsFuncInvoked)
			requireCacheMiss(t, d, nk)
		})

		t.Run("UpdateHostRefetchRequested invalidates by id", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			ds.UpdateHostRefetchRequestedFunc = func(_ context.Context, _ uint, _ bool) error { return nil }
			d := New(ds, pool, WithHostCache(30*time.Second))

			nk := "nk-refetch"
			primeCachedHost(t, d, 4, nk)
			require.NoError(t, d.UpdateHostRefetchRequested(ctx, 4, true))
			require.True(t, ds.UpdateHostRefetchRequestedFuncInvoked)
			requireCacheMiss(t, d, nk)
		})

		t.Run("UpdateHostRefetchCriticalQueriesUntil invalidates by id", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			ds.UpdateHostRefetchCriticalQueriesUntilFunc = func(_ context.Context, _ uint, _ *time.Time) error { return nil }
			d := New(ds, pool, WithHostCache(30*time.Second))

			nk := "nk-critical"
			primeCachedHost(t, d, 5, nk)
			until := time.Now().Add(time.Hour)
			require.NoError(t, d.UpdateHostRefetchCriticalQueriesUntil(ctx, 5, &until))
			require.True(t, ds.UpdateHostRefetchCriticalQueriesUntilFuncInvoked)
			requireCacheMiss(t, d, nk)
		})

		t.Run("EnrollOrbit invalidates for returned host", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			// mock.Store has a hand-written EnrollOrbit that returns (nil, nil)
			// regardless of EnrollOrbitFunc, so for this subtest we wrap the
			// auto-generated mock.DataStore directly.
			//
			// The mock returns a Host with ONLY ID populated — mirroring the
			// production mysql.EnrollOrbit, which doesn't set NodeKey or
			// OrbitNodeKey on its returned struct. This exercises the ID-based
			// reverse-index invalidation path that production actually takes.
			ds := new(mock.DataStore)
			ds.EnrollOrbitFunc = func(_ context.Context, _ ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
				return &fleet.Host{ID: 6}, nil
			}
			d := New(ds, pool, WithHostCache(30*time.Second))

			// Prime the cache under a node_key and verify EnrollOrbit clears
			// it via the reverse index (id2nk → nk), even though the returned
			// Host struct doesn't carry the NodeKey field.
			nk := "nk-orbit"
			primeCachedHost(t, d, 6, nk)
			h, err := d.EnrollOrbit(ctx)
			require.NoError(t, err)
			require.NotNil(t, h)
			requireCacheMiss(t, d, nk)
		})

		t.Run("AddHostsToTeam invalidates every host in the batch", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			ds.AddHostsToTeamFunc = func(_ context.Context, _ *fleet.AddHostsToTeamParams) error { return nil }
			d := New(ds, pool, WithHostCache(30*time.Second))

			nks := []string{"nk-team-a", "nk-team-b", "nk-team-c"}
			ids := []uint{10, 11, 12}
			for i, nk := range nks {
				primeCachedHost(t, d, ids[i], nk)
			}

			teamID := uint(7)
			params := fleet.NewAddHostsToTeamParams(&teamID, ids)
			require.NoError(t, d.AddHostsToTeam(ctx, params))
			require.True(t, ds.AddHostsToTeamFuncInvoked)
			for _, nk := range nks {
				requireCacheMiss(t, d, nk)
			}
		})

		t.Run("UpdateHostIdentityCertHostIDBySerial invalidates by host id", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			ds.UpdateHostIdentityCertHostIDBySerialFunc = func(_ context.Context, _ uint64, _ uint) error { return nil }
			d := New(ds, pool, WithHostCache(30*time.Second))

			nk := "nk-cert"
			primeCachedHost(t, d, 20, nk)
			require.NoError(t, d.UpdateHostIdentityCertHostIDBySerial(ctx, 12345, 20))
			require.True(t, ds.UpdateHostIdentityCertHostIDBySerialFuncInvoked)
			requireCacheMiss(t, d, nk)
		})

		t.Run("NewHost clears stale negative cache for new node_key", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			nk := "nk-new-with-neg"
			ds := new(mock.Store)
			ds.NewHostFunc = func(_ context.Context, h *fleet.Host) (*fleet.Host, error) {
				h.ID = 30
				return h, nil
			}
			d := New(ds, pool, WithHostCache(30*time.Second))

			// Someone probed this node_key before enrollment and we cached notFound.
			d.hostCachePutNotFound(ctx, nk)

			h, err := d.NewHost(ctx, &fleet.Host{NodeKey: &nk})
			require.NoError(t, err)
			require.NotNil(t, h)
			requireCacheMiss(t, d, nk) // negative cache must be gone
		})

		t.Run("EnrollOsquery invalidates for returned host on re-enroll", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			nk := "nk-osq-reenroll"
			ds := new(mock.Store)
			ds.EnrollOsqueryFunc = func(_ context.Context, _ ...fleet.DatastoreEnrollOsqueryOption) (*fleet.Host, error) {
				return &fleet.Host{ID: 40, NodeKey: &nk}, nil
			}
			d := New(ds, pool, WithHostCache(30*time.Second))

			primeCachedHost(t, d, 40, nk)
			h, err := d.EnrollOsquery(ctx)
			require.NoError(t, err)
			require.NotNil(t, h)
			requireCacheMiss(t, d, nk)
		})

		t.Run("DeleteHost clears cache entry", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			ds.DeleteHostFunc = func(_ context.Context, _ uint) error { return nil }
			d := New(ds, pool, WithHostCache(30*time.Second))

			nk := "nk-delete"
			primeCachedHost(t, d, 50, nk)
			require.NoError(t, d.DeleteHost(ctx, 50))
			requireCacheMiss(t, d, nk)
		})

		t.Run("DeleteHosts clears each cache entry", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			ds.DeleteHostsFunc = func(_ context.Context, _ []uint) error { return nil }
			d := New(ds, pool, WithHostCache(30*time.Second))

			nks := []string{"nk-del-1", "nk-del-2"}
			ids := []uint{60, 61}
			for i, nk := range nks {
				primeCachedHost(t, d, ids[i], nk)
			}
			require.NoError(t, d.DeleteHosts(ctx, ids))
			for _, nk := range nks {
				requireCacheMiss(t, d, nk)
			}
		})

		t.Run("CleanupExpiredHosts clears cache for each removed host", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			ds.CleanupExpiredHostsFunc = func(_ context.Context) ([]fleet.DeletedHostDetails, error) {
				return []fleet.DeletedHostDetails{{ID: 70}, {ID: 71}}, nil
			}
			d := New(ds, pool, WithHostCache(30*time.Second))

			nks := []string{"nk-exp-1", "nk-exp-2"}
			for i, nk := range nks {
				primeCachedHost(t, d, uint(70+i), nk)
			}
			_, err := d.CleanupExpiredHosts(ctx)
			require.NoError(t, err)
			for _, nk := range nks {
				requireCacheMiss(t, d, nk)
			}
		})

		t.Run("CleanupIncomingHosts clears cache for each removed host", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			ds.CleanupIncomingHostsFunc = func(_ context.Context, _ time.Time) ([]uint, error) {
				return []uint{80, 81}, nil
			}
			d := New(ds, pool, WithHostCache(30*time.Second))

			nks := []string{"nk-in-1", "nk-in-2"}
			for i, nk := range nks {
				primeCachedHost(t, d, uint(80+i), nk)
			}
			_, err := d.CleanupIncomingHosts(ctx, time.Now())
			require.NoError(t, err)
			for _, nk := range nks {
				requireCacheMiss(t, d, nk)
			}
		})

		t.Run("UpdateHost with both keys clears both osquery and orbit caches", func(t *testing.T) {
			// Regression test for the dual-invalidation design: a host
			// running both agents has both nk and onk entries in the cache.
			// UpdateHost should clear both.
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			ds.UpdateHostFunc = func(_ context.Context, _ *fleet.Host) error { return nil }
			d := New(ds, pool, WithHostCache(30*time.Second))

			nk := "nk-dual"
			onk := "onk-dual"
			host := &fleet.Host{ID: 101, NodeKey: &nk, OrbitNodeKey: &onk, Hostname: "dual"}
			d.hostCachePut(ctx, host)
			d.hostCachePutByOrbit(ctx, host)

			_, res := d.hostCacheGet(ctx, nk)
			require.Equal(t, hostCacheLookupHit, res)
			_, res = d.hostCacheGetByOrbitNodeKey(ctx, onk)
			require.Equal(t, hostCacheLookupHit, res)

			require.NoError(t, d.UpdateHost(ctx, host))

			_, res = d.hostCacheGet(ctx, nk)
			assert.Equal(t, hostCacheLookupMiss, res)
			_, res = d.hostCacheGetByOrbitNodeKey(ctx, onk)
			assert.Equal(t, hostCacheLookupMiss, res)
		})

		t.Run("inner error preserves cache", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			ds := new(mock.Store)
			boom := errors.New("boom")
			ds.UpdateHostFunc = func(_ context.Context, _ *fleet.Host) error { return boom }
			d := New(ds, pool, WithHostCache(30*time.Second))

			nk := "nk-preserve-on-err"
			primeCachedHost(t, d, 90, nk)
			err := d.UpdateHost(ctx, &fleet.Host{ID: 90, NodeKey: &nk})
			require.ErrorIs(t, err, boom)

			// Cache must still hold the pre-write value — a failed write must
			// not evict valid cached data.
			_, result := d.hostCacheGet(ctx, nk)
			assert.Equal(t, hostCacheLookupHit, result)
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
