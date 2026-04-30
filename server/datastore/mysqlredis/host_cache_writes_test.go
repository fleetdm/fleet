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
	d.hostCachePutByNodeKey(ctx, &fleet.Host{ID: id, NodeKey: &nk, Hostname: "primed-" + nk})
	_, result := d.hostCacheGetByNodeKey(ctx, nk)
	require.Equal(t, hostCacheLookupHit, result, "primeCachedHost failed to prime %q", nk)
}

func requireCacheMiss(t *testing.T, d *Datastore, nk string) {
	t.Helper()
	_, result := d.hostCacheGetByNodeKey(t.Context(), nk)
	require.Equal(t, hostCacheLookupMiss, result, "expected miss for %q, got %v", nk, result)
}

func TestWritePathInvalidation(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		ctx := t.Context()

		// Single-host wrapper invalidation cases. Each case verifies that the wrapper invalidates the
		// cache for the affected host after a successful inner call. Methods with materially different
		// invalidation paths (enrollment, batch, regression) get their own subtests below.
		singleHostWrappers := []struct {
			name      string
			id        uint
			setupMock func(*mock.Store)
			invoke    func(context.Context, *Datastore, uint, string) error
			invoked   func(*mock.Store) bool
		}{
			{
				name: "UpdateHost",
				id:   1,
				setupMock: func(ds *mock.Store) {
					ds.UpdateHostFunc = func(_ context.Context, _ *fleet.Host) error { return nil }
				},
				invoke: func(ctx context.Context, d *Datastore, id uint, nk string) error {
					return d.UpdateHost(ctx, &fleet.Host{ID: id, NodeKey: &nk})
				},
				invoked: func(ds *mock.Store) bool { return ds.UpdateHostFuncInvoked },
			},
			{
				name: "SerialUpdateHost",
				id:   2,
				setupMock: func(ds *mock.Store) {
					ds.SerialUpdateHostFunc = func(_ context.Context, _ *fleet.Host) error { return nil }
				},
				invoke: func(ctx context.Context, d *Datastore, id uint, nk string) error {
					return d.SerialUpdateHost(ctx, &fleet.Host{ID: id, NodeKey: &nk})
				},
				invoked: func(ds *mock.Store) bool { return ds.SerialUpdateHostFuncInvoked },
			},
			{
				name: "UpdateHostOsqueryIntervals",
				id:   3,
				setupMock: func(ds *mock.Store) {
					ds.UpdateHostOsqueryIntervalsFunc = func(_ context.Context, _ uint, _ fleet.HostOsqueryIntervals) error { return nil }
				},
				invoke: func(ctx context.Context, d *Datastore, id uint, _ string) error {
					return d.UpdateHostOsqueryIntervals(ctx, id, fleet.HostOsqueryIntervals{})
				},
				invoked: func(ds *mock.Store) bool { return ds.UpdateHostOsqueryIntervalsFuncInvoked },
			},
			{
				name: "UpdateHostRefetchRequested",
				id:   4,
				setupMock: func(ds *mock.Store) {
					ds.UpdateHostRefetchRequestedFunc = func(_ context.Context, _ uint, _ bool) error { return nil }
				},
				invoke: func(ctx context.Context, d *Datastore, id uint, _ string) error {
					return d.UpdateHostRefetchRequested(ctx, id, true)
				},
				invoked: func(ds *mock.Store) bool { return ds.UpdateHostRefetchRequestedFuncInvoked },
			},
			{
				name: "UpdateHostRefetchCriticalQueriesUntil",
				id:   5,
				setupMock: func(ds *mock.Store) {
					ds.UpdateHostRefetchCriticalQueriesUntilFunc = func(_ context.Context, _ uint, _ *time.Time) error { return nil }
				},
				invoke: func(ctx context.Context, d *Datastore, id uint, _ string) error {
					return d.UpdateHostRefetchCriticalQueriesUntil(ctx, id, new(time.Unix(1, 0)))
				},
				invoked: func(ds *mock.Store) bool { return ds.UpdateHostRefetchCriticalQueriesUntilFuncInvoked },
			},
			{
				name: "UpdateHostIdentityCertHostIDBySerial",
				id:   6,
				setupMock: func(ds *mock.Store) {
					ds.UpdateHostIdentityCertHostIDBySerialFunc = func(_ context.Context, _ uint64, _ uint) error { return nil }
				},
				invoke: func(ctx context.Context, d *Datastore, id uint, _ string) error {
					return d.UpdateHostIdentityCertHostIDBySerial(ctx, 12345, id)
				},
				invoked: func(ds *mock.Store) bool { return ds.UpdateHostIdentityCertHostIDBySerialFuncInvoked },
			},
			{
				name: "DeleteHost",
				id:   7,
				setupMock: func(ds *mock.Store) {
					ds.DeleteHostFunc = func(_ context.Context, _ uint) error { return nil }
				},
				invoke: func(ctx context.Context, d *Datastore, id uint, _ string) error {
					return d.DeleteHost(ctx, id)
				},
				invoked: func(ds *mock.Store) bool { return ds.DeleteHostFuncInvoked },
			},
		}

		for _, tc := range singleHostWrappers {
			t.Run(tc.name+" invalidates cache", func(t *testing.T) {
				t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
				ds := new(mock.Store)
				tc.setupMock(ds)
				d := New(ds, pool, WithHostCache(30*time.Second))

				nk := "nk-" + tc.name
				primeCachedHost(t, d, tc.id, nk)
				require.NoError(t, tc.invoke(ctx, d, tc.id, nk))
				require.True(t, tc.invoked(ds), "inner mock not invoked")
				requireCacheMiss(t, d, nk)
			})
		}

		t.Run("EnrollOrbit invalidates for returned host", func(t *testing.T) {
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			// The mock returns a Host with ONLY ID populated, mirroring the
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

		t.Run("EnrollOrbit clears stale negative cache for new orbit_node_key", func(t *testing.T) {
			// mysql.EnrollOrbit does NOT populate host.OrbitNodeKey on the returned struct
			// (unlike EnrollOsquery, which does a SELECT-back). The wrapper must extract orbit_node_key
			// from opts to fire the direct-keys clear; otherwise the helper short-circuits on empty key
			// and a pre-enrollment onk_miss:<K> entry survives, returning NotFound from the negative
			// cache for the freshly-enrolled host's first /orbit/* requests.
			t.Cleanup(func() { cleanupHostCacheKeys(t, pool) })
			onk := "onk-pre-enroll-race"
			ds := new(mock.DataStore)
			ds.EnrollOrbitFunc = func(_ context.Context, _ ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
				// Mirror mysql.EnrollOrbit's return: ID set, OrbitNodeKey nil.
				return &fleet.Host{ID: 99}, nil
			}
			d := New(ds, pool, WithHostCache(30*time.Second))

			// A probe arrived before enrollment committed and cached NotFound under the new key.
			d.hostCachePutNotFoundFamily(ctx, orbitCacheFamily, onk)
			_, before := d.hostCacheGetByOrbitNodeKey(ctx, onk)
			require.Equal(t, hostCacheLookupNegative, before, "precondition: negative cache must be primed")

			h, err := d.EnrollOrbit(ctx, fleet.WithEnrollOrbitNodeKey(onk))
			require.NoError(t, err)
			require.NotNil(t, h)

			// After EnrollOrbit, the negative entry must be gone so the agent's first /orbit/config
			// falls through to the DB instead of getting a stale NotFound.
			_, after := d.hostCacheGetByOrbitNodeKey(ctx, onk)
			require.Equal(t, hostCacheLookupMiss, after, "EnrollOrbit must clear the pre-enrollment negative cache")
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

			params := fleet.NewAddHostsToTeamParams(new(uint(7)), ids)
			require.NoError(t, d.AddHostsToTeam(ctx, params))
			require.True(t, ds.AddHostsToTeamFuncInvoked)
			for _, nk := range nks {
				requireCacheMiss(t, d, nk)
			}
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
			d.hostCachePutNotFoundByNodeKey(ctx, nk)

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
			d.hostCachePutByNodeKey(ctx, host)
			d.hostCachePutByOrbitNodeKey(ctx, host)

			_, res := d.hostCacheGetByNodeKey(ctx, nk)
			require.Equal(t, hostCacheLookupHit, res)
			_, res = d.hostCacheGetByOrbitNodeKey(ctx, onk)
			require.Equal(t, hostCacheLookupHit, res)

			require.NoError(t, d.UpdateHost(ctx, host))

			_, res = d.hostCacheGetByNodeKey(ctx, nk)
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

			// Cache must still hold the pre-written value. A failed write operation must not evict valid cached data.
			_, result := d.hostCacheGetByNodeKey(ctx, nk)
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
