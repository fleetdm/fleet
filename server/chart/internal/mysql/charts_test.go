package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindOnlineHostIDs covers the per-host online predicate and the
// disabledFleetIDs filter: NULL team_id hosts are always retained, hosts in
// disabled fleets are excluded, hosts whose seen_time falls outside their own
// check-in interval are excluded, and hosts without a host_seen_times row at
// all (mobile devices) are excluded.
func TestFindOnlineHostIDs(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)

	cases := []struct {
		name string
		fn   func(t *testing.T, tdb *testutils.TestDB, ds *Datastore)
	}{
		{"NoFilter", testFindOnlineEmptyDisabled},
		{"SingleDisabledFleet", testFindOnlineSingleDisabled},
		{"MultipleDisabledFleets", testFindOnlineMultipleDisabled},
		{"NullTeamHostsAlwaysIncluded", testFindOnlineNullTeamRetained},
		{"OfflineHostsExcluded", testFindOnlineOfflineExcluded},
		{"MobileHostsWithoutSeenTimeExcluded", testFindOnlineMobileExcluded},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer tdb.TruncateTables(t)
			c.fn(t, tdb, ds)
		})
	}
}

// hostSeed describes one row to insert into hosts (and optionally host_seen_times).
// distributedInterval is in seconds; when 0, the table defaults apply (which means
// the effective online window collapses to fleet.OnlineIntervalBuffer alone).
// When omitSeenTime is true, no host_seen_times row is inserted — simulating an
// MDM-only mobile device.
type hostSeed struct {
	teamID              uint // 0 means NULL
	seenTime            time.Time
	distributedInterval int
	omitSeenTime        bool
}

// seedHosts inserts a host per entry and returns the auto-assigned host ids in
// input order. teamID == 0 is treated as NULL.
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
		// detail_updated_at is set to the sentinel so it never spuriously
		// makes a host look freshly active to anyone reading from hosts.
		res, err := tdb.DB.ExecContext(ctx, `
			INSERT INTO hosts (osquery_host_id, node_key, uuid, hostname, detail_updated_at, created_at, team_id, distributed_interval, config_tls_refresh)
			VALUES (?, ?, ?, ?, '2000-01-01 00:00:00', ?, ?, ?, ?)
		`, "ohid-"+itoa(uint(i+1)), "nk-"+itoa(uint(i+1)), "uuid-"+itoa(uint(i+1)),
			"host-"+itoa(uint(i+1)), e.seenTime, teamArg, e.distributedInterval, e.distributedInterval)
		require.NoError(t, err)
		raw, err := res.LastInsertId()
		require.NoError(t, err)
		hostID := uint(raw) //nolint:gosec // G115: AUTO_INCREMENT primary key
		if !e.omitSeenTime {
			_, err = tdb.DB.ExecContext(ctx,
				`INSERT INTO host_seen_times (host_id, seen_time) VALUES (?, ?)`,
				hostID, e.seenTime)
			require.NoError(t, err)
		}
		ids = append(ids, hostID)
	}
	return ids
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

// onlineSeen is a seen_time recent enough that the host should be considered
// online given the default test distributedInterval below.
func onlineSeen(now time.Time) time.Time { return now.Add(-1 * time.Minute) }

// defaultInterval gives every online test host a 10-minute check-in interval,
// so the predicate's effective window is 10m + OnlineIntervalBuffer (60s).
const defaultInterval = 600

func testFindOnlineEmptyDisabled(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, seenTime: onlineSeen(now), distributedInterval: defaultInterval},
		{teamID: 2, seenTime: onlineSeen(now), distributedInterval: defaultInterval},
		{teamID: 0, seenTime: onlineSeen(now), distributedInterval: defaultInterval},
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, ids, got, "with no fleet filter all online hosts are returned")
}

func testFindOnlineSingleDisabled(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, seenTime: onlineSeen(now), distributedInterval: defaultInterval}, // 0: excluded
		{teamID: 2, seenTime: onlineSeen(now), distributedInterval: defaultInterval}, // 1: kept
		{teamID: 0, seenTime: onlineSeen(now), distributedInterval: defaultInterval}, // 2: kept (no team)
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, []uint{1})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[1], ids[2]}, got)
}

func testFindOnlineMultipleDisabled(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, seenTime: onlineSeen(now), distributedInterval: defaultInterval}, // 0: excluded
		{teamID: 2, seenTime: onlineSeen(now), distributedInterval: defaultInterval}, // 1: excluded
		{teamID: 3, seenTime: onlineSeen(now), distributedInterval: defaultInterval}, // 2: kept
		{teamID: 0, seenTime: onlineSeen(now), distributedInterval: defaultInterval}, // 3: kept (no team)
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, []uint{1, 2})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[2], ids[3]}, got)
}

func testFindOnlineNullTeamRetained(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 0, seenTime: onlineSeen(now), distributedInterval: defaultInterval}, // 0: kept (NULL team)
		{teamID: 1, seenTime: onlineSeen(now), distributedInterval: defaultInterval}, // 1: excluded
	})

	// Disabling every fleet that exists must still return the NULL-team host.
	got, err := ds.FindOnlineHostIDs(ctx, now, []uint{1})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[0]}, got)
}

func testFindOnlineOfflineExcluded(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	// All three hosts have a 10-min check-in interval (effective online
	// window: 660s). Only host 0 is within that window.
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 2, seenTime: onlineSeen(now), distributedInterval: defaultInterval},          // 0: online
		{teamID: 2, seenTime: now.Add(-1 * time.Hour), distributedInterval: defaultInterval},  // 1: offline (well past its window)
		{teamID: 0, seenTime: now.Add(-48 * time.Hour), distributedInterval: defaultInterval}, // 2: offline even though NULL team
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[0]}, got)
}

func testFindOnlineMobileExcluded(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	// Host 0 is an osquery host, online. Host 1 has no host_seen_times row —
	// representing an MDM-only mobile device — and must be excluded by the
	// INNER JOIN regardless of how recently it might have checked in via MDM.
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, seenTime: onlineSeen(now), distributedInterval: defaultInterval}, // 0: osquery online
		{teamID: 1, seenTime: now, omitSeenTime: true},                               // 1: mobile (no hst row)
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[0]}, got)
}
