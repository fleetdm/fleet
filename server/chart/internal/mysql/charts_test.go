package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindOnlineHostIDs covers the platform-specific online predicate and the
// disabledFleetIDs filter: NULL team_id hosts are always retained, hosts in
// disabled fleets are excluded, non-mobile hosts whose seen_time falls outside
// their own check-in interval are excluded, and mobile hosts are evaluated via
// their MDM activity signal (nano_enrollments.last_seen_at / detail_updated_at)
// rather than host_seen_times.
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
		{"NonMobileWithoutSeenTimeExcluded", testFindOnlineNonMobileNoSeenTimeExcluded},
		{"AppleMobileOnlineViaNanoLastSeen", testFindOnlineAppleMobileOnline},
		{"AppleMobileOfflineWhenNanoStale", testFindOnlineAppleMobileStale},
		{"AndroidOnlineViaDetailUpdatedAt", testFindOnlineAndroidOnline},
		{"AndroidOfflineWhenDetailNever", testFindOnlineAndroidNever},
		{"MobileNeverCheckedInExcluded", testFindOnlineMobileNeverCheckedIn},
		{"MobileDisabledEnrollmentExcluded", testFindOnlineMobileDisabledEnrollment},
		{"MobileDisabledFleetExcluded", testFindOnlineMobileDisabledFleet},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer tdb.TruncateTables(t)
			c.fn(t, tdb, ds)
		})
	}
}

// hostSeed describes one row to insert into hosts (and optionally
// host_seen_times / nano_enrollments).
//
// distributedInterval is in seconds; when 0, the table defaults apply (which
// means the effective online window collapses to fleet.OnlineIntervalBuffer
// alone). When omitSeenTime is true, no host_seen_times row is inserted —
// simulating an MDM-only mobile device.
//
// platform defaults to "" (treated as a non-mobile/osquery host). Set it to
// "ios", "ipados", or "android" to exercise the mobile predicate. For mobile
// hosts, nanoLastSeen seeds a nano_enrollments row (the Apple MDM check-in
// signal) and detailUpdatedAt overrides hosts.detail_updated_at (the Android
// status-report signal); leave either zero to omit it.
//
// nanoDisabled seeds the nano_enrollments row with enabled = 0 (simulating a
// device that checked out, which also bumps last_seen_at). The default is an
// enabled enrollment.
type hostSeed struct {
	teamID              uint // 0 means NULL
	seenTime            time.Time
	distributedInterval int
	omitSeenTime        bool
	platform            string
	nanoLastSeen        time.Time
	nanoDisabled        bool
	detailUpdatedAt     time.Time
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
		uuid := "uuid-" + itoa(uint(i+1))
		// detail_updated_at defaults to the sentinel so it never spuriously
		// makes a host look freshly active; a non-zero detailUpdatedAt (the
		// Android status-report signal) overrides it.
		var detailArg any = neverTimestamp
		if !e.detailUpdatedAt.IsZero() {
			detailArg = e.detailUpdatedAt
		}
		// created_at must be a valid timestamp. Mobile seeds omit seenTime, so
		// fall back to a recent value — which also exercises that the mobile
		// predicate does NOT treat a recent created_at as an online signal.
		createdArg := e.seenTime
		if createdArg.IsZero() {
			createdArg = time.Now().UTC()
		}
		res, err := tdb.DB.ExecContext(ctx, `
			INSERT INTO hosts (osquery_host_id, node_key, uuid, hostname, platform, detail_updated_at, created_at, team_id, distributed_interval, config_tls_refresh)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "ohid-"+itoa(uint(i+1)), "nk-"+itoa(uint(i+1)), uuid,
			"host-"+itoa(uint(i+1)), e.platform, detailArg, createdArg, teamArg, e.distributedInterval, e.distributedInterval)
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
		// Seed the Apple MDM check-in signal (nano_enrollments.last_seen_at),
		// joined to the host by uuid. nano_enrollments requires a nano_devices
		// row via FK, so insert that first.
		if !e.nanoLastSeen.IsZero() {
			_, err = tdb.DB.ExecContext(ctx,
				`INSERT INTO nano_devices (id, authenticate) VALUES (?, ?)`, uuid, "auth")
			require.NoError(t, err)
			enabled := 1
			if e.nanoDisabled {
				enabled = 0
			}
			_, err = tdb.DB.ExecContext(ctx, `
				INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, last_seen_at, enabled)
				VALUES (?, ?, 'Device', 'topic', 'magic', 'hex', ?, ?)`,
				uuid, uuid, e.nanoLastSeen, enabled)
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

func testFindOnlineNonMobileNoSeenTimeExcluded(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	// Host 0 is an osquery host, online. Host 1 is a non-mobile (darwin) host
	// with no host_seen_times row — the non-mobile branch requires a seen_time,
	// so it's excluded. The mobile branch (which would consult MDM signals)
	// must not rescue a non-mobile platform.
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, seenTime: onlineSeen(now), distributedInterval: defaultInterval, platform: "darwin"}, // 0: osquery online
		{teamID: 1, platform: "darwin", omitSeenTime: true, nanoLastSeen: now},                           // 1: no hst row, has nano signal but not mobile
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[0]}, got)
}

// mobileRecent / mobileStale are activity-signal timestamps relative to the
// mobile online window (mobileOnlineWindowSeconds ≈ 61 minutes).
func mobileRecent(now time.Time) time.Time { return now.Add(-5 * time.Minute) }
func mobileStale(now time.Time) time.Time  { return now.Add(-2 * time.Hour) }

func testFindOnlineAppleMobileOnline(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	// iOS and iPadOS hosts with a recent nano_enrollments.last_seen_at and no
	// host_seen_times row are online via the MDM signal.
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, platform: "ios", omitSeenTime: true, nanoLastSeen: mobileRecent(now)},    // 0: online
		{teamID: 1, platform: "ipados", omitSeenTime: true, nanoLastSeen: mobileRecent(now)}, // 1: online
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[0], ids[1]}, got)
}

func testFindOnlineAppleMobileStale(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	// iOS host whose last MDM check-in is older than the mobile window is offline.
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, platform: "ios", omitSeenTime: true, nanoLastSeen: mobileRecent(now)}, // 0: online
		{teamID: 1, platform: "ios", omitSeenTime: true, nanoLastSeen: mobileStale(now)},  // 1: offline (stale)
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[0]}, got)
}

func testFindOnlineAndroidOnline(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	// Android has no nano_enrollments row; its signal is detail_updated_at
	// (written on status reports). A recent value within the window is online;
	// a stale one is offline.
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, platform: "android", omitSeenTime: true, detailUpdatedAt: mobileRecent(now)}, // 0: online
		{teamID: 1, platform: "android", omitSeenTime: true, detailUpdatedAt: mobileStale(now)},  // 1: offline
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[0]}, got)
}

func testFindOnlineAndroidNever(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	// Android host whose detail_updated_at is still the NeverTimestamp sentinel
	// (no status report yet) and has no nano signal is offline — NULLIF drops
	// the sentinel and there is no created_at fallback.
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, platform: "android", omitSeenTime: true, detailUpdatedAt: mobileRecent(now)}, // 0: online
		{teamID: 1, platform: "android", omitSeenTime: true},                                     // 1: sentinel detail, offline
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[0]}, got)
}

func testFindOnlineMobileNeverCheckedIn(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	// A freshly enrolled iOS host with no MDM signal at all (sentinel
	// detail_updated_at, no nano row, no seen_time) must NOT be online — the
	// mobile predicate deliberately has no created_at fallback.
	seedHosts(t, tdb, []hostSeed{
		{teamID: 1, platform: "ios", omitSeenTime: true}, // 0: never checked in
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func testFindOnlineMobileDisabledEnrollment(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	// Disabling an enrollment (e.g. on checkout) sets nano_enrollments.enabled = 0
	// AND bumps last_seen_at to CURRENT_TIMESTAMP. The predicate only joins
	// enabled enrollments, so a device that just checked out must NOT count as
	// online even though its last_seen_at is recent.
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, platform: "ios", omitSeenTime: true, nanoLastSeen: mobileRecent(now)},                     // 0: online (enabled)
		{teamID: 1, platform: "ios", omitSeenTime: true, nanoLastSeen: mobileRecent(now), nanoDisabled: true}, // 1: offline (disabled, last_seen_at bumped on checkout)
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[0]}, got)
}

func testFindOnlineMobileDisabledFleet(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	// The disabled-fleet exclusion applies to mobile hosts too: the iOS host in
	// fleet 1 is dropped while the NULL-team iOS host is retained.
	ids := seedHosts(t, tdb, []hostSeed{
		{teamID: 1, platform: "ios", omitSeenTime: true, nanoLastSeen: mobileRecent(now)}, // 0: excluded (disabled fleet)
		{teamID: 0, platform: "ios", omitSeenTime: true, nanoLastSeen: mobileRecent(now)}, // 1: kept (NULL team)
	})

	got, err := ds.FindOnlineHostIDs(ctx, now, []uint{1})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{ids[1]}, got)
}
