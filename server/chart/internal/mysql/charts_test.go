package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindRecentlySeenHostIDs covers the disabledFleetIDs filter on the
// recently-seen query: that NULL team_id hosts are always retained, that hosts
// in disabled fleets are excluded, and that the activity-signal cutoff still
// applies regardless of fleet membership.
func TestFindRecentlySeenHostIDs(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)

	cases := []struct {
		name string
		fn   func(t *testing.T, tdb *testutils.TestDB, ds *Datastore)
	}{
		{"NoFilter", testFindRecentEmptyDisabled},
		{"SingleDisabledFleet", testFindRecentSingleDisabled},
		{"MultipleDisabledFleets", testFindRecentMultipleDisabled},
		{"NullTeamHostsAlwaysIncluded", testFindRecentNullTeamRetained},
		{"OldHostsExcludedRegardlessOfFleet", testFindRecentOldHostsFiltered},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer tdb.TruncateTables(t)
			c.fn(t, tdb, ds)
		})
	}
}

// seedHosts inserts a host per (teamID, lastSeen) entry and returns the
// auto-assigned host ids in input order. teamID == 0 is treated as NULL.
func seedHosts(t *testing.T, tdb *testutils.TestDB, entries []hostSeed) []uint {
	t.Helper()
	ctx := t.Context()

	// Create teams referenced by the entries (idempotent within the call).
	teams := map[uint]struct{}{}
	for _, e := range entries {
		if e.teamID != 0 {
			teams[e.teamID] = struct{}{}
		}
	}
	for id := range teams {
		_, err := tdb.DB.ExecContext(ctx,
			`INSERT INTO teams (id, name) VALUES (?, ?)`, id, "team-"+itoa(id))
		require.NoError(t, err)
	}

	ids := make([]uint, 0, len(entries))
	for i, e := range entries {
		var teamArg any
		if e.teamID != 0 {
			teamArg = e.teamID
		}
		// detail_updated_at is set to the sentinel so the WHERE clause's
		// COALESCE falls through to either host_seen_times or created_at.
		res, err := tdb.DB.ExecContext(ctx, `
			INSERT INTO hosts (osquery_host_id, node_key, uuid, hostname, detail_updated_at, created_at, team_id)
			VALUES (?, ?, ?, ?, '2000-01-01 00:00:00', ?, ?)
		`, "ohid-"+itoa(uint(i+1)), "nk-"+itoa(uint(i+1)), "uuid-"+itoa(uint(i+1)),
			"host-"+itoa(uint(i+1)), e.lastSeen, teamArg)
		require.NoError(t, err)
		raw, err := res.LastInsertId()
		require.NoError(t, err)
		hostID := uint(raw) //nolint:gosec // G115: AUTO_INCREMENT primary key
		_, err = tdb.DB.ExecContext(ctx,
			`INSERT INTO host_seen_times (host_id, seen_time) VALUES (?, ?)`,
			hostID, e.lastSeen)
		require.NoError(t, err)
		ids = append(ids, hostID)
	}
	return ids
}

type hostSeed struct {
	teamID   uint // 0 means NULL
	lastSeen time.Time
}

func itoa(u uint) string {
	const digits = "0123456789"
	if u == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for u > 0 {
		i--
		b[i] = digits[u%10]
		u /= 10
	}
	return string(b[i:])
}

func testFindRecentEmptyDisabled(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, lastSeen: now.Add(-1 * time.Hour)},
		{teamID: 2, lastSeen: now.Add(-1 * time.Hour)},
		{teamID: 0, lastSeen: now.Add(-1 * time.Hour)},
	})

	got, err := ds.FindRecentlySeenHostIDs(ctx, now.Add(-24*time.Hour), nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, ids, got, "with no fleet filter all recently-seen hosts are returned")
}

func testFindRecentSingleDisabled(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, lastSeen: now.Add(-1 * time.Hour)}, // 0: excluded
		{teamID: 2, lastSeen: now.Add(-1 * time.Hour)}, // 1: kept
		{teamID: 0, lastSeen: now.Add(-1 * time.Hour)}, // 2: kept (no team)
	})

	got, err := ds.FindRecentlySeenHostIDs(ctx, now.Add(-24*time.Hour), []uint{1})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[1], ids[2]}, got)
}

func testFindRecentMultipleDisabled(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, lastSeen: now.Add(-1 * time.Hour)}, // 0: excluded
		{teamID: 2, lastSeen: now.Add(-1 * time.Hour)}, // 1: excluded
		{teamID: 3, lastSeen: now.Add(-1 * time.Hour)}, // 2: kept
		{teamID: 0, lastSeen: now.Add(-1 * time.Hour)}, // 3: kept (no team)
	})

	got, err := ds.FindRecentlySeenHostIDs(ctx, now.Add(-24*time.Hour), []uint{1, 2})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[2], ids[3]}, got)
}

func testFindRecentNullTeamRetained(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 0, lastSeen: now.Add(-1 * time.Hour)}, // 0: kept (NULL team)
		{teamID: 1, lastSeen: now.Add(-1 * time.Hour)}, // 1: excluded
	})

	// Disabling every fleet that exists must still return the NULL-team host.
	got, err := ds.FindRecentlySeenHostIDs(ctx, now.Add(-24*time.Hour), []uint{1})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[0]}, got)
}

func testFindRecentOldHostsFiltered(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 2, lastSeen: now.Add(-1 * time.Hour)},   // 0: kept (recent, not disabled)
		{teamID: 2, lastSeen: now.Add(-48 * time.Hour)},  // 1: excluded by recency
		{teamID: 0, lastSeen: now.Add(-48 * time.Hour)},  // 2: excluded by recency (NULL team doesn't bypass `since`)
	})
	_ = ids

	got, err := ds.FindRecentlySeenHostIDs(ctx, now.Add(-24*time.Hour), []uint{1})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[0]}, got)
}
