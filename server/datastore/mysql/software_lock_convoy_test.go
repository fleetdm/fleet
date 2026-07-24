package mysql

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// TestSoftwareTitlesInsertIgnoreLockConvoy reproduces the lock convoy described in #48719.
//
// Setup: N hosts all report the SAME software catalog (homogeneous fleet, like imaged
// corporate Windows machines). The software_titles table starts empty (cold start).
// All hosts race to INSERT IGNORE the same titles concurrently.
//
// Expected: With the current code, concurrent INSERT IGNORE statements on the same
// unique-index rows serialize and cause high contention. This test measures timing
// to confirm the convoy is observable even at modest concurrency.
func TestSoftwareTitlesInsertIgnoreLockConvoy(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	const (
		hostCount     = 50 // concurrent hosts
		softwareCount = 100 // software items per host (all identical across hosts)
	)

	// Create hosts
	hosts := make([]*fleet.Host, hostCount)
	for i := 0; i < hostCount; i++ {
		h, err := ds.NewHost(ctx, &fleet.Host{
			OsqueryHostID:   ptr.String(fmt.Sprintf("convoy-host-%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("convoy-key-%d", i)),
			Platform:        "windows",
			Hostname:        fmt.Sprintf("convoy-host-%d", i),
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
		})
		require.NoError(t, err)
		hosts[i] = h
	}

	// Build a SINGLE software catalog shared by ALL hosts (homogeneous fleet).
	// This is the key condition for the lock convoy: every host tries to INSERT IGNORE
	// the same software_titles rows.
	sharedSoftware := make([]fleet.Software, softwareCount)
	for i := 0; i < softwareCount; i++ {
		sharedSoftware[i] = fleet.Software{
			Name:    fmt.Sprintf("ConvoyApp %d", i),
			Version: "1.0.0",
			Source:  "programs",
		}
	}

	// --- Cold-start convoy: all hosts ingest simultaneously with empty software_titles ---
	t.Log("Starting cold-start convoy test...")
	t.Logf("  Hosts: %d, Software items per host: %d", hostCount, softwareCount)

	var (
		g          errgroup.Group
		maxElapsed atomic.Int64
		totalMs    atomic.Int64
		ready      = make(chan struct{}) // barrier to synchronize start
	)

	for i := 0; i < hostCount; i++ {
		hostID := hosts[i].ID
		g.Go(func() error {
			<-ready // wait for all goroutines to be ready
			start := time.Now()
			_, err := ds.UpdateHostSoftware(ctx, hostID, sharedSoftware)
			elapsed := time.Since(start)
			ms := elapsed.Milliseconds()
			totalMs.Add(ms)
			for {
				old := maxElapsed.Load()
				if ms <= old || maxElapsed.CompareAndSwap(old, ms) {
					break
				}
			}
			if err != nil {
				return fmt.Errorf("host %d: %w", hostID, err)
			}
			return nil
		})
	}

	start := time.Now()
	close(ready) // release all goroutines at once
	err := g.Wait()
	wallTime := time.Since(start)

	require.NoError(t, err)

	t.Logf("  Cold-start results:")
	t.Logf("    Wall time: %s", wallTime)
	t.Logf("    Max single-host ingestion: %dms", maxElapsed.Load())
	t.Logf("    Avg per-host ingestion: %dms", totalMs.Load()/int64(hostCount))

	// Verify all titles were created
	var titleCount int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &titleCount, `SELECT COUNT(*) FROM software_titles WHERE source = 'programs'`)
	})
	t.Logf("    Software titles created: %d (expected %d)", titleCount, softwareCount)
	require.Equal(t, softwareCount, titleCount)

	// --- Steady-state: re-ingest same software (should be fast, read-only path) ---
	t.Log("Starting steady-state re-ingestion test...")
	maxElapsed.Store(0)
	totalMs.Store(0)
	ready2 := make(chan struct{})

	var g2 errgroup.Group
	for i := 0; i < hostCount; i++ {
		hostID := hosts[i].ID
		g2.Go(func() error {
			<-ready2
			start := time.Now()
			_, err := ds.UpdateHostSoftware(ctx, hostID, sharedSoftware)
			elapsed := time.Since(start)
			ms := elapsed.Milliseconds()
			totalMs.Add(ms)
			for {
				old := maxElapsed.Load()
				if ms <= old || maxElapsed.CompareAndSwap(old, ms) {
					break
				}
			}
			if err != nil {
				return fmt.Errorf("host %d: %w", hostID, err)
			}
			return nil
		})
	}

	start2 := time.Now()
	close(ready2)
	err = g2.Wait()
	wallTime2 := time.Since(start2)

	require.NoError(t, err)

	t.Logf("  Steady-state results:")
	t.Logf("    Wall time: %s", wallTime2)
	t.Logf("    Max single-host ingestion: %dms", maxElapsed.Load())
	t.Logf("    Avg per-host ingestion: %dms", totalMs.Load()/int64(hostCount))

	// The cold-start should be significantly slower than steady-state due to lock contention
	t.Logf("\n  Convoy factor (cold wall / steady wall): %.1fx", float64(wallTime.Milliseconds())/float64(wallTime2.Milliseconds()))
}

// TestHostSoftwareInstalledPathsDeleteExplosion reproduces #49805.
//
// A host with many installed paths gets re-enrolled or its software changes significantly,
// triggering a DELETE FROM host_software_installed_paths WHERE id IN (thousands of IDs)
// in a single unbatched statement.
func TestHostSoftwareInstalledPathsDeleteExplosion(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	// Create a host
	host := test.NewHost(t, ds, "delete-explosion-host", "", "de-key", "de-uuid", time.Now())

	// Insert a large number of software items, each with an installed path
	const softwareCount = 500 // a more modest number than 30k for local testing
	software := make([]fleet.Software, softwareCount)
	for i := 0; i < softwareCount; i++ {
		software[i] = fleet.Software{
			Name:    fmt.Sprintf("DeleteTestApp %d", i),
			Version: "1.0.0",
			Source:  "apps",
		}
	}

	// First ingestion: establish software
	_, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	// Get the software IDs that were created
	var swIDs []struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &swIDs,
			`SELECT id, name FROM software WHERE name LIKE 'DeleteTestApp%' AND source = 'apps'`)
	})
	t.Logf("Created %d software entries", len(swIDs))

	// Directly insert installed paths to build up the table
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		for _, sw := range swIDs {
			_, err := q.ExecContext(ctx,
				`INSERT INTO host_software_installed_paths (host_id, software_id, installed_path) VALUES (?, ?, ?)`,
				host.ID, sw.ID, fmt.Sprintf("/Applications/%s.app", sw.Name))
			if err != nil {
				return err
			}
		}
		return nil
	})

	// Verify the paths are there
	var pathCount int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &pathCount,
			`SELECT COUNT(*) FROM host_software_installed_paths WHERE host_id = ?`, host.ID)
	})
	t.Logf("Installed paths for host: %d", pathCount)
	require.Equal(t, len(swIDs), pathCount)

	// Now simulate a "full replacement" by reporting all-new software with no overlap.
	// This causes ALL existing paths to be deleted in one shot.
	newSoftware := make([]fleet.Software, softwareCount)
	for i := 0; i < softwareCount; i++ {
		newSoftware[i] = fleet.Software{
			Name:    fmt.Sprintf("ReplacementApp %d", i),
			Version: "2.0.0",
			Source:  "apps",
		}
	}

	// This should trigger a massive DELETE of all old installed paths
	start := time.Now()
	_, err = ds.UpdateHostSoftware(ctx, host.ID, newSoftware)
	elapsed := time.Since(start)
	require.NoError(t, err)
	t.Logf("Full software replacement took: %s", elapsed)

	// Now test with concurrent hosts doing the same thing
	t.Log("Testing concurrent large deletes...")
	const concurrentHosts = 10
	var wg sync.WaitGroup
	wg.Add(concurrentHosts)

	for i := 0; i < concurrentHosts; i++ {
		go func(idx int) {
			defer wg.Done()
			h := test.NewHost(t, ds, fmt.Sprintf("concurrent-del-%d", idx), "", fmt.Sprintf("cd-key-%d", idx), fmt.Sprintf("cd-uuid-%d", idx), time.Now())

			// First, ingest software with paths
			_, err := ds.UpdateHostSoftware(ctx, h.ID, software)
			if err != nil {
				t.Logf("Host %d initial ingest error: %v", idx, err)
				return
			}

			// Insert paths
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				for _, sw := range swIDs {
					_, err := q.ExecContext(ctx,
						`INSERT IGNORE INTO host_software_installed_paths (host_id, software_id, installed_path) VALUES (?, ?, ?)`,
						h.ID, sw.ID, fmt.Sprintf("/Applications/%s.app", sw.Name))
					if err != nil {
						return err
					}
				}
				return nil
			})

			// Now replace everything
			start := time.Now()
			_, err = ds.UpdateHostSoftware(ctx, h.ID, newSoftware)
			elapsed := time.Since(start)
			t.Logf("  Host %d replacement took: %s", idx, elapsed)
			if err != nil {
				t.Logf("  Host %d replacement error: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()
}
