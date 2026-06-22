package live_query

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/test"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisLiveQuery(t *testing.T) {
	// Run every interface-contract test against both storage models: the legacy
	// bitfield and the reverse per-host index. The reverse index uses a large
	// threshold so the small target sets used in these tests are all stored as
	// reverse queries.
	models := []struct {
		name    string
		reverse bool
	}{
		{"bitfield", false},
		{"reverse", true},
	}
	for _, f := range testFunctions {
		t.Run(test.FunctionName(f), func(t *testing.T) {
			for _, m := range models {
				t.Run(m.name, func(t *testing.T) {
					t.Run("standalone", func(t *testing.T) {
						store := setupRedisLiveQuery(t, false, m.reverse)
						f(t, store)
					})

					t.Run("cluster", func(t *testing.T) {
						store := setupRedisLiveQuery(t, true, m.reverse)
						f(t, store)
					})
				})
			}
		})
	}
}

func setupRedisLiveQuery(t *testing.T, cluster, reverseEnabled bool) *redisLiveQuery {
	// A 0 threshold disables the reverse index (bitfield model); a large threshold
	// ensures the small target sets used in the contract tests all qualify for the
	// reverse index.
	threshold := 0
	if reverseEnabled {
		threshold = 1 << 30
	}
	return setupRedisLiveQueryThreshold(t, cluster, threshold)
}

func setupRedisLiveQueryThreshold(t *testing.T, cluster bool, threshold int) *redisLiveQuery {
	return setupRedisLiveQueryThresholdTTL(t, cluster, threshold, 0)
}

func setupRedisLiveQueryThresholdTTL(t *testing.T, cluster bool, threshold int, ttl time.Duration) *redisLiveQuery {
	pool := redistest.SetupRedis(t, "*livequery", cluster, true, true)
	return NewRedisLiveQuery(pool, slog.New(slog.DiscardHandler), ttl, threshold)
}

func TestMapBitfield(t *testing.T) {
	// empty
	assert.Equal(t, []byte{}, mapBitfield(nil))
	assert.Equal(t, []byte{}, mapBitfield([]uint{}))

	// one byte
	assert.Equal(t, []byte("\x80"), mapBitfield([]uint{0}))
	assert.Equal(t, []byte("\x40"), mapBitfield([]uint{1}))
	assert.Equal(t, []byte("\xc0"), mapBitfield([]uint{0, 1}))

	assert.Equal(t, []byte("\x08"), mapBitfield([]uint{4}))
	assert.Equal(t, []byte("\xf8"), mapBitfield([]uint{0, 1, 2, 3, 4}))
	assert.Equal(t, []byte("\xff"), mapBitfield([]uint{0, 1, 2, 3, 4, 5, 6, 7}))

	// two bytes
	assert.Equal(t, []byte("\x00\x80"), mapBitfield([]uint{8}))
	assert.Equal(t, []byte("\xff\x80"), mapBitfield([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8}))

	// more bytes
	assert.Equal(
		t,
		[]byte("\xff\x80\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00 "),
		mapBitfield([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 170}),
	)
	assert.Equal(
		t,
		[]byte("\xff\x80\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00@\x00\x00\x00\x00\x00\x00 "),
		mapBitfield([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 113, 170}),
	)
	assert.Equal(
		t,
		[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01"),
		mapBitfield([]uint{79}),
	)
}

// TestReverseIndexThreshold verifies that queries at or below the small-target
// threshold are stored in the per-host reverse index (no bitfield) while larger
// queries keep the bitfield, and that the read path returns the union of both.
func TestReverseIndexThreshold(t *testing.T) {
	for _, cluster := range []bool{false, true} {
		clusterName := "standalone"
		if cluster {
			clusterName = "cluster"
		}
		t.Run(clusterName, func(t *testing.T) {
			// threshold 2: up to 2 targeted hosts use the reverse index.
			store := setupRedisLiveQueryThreshold(t, cluster, 2)
			conn := redis.ConfigureDoer(store.pool, store.pool.Get())
			defer conn.Close()

			// Small-target query (2 hosts == threshold) -> reverse index.
			require.NoError(t, store.RunQuery("small", "SELECT 1", []uint{1, 2}))

			for _, h := range []uint{1, 2} {
				isMember, err := redigo.Bool(conn.Do("SISMEMBER", reverseHostKey(h), "small"))
				require.NoError(t, err)
				assert.True(t, isMember, "host %d should be in the reverse set", h)
			}
			isReverse, err := redigo.Bool(conn.Do("SISMEMBER", activeReverseQueriesKey, "small"))
			require.NoError(t, err)
			assert.True(t, isReverse, "small-target query should be marked reverse")
			bitfieldExists, err := redigo.Int(conn.Do("EXISTS", queryKeyPrefix+"{small}"))
			require.NoError(t, err)
			assert.Zero(t, bitfieldExists, "small-target query must not create a bitfield")

			// Broadcast query (3 hosts > threshold) -> bitfield.
			require.NoError(t, store.RunQuery("big", "SELECT 2", []uint{1, 2, 3}))

			bitfieldExists, err = redigo.Int(conn.Do("EXISTS", queryKeyPrefix+"{big}"))
			require.NoError(t, err)
			assert.Equal(t, 1, bitfieldExists, "broadcast query must create a bitfield")
			isReverse, err = redigo.Bool(conn.Do("SISMEMBER", activeReverseQueriesKey, "big"))
			require.NoError(t, err)
			assert.False(t, isReverse, "broadcast query should not be marked reverse")
			isMember, err := redigo.Bool(conn.Do("SISMEMBER", reverseHostKey(1), "big"))
			require.NoError(t, err)
			assert.False(t, isMember, "broadcast query must not be added to per-host sets")

			// Read path returns the union for a host targeted by both models.
			queries, err := store.QueriesForHost(1)
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"small": "SELECT 1", "big": "SELECT 2"}, queries)

			// Host targeted only by the broadcast query.
			queries, err = store.QueriesForHost(3)
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"big": "SELECT 2"}, queries)
		})
	}
}

// TestReverseIndexStaleEntryFiltering verifies that after StopQuery, a reverse
// query is no longer returned to a targeted host even though its campaign ID is
// intentionally left lingering in that host's per-host set (StopQuery cannot
// enumerate the per-host sets). The read-time filter against the active set is
// what keeps this correct.
func TestReverseIndexStaleEntryFiltering(t *testing.T) {
	for _, cluster := range []bool{false, true} {
		clusterName := "standalone"
		if cluster {
			clusterName = "cluster"
		}
		t.Run(clusterName, func(t *testing.T) {
			store := setupRedisLiveQueryThreshold(t, cluster, 2)
			conn := redis.ConfigureDoer(store.pool, store.pool.Get())
			defer conn.Close()

			require.NoError(t, store.RunQuery("small", "SELECT 1", []uint{1}))

			queries, err := store.QueriesForHost(1)
			require.NoError(t, err)
			require.Equal(t, map[string]string{"small": "SELECT 1"}, queries)

			require.NoError(t, store.StopQuery("small"))

			// The per-host set still contains the (now stale) campaign ID: StopQuery
			// deliberately does not clean it up.
			isMember, err := redigo.Bool(conn.Do("SISMEMBER", reverseHostKey(1), "small"))
			require.NoError(t, err)
			require.True(t, isMember, "stale entry should remain in the per-host set after StopQuery")

			// Despite the lingering membership, the query must not be delivered: it is
			// filtered out because it is no longer in the active set / SQL cache.
			queries, err = store.QueriesForHost(1)
			require.NoError(t, err)
			require.Empty(t, queries, "stopped reverse query must not be returned despite stale per-host membership")
		})
	}
}

// TestReverseIndexQueryCompletedByHost verifies that QueryCompletedByHost on a
// reverse query removes only the completing host's per-host membership, so that
// host stops receiving the query while other targeted hosts still do.
func TestReverseIndexQueryCompletedByHost(t *testing.T) {
	for _, cluster := range []bool{false, true} {
		clusterName := "standalone"
		if cluster {
			clusterName = "cluster"
		}
		t.Run(clusterName, func(t *testing.T) {
			store := setupRedisLiveQueryThreshold(t, cluster, 2)
			conn := redis.ConfigureDoer(store.pool, store.pool.Get())
			defer conn.Close()

			require.NoError(t, store.RunQuery("small", "SELECT 1", []uint{1, 2}))

			// Host 1 completes the query.
			require.NoError(t, store.QueryCompletedByHost("small", 1))

			// Host 1's per-host membership is removed, host 2's remains.
			isMember, err := redigo.Bool(conn.Do("SISMEMBER", reverseHostKey(1), "small"))
			require.NoError(t, err)
			require.False(t, isMember, "completing host's membership should be removed")
			isMember, err = redigo.Bool(conn.Do("SISMEMBER", reverseHostKey(2), "small"))
			require.NoError(t, err)
			require.True(t, isMember, "other targeted host's membership should remain")

			// The query is still active, so the bitfield no-op SETBIT in
			// QueryCompletedByHost must not have created a lingering bitfield key.
			bitfieldExists, err := redigo.Int(conn.Do("EXISTS", queryKeyPrefix+"{small}"))
			require.NoError(t, err)
			require.Zero(t, bitfieldExists, "reverse query must not gain a bitfield from completion")

			// Host 1 no longer receives the query; host 2 still does.
			queries, err := store.QueriesForHost(1)
			require.NoError(t, err)
			require.Empty(t, queries)
			queries, err = store.QueriesForHost(2)
			require.NoError(t, err)
			require.Equal(t, map[string]string{"small": "SELECT 1"}, queries)
		})
	}
}

// TestReverseIndexCleanupInactiveQueries verifies that CleanupInactiveQueries
// removes a reverse campaign from both the active set and the reverse-model set.
func TestReverseIndexCleanupInactiveQueries(t *testing.T) {
	for _, cluster := range []bool{false, true} {
		clusterName := "standalone"
		if cluster {
			clusterName = "cluster"
		}
		t.Run(clusterName, func(t *testing.T) {
			store := setupRedisLiveQueryThreshold(t, cluster, 2)
			conn := redis.ConfigureDoer(store.pool, store.pool.Get())
			defer conn.Close()

			// Campaign IDs must be numeric so they match the uint IDs passed to
			// CleanupInactiveQueries.
			require.NoError(t, store.RunQuery("5", "SELECT 1", []uint{1}))

			isReverse, err := redigo.Bool(conn.Do("SISMEMBER", activeReverseQueriesKey, "5"))
			require.NoError(t, err)
			require.True(t, isReverse)

			require.NoError(t, store.CleanupInactiveQueries(t.Context(), []uint{5}))

			isActive, err := redigo.Bool(conn.Do("SISMEMBER", activeQueriesKey, "5"))
			require.NoError(t, err)
			require.False(t, isActive, "inactive campaign should be removed from the active set")
			isReverse, err = redigo.Bool(conn.Do("SISMEMBER", activeReverseQueriesKey, "5"))
			require.NoError(t, err)
			require.False(t, isReverse, "inactive campaign should be removed from the reverse-model set")
		})
	}
}

// TestReverseIndexKillSwitch verifies that a threshold of 0 disables the reverse
// index and forces the legacy bitfield model even for single-host queries.
func TestReverseIndexKillSwitch(t *testing.T) {
	for _, cluster := range []bool{false, true} {
		clusterName := "standalone"
		if cluster {
			clusterName = "cluster"
		}
		t.Run(clusterName, func(t *testing.T) {
			store := setupRedisLiveQueryThreshold(t, cluster, 0)
			conn := redis.ConfigureDoer(store.pool, store.pool.Get())
			defer conn.Close()

			require.NoError(t, store.RunQuery("q", "SELECT 1", []uint{1}))

			bitfieldExists, err := redigo.Int(conn.Do("EXISTS", queryKeyPrefix+"{q}"))
			require.NoError(t, err)
			assert.Equal(t, 1, bitfieldExists, "threshold 0 should force the bitfield model")
			isReverse, err := redigo.Bool(conn.Do("SISMEMBER", activeReverseQueriesKey, "q"))
			require.NoError(t, err)
			assert.False(t, isReverse)
			isMember, err := redigo.Bool(conn.Do("SISMEMBER", reverseHostKey(1), "q"))
			require.NoError(t, err)
			assert.False(t, isMember)

			queries, err := store.QueriesForHost(1)
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"q": "SELECT 1"}, queries)
		})
	}
}

// TestTestBit checks that testBit reads back exactly the bits mapBitfield sets,
// since the in-memory read path relies on the two agreeing.
func TestTestBit(t *testing.T) {
	cases := [][]uint{
		{0}, {1}, {4}, {7}, {8}, {0, 1, 2, 3, 4, 5, 6, 7, 8}, {79}, {2, 5, 9}, {100000},
	}
	for _, ids := range cases {
		bitfield := mapBitfield(ids)
		targeted := make(map[uint]struct{}, len(ids))
		for _, id := range ids {
			targeted[id] = struct{}{}
			assert.True(t, testBit(bitfield, id), "bit %d should be set for %v", id, ids)
		}
		// Every other bit up to a bit past the last ID must read as 0.
		for probe := uint(0); probe <= ids[len(ids)-1]+10; probe++ {
			if _, ok := targeted[probe]; ok {
				continue
			}
			assert.False(t, testBit(bitfield, probe), "bit %d should not be set for %v", probe, ids)
		}
	}

	// A bit beyond the end of the bitfield reads as 0 (matching GETBIT).
	assert.False(t, testBit(mapBitfield(nil), 0))
	assert.False(t, testBit([]byte{}, 5))
	assert.False(t, testBit(mapBitfield([]uint{1}), 1000))
}

// TestBitfieldCacheFallbackToGetbit verifies that when a broadcast query's
// bitfield is not in the in-memory cache (e.g. it expired while the campaign is
// still active), QueriesForHost falls back to a live GETBIT rather than missing
// the query.
func TestBitfieldCacheFallbackToGetbit(t *testing.T) {
	for _, cluster := range []bool{false, true} {
		clusterName := "standalone"
		if cluster {
			clusterName = "cluster"
		}
		t.Run(clusterName, func(t *testing.T) {
			// threshold 0 => broadcast (bitfield) model; long TTL so the cache is not
			// reloaded between the two calls below.
			store := setupRedisLiveQueryThresholdTTL(t, cluster, 0, time.Hour)

			require.NoError(t, store.RunQuery("q", "SELECT 1", []uint{1}))

			// Prime the cache (loads the bitfield into memory).
			queries, err := store.QueriesForHost(1)
			require.NoError(t, err)
			require.Equal(t, map[string]string{"q": "SELECT 1"}, queries)

			// Evict only the cached bitfield to simulate the rare race where the
			// campaign is still active but its bitfield is not cached. The bitfield
			// still exists in Redis, so the read path must fall back to GETBIT.
			store.cache.mu.Lock()
			delete(store.cache.bitfields, "q")
			store.cache.mu.Unlock()

			queries, err = store.QueriesForHost(1)
			require.NoError(t, err)
			require.Equal(t, map[string]string{"q": "SELECT 1"}, queries, "fallback GETBIT should still return the query")
		})
	}
}

// TestLoadBatchIntoCacheDrain exercises the conditional reply-draining in
// loadBatchIntoCache: a broadcast query whose SQL has expired (its bitfield GET
// reply must still be drained) batched together, in both orders, with a live
// broadcast query in the same pipeline. A missed drain would desync the pipeline
// and corrupt the live query's result.
func TestLoadBatchIntoCacheDrain(t *testing.T) {
	for _, expiredFirst := range []bool{true, false} {
		name := "liveFirst"
		if expiredFirst {
			name = "expiredFirst"
		}
		t.Run(name, func(t *testing.T) {
			store := setupRedisLiveQueryThreshold(t, false, 0) // broadcast model

			// "live" gets a SQL + bitfield; "expired" is never created, so its SQL
			// (and bitfield) GETs return nil.
			require.NoError(t, store.RunQuery("live", "SELECT 1", []uint{1}))

			liveKey, _ := generateKeys("live")
			expiredKey, _ := generateKeys("expired")
			targetKeys := []string{liveKey, expiredKey}
			if expiredFirst {
				targetKeys = []string{expiredKey, liveKey}
			}

			sqlCache := map[string]string{}
			bitfields := map[string][]byte{}
			expired := map[string]struct{}{}
			err := store.loadBatchIntoCache(targetKeys, map[string]struct{}{}, sqlCache, bitfields, expired)
			require.NoError(t, err)

			// "live" resolved correctly despite "expired" sharing the batch.
			require.Equal(t, map[string]string{"live": "SELECT 1"}, sqlCache)
			require.Contains(t, bitfields, "live")
			require.True(t, testBit(bitfields["live"], 1))
			require.NotContains(t, bitfields, "expired")
			require.Contains(t, expired, "expired")
		})
	}
}

// TestBitfieldServedFromCacheWithoutGetbit proves the read path serves a
// broadcast query from the in-memory bitfield cache rather than a live GETBIT:
// after priming the cache, the bitfield key is deleted from Redis, yet the query
// is still returned (a live GETBIT would read 0 and drop it).
func TestBitfieldServedFromCacheWithoutGetbit(t *testing.T) {
	store := setupRedisLiveQueryThresholdTTL(t, false, 0, time.Hour) // broadcast model, long TTL

	require.NoError(t, store.RunQuery("q", "SELECT 1", []uint{1}))

	// Prime the in-memory cache.
	queries, err := store.QueriesForHost(1)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"q": "SELECT 1"}, queries)

	// Delete the bitfield from Redis. A cache hit will still return the query; a
	// live GETBIT would now read 0 and miss it.
	conn := redis.ConfigureDoer(store.pool, store.pool.Get())
	defer conn.Close()
	_, err = conn.Do("DEL", queryKeyPrefix+"{q}")
	require.NoError(t, err)

	queries, err = store.QueriesForHost(1)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"q": "SELECT 1"}, queries, "broadcast query must be served from the in-memory cache, not a live GETBIT")
}

// TestConcurrentQueriesForHost exercises concurrent checkins (and thus the
// singleflight-coalesced cache reload) to surface races under -race.
func TestConcurrentQueriesForHost(t *testing.T) {
	store := setupRedisLiveQuery(t, false, false) // 0 TTL => every call reloads the cache

	require.NoError(t, store.RunQuery("q", "SELECT 1", []uint{1, 2, 3}))

	const goroutines = 25
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make(chan error, goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			queries, err := store.QueriesForHost(1)
			if err != nil {
				errs <- err
				return
			}
			if queries["q"] != "SELECT 1" {
				errs <- assert.AnError
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}
