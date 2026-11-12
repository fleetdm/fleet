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
// Performance
// ---------------------------------------------------------------------------

func TestSaveHostPackStatsScaling(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()
	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(), LabelUpdatedAt: time.Now(),
		PolicyUpdatedAt: time.Now(), SeenTime: time.Now(),
		NodeKey: new("scale-host-1"), UUID: "scale-uuid-1",
		Hostname: "scale.local", PrimaryIP: "10.0.0.1",
		PrimaryMac: "AA-BB-CC-DD-EE-FF", Platform: "darwin",
	})
	require.NoError(t, err)

	for _, n := range []int{10, 100, 500} {
		t.Run(fmt.Sprintf("queries=%d", n), func(t *testing.T) {
			qs := makeGlobalQueryStats(t, ds, n, fmt.Sprintf("scale-%d", n))
			ps := []fleet.PackStats{{PackName: "Global", QueryStats: qs}}
			const iters = 5
			var total int64
			for range iters {
				t0 := time.Now()
				require.NoError(t, ds.SaveHostPackStats(ctx, host.TeamID, host.ID, ps))
				total += time.Since(t0).Milliseconds()
			}
			t.Logf("  %d queries: avg %dms (%d iters)", n, total/iters, iters)
		})
	}
}

func TestConcurrentStatsIngestion(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()
	const nQ, nH, conc = 500, 50, 20
	qs := makeGlobalQueryStats(t, ds, nQ, "conc")
	hosts := makeHosts(t, ds, nH, "conc")
	ps := []fleet.PackStats{{PackName: "Global", QueryStats: qs}}

	var wg sync.WaitGroup
	sem := make(chan struct{}, conc)
	var errN, totalMs atomic.Int64

	t0 := time.Now()
	for _, h := range hosts {
		wg.Add(1)
		go func(h *fleet.Host) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			s := time.Now()
			if ds.SaveHostPackStats(ctx, h.TeamID, h.ID, ps) != nil {
				errN.Add(1)
			}
			totalMs.Add(time.Since(s).Milliseconds())
		}(h)
	}
	wg.Wait()
	t.Logf("=== %d hosts x %d queries, concurrency %d ===", nH, nQ, conc)
	t.Logf("  Wall: %v  Avg: %dms  Errors: %d", time.Since(t0), totalMs.Load()/int64(nH), errN.Load())
}

func TestHighLoadStatsIngestion(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()
	const nQ, nH, conc = 500, 200, 50
	qs := makeGlobalQueryStats(t, ds, nQ, "hload")
	hosts := makeHosts(t, ds, nH, "hload")
	ps := []fleet.PackStats{{PackName: "Global", QueryStats: qs}}

	var wg sync.WaitGroup
	sem := make(chan struct{}, conc)
	var errN atomic.Int64
	dur := make([]int64, 0, nH)
	var mu sync.Mutex

	t0 := time.Now()
	for _, h := range hosts {
		wg.Add(1)
		go func(h *fleet.Host) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			s := time.Now()
			if ds.SaveHostPackStats(ctx, h.TeamID, h.ID, ps) != nil {
				errN.Add(1)
			}
			mu.Lock()
			dur = append(dur, time.Since(s).Milliseconds())
			mu.Unlock()
		}(h)
	}
	wg.Wait()
	slices.Sort(dur)
	t.Logf("=== %d hosts x %d queries, concurrency %d ===", nH, nQ, conc)
	t.Logf("  Wall: %v  p50: %dms  p95: %dms  p99: %dms  max: %dms  Errors: %d",
		time.Since(t0), dur[len(dur)/2], dur[int(float64(len(dur))*0.95)],
		dur[int(float64(len(dur))*0.99)], dur[len(dur)-1], errN.Load())
	assert.Zero(t, errN.Load())
}

func TestStatsIngestionMemory(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()
	const nQ, nH = 500, 100
	qs := makeGlobalQueryStats(t, ds, nQ, "mem")
	hosts := makeHosts(t, ds, nH, "mem")
	ps := []fleet.PackStats{{PackName: "Global", QueryStats: qs}}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	for _, h := range hosts {
		require.NoError(t, ds.SaveHostPackStats(ctx, h.TeamID, h.ID, ps))
	}
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	allocMB := float64(after.TotalAlloc-before.TotalAlloc) / (1024 * 1024)
	t.Logf("=== Memory: %d hosts x %d queries ===", nH, nQ)
	t.Logf("  Total alloc: %.1f MB  Per host: %.1f KB", allocMB, allocMB*1024/float64(nH))
}

// ---------------------------------------------------------------------------
// EXPLAIN index verification
// ---------------------------------------------------------------------------

func TestResolveQueryUsesIndex(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	for i := range 10 {
		test.NewQuery(t, ds, nil, fmt.Sprintf("idx-q%d", i), "SELECT 1", 0, true)
	}
	var rows []map[string]any
	ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
		qr, err := db.QueryxContext(t.Context(),
			`EXPLAIN SELECT id, name, team_id_char FROM queries WHERE (name, team_id_char) IN (('idx-q0',''),('idx-q1',''))`)
		if err != nil {
			return err
		}
		defer qr.Close()
		for qr.Next() {
			r := make(map[string]any)
			if err := qr.MapScan(r); err != nil {
				return err
			}
			rows = append(rows, r)
		}
		return qr.Err()
	})
	require.NotEmpty(t, rows)
	s := func(v any) string {
		if b, ok := v.([]byte); ok {
			return string(b)
		}
		return fmt.Sprint(v)
	}
	tp, key := s(rows[0]["type"]), s(rows[0]["key"])
	t.Logf("EXPLAIN: type=%s key=%s extra=%s", tp, key, s(rows[0]["Extra"]))
	assert.NotEqual(t, "ALL", tp, "should not full-scan")
	assert.Contains(t, key, "idx_", "should use an index")
}

// ---------------------------------------------------------------------------
// Edge cases
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

	t.Run("empty", func(t *testing.T) {
		require.NoError(t, ds.SaveHostPackStats(ctx, host.TeamID, host.ID, nil))
		require.NoError(t, ds.SaveHostPackStats(ctx, host.TeamID, host.ID, []fleet.PackStats{}))
	})

	t.Run("global queries", func(t *testing.T) {
		q := test.NewQuery(t, ds, nil, "edge-global", "SELECT 1", 0, true)
		require.NoError(t, ds.SaveHostPackStats(ctx, host.TeamID, host.ID, []fleet.PackStats{{
			PackName: "Global",
			QueryStats: []fleet.ScheduledQueryStats{{
				ScheduledQueryName: q.Name, PackName: "Global",
				Executions: 10, Interval: 3600, LastExecuted: time.Now().UTC(),
			}},
		}}))
		var count int
		ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, db, &count,
				`SELECT COUNT(*) FROM scheduled_query_stats WHERE host_id = ? AND scheduled_query_id = ?`, host.ID, q.ID)
		})
		assert.Equal(t, 1, count)
	})

	t.Run("team-scoped queries", func(t *testing.T) {
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "edge-team"})
		require.NoError(t, err)
		q := test.NewQuery(t, ds, &team.ID, "edge-team-q", "SELECT 1", 0, true)
		require.NoError(t, ds.SaveHostPackStats(ctx, &team.ID, host.ID, []fleet.PackStats{{
			PackName: fmt.Sprintf("team-%d", team.ID),
			QueryStats: []fleet.ScheduledQueryStats{{
				ScheduledQueryName: q.Name, PackName: fmt.Sprintf("team-%d", team.ID),
				Executions: 5, Interval: 3600, LastExecuted: time.Now().UTC(),
			}},
		}}))
	})

	t.Run("mixed global and team", func(t *testing.T) {
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "edge-mix-team"})
		require.NoError(t, err)
		gq := test.NewQuery(t, ds, nil, "edge-mix-g", "SELECT 1", 0, true)
		tq := test.NewQuery(t, ds, &team.ID, "edge-mix-t", "SELECT 1", 0, true)
		require.NoError(t, ds.SaveHostPackStats(ctx, &team.ID, host.ID, []fleet.PackStats{
			{PackName: "Global", QueryStats: []fleet.ScheduledQueryStats{{
				ScheduledQueryName: gq.Name, PackName: "Global", Executions: 1, Interval: 3600, LastExecuted: time.Now().UTC()}}},
			{PackName: fmt.Sprintf("team-%d", team.ID), QueryStats: []fleet.ScheduledQueryStats{{
				ScheduledQueryName: tq.Name, PackName: fmt.Sprintf("team-%d", team.ID), Executions: 2, Interval: 3600, LastExecuted: time.Now().UTC()}}},
		}))
	})

	t.Run("deleted query silently skipped", func(t *testing.T) {
		require.NoError(t, ds.SaveHostPackStats(ctx, host.TeamID, host.ID, []fleet.PackStats{{
			PackName: "Global",
			QueryStats: []fleet.ScheduledQueryStats{{
				ScheduledQueryName: "nonexistent-xyz", PackName: "Global",
				Executions: 1, Interval: 3600, LastExecuted: time.Now().UTC(),
			}},
		}}))
	})

	t.Run("upsert overwrites", func(t *testing.T) {
		q := test.NewQuery(t, ds, nil, "edge-upsert", "SELECT 1", 0, true)
		mk := func(ex uint64) []fleet.PackStats {
			return []fleet.PackStats{{PackName: "Global", QueryStats: []fleet.ScheduledQueryStats{{
				ScheduledQueryName: q.Name, PackName: "Global",
				Executions: ex, Interval: 3600, LastExecuted: time.Now().UTC(), WallTimeMs: 100,
			}}}}
		}
		require.NoError(t, ds.SaveHostPackStats(ctx, host.TeamID, host.ID, mk(10)))
		require.NoError(t, ds.SaveHostPackStats(ctx, host.TeamID, host.ID, mk(20)))
		var ex uint64
		ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, db, &ex,
				`SELECT executions FROM scheduled_query_stats WHERE host_id = ? AND scheduled_query_id = ?`, host.ID, q.ID)
		})
		assert.Equal(t, uint64(20), ex)
	})

	t.Run("large batch 1000", func(t *testing.T) {
		qs := makeGlobalQueryStats(t, ds, 1000, "edge-large")
		require.NoError(t, ds.SaveHostPackStats(ctx, host.TeamID, host.ID, []fleet.PackStats{{PackName: "Global", QueryStats: qs}}))
	})

	t.Run("wall_time fallback", func(t *testing.T) {
		q := test.NewQuery(t, ds, nil, "edge-wt", "SELECT 1", 0, true)
		require.NoError(t, ds.SaveHostPackStats(ctx, host.TeamID, host.ID, []fleet.PackStats{{
			PackName: "Global",
			QueryStats: []fleet.ScheduledQueryStats{{
				ScheduledQueryName: q.Name, PackName: "Global",
				Executions: 1, Interval: 3600, LastExecuted: time.Now().UTC(),
				WallTime: 5, WallTimeMs: 0,
			}},
		}}))
		var wt uint64
		ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, db, &wt,
				`SELECT wall_time FROM scheduled_query_stats WHERE host_id = ? AND scheduled_query_id = ?`, host.ID, q.ID)
		})
		assert.Equal(t, uint64(5000), wt)
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeGlobalQueryStats(t *testing.T, ds *Datastore, n int, prefix string) []fleet.ScheduledQueryStats {
	t.Helper()
	out := make([]fleet.ScheduledQueryStats, 0, n)
	for i := range n {
		q := test.NewQuery(t, ds, nil, fmt.Sprintf("%s-q%d", prefix, i), "SELECT 1", 0, true)
		out = append(out, fleet.ScheduledQueryStats{
			ScheduledQueryName: q.Name, PackName: "Global",
			AverageMemory: 8000, Executions: uint64(i + 1), Interval: 3600,
			LastExecuted: time.Now().UTC(), OutputSize: 1337,
			SystemTime: 150, UserTime: 180, WallTimeMs: 5000,
		})
	}
	return out
}

func makeHosts(t *testing.T, ds *Datastore, n int, prefix string) []*fleet.Host {
	t.Helper()
	out := make([]*fleet.Host, 0, n)
	for i := range n {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(), LabelUpdatedAt: time.Now(),
			PolicyUpdatedAt: time.Now(), SeenTime: time.Now(),
			NodeKey: new(fmt.Sprintf("%s-h%d", prefix, i)),
			UUID: fmt.Sprintf("%s-u%d", prefix, i),
			Hostname: fmt.Sprintf("%s-%d.local", prefix, i),
			PrimaryIP: fmt.Sprintf("10.%d.%d.%d", i/65536, (i/256)%256, i%256),
			PrimaryMac: fmt.Sprintf("AA-%02X-%02X-%02X-00-00", i/65536, (i/256)%256, i%256),
			Platform: "darwin",
		})
		require.NoError(t, err)
		out = append(out, h)
	}
	return out
}
