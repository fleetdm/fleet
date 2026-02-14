package mysql

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// TestBulkSetPendingDeclarationStatePerformance measures the time taken by
// BulkSetPendingMDMHostProfiles when processing declarations, with a large
// number of bystander hosts in another team. This test validates the performance
// fix that scopes declaration state queries to only affected hosts.
//
// Run with:
//
//	FLEET_PERF_TESTS=1 make run-go-tests PKG_TO_TEST="server/datastore/mysql" TESTS_TO_RUN="TestBulkSetPendingDeclarationStatePerformance"
//
// Compare results between main (full-fleet scan) and the fix branch (scoped scan).
func TestBulkSetPendingDeclarationStatePerformance(t *testing.T) {
	if os.Getenv("FLEET_PERF_TESTS") == "" {
		t.Skip("skipping performance test; set FLEET_PERF_TESTS=1 to run")
	}

	ds := CreateMySQLDS(t)
	TruncateTables(t, ds)

	ctx := t.Context()
	const numBystanderHosts = 50_000
	const batchSize = 1000

	// Create two teams: a large "bystander" team and a small "target" team
	largeTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: "large-team"})
	require.NoError(t, err)
	smallTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: "small-team"})
	require.NoError(t, err)

	// Batch-insert 50K enrolled darwin hosts into the large team.
	// Uses raw SQL for speed (ds.NewHost is too slow for 50K hosts).
	t.Logf("Creating %d bystander hosts in large team...", numBystanderHosts)
	setupStart := time.Now()

	now := time.Now().Truncate(time.Second)
	for batchStart := 0; batchStart < numBystanderHosts; batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > numBystanderHosts {
			batchEnd = numBystanderHosts
		}
		count := batchEnd - batchStart

		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			// Insert hosts
			hostValues := make([]string, 0, count)
			hostArgs := make([]any, 0, count*8)
			for i := batchStart; i < batchEnd; i++ {
				hostUUID := fmt.Sprintf("perf-host-%d", i)
				hostValues = append(hostValues, "(?, ?, ?, ?, 'darwin', ?, ?, ?, ?)")
				hostArgs = append(hostArgs,
					hostUUID,     // osquery_host_id
					hostUUID,     // uuid
					hostUUID,     // node_key
					hostUUID,     // hostname
					largeTeam.ID, // team_id
					now,          // detail_updated_at
					now,          // label_updated_at
					now,          // policy_updated_at
				)
			}
			_, err := q.ExecContext(ctx, fmt.Sprintf(`
				INSERT INTO hosts (osquery_host_id, uuid, node_key, hostname, platform, team_id, detail_updated_at, label_updated_at, policy_updated_at)
				VALUES %s`, strings.Join(hostValues, ",")),
				hostArgs...)
			if err != nil {
				return fmt.Errorf("inserting hosts batch %d: %w", batchStart, err)
			}

			// Insert nano_devices
			deviceValues := make([]string, 0, count)
			deviceArgs := make([]any, 0, count*2)
			for i := batchStart; i < batchEnd; i++ {
				hostUUID := fmt.Sprintf("perf-host-%d", i)
				deviceValues = append(deviceValues, "(?, ?, 'test', 'darwin')")
				deviceArgs = append(deviceArgs, hostUUID, hostUUID)
			}
			_, err = q.ExecContext(ctx, fmt.Sprintf(`
				INSERT INTO nano_devices (id, serial_number, authenticate, platform)
				VALUES %s`, strings.Join(deviceValues, ",")),
				deviceArgs...)
			if err != nil {
				return fmt.Errorf("inserting nano_devices batch %d: %w", batchStart, err)
			}

			// Insert nano_enrollments
			enrollValues := make([]string, 0, count)
			enrollArgs := make([]any, 0, count*3)
			for i := batchStart; i < batchEnd; i++ {
				hostUUID := fmt.Sprintf("perf-host-%d", i)
				enrollValues = append(enrollValues, "(?, ?, 'Device', 'topic', 'magic', 'token', 1, 1, ?)")
				enrollArgs = append(enrollArgs, hostUUID, hostUUID, now)
			}
			_, err = q.ExecContext(ctx, fmt.Sprintf(`
				INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled, token_update_tally, last_seen_at)
				VALUES %s`, strings.Join(enrollValues, ",")),
				enrollArgs...)
			if err != nil {
				return fmt.Errorf("inserting nano_enrollments batch %d: %w", batchStart, err)
			}

			return nil
		})
	}
	t.Logf("Created %d bystander hosts in %s", numBystanderHosts, time.Since(setupStart))

	// Create 5 enrolled hosts in the small team using standard helpers
	var smallTeamHosts []*fleet.Host
	for i := 0; i < 5; i++ {
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      fmt.Sprintf("small-team-host-%d", i),
			OsqueryHostID: ptr.String(fmt.Sprintf("small-osquery-%d", i)),
			NodeKey:       ptr.String(fmt.Sprintf("small-nodekey-%d", i)),
			UUID:          fmt.Sprintf("small-uuid-%d", i),
			Platform:      "darwin",
			TeamID:        &smallTeam.ID,
		})
		require.NoError(t, err)
		nanoEnroll(t, ds, h, false)
		smallTeamHosts = append(smallTeamHosts, h)
	}
	t.Logf("Created %d target hosts in small team", len(smallTeamHosts))

	// Create 2 declarations for the small team
	_, err = ds.BatchSetMDMProfiles(ctx, &smallTeam.ID,
		nil, // no Apple config profiles
		nil, // no Windows profiles
		[]*fleet.MDMAppleDeclaration{
			declForTest("PerfDecl1", "PerfDecl1", "payload1"),
			declForTest("PerfDecl2", "PerfDecl2", "payload2"),
		},
		nil, // no Android profiles
		nil, // no labels
	)
	require.NoError(t, err)

	// Get the declaration UUIDs
	profs, _, err := ds.ListMDMConfigProfiles(ctx, &smallTeam.ID, fleet.ListOptions{})
	require.NoError(t, err)
	var declUUIDs []string
	for _, p := range profs {
		if strings.HasPrefix(p.ProfileUUID, fleet.MDMAppleDeclarationUUIDPrefix) {
			declUUIDs = append(declUUIDs, p.ProfileUUID)
		}
	}
	require.Len(t, declUUIDs, 2, "expected 2 declaration UUIDs")
	t.Logf("Declaration UUIDs: %v", declUUIDs)

	// Measure BulkSetPendingMDMHostProfiles with declaration UUIDs.
	// This simulates the gitops flow: editTeamFromSpec → mdmAppleEditedAppleOSUpdates → BulkSetPendingMDMHostProfiles
	// On main (before fix): scans ALL 50K+ hosts
	// On fix branch: scans only the 5 small-team hosts
	t.Logf("Running BulkSetPendingMDMHostProfiles with %d bystander hosts...", numBystanderHosts)

	start := time.Now()
	updates, err := ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, declUUIDs, nil)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.True(t, updates.AppleDeclaration, "expected AppleDeclaration updates")
	t.Logf("BulkSetPendingMDMHostProfiles with %d bystander hosts took %s", numBystanderHosts, elapsed)
}
