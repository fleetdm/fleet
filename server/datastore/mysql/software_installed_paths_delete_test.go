package mysql

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestHostSoftwareInstalledPathsDeleteExplosion(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	// Create a host
	host := test.NewHost(t, ds, "delete-explosion-host", "", "de-key", "de-uuid", time.Now())

	// Insert a large number of software items, each with an installed path
	const softwareCount = 500 // a more modest number than 30k for local testing
	software := make([]fleet.Software, softwareCount)
	for i := range softwareCount {
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
	for i := range softwareCount {
		newSoftware[i] = fleet.Software{
			Name:    fmt.Sprintf("ReplacementApp %d", i),
			Version: "2.0.0",
			Source:  "apps",
		}
	}

	// This should trigger a massive DELETE of all old installed paths
	startDel := time.Now()
	_, err = ds.UpdateHostSoftware(ctx, host.ID, newSoftware)
	elapsed := time.Since(startDel)
	require.NoError(t, err)
	t.Logf("Full software replacement took: %s", elapsed)

	// Now test with concurrent hosts doing the same thing
	t.Log("Testing concurrent large deletes...")
	const concurrentHosts = 10
	var wg sync.WaitGroup
	wg.Add(concurrentHosts)

	// Create hosts and insert paths synchronously to avoid require.*/ExecAdhocSQL panics from goroutines.
	concurrentTestHosts := make([]*fleet.Host, concurrentHosts)
	for i := range concurrentHosts {
		concurrentTestHosts[i] = test.NewHost(t, ds, fmt.Sprintf("concurrent-del-%d", i), "", fmt.Sprintf("cd-key-%d", i), fmt.Sprintf("cd-uuid-%d", i), time.Now())

		swCopy := slices.Clone(software)
		_, err := ds.UpdateHostSoftware(ctx, concurrentTestHosts[i].ID, swCopy)
		require.NoError(t, err)

		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			for _, sw := range swIDs {
				_, err := q.ExecContext(ctx,
					`INSERT IGNORE INTO host_software_installed_paths (host_id, software_id, installed_path) VALUES (?, ?, ?)`,
					concurrentTestHosts[i].ID, sw.ID, fmt.Sprintf("/Applications/%s.app", sw.Name))
				if err != nil {
					return err
				}
			}
			return nil
		})
	}

	for i := range concurrentHosts {
		go func(idx int) {
			defer wg.Done()
			h := concurrentTestHosts[idx]
			newSwCopy := slices.Clone(newSoftware)

			startReplace := time.Now()
			_, err := ds.UpdateHostSoftware(ctx, h.ID, newSwCopy)
			elapsedReplace := time.Since(startReplace)
			t.Logf("  Host %d replacement took: %s", idx, elapsedReplace)
			if err != nil {
				t.Errorf("  Host %d replacement error: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()
}
