package mysql

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
	for i := range 5 {
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
	for i := range 5 {
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
	for i := range 3 {
		_, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Identifier: fmt.Sprintf("com.fleet.scope-test.team-a.%d", i),
			Name:       fmt.Sprintf("scope-decl-a-%d", i),
			RawJSON:    fmt.Appendf(nil, `{"Type":"com.apple.configuration.decl.a.%d","Identifier":"com.fleet.scope-test.team-a.%d","Payload":{"ServiceType":"com.apple.service.a.%d"}}`, i, i, i),
			TeamID:     &teamA.ID,
		})
		require.NoError(t, err)
	}

	// Create 3 declarations for team B
	for i := range 3 {
		_, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Identifier: fmt.Sprintf("com.fleet.scope-test.team-b.%d", i),
			Name:       fmt.Sprintf("scope-decl-b-%d", i),
			RawJSON:    fmt.Appendf(nil, `{"Type":"com.apple.configuration.decl.b.%d","Identifier":"com.fleet.scope-test.team-b.%d","Payload":{"ServiceType":"com.apple.service.b.%d"}}`, i, i, i),
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
	for i := range numHosts {
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

	for i := range numDecls {
		_, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Identifier: fmt.Sprintf("com.fleet.bench.%s.%d", teamName, i),
			Name:       fmt.Sprintf("%s-decl-%d", teamName, i),
			RawJSON:    fmt.Appendf(nil, `{"Type":"com.apple.configuration.decl.%d","Identifier":"com.fleet.bench.%s.%d","Payload":{"ServiceType":"com.apple.svc.%d"}}`, i, teamName, i, i),
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
	for i := range numBystanderTeams {
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

// bulkInsertHosts inserts hosts in batches of 1000 using multi-row INSERT for speed.
// Returns the host UUIDs created.
func bulkInsertHosts(tb testing.TB, ds *Datastore, teamID uint, prefix string, count int) []string {
	tb.Helper()
	ctx := context.Background()
	now := time.Now()

	uuids := make([]string, count)
	const batchSize = 1000

	for start := 0; start < count; start += batchSize {
		end := min(start+batchSize, count)
		batch := end - start

		// Build multi-row INSERT for hosts
		hostVals := make([]string, 0, batch)
		hostArgs := make([]any, 0, batch*10)
		for i := start; i < end; i++ {
			uuid := fmt.Sprintf("%s-uuid-%d", prefix, i)
			uuids[i] = uuid
			hostVals = append(hostVals, "(?, ?, ?, ?, ?, ?, ?, ?)")
			hostArgs = append(hostArgs,
				fmt.Sprintf("%s-osquery-%d", prefix, i), // osquery_host_id
				now, // detail_updated_at
				now, // label_updated_at
				now, // policy_updated_at
				fmt.Sprintf("%s-nodekey-%d", prefix, i), // node_key
				uuid,     // uuid
				"darwin", // platform
				teamID,   // team_id
			)
		}

		stmt := "INSERT INTO hosts (osquery_host_id, detail_updated_at, label_updated_at, policy_updated_at, node_key, uuid, platform, team_id) VALUES " +
			strings.Join(hostVals, ",")
		_, err := ds.writer(ctx).ExecContext(ctx, stmt, hostArgs...)
		require.NoError(tb, err)

		// Get the inserted host IDs for host_seen_times
		var insertedHosts []struct {
			ID   uint   `db:"id"`
			UUID string `db:"uuid"`
		}
		batchUUIDs := uuids[start:end]
		query, args, err := sqlx.In("SELECT id, uuid FROM hosts WHERE uuid IN (?)", batchUUIDs)
		require.NoError(tb, err)
		err = sqlx.SelectContext(ctx, ds.reader(ctx), &insertedHosts, query, args...)
		require.NoError(tb, err)

		// Bulk insert host_seen_times
		seenVals := make([]string, 0, len(insertedHosts))
		seenArgs := make([]any, 0, len(insertedHosts)*2)
		for _, h := range insertedHosts {
			seenVals = append(seenVals, "(?, ?)")
			seenArgs = append(seenArgs, h.ID, now)
		}
		_, err = ds.writer(ctx).ExecContext(ctx,
			"INSERT INTO host_seen_times (host_id, seen_time) VALUES "+strings.Join(seenVals, ","),
			seenArgs...)
		require.NoError(tb, err)

		// Bulk insert nano_devices
		nanoDevVals := make([]string, 0, len(insertedHosts))
		nanoDevArgs := make([]any, 0, len(insertedHosts)*3)
		for _, h := range insertedHosts {
			nanoDevVals = append(nanoDevVals, "(?, 'test', ?, ?)")
			nanoDevArgs = append(nanoDevArgs, h.UUID, "darwin", teamID)
		}
		_, err = ds.writer(ctx).ExecContext(ctx,
			"INSERT INTO nano_devices (id, authenticate, platform, enroll_team_id) VALUES "+strings.Join(nanoDevVals, ","),
			nanoDevArgs...)
		require.NoError(tb, err)

		// Bulk insert nano_enrollments
		nanoEnrVals := make([]string, 0, len(insertedHosts))
		nanoEnrArgs := make([]any, 0, len(insertedHosts)*5)
		enrollTime := now.Add(-2 * time.Second).Truncate(time.Second)
		for _, h := range insertedHosts {
			nanoEnrVals = append(nanoEnrVals, "(?, ?, 'Device', ?, ?, ?, 1, ?)")
			nanoEnrArgs = append(nanoEnrArgs,
				h.UUID,             // id
				h.UUID,             // device_id
				h.UUID+".topic",    // topic
				h.UUID+".magic",    // push_magic
				h.UUID,             // token_hex
				enrollTime,         // last_seen_at
			)
		}
		_, err = ds.writer(ctx).ExecContext(ctx,
			"INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, token_update_tally, last_seen_at) VALUES "+strings.Join(nanoEnrVals, ","),
			nanoEnrArgs...)
		require.NoError(tb, err)
	}

	return uuids
}

// TestReproduceIssue39921 reproduces the exact production scenario reported in
// issue #39921: ~70k hosts and ~30 declarations causing 107+ second API responses
// on POST /api/latest/fleet/spec/teams.
//
// The root cause was that mdmAppleBatchSetHostDeclarationStateDB computed
// desired state for ALL hosts regardless of which team was modified. The
// 4-way UNION in generateDesiredStateQuery() joins every host against every
// declaration in their team, producing N_hosts * N_declarations row candidates.
//
// This test reproduces the EXACT scenario from issue #39921:
//   - 70,000 hosts (50 teams × 1,400 hosts each)
//   - 30 declarations per team (1,500 total)
//   - One team modified via GitOps (POST /api/latest/fleet/spec/teams)
//   - Reported: 107 second API response time
//
// It compares:
//   - Scoped (fix): process only the target team's 1,400 hosts
//   - Unscoped (bug): process ALL 70,000 hosts
//
// Run with:
//
//	MYSQL_TEST=1 FLEET_MYSQL_ADDRESS=127.0.0.1:3307 FLEET_MYSQL_DATABASE=fleet \
//	  go test -run TestReproduceIssue39921 -v -count=1 -timeout 1200s \
//	  ./server/datastore/mysql/
func TestReproduceIssue39921(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)

	ctx := context.Background()

	// Exact reproduction of issue #39921:
	// - ~70k hosts total across multiple teams
	// - ~30 profiles/declarations per team
	// - POST /api/latest/fleet/spec/teams modifies ONE team via GitOps
	// - The query scans ALL 70k hosts instead of just the affected team
	// - Reported: 107 second response time
	//
	// We reproduce the EXACT numbers: 70,000 hosts across 50 teams (1,400
	// hosts per team), with 30 declarations per team. One team (the target)
	// is being modified via GitOps — the fix ensures only that team's 1,400
	// hosts are scanned instead of all 70,000.

	const (
		targetTeamHosts    = 1400
		numBystanderTeams  = 49
		bystanderHosts     = 1400 // per team
		declsPerTeam       = 30   // matches "~30 profiles" from the issue
	)

	totalHosts := targetTeamHosts + numBystanderTeams*bystanderHosts
	totalDecls := (1 + numBystanderTeams) * declsPerTeam

	t.Logf("Setting up %d hosts: target team (%d hosts) + %d bystander teams (%d hosts each), %d declarations/team (%d total)...",
		totalHosts, targetTeamHosts, numBystanderTeams, bystanderHosts, declsPerTeam, totalDecls)
	setupStart := time.Now()

	// Create target team (the one being modified via GitOps)
	targetTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: "repro-target"})
	require.NoError(t, err)
	targetUUIDs := bulkInsertHosts(t, ds, targetTeam.ID, "repro-target", targetTeamHosts)
	for d := range declsPerTeam {
		_, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Identifier: fmt.Sprintf("com.fleet.repro.target.%d", d),
			Name:       fmt.Sprintf("repro-target-decl-%d", d),
			RawJSON:    fmt.Appendf(nil, `{"Type":"com.apple.configuration.decl.%d","Identifier":"com.fleet.repro.target.%d","Payload":{"ServiceType":"com.apple.svc.%d"}}`, d, d, d),
			TeamID:     &targetTeam.ID,
		})
		require.NoError(t, err)
	}

	// Create bystander teams (these create the load that the old code scanned needlessly)
	for i := range numBystanderTeams {
		teamName := fmt.Sprintf("repro-bystander-%02d", i)
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: teamName})
		require.NoError(t, err)
		bulkInsertHosts(t, ds, team.ID, teamName, bystanderHosts)
		for d := range declsPerTeam {
			_, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
				Identifier: fmt.Sprintf("com.fleet.repro.%s.%d", teamName, d),
				Name:       fmt.Sprintf("%s-decl-%d", teamName, d),
				RawJSON:    fmt.Appendf(nil, `{"Type":"com.apple.configuration.decl.%d","Identifier":"com.fleet.repro.%s.%d","Payload":{"ServiceType":"com.apple.svc.%d"}}`, d, teamName, d, d),
				TeamID:     &team.ID,
			})
			require.NoError(t, err)
		}
	}

	t.Logf("Setup took %v", time.Since(setupStart))

	// === SCOPED PATH (the fix) ===
	// Only process the target team's hosts — this is what happens after the fix
	// when BulkSetPendingMDMHostProfiles passes hostUUIDs from the team lookup.
	var scopedResult []*fleet.MDMAppleHostDeclaration
	scopedStart := time.Now()
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		var err error
		scopedResult, err = mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, targetUUIDs)
		return err
	})
	scopedDuration := time.Since(scopedStart)

	// === UNSCOPED PATH (the bug) ===
	// Process ALL hosts — this is what happened before the fix when
	// mdmAppleBatchSetHostDeclarationStateDB was called with nil hostUUIDs.
	var unscopedResult []*fleet.MDMAppleHostDeclaration
	unscopedStart := time.Now()
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		var err error
		unscopedResult, err = mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, nil)
		return err
	})
	unscopedDuration := time.Since(unscopedStart)

	// === VERIFY CORRECTNESS ===
	scopedHosts := make(map[string]bool)
	for _, d := range scopedResult {
		scopedHosts[d.HostUUID] = true
	}
	unscopedHosts := make(map[string]bool)
	for _, d := range unscopedResult {
		unscopedHosts[d.HostUUID] = true
	}

	assert.Equal(t, targetTeamHosts, len(scopedHosts),
		"scoped: expected %d unique hosts (target team), got %d", targetTeamHosts, len(scopedHosts))
	assert.Equal(t, totalHosts, len(unscopedHosts),
		"unscoped: expected %d unique hosts (all teams), got %d", totalHosts, len(unscopedHosts))

	// Scoped returns only target team hosts
	for _, uuid := range targetUUIDs {
		assert.True(t, scopedHosts[uuid], "target host %s missing from scoped results", uuid)
	}

	// Scoped returns correct number of declarations per host
	scopedDeclsPerHost := make(map[string]int)
	for _, d := range scopedResult {
		scopedDeclsPerHost[d.HostUUID]++
	}
	for _, uuid := range targetUUIDs {
		assert.Equal(t, declsPerTeam, scopedDeclsPerHost[uuid],
			"host %s: expected %d declarations, got %d", uuid, declsPerTeam, scopedDeclsPerHost[uuid])
	}

	// === REPORT ===
	speedup := float64(unscopedDuration) / float64(scopedDuration)

	t.Logf("")
	t.Logf("╔══════════════════════════════════════════════════════════════════════════╗")
	t.Logf("║  REPRODUCTION OF ISSUE #39921                                          ║")
	t.Logf("║  'POST /api/latest/fleet/spec/teams taking ~107s with ~70k hosts'      ║")
	t.Logf("╠══════════════════════════════════════════════════════════════════════════╣")
	t.Logf("║  Setup: %d hosts, %d teams, %d decls/team (%d total)",
		totalHosts, 1+numBystanderTeams, declsPerTeam, totalDecls)
	t.Logf("╠══════════════════════════════════════════════════════════════════════════╣")
	t.Logf("║  SCOPED   (fix)  : %6d hosts → %v", targetTeamHosts, scopedDuration.Truncate(time.Millisecond))
	t.Logf("║  UNSCOPED (bug)  : %6d hosts → %v  (reported: ~107s)", totalHosts, unscopedDuration.Truncate(time.Millisecond))
	t.Logf("║  Speedup         : %.1fx", speedup)
	t.Logf("╚══════════════════════════════════════════════════════════════════════════╝")

	// The fix must be faster
	assert.Less(t, scopedDuration, unscopedDuration,
		"scoped (%v) must be faster than unscoped (%v)", scopedDuration, unscopedDuration)
}

// BenchmarkDeclarationProcessingAtScale benchmarks scoped vs unscoped
// declaration processing at multiple host counts to show how the performance
// gap widens with scale.
//
// Run with:
//
//	MYSQL_TEST=1 FLEET_MYSQL_ADDRESS=127.0.0.1:3307 FLEET_MYSQL_DATABASE=fleet \
//	  go test -bench BenchmarkDeclarationProcessingAtScale -benchtime 3x -run ^$ -timeout 600s \
//	  ./server/datastore/mysql/
func BenchmarkDeclarationProcessingAtScale(b *testing.B) {
	ds := CreateMySQLDS(b)
	defer TruncateTables(b, ds)

	const declsPerTeam = 3

	// Scale tiers showing how unscoped performance degrades with fleet size.
	// The target team always has 500 hosts. Only bystander count changes.
	tiers := []struct {
		name              string
		numBystanderTeams int
		hostsPerTeam      int
	}{
		{"1000hosts_10teams", 9, 100},
		{"2000hosts_10teams", 9, 200},
		{"5000hosts_10teams", 9, 500},
		{"10000hosts_10teams", 9, 1000},
	}

	for _, tier := range tiers {
		b.Run(tier.name, func(b *testing.B) {
			TruncateTables(b, ds)
			ctx := context.Background()

			// Create target team with its hosts
			targetTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: fmt.Sprintf("target-%s", tier.name)})
			require.NoError(b, err)
			targetUUIDs := bulkInsertHosts(b, ds, targetTeam.ID, fmt.Sprintf("target-%s", tier.name), tier.hostsPerTeam)
			for d := range declsPerTeam {
				_, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
					Identifier: fmt.Sprintf("com.fleet.bench.target-%s.%d", tier.name, d),
					Name:       fmt.Sprintf("target-%s-decl-%d", tier.name, d),
					RawJSON:    fmt.Appendf(nil, `{"Type":"com.apple.configuration.decl.%d","Identifier":"com.fleet.bench.target-%s.%d","Payload":{}}`, d, tier.name, d),
					TeamID:     &targetTeam.ID,
				})
				require.NoError(b, err)
			}

			// Create bystander teams
			for i := range tier.numBystanderTeams {
				name := fmt.Sprintf("bystander-%s-%02d", tier.name, i)
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: name})
				require.NoError(b, err)
				bulkInsertHosts(b, ds, team.ID, name, tier.hostsPerTeam)
				for d := range declsPerTeam {
					_, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
						Identifier: fmt.Sprintf("com.fleet.bench.%s.%d", name, d),
						Name:       fmt.Sprintf("%s-decl-%d", name, d),
						RawJSON:    fmt.Appendf(nil, `{"Type":"com.apple.configuration.decl.%d","Identifier":"com.fleet.bench.%s.%d","Payload":{}}`, d, name, d),
						TeamID:     &team.ID,
					})
					require.NoError(b, err)
				}
			}

			totalHosts := tier.hostsPerTeam * (1 + tier.numBystanderTeams)

			b.Run(fmt.Sprintf("Scoped_%dhosts", tier.hostsPerTeam), func(b *testing.B) {
				b.ResetTimer()
				for range b.N {
					ExecAdhocSQL(b, ds, func(q sqlx.ExtContext) error {
						_, err := mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, targetUUIDs)
						return err
					})
				}
			})

			b.Run(fmt.Sprintf("Unscoped_%dhosts", totalHosts), func(b *testing.B) {
				b.ResetTimer()
				for range b.N {
					ExecAdhocSQL(b, ds, func(q sqlx.ExtContext) error {
						_, err := mdmAppleGetHostsWithChangedDeclarationsDB(ctx, q, nil)
						return err
					})
				}
			})
		})
	}
}
