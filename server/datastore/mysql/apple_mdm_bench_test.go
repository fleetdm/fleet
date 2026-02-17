package mysql

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nanoEnrollTB is a testing.TB-compatible version of nanoEnroll for use in benchmarks.
func nanoEnrollTB(tb testing.TB, ds *Datastore, host *fleet.Host) {
	tb.Helper()
	ctx := context.Background()
	_, err := ds.writer(ctx).Exec(
		`INSERT INTO nano_devices (id, serial_number, authenticate, platform, enroll_team_id) VALUES (?, NULLIF(?, ''), 'test', ?, ?)`,
		host.UUID, host.HardwareSerial, host.Platform, host.TeamID,
	)
	require.NoError(tb, err)

	_, err = ds.writer(ctx).Exec(`
INSERT INTO nano_enrollments
	(id, device_id, user_id, type, topic, push_magic, token_hex, token_update_tally, last_seen_at)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		host.UUID,
		host.UUID,
		nil,
		"Device",
		host.UUID+".topic",
		host.UUID+".magic",
		host.UUID,
		1,
		time.Now().Add(-2*time.Second).Truncate(time.Second),
	)
	require.NoError(tb, err)
}

// TestScopedDeclarationProcessing is a unit test that asserts the correctness
// of scoping declaration processing to specific hosts. It verifies that:
//   - When called with specific host UUIDs (scoped), only those hosts are returned
//   - When called with nil (unscoped), all hosts with changed declarations are returned
func TestScopedDeclarationProcessing(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()

	// Create two teams
	teamA, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-A-scope-test"})
	require.NoError(t, err)
	teamB, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-B-scope-test"})
	require.NoError(t, err)

	// Create 5 hosts in team A and 5 hosts in team B
	var teamAHosts []*fleet.Host
	var teamBHosts []*fleet.Host
	for i := 0; i < 5; i++ {
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      fmt.Sprintf("scope-host-a-%d", i),
			OsqueryHostID: ptr.String(fmt.Sprintf("scope-osquery-a-%d", i)),
			NodeKey:       ptr.String(fmt.Sprintf("scope-nodekey-a-%d", i)),
			UUID:          fmt.Sprintf("scope-uuid-a-%d", i),
			Platform:      "darwin",
			TeamID:        &teamA.ID,
		})
		require.NoError(t, err)
		nanoEnroll(t, ds, h, false)
		teamAHosts = append(teamAHosts, h)
	}
	for i := 0; i < 5; i++ {
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      fmt.Sprintf("scope-host-b-%d", i),
			OsqueryHostID: ptr.String(fmt.Sprintf("scope-osquery-b-%d", i)),
			NodeKey:       ptr.String(fmt.Sprintf("scope-nodekey-b-%d", i)),
			UUID:          fmt.Sprintf("scope-uuid-b-%d", i),
			Platform:      "darwin",
			TeamID:        &teamB.ID,
		})
		require.NoError(t, err)
		nanoEnroll(t, ds, h, false)
		teamBHosts = append(teamBHosts, h)
	}

	// Create 3 declarations for team A
	for i := 0; i < 3; i++ {
		_, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Identifier: fmt.Sprintf("com.fleet.scope-test.team-a.%d", i),
			Name:       fmt.Sprintf("scope-decl-a-%d", i),
			RawJSON:    []byte(fmt.Sprintf(`{"Type":"com.apple.configuration.decl.a.%d","Identifier":"com.fleet.scope-test.team-a.%d","Payload":{"ServiceType":"com.apple.service.a.%d"}}`, i, i, i)),
			TeamID:     &teamA.ID,
		})
		require.NoError(t, err)
	}

	// Create 3 declarations for team B
	for i := 0; i < 3; i++ {
		_, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Identifier: fmt.Sprintf("com.fleet.scope-test.team-b.%d", i),
			Name:       fmt.Sprintf("scope-decl-b-%d", i),
			RawJSON:    []byte(fmt.Sprintf(`{"Type":"com.apple.configuration.decl.b.%d","Identifier":"com.fleet.scope-test.team-b.%d","Payload":{"ServiceType":"com.apple.service.b.%d"}}`, i, i, i)),
			TeamID:     &teamB.ID,
		})
		require.NoError(t, err)
	}

	// At this point, all hosts should have "changed" declarations because they have
	// declarations assigned (via their team) but no host_mdm_apple_declarations rows yet.

	// Test 1: Scoped call with only team A host UUIDs should return only team A hosts
	t.Run("ScopedToTeamA", func(t *testing.T) {
		teamAUUIDs := make([]string, len(teamAHosts))
		for i, h := range teamAHosts {
			teamAUUIDs[i] = h.UUID
		}

		var changedDecls []*fleet.MDMAppleHostDeclaration
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			var err error
			changedDecls, err = mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, teamAUUIDs)
			return err
		})

		// Should only contain team A hosts
		hostUUIDSet := make(map[string]bool)
		for _, d := range changedDecls {
			hostUUIDSet[d.HostUUID] = true
		}

		// All team A hosts should be present
		for _, h := range teamAHosts {
			assert.True(t, hostUUIDSet[h.UUID], "expected team A host %s in results", h.UUID)
		}

		// No team B hosts should be present
		for _, h := range teamBHosts {
			assert.False(t, hostUUIDSet[h.UUID], "did not expect team B host %s in results", h.UUID)
		}
	})

	// Test 2: Scoped call with only team B host UUIDs should return only team B hosts
	t.Run("ScopedToTeamB", func(t *testing.T) {
		teamBUUIDs := make([]string, len(teamBHosts))
		for i, h := range teamBHosts {
			teamBUUIDs[i] = h.UUID
		}

		var changedDecls []*fleet.MDMAppleHostDeclaration
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			var err error
			changedDecls, err = mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, teamBUUIDs)
			return err
		})

		hostUUIDSet := make(map[string]bool)
		for _, d := range changedDecls {
			hostUUIDSet[d.HostUUID] = true
		}

		// All team B hosts should be present
		for _, h := range teamBHosts {
			assert.True(t, hostUUIDSet[h.UUID], "expected team B host %s in results", h.UUID)
		}

		// No team A hosts should be present
		for _, h := range teamAHosts {
			assert.False(t, hostUUIDSet[h.UUID], "did not expect team A host %s in results", h.UUID)
		}
	})

	// Test 3: Unscoped call (nil) should return hosts from BOTH teams
	t.Run("Unscoped", func(t *testing.T) {
		var changedDecls []*fleet.MDMAppleHostDeclaration
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			var err error
			changedDecls, err = mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, nil)
			return err
		})

		hostUUIDSet := make(map[string]bool)
		for _, d := range changedDecls {
			hostUUIDSet[d.HostUUID] = true
		}

		// Both team A and team B hosts should be present
		for _, h := range teamAHosts {
			assert.True(t, hostUUIDSet[h.UUID], "expected team A host %s in unscoped results", h.UUID)
		}
		for _, h := range teamBHosts {
			assert.True(t, hostUUIDSet[h.UUID], "expected team B host %s in unscoped results", h.UUID)
		}
	})

	// Test 4: Empty slice should behave like nil (all hosts)
	t.Run("EmptySlice", func(t *testing.T) {
		var changedDecls []*fleet.MDMAppleHostDeclaration
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			var err error
			changedDecls, err = mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, []string{})
			return err
		})

		hostUUIDSet := make(map[string]bool)
		for _, d := range changedDecls {
			hostUUIDSet[d.HostUUID] = true
		}

		// Both team A and team B hosts should be present
		for _, h := range teamAHosts {
			assert.True(t, hostUUIDSet[h.UUID], "expected team A host %s in empty-slice results", h.UUID)
		}
		for _, h := range teamBHosts {
			assert.True(t, hostUUIDSet[h.UUID], "expected team B host %s in empty-slice results", h.UUID)
		}
	})

	// Test 5: Scoped call correctly counts declarations per host
	t.Run("ScopedDeclarationCount", func(t *testing.T) {
		teamAUUIDs := make([]string, len(teamAHosts))
		for i, h := range teamAHosts {
			teamAUUIDs[i] = h.UUID
		}

		var changedDecls []*fleet.MDMAppleHostDeclaration
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			var err error
			changedDecls, err = mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, teamAUUIDs)
			return err
		})

		// Each team A host should have 3 declarations (all install operations)
		declCountPerHost := make(map[string]int)
		for _, d := range changedDecls {
			declCountPerHost[d.HostUUID]++
		}
		for _, h := range teamAHosts {
			assert.Equal(t, 3, declCountPerHost[h.UUID],
				"expected 3 declarations for team A host %s, got %d", h.UUID, declCountPerHost[h.UUID])
		}
	})

	// Test 6: End-to-end through mdmAppleBatchSetHostDeclarationStateDB with scoped hosts
	t.Run("BatchSetWithScope", func(t *testing.T) {
		teamAUUIDs := make([]string, len(teamAHosts))
		for i, h := range teamAHosts {
			teamAUUIDs[i] = h.UUID
		}

		var uuids []string
		err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			var err error
			uuids, _, err = mdmAppleBatchSetHostDeclarationStateDB(ctx, tx, 1000, &fleet.MDMDeliveryPending, teamAUUIDs)
			return err
		})
		require.NoError(t, err)

		// Sort for stable comparison
		sort.Strings(uuids)
		sort.Strings(teamAUUIDs)
		assert.Equal(t, teamAUUIDs, uuids, "scoped batch set should return only team A host UUIDs")

		// Verify team A hosts have pending declarations
		for _, h := range teamAHosts {
			var count int
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(ctx, q, &count,
					`SELECT COUNT(*) FROM host_mdm_apple_declarations WHERE host_uuid = ? AND status = 'pending'`,
					h.UUID)
			})
			assert.Equal(t, 3, count, "team A host %s should have 3 pending declarations", h.UUID)
		}

		// Verify team B hosts have NO pending declarations (they were not in scope)
		for _, h := range teamBHosts {
			var count int
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(ctx, q, &count,
					`SELECT COUNT(*) FROM host_mdm_apple_declarations WHERE host_uuid = ?`,
					h.UUID)
			})
			assert.Equal(t, 0, count, "team B host %s should have 0 declarations", h.UUID)
		}
	})
}

// createBenchmarkTeamWithHosts sets up a team with hosts enrolled in MDM and
// declarations assigned. Returns the host UUIDs for that team.
func createBenchmarkTeamWithHosts(tb testing.TB, ds *Datastore, teamName string, numHosts, numDecls int) (teamID uint, hostUUIDs []string) {
	tb.Helper()
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: teamName})
	require.NoError(tb, err)
	teamID = team.ID

	hostUUIDs = make([]string, numHosts)
	for i := 0; i < numHosts; i++ {
		hostUUID := fmt.Sprintf("%s-uuid-%d", teamName, i)
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      fmt.Sprintf("%s-host-%d", teamName, i),
			OsqueryHostID: ptr.String(fmt.Sprintf("%s-osquery-%d", teamName, i)),
			NodeKey:       ptr.String(fmt.Sprintf("%s-nodekey-%d", teamName, i)),
			UUID:          hostUUID,
			Platform:      "darwin",
			TeamID:        &team.ID,
		})
		require.NoError(tb, err)
		nanoEnrollTB(tb, ds, h)
		hostUUIDs[i] = hostUUID
	}

	for i := 0; i < numDecls; i++ {
		_, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Identifier: fmt.Sprintf("com.fleet.bench.%s.%d", teamName, i),
			Name:       fmt.Sprintf("%s-decl-%d", teamName, i),
			RawJSON:    []byte(fmt.Sprintf(`{"Type":"com.apple.configuration.decl.%d","Identifier":"com.fleet.bench.%s.%d","Payload":{"ServiceType":"com.apple.svc.%d"}}`, i, teamName, i, i)),
			TeamID:     &team.ID,
		})
		require.NoError(tb, err)
	}

	return teamID, hostUUIDs
}

// TestScopedVsUnscopedPerformance is an explicit test (not just a benchmark)
// that demonstrates the performance difference at scale. It creates many teams
// with many hosts to simulate a production scenario, then measures wall-clock
// time for scoped vs unscoped declaration processing.
//
// The original bug (#39921) was: with 70k hosts and 30 declarations, the
// unscoped 4-way UNION query in generateDesiredStateQuery() processes
// N_hosts * N_declarations rows across 4 UNION arms. With scoping, only
// the affected team's hosts are in the query, reducing rows dramatically.
//
// Run with:
//
//	MYSQL_TEST=1 FLEET_MYSQL_ADDRESS=127.0.0.1:3307 FLEET_MYSQL_DATABASE=fleet \
//	  go test -run TestScopedVsUnscopedPerformance -v -count=1 -timeout 300s \
//	  ./server/datastore/mysql/
func TestScopedVsUnscopedPerformance(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()

	// Create a "target" team (the one being modified — small) and many
	// "bystander" teams (unaffected — they make the unscoped query slow).
	//
	// Production scenario: 1 team modified, 70k total hosts across many teams.
	// Test scenario: 1 target team (50 hosts), 19 bystander teams (50 hosts each)
	// = 1000 total hosts, 200 declarations (10 per team).
	//
	// The 4-way UNION in generateDesiredStateQuery joins every host against
	// every declaration in their team. Unscoped: all 1000 hosts evaluated.
	// Scoped: only 50 hosts evaluated.

	const numBystanderTeams = 19
	const hostsPerTeam = 50
	const declsPerTeam = 10

	// Create the target team
	_, targetUUIDs := createBenchmarkTeamWithHosts(t, ds, "target", hostsPerTeam, declsPerTeam)

	// Create bystander teams (these make the unscoped path slow)
	for i := 0; i < numBystanderTeams; i++ {
		createBenchmarkTeamWithHosts(t, ds, fmt.Sprintf("bystander-%02d", i), hostsPerTeam, declsPerTeam)
	}

	totalHosts := hostsPerTeam * (1 + numBystanderTeams) // 1000
	t.Logf("Setup complete: %d total hosts across %d teams, %d declarations per team",
		totalHosts, 1+numBystanderTeams, declsPerTeam)

	// Measure scoped (only target team's 50 hosts)
	var scopedResult []*fleet.MDMAppleHostDeclaration
	scopedStart := time.Now()
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		var err error
		scopedResult, err = mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, targetUUIDs)
		return err
	})
	scopedDuration := time.Since(scopedStart)

	// Measure unscoped (all 1000 hosts — the old behavior)
	var unscopedResult []*fleet.MDMAppleHostDeclaration
	unscopedStart := time.Now()
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		var err error
		unscopedResult, err = mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, nil)
		return err
	})
	unscopedDuration := time.Since(unscopedStart)

	// Verify correctness
	scopedHosts := make(map[string]bool)
	for _, d := range scopedResult {
		scopedHosts[d.HostUUID] = true
	}
	unscopedHosts := make(map[string]bool)
	for _, d := range unscopedResult {
		unscopedHosts[d.HostUUID] = true
	}

	// Scoped should have exactly the target team's hosts
	assert.Equal(t, hostsPerTeam, len(scopedHosts),
		"scoped result should have exactly %d unique hosts (target team only)", hostsPerTeam)

	// Unscoped should have all hosts
	assert.Equal(t, totalHosts, len(unscopedHosts),
		"unscoped result should have all %d hosts", totalHosts)

	// Scoped should be a subset of unscoped
	for uuid := range scopedHosts {
		assert.True(t, unscopedHosts[uuid],
			"scoped host %s should also appear in unscoped results", uuid)
	}

	// Scoped should return declsPerTeam declarations per host
	scopedDeclsPerHost := make(map[string]int)
	for _, d := range scopedResult {
		scopedDeclsPerHost[d.HostUUID]++
	}
	for _, uuid := range targetUUIDs {
		assert.Equal(t, declsPerTeam, scopedDeclsPerHost[uuid],
			"target host %s should have %d declarations", uuid, declsPerTeam)
	}

	// Log the performance comparison
	t.Logf("")
	t.Logf("=== PERFORMANCE COMPARISON ===")
	t.Logf("Scoped   (%3d hosts): %v  (%d declaration results)",
		hostsPerTeam, scopedDuration, len(scopedResult))
	t.Logf("Unscoped (%3d hosts): %v  (%d declaration results)",
		totalHosts, unscopedDuration, len(unscopedResult))
	t.Logf("Speedup: %.1fx", float64(unscopedDuration)/float64(scopedDuration))
	t.Logf("")
	t.Logf("The scoped query processes %d hosts × %d declarations = %d row candidates",
		hostsPerTeam, declsPerTeam, hostsPerTeam*declsPerTeam)
	t.Logf("The unscoped query processes %d hosts × %d declarations = %d row candidates",
		totalHosts, declsPerTeam*20, totalHosts*declsPerTeam)
	t.Logf("Row reduction: %.0fx fewer rows in scoped path",
		float64(totalHosts*declsPerTeam)/float64(hostsPerTeam*declsPerTeam))

	// The scoped path should be faster than unscoped at this scale
	// (1000 hosts is enough to see the difference)
	assert.Less(t, scopedDuration, unscopedDuration,
		"scoped (%v) should be faster than unscoped (%v) at %d total hosts",
		scopedDuration, unscopedDuration, totalHosts)
}

// BenchmarkDeclarationProcessingAtScale benchmarks scoped vs unscoped
// declaration processing at multiple host counts to show how the performance
// gap widens with scale.
//
// Run with:
//
//	MYSQL_TEST=1 FLEET_MYSQL_ADDRESS=127.0.0.1:3307 FLEET_MYSQL_DATABASE=fleet \
//	  go test -bench BenchmarkDeclarationProcessingAtScale -benchtime 5x -run ^$ -timeout 600s \
//	  ./server/datastore/mysql/
func BenchmarkDeclarationProcessingAtScale(b *testing.B) {
	ds := CreateMySQLDS(b)
	defer TruncateTables(b, ds)

	const declsPerTeam = 10

	// Scale tiers: each adds more bystander teams while the target team stays small.
	// This simulates the production scenario where modifying one team's declarations
	// shouldn't require scanning hosts from all other teams.
	tiers := []struct {
		name            string
		numBystanderTeams int
		hostsPerTeam    int
	}{
		{"500hosts_10teams", 9, 50},
		{"1000hosts_20teams", 19, 50},
		{"2000hosts_20teams", 19, 100},
	}

	for _, tier := range tiers {
		b.Run(tier.name, func(b *testing.B) {
			// Clean slate for each tier
			TruncateTables(b, ds)

			// Create target team
			_, targetUUIDs := createBenchmarkTeamWithHosts(b, ds, fmt.Sprintf("target-%s", tier.name), tier.hostsPerTeam, declsPerTeam)

			// Create bystander teams
			for i := 0; i < tier.numBystanderTeams; i++ {
				createBenchmarkTeamWithHosts(b, ds, fmt.Sprintf("bystander-%s-%02d", tier.name, i), tier.hostsPerTeam, declsPerTeam)
			}

			totalHosts := tier.hostsPerTeam * (1 + tier.numBystanderTeams)
			ctx := context.Background()

			b.Run(fmt.Sprintf("Scoped_%dhosts", tier.hostsPerTeam), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					ExecAdhocSQL(b, ds, func(q sqlx.ExtContext) error {
						_, err := mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, targetUUIDs)
						return err
					})
				}
			})

			b.Run(fmt.Sprintf("Unscoped_%dhosts", totalHosts), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					ExecAdhocSQL(b, ds, func(q sqlx.ExtContext) error {
						_, err := mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, nil)
						return err
					})
				}
			})
		})
	}
}
