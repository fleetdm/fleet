package mysql

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

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

			// Time 5 iterations.
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

// TestConcurrentStatsIngestion simulates many hosts concurrently ingesting
// stats for 500 reports, reproducing the connection pool exhaustion and
// goroutine pile-up described in #54969.
func TestConcurrentStatsIngestion(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()
	const numQueries = 500
	const numHosts = 50
	const concurrency = 20

	// Create 500 global queries.
	queryStats := make([]fleet.ScheduledQueryStats, 0, numQueries)
	for i := range numQueries {
		q := test.NewQuery(t, ds, nil, fmt.Sprintf("conc-query-%d", i), "SELECT 1", 0, true)
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

	// Create hosts.
	hosts := make([]*fleet.Host, 0, numHosts)
	for i := range numHosts {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         new(fmt.Sprintf("conc-host-%d", i)),
			UUID:            fmt.Sprintf("conc-uuid-%d", i),
			Hostname:        fmt.Sprintf("conc-%d.local", i),
			PrimaryIP:       fmt.Sprintf("10.0.%d.%d", i/256, i%256),
			PrimaryMac:      fmt.Sprintf("AA-BB-CC-%02X-%02X-%02X", i/256, i%256, i%256),
			Platform:        "darwin",
		})
		require.NoError(t, err)
		hosts = append(hosts, h)
	}

	packStats := []fleet.PackStats{{
		PackName:   "Global",
		QueryStats: queryStats,
	}}

	// Fire all hosts concurrently with limited concurrency to simulate pool pressure.
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
