package mysql

import (
	"context"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// 1. Performance scaling test
// ---------------------------------------------------------------------------

// TestSaveHostPackStatsScaling measures saveHostPackStatsDB performance at
// different report counts to demonstrate the 500-subquery bottleneck (#54969).
func TestSaveHostPackStatsScaling(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()
	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("scale-host-1"),
		UUID:            "scale-uuid-1",
		Hostname:        "scale.local",
		PrimaryIP:       "10.0.0.1",
		PrimaryMac:      "AA-BB-CC-DD-EE-FF",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	for _, numQueries := range []int{10, 100, 500} {
		t.Run(fmt.Sprintf("queries=%d", numQueries), func(t *testing.T) {
			queryStats := make([]fleet.ScheduledQueryStats, 0, numQueries)
			for i := range numQueries {
				q := test.NewQuery(t, ds, nil, fmt.Sprintf("scale-%d-q%d", numQueries, i), "SELECT 1", 0, true)
				queryStats = append(queryStats, fleet.ScheduledQueryStats{
					ScheduledQueryName: q.Name,
					PackName:           "Global",
					AverageMemory:      8000,
					Denylisted:         false,
					Executions:         uint64(i + 1),
					Interval:           3600,
					LastExecuted:       time.Now().UTC(),
					OutputSize:         1337,
					SystemTime:         150,
					UserTime:           180,
					WallTimeMs:         5000,
				})
			}
			packStats := []fleet.PackStats{{
				PackName:   "Global",
				QueryStats: queryStats,
			}}

			const iters = 5
			var totalMs int64
			for range iters {
				start := time.Now()
				err := ds.SaveHostPackStats(ctx, host.TeamID, host.ID, packStats)
				require.NoError(t, err)
				totalMs += time.Since(start).Milliseconds()
			}
			avgMs := totalMs / iters
			t.Logf("  %d queries: avg %dms per call (%d iterations)", numQueries, avgMs, iters)
		})
	}
}

// ---------------------------------------------------------------------------
// 2. Concurrent load test
// ---------------------------------------------------------------------------

// TestConcurrentStatsIngestion simulates many hosts concurrently ingesting
// stats for 500 reports, reproducing the connection pool exhaustion (#54969).
func TestConcurrentStatsIngestion(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()
	const numQueries = 500
	const numHosts = 50
	const concurrency = 20

	queryStats := makeGlobalQueryStats(t, ds, numQueries, "conc")
	hosts := makeHosts(t, ds, numHosts, "conc")

	packStats := []fleet.PackStats{{
		PackName:   "Global",
		QueryStats: queryStats,
	}}

	var (
		wg       sync.WaitGroup
		sem      = make(chan struct{}, concurrency)
		errCount atomic.Int64
		totalMs  atomic.Int64
	)

	start := time.Now()
	for _, h := range hosts {
		wg.Add(1)
		go func(host *fleet.Host) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			t0 := time.Now()
			err := ds.SaveHostPackStats(ctx, host.TeamID, host.ID, packStats)
			totalMs.Add(time.Since(t0).Milliseconds())
			if err != nil {
				errCount.Add(1)
			}
		}(h)
	}
	wg.Wait()
	wallTime := time.Since(start)

	avgMs := totalMs.Load() / int64(numHosts)
	t.Logf("=== Concurrent stats ingestion: %d hosts x %d queries ===", numHosts, numQueries)
	t.Logf("  Wall time:    %v", wallTime)
	t.Logf("  Avg per host: %dms", avgMs)
	t.Logf("  Errors:       %d/%d", errCount.Load(), numHosts)
	t.Logf("  Concurrency:  %d", concurrency)
}

// ---------------------------------------------------------------------------
// 3. Edge case tests
// ---------------------------------------------------------------------------

func TestSaveHostPackStatsEdgeCases(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()
	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(), LabelUpdatedAt: time.Now(),
		PolicyUpdatedAt: time.Now(), SeenTime: time.Now(),
		NodeKey: new("edge-host"), UUID: "edge-uuid",
		Hostname: "edge.local", PrimaryIP: "10.0.0.1",
		PrimaryMac: "AA-BB-CC-DD-EE-01", Platform: "darwin",
	})
	require.NoError(t, err)

	t.Run("empty stats", func(t *testing.T) {
		err := ds.SaveHostPackStats(ctx, host.TeamID, host.ID, nil)
		require.NoError(t, err)
		err = ds.SaveHostPackStats(ctx, host.TeamID, host.ID, []fleet.PackStats{})
		require.NoError(t, err)
	})

	t.Run("global queries (NULL team_id)", func(t *testing.T) {
		q := test.NewQuery(t, ds, nil, "edge-global-q", "SELECT 1", 0, true)
		stats := []fleet.PackStats{{
			PackName: "Global",
			QueryStats: []fleet.ScheduledQueryStats{{
				ScheduledQueryName: q.Name,
				PackName:           "Global",
				Executions:         10,
				Interval:           3600,
				LastExecuted:       time.Now().UTC(),
				WallTimeMs:         100,
			}},
		}}
		err := ds.SaveHostPackStats(ctx, host.TeamID, host.ID, stats)
		require.NoError(t, err)

		// Verify the row was inserted.
		var count int
		ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, db, &count,
				`SELECT COUNT(*) FROM scheduled_query_stats WHERE host_id = ? AND scheduled_query_id = ?`,
				host.ID, q.ID)
		})
		assert.Equal(t, 1, count)
	})

	t.Run("team-scoped queries", func(t *testing.T) {
		// Create a real team so FK constraint is satisfied.
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "edge-team-42"})
		require.NoError(t, err)
		q := test.NewQuery(t, ds, &team.ID, "edge-team-q", "SELECT 1", 0, true)
		stats := []fleet.PackStats{{
			PackName: fmt.Sprintf("team-%d", team.ID),
			QueryStats: []fleet.ScheduledQueryStats{{
				ScheduledQueryName: q.Name,
				PackName:           fmt.Sprintf("team-%d", team.ID),
				Executions:         5,
				Interval:           3600,
				LastExecuted:       time.Now().UTC(),
				WallTimeMs:         50,
			}},
		}}
		err = ds.SaveHostPackStats(ctx, &team.ID, host.ID, stats)
		require.NoError(t, err)
	})

	t.Run("mixed global and team queries", func(t *testing.T) {
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "edge-team-99"})
		require.NoError(t, err)
		gq := test.NewQuery(t, ds, nil, "edge-mix-global", "SELECT 1", 0, true)
		tq := test.NewQuery(t, ds, &team.ID, "edge-mix-team", "SELECT 1", 0, true)
		stats := []fleet.PackStats{
			{
				PackName: "Global",
				QueryStats: []fleet.ScheduledQueryStats{{
					ScheduledQueryName: gq.Name,
					PackName:           "Global",
					Executions:         1,
					Interval:           3600,
					LastExecuted:       time.Now().UTC(),
				}},
			},
			{
				PackName: fmt.Sprintf("team-%d", team.ID),
				QueryStats: []fleet.ScheduledQueryStats{{
					ScheduledQueryName: tq.Name,
					PackName:           fmt.Sprintf("team-%d", team.ID),
					Executions:         2,
					Interval:           3600,
					LastExecuted:       time.Now().UTC(),
				}},
			},
		}
		err = ds.SaveHostPackStats(ctx, &team.ID, host.ID, stats)
		require.NoError(t, err)
	})

	t.Run("deleted query is silently skipped", func(t *testing.T) {
		// Stats reference a query name that doesn't exist in the DB.
		stats := []fleet.PackStats{{
			PackName: "Global",
			QueryStats: []fleet.ScheduledQueryStats{{
				ScheduledQueryName: "nonexistent-query-xyz",
				PackName:           "Global",
				Executions:         1,
				Interval:           3600,
				LastExecuted:       time.Now().UTC(),
			}},
		}}
		err := ds.SaveHostPackStats(ctx, host.TeamID, host.ID, stats)
		require.NoError(t, err, "stats for non-existent query should not error")
	})

	t.Run("duplicate query names in same batch", func(t *testing.T) {
		q := test.NewQuery(t, ds, nil, "edge-dup-q", "SELECT 1", 0, true)
		stats := []fleet.PackStats{{
			PackName: "Global",
			QueryStats: []fleet.ScheduledQueryStats{
				{
					ScheduledQueryName: q.Name,
					PackName:           "Global",
					Executions:         10,
					Interval:           3600,
					LastExecuted:       time.Now().UTC(),
				},
				{
					ScheduledQueryName: q.Name,
					PackName:           "Global",
					Executions:         20,
					Interval:           3600,
					LastExecuted:       time.Now().UTC(),
				},
			},
		}}
		// INSERT IGNORE means the first one wins, second is silently skipped.
		err := ds.SaveHostPackStats(ctx, host.TeamID, host.ID, stats)
		require.NoError(t, err)
	})

	t.Run("upsert overwrites existing stats", func(t *testing.T) {
		q := test.NewQuery(t, ds, nil, "edge-upsert-q", "SELECT 1", 0, true)
		mkStats := func(execs uint64) []fleet.PackStats {
			return []fleet.PackStats{{
				PackName: "Global",
				QueryStats: []fleet.ScheduledQueryStats{{
					ScheduledQueryName: q.Name,
					PackName:           "Global",
					Executions:         execs,
					Interval:           3600,
					LastExecuted:       time.Now().UTC(),
					WallTimeMs:         100,
				}},
			}}
		}
		require.NoError(t, ds.SaveHostPackStats(ctx, host.TeamID, host.ID, mkStats(10)))
		require.NoError(t, ds.SaveHostPackStats(ctx, host.TeamID, host.ID, mkStats(20)))

		// Verify the execution count was updated, not duplicated.
		var execs uint64
		ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, db, &execs,
				`SELECT executions FROM scheduled_query_stats WHERE host_id = ? AND scheduled_query_id = ?`,
				host.ID, q.ID)
		})
		assert.Equal(t, uint64(20), execs, "upsert should update executions")
	})

	t.Run("large batch 1000 queries", func(t *testing.T) {
		const n = 1000
		queryStats := makeGlobalQueryStats(t, ds, n, "edge-large")
		stats := []fleet.PackStats{{
			PackName:   "Global",
			QueryStats: queryStats,
		}}
		err := ds.SaveHostPackStats(ctx, host.TeamID, host.ID, stats)
		require.NoError(t, err, "1000-query batch should succeed")
	})

	t.Run("wall_time_ms fallback from wall_time", func(t *testing.T) {
		q := test.NewQuery(t, ds, nil, "edge-walltime-q", "SELECT 1", 0, true)
		stats := []fleet.PackStats{{
			PackName: "Global",
			QueryStats: []fleet.ScheduledQueryStats{{
				ScheduledQueryName: q.Name,
				PackName:           "Global",
				Executions:         1,
				Interval:           3600,
				LastExecuted:       time.Now().UTC(),
				WallTime:          5, // seconds, WallTimeMs is 0
				WallTimeMs:         0,
			}},
		}}
		require.NoError(t, ds.SaveHostPackStats(ctx, host.TeamID, host.ID, stats))

		var wallTime uint64
		ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, db, &wallTime,
				`SELECT wall_time FROM scheduled_query_stats WHERE host_id = ? AND scheduled_query_id = ?`,
				host.ID, q.ID)
		})
		assert.Equal(t, uint64(5000), wallTime, "wall_time should be wall_time_ms = wall_time * 1000")
	})
}

// ---------------------------------------------------------------------------
// 4. EXPLAIN index verification
// ---------------------------------------------------------------------------

func TestResolveQueryUsesIndex(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	// Create a few queries so the optimizer has data to work with.
	for i := range 10 {
		test.NewQuery(t, ds, nil, fmt.Sprintf("idx-q%d", i), "SELECT 1", 0, true)
	}

	// Use a map to avoid struct field mapping issues with EXPLAIN column names.
	var rows []map[string]any
	ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
		qRows, err := db.QueryxContext(t.Context(),
			`EXPLAIN SELECT id, name, team_id_char FROM queries WHERE (name, team_id_char) IN (('idx-q0',''), ('idx-q1',''))`)
		if err != nil {
			return err
		}
		defer qRows.Close()
		for qRows.Next() {
			row := make(map[string]any)
			if err := qRows.MapScan(row); err != nil {
				return err
			}
			rows = append(rows, row)
		}
		return qRows.Err()
	})

	require.NotEmpty(t, rows)
	row := rows[0]

	toString := func(v any) string {
		if b, ok := v.([]byte); ok {
			return string(b)
		}
		return fmt.Sprint(v)
	}
	accessType := toString(row["type"])
	key := toString(row["key"])
	extra := toString(row["Extra"])
	t.Logf("EXPLAIN: type=%s key=%s extra=%s", accessType, key, extra)

	assert.NotEqual(t, "ALL", accessType, "resolve query should not do a full table scan")
	assert.Contains(t, key, "idx_", "should use an index")
}

// ---------------------------------------------------------------------------
// 5. High-load simulation (small pool, many hosts)
// ---------------------------------------------------------------------------

func TestHighLoadStatsIngestion(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()
	const numQueries = 500
	const numHosts = 200
	const concurrency = 50

	queryStats := makeGlobalQueryStats(t, ds, numQueries, "hload")
	hosts := makeHosts(t, ds, numHosts, "hload")

	packStats := []fleet.PackStats{{
		PackName:   "Global",
		QueryStats: queryStats,
	}}

	var (
		wg       sync.WaitGroup
		sem      = make(chan struct{}, concurrency)
		errCount atomic.Int64
		durations []int64
		mu        sync.Mutex
	)
	durations = make([]int64, 0, numHosts)

	start := time.Now()
	for _, h := range hosts {
		wg.Add(1)
		go func(host *fleet.Host) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			t0 := time.Now()
			err := ds.SaveHostPackStats(ctx, host.TeamID, host.ID, packStats)
			d := time.Since(t0).Milliseconds()
			mu.Lock()
			durations = append(durations, d)
			mu.Unlock()
			if err != nil {
				errCount.Add(1)
			}
		}(h)
	}
	wg.Wait()
	wallTime := time.Since(start)

	slices.Sort(durations)
	p50 := durations[len(durations)/2]
	p95 := durations[int(float64(len(durations))*0.95)]
	p99 := durations[int(float64(len(durations))*0.99)]
	max := durations[len(durations)-1]

	t.Logf("=== High-load stats ingestion: %d hosts x %d queries, concurrency %d ===", numHosts, numQueries, concurrency)
	t.Logf("  Wall time: %v", wallTime)
	t.Logf("  p50: %dms  p95: %dms  p99: %dms  max: %dms", p50, p95, p99, max)
	t.Logf("  Errors: %d/%d", errCount.Load(), numHosts)

	assert.Zero(t, errCount.Load(), "no errors expected")
}

// ---------------------------------------------------------------------------
// 6. Memory measurement
// ---------------------------------------------------------------------------

func TestStatsIngestionMemory(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()
	const numQueries = 500
	const numHosts = 100

	queryStats := makeGlobalQueryStats(t, ds, numQueries, "mem")
	hosts := makeHosts(t, ds, numHosts, "mem")

	packStats := []fleet.PackStats{{
		PackName:   "Global",
		QueryStats: queryStats,
	}}

	// Force GC and measure baseline.
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	// Run all hosts sequentially to isolate memory from concurrency overhead.
	for _, h := range hosts {
		err := ds.SaveHostPackStats(ctx, h.TeamID, h.ID, packStats)
		require.NoError(t, err)
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	heapDeltaMB := (float64(after.HeapAlloc) - float64(before.HeapAlloc)) / (1024 * 1024) //nolint:gosec
	totalAllocMB := float64(after.TotalAlloc-before.TotalAlloc) / (1024 * 1024)
	t.Logf("=== Memory: %d hosts x %d queries ===", numHosts, numQueries)
	t.Logf("  Heap delta: %.2f MB", heapDeltaMB)
	t.Logf("  Total alloc: %.2f MB", totalAllocMB)
	t.Logf("  Alloc per host: %.2f KB", totalAllocMB*1024/float64(numHosts))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeGlobalQueryStats(t *testing.T, ds *Datastore, n int, prefix string) []fleet.ScheduledQueryStats {
	t.Helper()
	stats := make([]fleet.ScheduledQueryStats, 0, n)
	for i := range n {
		q := test.NewQuery(t, ds, nil, fmt.Sprintf("%s-q%d", prefix, i), "SELECT 1", 0, true)
		stats = append(stats, fleet.ScheduledQueryStats{
			ScheduledQueryName: q.Name,
			PackName:           "Global",
			AverageMemory:      8000,
			Executions:         uint64(i + 1),
			Interval:           3600,
			LastExecuted:       time.Now().UTC(),
			OutputSize:         1337,
			SystemTime:         150,
			UserTime:           180,
			WallTimeMs:         5000,
		})
	}
	return stats
}

func makeHosts(t *testing.T, ds *Datastore, n int, prefix string) []*fleet.Host {
	t.Helper()
	hosts := make([]*fleet.Host, 0, n)
	for i := range n {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(), LabelUpdatedAt: time.Now(),
			PolicyUpdatedAt: time.Now(), SeenTime: time.Now(),
			NodeKey:    new(fmt.Sprintf("%s-host-%d", prefix, i)),
			UUID:       fmt.Sprintf("%s-uuid-%d", prefix, i),
			Hostname:   fmt.Sprintf("%s-%d.local", prefix, i),
			PrimaryIP:  fmt.Sprintf("10.%d.%d.%d", i/65536, (i/256)%256, i%256),
			PrimaryMac: fmt.Sprintf("AA-%02X-%02X-%02X-%02X-%02X", i/65536, (i/256)%256, i%256, 0, 0),
			Platform:   "darwin",
		})
		require.NoError(t, err)
		hosts = append(hosts, h)
	}
	return hosts
}

func ptrStr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func ptrInt(n *int) string {
	if n == nil {
		return "<nil>"
	}
	return fmt.Sprint(*n)
}
