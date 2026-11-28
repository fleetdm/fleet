package mysql

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/tables"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestVPPAppInstallState(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"RefreshPointers", testVPPAppInstallStateRefreshPointers},
		{"NullSnapshots", testVPPAppInstallStateNullSnapshots},
		{"ReadCutover", testVPPAppInstallStateReadCutover},
		{"HostDelete", testVPPAppInstallStateHostDelete},
		{"Serialization", testVPPAppInstallStateSerialization},
		{"DiscoverySerialization", testVPPAppInstallDiscoverySerialization},
		{"DiscoveryChangedTargets", testVPPAppInstallDiscoveryChangedTargets},
		{"DiscoveryRetryLimit", testVPPAppInstallDiscoveryRetryLimit},
		{"DiscoveryQueueSnapshot", testVPPAppInstallDiscoveryQueueSnapshot},
		{"DiscoveryFailureSnapshot", testVPPAppInstallDiscoveryFailureSnapshot},
		{"AttemptExpiry", testVPPAppInstallAttemptExpiry},
		{"CanceledMassFailure", testVPPAppInstallCanceledMassFailure},
		{"AttemptLog", testVPPAppInstallAttemptLog},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			defer assertVPPAppInstallStateMatchesRequests(t, ds)
			c.fn(t, ds)
		})
	}
}

// assertVPPAppInstallStateMatchesRequests recomputes all state with the migration's backfill query and compares it
// to the read-equivalent state, merging buckets whose version FKs were nulled.
func assertVPPAppInstallStateMatchesRequests(t *testing.T, ds *Datastore) {
	t.Helper()
	ctx := t.Context()
	const order = ` ORDER BY host_id, adam_id, platform, version_bucket`
	var want, got []hostVPPAppInstallState
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &want, `SELECT * FROM (`+tables.VPPAppInstallStateQuery+`) s`+order))
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &got, `
SELECT host_id, adam_id, platform, version_bucket, vpp_app_team_id,
	installed_install_id, failed_install_id, unfinished_install_id,
	visible_install_id, latest_install_id, unfailed_install_id,
	visible_created_at, visible_command_uuid, visible_verification_at, visible_verification_failed_at,
	latest_created_at, latest_command_uuid, latest_verification_at, latest_verification_failed_at
FROM host_vpp_app_installs`+order))
	var ranked []uint
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &ranked,
		`SELECT id FROM host_vpp_software_installs ORDER BY created_at DESC, id DESC`))
	ranks := make(map[uint]int, len(ranked))
	for rank, id := range ranked {
		ranks[id] = rank
	}
	normalize := func(rows []hostVPPAppInstallState) []hostVPPAppInstallState {
		type key struct {
			hostVPPAppKey
			Version uint
		}
		merged := make(map[key]hostVPPAppInstallState)
		maxID := func(a, b *uint) *uint {
			if a == nil || (b != nil && *b > *a) {
				return b
			}
			return a
		}
		firstID := func(a, b *uint) *uint {
			if a == nil || (b != nil && ranks[*b] < ranks[*a]) {
				return b
			}
			return a
		}
		for _, row := range rows {
			row.VersionBucket = 0
			if row.VPPAppTeamID != nil {
				row.VersionBucket = *row.VPPAppTeamID
			}
			k := key{hostVPPAppKey{row.HostID, row.AdamID, row.Platform}, row.VersionBucket}
			if prev, ok := merged[k]; ok {
				row.InstalledInstallID = maxID(prev.InstalledInstallID, row.InstalledInstallID)
				row.FailedInstallID = maxID(prev.FailedInstallID, row.FailedInstallID)
				row.UnfinishedInstallID = maxID(prev.UnfinishedInstallID, row.UnfinishedInstallID)
				row.VisibleInstallID = firstID(prev.VisibleInstallID, row.VisibleInstallID)
				if row.VisibleInstallID == prev.VisibleInstallID {
					row.VisibleCreatedAt = prev.VisibleCreatedAt
					row.VisibleCommandUUID = prev.VisibleCommandUUID
					row.VisibleVerificationAt = prev.VisibleVerificationAt
					row.VisibleVerificationFailedAt = prev.VisibleVerificationFailedAt
				}
				row.LatestInstallID = firstID(prev.LatestInstallID, row.LatestInstallID)
				if row.LatestInstallID == prev.LatestInstallID {
					row.LatestCreatedAt = prev.LatestCreatedAt
					row.LatestCommandUUID = prev.LatestCommandUUID
					row.LatestVerificationAt = prev.LatestVerificationAt
					row.LatestVerificationFailedAt = prev.LatestVerificationFailedAt
				}
				row.UnfailedInstallID = firstID(prev.UnfailedInstallID, row.UnfailedInstallID)
			}
			merged[k] = row
		}
		result := make([]hostVPPAppInstallState, 0, len(merged))
		for _, row := range merged {
			result = append(result, row)
		}
		slices.SortFunc(result, func(a, b hostVPPAppInstallState) int {
			return cmp.Or(cmp.Compare(a.HostID, b.HostID), cmp.Compare(a.AdamID, b.AdamID),
				cmp.Compare(a.Platform, b.Platform), cmp.Compare(a.VersionBucket, b.VersionBucket))
		})
		return result
	}
	require.Equal(t, normalize(want), normalize(got))
	assertVPPAppInstallReadsMatchRequests(t, ds)
}

func lockAndRefreshVPPAppInstallState(t *testing.T, ds *Datastore, keys ...hostVPPAppKey) {
	t.Helper()
	ctx := t.Context()
	require.NoError(t, ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := lockVPPAppInstallKeys(ctx, tx, keys); err != nil {
			return err
		}
		return refreshVPPAppInstallState(ctx, tx, keys)
	}))
}

func testVPPAppInstallStateNullSnapshots(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('null-snapshots', 'ios')`)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `
INSERT INTO host_vpp_software_installs
	(host_id, adam_id, platform, command_uuid, created_at, verification_at, canceled, removed)
VALUES
	(1, 'null-snapshots', 'ios', 'removed-only', '2026-01-01 09:00:00', '2026-01-01 10:00:00', 0, 1),
	(2, 'null-snapshots', 'ios', 'canceled-only', '2026-01-01 09:00:00', '2026-01-01 10:00:00', 1, 0)`)
	require.NoError(t, err)
	lockAndRefreshVPPAppInstallState(t, ds,
		hostVPPAppKey{1, "null-snapshots", "ios"}, hostVPPAppKey{2, "null-snapshots", "ios"})

	var rows []hostVPPAppInstallState
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &rows, `
SELECT visible_install_id, visible_created_at, visible_command_uuid,
	visible_verification_at, visible_verification_failed_at,
	latest_install_id, latest_created_at, latest_command_uuid,
	latest_verification_at, latest_verification_failed_at
FROM host_vpp_app_installs ORDER BY host_id`))
	require.Len(t, rows, 2)
	for _, row := range rows {
		require.Nil(t, row.VisibleInstallID)
		require.Nil(t, row.VisibleCreatedAt)
		require.Nil(t, row.VisibleCommandUUID)
		require.Nil(t, row.VisibleVerificationAt)
		require.Nil(t, row.VisibleVerificationFailedAt)
	}
	// Removed requests remain eligible for the latest pointer.
	require.NotNil(t, rows[0].LatestInstallID)
	require.NotNil(t, rows[0].LatestCreatedAt)
	require.Equal(t, new("removed-only"), rows[0].LatestCommandUUID)
	require.NotNil(t, rows[0].LatestVerificationAt)
	require.Nil(t, rows[0].LatestVerificationFailedAt)
	require.Nil(t, rows[1].LatestInstallID)
	require.Nil(t, rows[1].LatestCreatedAt)
	require.Nil(t, rows[1].LatestCommandUUID)
	require.Nil(t, rows[1].LatestVerificationAt)
	require.Nil(t, rows[1].LatestVerificationFailedAt)
}

func testVPPAppInstallStateRefreshPointers(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	exec := func(stmt string, args ...any) uint {
		res, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return uint(id) //nolint:gosec // dismiss G115
	}

	exec(`INSERT INTO vpp_apps (adam_id, platform) VALUES ('app-a', 'ios'), ('app-b', 'android')`)
	version := func(adamID, platform, name string) uint {
		return exec(`INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, name) VALUES (?, ?, 0, ?)`, adamID, platform, name)
	}
	vA1, vA2, vA3 := version("app-a", "ios", "a1"), version("app-a", "ios", "a2"), version("app-a", "ios", "a3")
	vB1 := version("app-b", "android", "b1")

	const h1, h2 = uint(1), uint(2)
	var n int
	request := func(hostID uint, adamID, platform string, versionID *uint, createdAt *string) uint {
		n++
		return exec(`INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid, vpp_app_team_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`, hostID, adamID, platform, fmt.Sprint("cmd-", n), versionID, createdAt)
	}
	at := func(hour string) *string { return new("2026-01-01 " + hour + ":00:00") }
	set := func(id uint, assignments string) {
		exec(`UPDATE host_vpp_software_installs SET `+assignments+` WHERE id = ?`, id)
	}
	const verified, failed = `verification_at = NOW(6)`, `verification_failed_at = NOW(6)`

	// vA2 rows get lower ids but later created_at than vA1 rows.
	b1 := request(h1, "app-a", "ios", &vA2, at("13"))
	set(b1, failed)
	b2 := request(h1, "app-a", "ios", &vA2, at("12"))
	a1 := request(h1, "app-a", "ios", &vA1, at("09"))
	set(a1, failed)
	a2 := request(h1, "app-a", "ios", &vA1, at("10")) // verified after a failed attempt
	set(a2, verified)
	a3 := request(h1, "app-a", "ios", &vA1, at("08"))
	a4 := request(h1, "app-a", "ios", &vA1, at("11"))
	set(a4, `canceled = 1`)
	a5 := request(h1, "app-a", "ios", &vA1, at("12"))
	set(a5, `removed = 1, `+verified)
	c1 := request(h1, "app-a", "ios", nil, nil)
	c2 := request(h1, "app-a", "ios", &vA3, at("06"))
	set(c2, verified)
	exec(`DELETE FROM vpp_apps_teams WHERE id = ?`, vA3)
	p1 := request(h2, "app-a", "ios", &vA1, at("10")) // preflight failure
	set(p1, failed)
	d1 := request(h2, "app-a", "ios", &vA2, at("09"))
	set(d1, `canceled = 1`)
	e1 := request(h2, "app-b", "android", &vB1, at("10"))
	set(e1, failed)
	e2 := request(h2, "app-b", "android", &vB1, at("11"))

	keyA1, keyA2 := hostVPPAppKey{h1, "app-a", "ios"}, hostVPPAppKey{h2, "app-a", "ios"}
	keyB2 := hostVPPAppKey{h2, "app-b", "android"}
	lockAndRefreshVPPAppInstallState(t, ds, keyB2, keyA1, keyA2, keyA1)

	state := func(hostID uint, adamID, platform string, bucket uint, versionID, installed, failed, unfinished, visible, latest, unfailed *uint) hostVPPAppInstallState {
		return hostVPPAppInstallState{
			HostID: hostID, AdamID: adamID, Platform: platform, VersionBucket: bucket, VPPAppTeamID: versionID,
			InstalledInstallID: installed, FailedInstallID: failed, UnfinishedInstallID: unfinished,
			VisibleInstallID: visible, LatestInstallID: latest, UnfailedInstallID: unfailed,
		}
	}
	readState := func() []hostVPPAppInstallState {
		var got []hostVPPAppInstallState
		require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &got, `
SELECT host_id, adam_id, platform, version_bucket, vpp_app_team_id,
	installed_install_id, failed_install_id, unfinished_install_id,
	visible_install_id, latest_install_id, unfailed_install_id
FROM host_vpp_app_installs ORDER BY host_id, adam_id, platform, version_bucket`))
		return got
	}

	// c1's NULL created_at sorts last in a DESC order, so c2 wins the ordered pointers.
	require.Equal(t, []hostVPPAppInstallState{
		state(h1, "app-a", "ios", 0, nil, &c2, nil, &c1, &c2, &c2, &c2),
		state(h1, "app-a", "ios", vA1, &vA1, &a2, &a1, &a3, &a2, &a5, &a5),
		state(h1, "app-a", "ios", vA2, &vA2, nil, &b1, &b2, &b1, &b1, &b2),
		state(h2, "app-a", "ios", vA1, &vA1, nil, &p1, nil, &p1, &p1, nil),
		state(h2, "app-a", "ios", vA2, &vA2, nil, nil, nil, nil, nil, nil),
		state(h2, "app-b", "android", vB1, &vB1, nil, &e1, &e2, &e2, &e2, &e2),
	}, readState())
	assertVPPAppInstallStateMatchesRequests(t, ds)

	// Deleting a version nulls the state row's version; the next refresh merges its requests into bucket 0.
	exec(`DELETE FROM vpp_apps_teams WHERE id = ?`, vA2)
	var nulled int
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &nulled,
		`SELECT COUNT(*) FROM host_vpp_app_installs WHERE version_bucket = ? AND vpp_app_team_id IS NULL`, vA2))
	require.Equal(t, 2, nulled)
	assertVPPAppInstallStateMatchesRequests(t, ds)

	// Canceling a2 leaves vA1 without an installed request. Deleting d1 leaves h2's vA2 bucket without requests.
	set(a2, `canceled = 1`)
	exec(`DELETE FROM host_vpp_software_installs WHERE id = ?`, d1)
	lockAndRefreshVPPAppInstallState(t, ds, keyA1, keyA2, keyB2)

	require.Equal(t, []hostVPPAppInstallState{
		state(h1, "app-a", "ios", 0, nil, &c2, &b1, &c1, &b1, &b1, &b2),
		state(h1, "app-a", "ios", vA1, &vA1, nil, &a1, &a3, &a1, &a5, &a5),
		state(h2, "app-a", "ios", vA1, &vA1, nil, &p1, nil, &p1, &p1, nil),
		state(h2, "app-b", "android", vB1, &vB1, nil, &e1, &e2, &e2, &e2, &e2),
	}, readState())
	assertVPPAppInstallStateMatchesRequests(t, ds)

	// A key with no requests left loses all its rows.
	exec(`DELETE FROM host_vpp_software_installs WHERE host_id = ? AND adam_id = 'app-b'`, h2)
	lockAndRefreshVPPAppInstallState(t, ds, keyB2)
	require.Len(t, readState(), 3)
	assertVPPAppInstallStateMatchesRequests(t, ds)
}

func testVPPAppInstallStateReadCutover(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	host := test.NewHost(t, ds, "read-cutover", "", "read-cutover", "read-cutover", time.Now(), test.WithPlatform("ios"))
	app, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		AdamID: "read-cutover", Platform: fleet.IOSPlatform, Name: "Read cutover", BundleIdentifier: "com.example.readcutover",
	}, nil, nil)
	require.NoError(t, err)
	other, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		AdamID: app.AdamID, Platform: app.Platform, Name: app.Name, BundleIdentifier: app.BundleIdentifier, VersionName: "Other",
	}, nil, nil)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{{Name: app.Name, Version: "1", Source: "ios_apps", BundleIdentifier: app.BundleIdentifier}})
	require.NoError(t, err)

	insert := func(command string, version uint, hour int) {
		_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_vpp_software_installs
			(host_id, adam_id, platform, command_uuid, vpp_app_team_id, created_at, verification_at)
			VALUES (?, ?, 'ios', ?, ?, ?, NOW())`, host.ID, app.AdamID, command, version, fmt.Sprintf("2026-01-01 %02d:00:00", hour))
		require.NoError(t, err)
	}
	// A lower id has the later timestamp, so numeric pointer maxima would pick the wrong version.
	insert("later-time", other.AppTeamID, 13)
	insert("higher-id", app.AppTeamID, 12)
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE host_vpp_software_installs SET verification_at = NULL WHERE command_uuid = 'higher-id'`)
	require.NoError(t, err)
	refreshVPPAppInstallStateForTest(t, ds)
	assertVPPAppInstallStateMatchesRequests(t, ds)
	installs, err := hostVPPInstalls(ds, ctx, host.ID, 0, false, true)
	require.NoError(t, err)
	require.Len(t, installs, 1)
	require.Equal(t, "later-time", *installs[0].LastInstallInstallUUID)
	versions, err := ds.ListHostAppStoreAppInstallVersions(ctx, 0, []uint{host.ID}, []fleet.InstallableDevicePlatform{fleet.IOSPlatform})
	require.NoError(t, err)
	require.Equal(t, new(other.AppTeamID), versions[app.TitleID][host.ID].VPPAppTeamID)

	// Keep the nulled version bucket alongside the remaining live version before refreshing.
	_, err = ds.writer(ctx).ExecContext(ctx, `DELETE FROM vpp_apps_teams WHERE id = ?`, other.AppTeamID)
	require.NoError(t, err)
	assertVPPAppInstallStateMatchesRequests(t, ds)
	versions, err = ds.ListHostAppStoreAppInstallVersions(ctx, 0, []uint{host.ID}, []fleet.InstallableDevicePlatform{fleet.IOSPlatform})
	require.NoError(t, err)
	require.Nil(t, versions[app.TitleID][host.ID].VPPAppTeamID)

	params := map[string]any{
		"host_id": host.ID, "global_or_team_id": 0, "version_ids": []uint{app.AppTeamID},
		"software_status_installed": fleet.SoftwareInstalled, "software_status_failed": fleet.SoftwareInstallFailed,
		"software_status_pending": fleet.SoftwareInstallPending, "mdm_status_acknowledged": fleet.MDMAppleStatusAcknowledged,
		"mdm_status_error": fleet.MDMAppleStatusError, "mdm_status_format_error": fleet.MDMAppleStatusCommandFormatError,
	}
	explain := func(name, query string, args ...any) {
		var plans []string
		require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &plans, `EXPLAIN FORMAT=TREE `+query, args...))
		plan := strings.Join(plans, "\n")
		if name == "S38" || name == "S47" {
			require.NotContains(t, query, "host_vpp_software_installs")
		} else if name == "S74" {
			require.NotContains(t, plan, "Table scan on hvsi")
		}
		t.Logf("EXPLAIN %s\n%s", name, plan)
	}
	for _, read := range []struct{ name, query string }{
		{"S49", vppAppVersionHostStatusesQuery},
		{"S38", fmt.Sprintf(hostVPPInstallsQuery, "", vppAppHostStatusNamedQuery("hvsi", "ncr", "status"), "")},
	} {
		query, args, err := sqlx.Named(read.query, params)
		require.NoError(t, err)
		query, args, err = sqlx.In(query, args...)
		require.NoError(t, err)
		explain(read.name, query, args...)
	}
	for _, read := range []struct {
		name    string
		version *uint
	}{{"S45", new(app.AppTeamID)}, {"S47", nil}} {
		join, args, err := ds.vppAppJoin(ctx, app.VPPAppID, read.version, fleet.SoftwareInstalled)
		require.NoError(t, err)
		explain(read.name, `SELECT h.id FROM hosts h `+join, args...)
	}
	query, args, err := sqlx.In(hostAppStoreAppInstallVersionsQuery, 0, []string{"ios"}, []uint{host.ID}, nil)
	require.NoError(t, err)
	explain("S74", query, args...)
}

func testVPPAppInstallStateHostDelete(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	other := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())

	_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('app-a', 'ios')`)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid)
		VALUES (?, 'app-a', 'ios', 'cmd-1'), (?, 'app-a', 'ios', 'cmd-2')`, host.ID, other.ID)
	require.NoError(t, err)
	lockAndRefreshVPPAppInstallState(t, ds, hostVPPAppKey{host.ID, "app-a", "ios"}, hostVPPAppKey{other.ID, "app-a", "ios"})

	require.NoError(t, ds.DeleteHosts(ctx, []uint{host.ID}))

	var hostIDs []uint
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &hostIDs, `SELECT host_id FROM host_vpp_app_installs`))
	require.Equal(t, []uint{other.ID}, hostIDs)
	assertVPPAppInstallStateMatchesRequests(t, ds)
}

func refreshVPPAppInstallStateForTest(t *testing.T, ds *Datastore) {
	t.Helper()
	ctx := t.Context()
	var keys []hostVPPAppKey
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &keys,
		`SELECT DISTINCT host_id, adam_id, platform FROM host_vpp_software_installs
		UNION SELECT DISTINCT host_id, adam_id, platform FROM host_vpp_app_installs`))
	lockAndRefreshVPPAppInstallState(t, ds, keys...)
}

func testVPPAppInstallStateSerialization(t *testing.T, ds *Datastore) {
	for _, scenario := range []string{"verify-fail", "complete-remove", "backfill-activate"} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reverse=%t", scenario, reverse), func(t *testing.T) {
				defer TruncateTables(t, ds)
				ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
				defer cancel()
				_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('barrier-app', 'ios')`)
				require.NoError(t, err)
				_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_vpp_software_installs
					(host_id, adam_id, platform, command_uuid) VALUES
					(1, 'barrier-app', 'ios', 'barrier-a'), (1, 'barrier-app', 'ios', 'barrier-b')`)
				require.NoError(t, err)
				host := test.NewHost(t, ds, "barrier-host", "", "barrier-key", "barrier-uuid", time.Now())
				require.Equal(t, uint(1), host.ID)
				res, err := ds.writer(ctx).ExecContext(ctx,
					`INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, name) VALUES ('barrier-app', 'ios', 0, 'Barrier app')`)
				require.NoError(t, err)
				versionID, err := res.LastInsertId()
				require.NoError(t, err)
				_, err = ds.writer(ctx).ExecContext(ctx,
					`UPDATE host_vpp_software_installs SET vpp_app_team_id = ? WHERE host_id = 1`, versionID)
				require.NoError(t, err)
				keys := []hostVPPAppKey{{1, "barrier-app", "ios"}}
				lockAndRefreshVPPAppInstallState(t, ds, keys...)

				firstLocked, secondSnapshot := make(chan struct{}), make(chan struct{})
				var lockedOnce, snapshotOnce sync.Once
				done := make(chan error, 2)
				for worker := range 2 {
					go func() {
						done <- ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
							// Establish both snapshots before the first mutation commits.
							var count int
							if err := sqlx.GetContext(ctx, tx, &count, `SELECT COUNT(*) FROM host_vpp_software_installs`); err != nil {
								return err
							}
							if worker == 1 {
								select {
								case <-firstLocked:
								case <-ctx.Done():
									return ctx.Err()
								}
								snapshotOnce.Do(func() { close(secondSnapshot) })
							}
							if err := lockVPPAppInstallKeys(ctx, tx, keys); err != nil {
								return err
							}
							if worker == 0 {
								lockedOnce.Do(func() { close(firstLocked) })
								select {
								case <-secondSnapshot:
								case <-ctx.Done():
									return ctx.Err()
								}
							}
							operation := worker
							if reverse {
								operation = 1 - operation
							}
							var stmt string
							switch {
							case scenario == "backfill-activate" && operation == 0:
							case scenario == "backfill-activate":
								stmt = `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid)
									VALUES (1, 'barrier-app', 'ios', 'barrier-c')`
							case operation == 0:
								stmt = `UPDATE host_vpp_software_installs SET verification_at = NOW(6) WHERE command_uuid = 'barrier-a'`
							case scenario == "verify-fail":
								stmt = `UPDATE host_vpp_software_installs SET verification_failed_at = NOW(6) WHERE command_uuid = 'barrier-b'`
							default:
								stmt = `UPDATE host_vpp_software_installs SET removed = 1 WHERE host_id = 1`
							}
							if stmt != "" {
								if _, err := tx.ExecContext(ctx, stmt); err != nil {
									return err
								}
							}
							return refreshVPPAppInstallState(ctx, tx, keys)
						})
					}()
				}
				firstErr, secondErr := <-done, <-done
				require.NoError(t, firstErr)
				require.NoError(t, secondErr)
				assertVPPAppInstallStateMatchesRequests(t, ds)
				type summary struct {
					Installed  *uint `db:"installed"`
					Failed     *uint `db:"failed"`
					Unfinished *uint `db:"unfinished"`
				}
				var legacy, state summary
				require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &legacy, `SELECT
					MAX(IF(verification_at IS NOT NULL, id, NULL)) AS installed,
					MAX(IF(verification_at IS NULL AND verification_failed_at IS NOT NULL, id, NULL)) AS failed,
					MAX(IF(verification_at IS NULL AND verification_failed_at IS NULL, id, NULL)) AS unfinished
					FROM host_vpp_software_installs WHERE host_id = 1 AND adam_id = 'barrier-app' AND platform = 'ios'
					AND canceled = 0 AND removed = 0`))
				require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &state, `SELECT
					MAX(installed_install_id) AS installed, MAX(failed_install_id) AS failed,
					MAX(unfinished_install_id) AS unfinished FROM host_vpp_app_installs
					WHERE host_id = 1 AND adam_id = 'barrier-app' AND platform = 'ios'`))
				require.Equal(t, legacy, state)
			})
		}
	}
}

// vppAppInstallDiscoveryBarrier pauses after each nonlocking discovery, before either key lock.
type vppAppInstallDiscoveryBarrier struct {
	discovered bool
	sqlx.ExtContext
	discoveryQuery string
	afterDiscovery func() error
	afterKeyLock   func()
}

func (b *vppAppInstallDiscoveryBarrier) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	rows, err := b.ExtContext.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if !b.discovered && (query == b.discoveryQuery || query == b.discoveryQuery+` FOR SHARE`) {
		b.discovered = true
		if err := b.afterDiscovery(); err != nil {
			_ = rows.Close() //nolint:sqlclosecheck
			return nil, err
		}
	} else if strings.HasSuffix(query, "FOR UPDATE") && b.afterKeyLock != nil {
		b.afterKeyLock()
	}
	return rows, nil
}

func testVPPAppInstallDiscoverySerialization(t *testing.T, ds *Datastore) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprintf("reverse=%t", reverse), func(t *testing.T) {
			defer TruncateTables(t, ds)
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			host := test.NewHost(t, ds, "discovery-host", "", "discovery-key", "discovery-uuid", time.Now())
			_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('discovery-app', 'ios')`)
			require.NoError(t, err)
			_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_vpp_software_installs
    (host_id, adam_id, platform, command_uuid) VALUES
    (?, 'discovery-app', 'ios', 'discovery-a'), (?, 'discovery-app', 'ios', 'discovery-b')`, host.ID, host.ID)
			require.NoError(t, err)
			lockAndRefreshVPPAppInstallState(t, ds, hostVPPAppKey{host.ID, "discovery-app", "ios"})
			discovered := [2]chan struct{}{make(chan struct{}), make(chan struct{})}
			locked := make(chan struct{})
			var discoveredOnce [2]sync.Once
			var lockedOnce sync.Once
			var attempts [2]atomic.Int32
			done := make(chan error, 2)
			const query = `SELECT id, host_id, adam_id, platform FROM host_vpp_software_installs WHERE command_uuid = ?`
			for worker := range 2 {
				go func() {
					barrier := &vppAppInstallDiscoveryBarrier{
						ExtContext: ds.writer(ctx), discoveryQuery: query,
						afterDiscovery: func() error {
							discoveredOnce[worker].Do(func() { close(discovered[worker]) })
							wait := discovered[1]
							if worker == 1 {
								wait = locked
							}
							select {
							case <-wait:
								return nil
							case <-ctx.Done():
								return ctx.Err()
							}
						},
						afterKeyLock: func() {
							if worker == 0 {
								lockedOnce.Do(func() { close(locked) })
							}
						},
					}
					operation := worker
					if reverse {
						operation = 1 - operation
					}
					command, assignment := "discovery-a", "verification_at = NOW(6)"
					if operation == 1 {
						command, assignment = "discovery-b", "verification_failed_at = NOW(6)"
					}
					keys, err := discoverVPPAppInstallKeys(ctx, barrier, query, command)
					if err != nil {
						done <- err
						return
					}
					done <- ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
						attempts[worker].Add(1)
						barrier.ExtContext = tx
						_, _, err := lockVPPAppInstallRequests(ctx, barrier, keys, query, command)
						if err != nil {
							return err
						}
						if _, err := tx.ExecContext(ctx, `UPDATE host_vpp_software_installs SET `+assignment+` WHERE command_uuid = ?`, command); err != nil {
							return err
						}
						return refreshVPPAppInstallState(ctx, tx, keys)
					})
				}()
			}
			firstErr, secondErr := <-done, <-done
			require.NoError(t, firstErr)
			require.NoError(t, secondErr)
			require.Equal(t, int32(1), attempts[0].Load())
			require.Equal(t, int32(1), attempts[1].Load())
			assertVPPAppInstallStateMatchesRequests(t, ds)
			assertVPPAppInstallReadsMatchRequests(t, ds)
		})
	}
}

func testVPPAppInstallDiscoveryChangedTargets(t *testing.T, ds *Datastore) {
	for _, verification := range []bool{true, false} {
		t.Run(fmt.Sprintf("verification_uuid=%t", verification), func(t *testing.T) {
			defer TruncateTables(t, ds)
			defer assertVPPAppInstallStateMatchesRequests(t, ds)
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			test.CreateInsertGlobalVPPToken(t, ds)
			platform := fleet.AndroidPlatform
			var host *fleet.Host
			if verification {
				platform = fleet.MacOSPlatform
				host = vppSplitHost(t, ds, "discovery-host", "darwin", nil)
			} else {
				androidHost, err := ds.NewAndroidHost(ctx, createAndroidHost("discovery-host"), false)
				require.NoError(t, err)
				host = androidHost.Host
			}
			oldApp := vppSplitApp(t, ds, "discovery-old", platform, nil)
			newApp := vppSplitApp(t, ds, "discovery-new", platform, nil)
			_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_vpp_software_installs
  (host_id, adam_id, platform, command_uuid, vpp_app_team_id) VALUES (?, ?, ?, 'discovery-old-command', ?)`,
				host.ID, oldApp.AdamID, platform, oldApp.AppTeamID)
			require.NoError(t, err)
			query := pendingVPPAppInstallRequestsForHostQuery
			args := []any{host.ID}
			join := func() error {
				return ds.InsertAndroidSetupExperienceSoftwareInstall(ctx, &fleet.HostAndroidVPPSoftwareInstall{
					HostID: host.ID, AdamID: newApp.AdamID, CommandUUID: "discovery-new-command",
					AssociatedEventID: "5", VPPAppTeamID: newApp.AppTeamID,
				})
			}
			if verification {
				_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_vpp_software_installs
  (host_id, adam_id, platform, command_uuid, vpp_app_team_id) VALUES (?, ?, ?, 'discovery-new-command', ?)`,
					host.ID, newApp.AdamID, platform, newApp.AppTeamID)
				require.NoError(t, err)
				_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE host_vpp_software_installs SET verification_command_uuid = 'discovery-verification' WHERE command_uuid = 'discovery-old-command'`)
				require.NoError(t, err)
				query = `SELECT id, host_id, adam_id, platform FROM host_vpp_software_installs WHERE verification_command_uuid = ?`
				args = []any{"discovery-verification"}
				join = func() error {
					return ds.AssociateMDMInstallToVerificationUUID(ctx, "discovery-new-command", "discovery-verification", host.UUID)
				}
			}
			refreshVPPAppInstallStateForTest(t, ds)
			discovered, joined := make(chan struct{}), make(chan struct{})
			joinErr := make(chan error, 1)
			go func() {
				select {
				case <-discovered:
				case <-ctx.Done():
					joinErr <- ctx.Err()
					return
				}
				joinErr <- join()
				close(joined)
			}()
			barrier := &vppAppInstallDiscoveryBarrier{
				ExtContext: ds.writer(ctx), discoveryQuery: query,
				afterDiscovery: func() error {
					close(discovered)
					select {
					case <-joined:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				},
			}
			var attempts, discoveries int
			err = ds.withVPPAppInstallKeyRetry(ctx, func() ([]hostVPPAppKey, error) {
				discoveries++
				return discoverVPPAppInstallKeys(ctx, barrier, query, args...)
			}, func(tx sqlx.ExtContext, keys []hostVPPAppKey) error {
				attempts++
				if !verification {
					_, activities, err := ds.markAllPendingVPPInstallsAsFailedForHost(ctx, tx, host.ID, "android", softwareTypeVPP, keys)
					if err == nil {
						require.Len(t, activities, 2)
					}
					return err
				}
				ids, keys, err := lockVPPAppInstallRequests(ctx, tx, keys, query, args...)
				if err != nil {
					return err
				}
				for _, id := range ids {
					var command string
					if err := sqlx.GetContext(ctx, tx, &command, `SELECT command_uuid FROM host_vpp_software_installs WHERE id = ?`, id); err != nil {
						return err
					}
					if err := ds.setVPPInstallAsVerified(ctx, tx, host.ID, command, "discovery-verification", keys); err != nil {
						return err
					}
				}
				return nil
			})
			require.NoError(t, <-joinErr)
			require.NoError(t, err)
			require.Equal(t, 2, discoveries)
			require.Equal(t, 2, attempts)
			column, outcome := "verification_failed_at", "failed"
			if verification {
				column, outcome = "verification_at", "installed"
			}
			require.Equal(t, 2, vppSplitCount(t, ds, `SELECT COUNT(*) FROM host_vpp_software_installs WHERE `+column+` IS NOT NULL`))
			var outcomes []string
			require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &outcomes, `SELECT outcome FROM log_vpp_app_install_attempts ORDER BY id`))
			require.Equal(t, []string{outcome, outcome}, outcomes)
			assertVPPAppInstallStateMatchesRequests(t, ds)
			assertVPPAppInstallReadsMatchRequests(t, ds)
		})
	}
}

func testVPPAppInstallDiscoveryRetryLimit(t *testing.T, ds *Datastore) {
	permanentErr := errors.New("permanent transition failure")
	for _, transitionErr := range []error{errVPPAppInstallKeysChanged, permanentErr} {
		var discoveries, attempts int
		err := ds.withVPPAppInstallKeyRetry(t.Context(), func() ([]hostVPPAppKey, error) {
			discoveries++
			return nil, nil
		}, func(tx sqlx.ExtContext, _ []hostVPPAppKey) error {
			attempts++
			_, err := tx.ExecContext(t.Context(), `INSERT INTO vpp_apps (adam_id, platform) VALUES ('rollback-app', 'ios')`)
			if err != nil {
				return err
			}
			return transitionErr
		})
		require.ErrorIs(t, err, transitionErr)
		want := 1
		if errors.Is(transitionErr, errVPPAppInstallKeysChanged) {
			want = 3
			require.ErrorContains(t, err, "retry limit reached")
		}
		require.Equal(t, want, discoveries)
		require.Equal(t, want, attempts)
		require.Zero(t, vppSplitCount(t, ds, `SELECT COUNT(*) FROM vpp_apps WHERE adam_id = 'rollback-app'`))
	}
}

func testVPPAppInstallDiscoveryQueueSnapshot(t *testing.T, ds *Datastore) {
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	test.CreateInsertGlobalVPPToken(t, ds)
	app := vppSplitApp(t, ds, "snapshot-queue-app", fleet.MacOSPlatform, nil)
	host := vppSplitHost(t, ds, "snapshot-queue-host", "darwin", nil)
	commands := make([]string, 3)
	for i := range commands {
		commands[i] = vppSplitQueue(t, ds, host, app, fleet.HostSoftwareInstallOptions{DeferActivation: true})
	}
	_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE upcoming_activities SET activated_at = NOW(6)
  WHERE execution_id IN (?, ?)`, commands[0], commands[1])
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_vpp_software_installs
  (host_id, adam_id, platform, command_uuid, vpp_app_team_id) VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)`,
		host.ID, app.AdamID, app.Platform, commands[0], app.AppTeamID,
		host.ID, app.AdamID, app.Platform, commands[1], app.AppTeamID)
	require.NoError(t, err)
	refreshVPPAppInstallStateForTest(t, ds)
	discovered, completed := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	const query = `SELECT id, host_id, adam_id, platform FROM host_vpp_software_installs WHERE command_uuid = ?`
	go func() {
		barrier := &vppAppInstallDiscoveryBarrier{
			ExtContext: ds.writer(ctx), discoveryQuery: query,
			afterDiscovery: func() error {
				close(discovered)
				select {
				case <-completed:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}
		keys, err := discoverVPPAppInstallKeys(ctx, barrier, query, commands[0])
		if err != nil {
			done <- err
			return
		}
		done <- ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			return ds.setVPPInstallAsVerified(ctx, tx, host.ID, commands[0], "snapshot-verify-a", keys)
		})
	}()
	select {
	case <-discovered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	require.NoError(t, ds.SetVPPInstallAsVerified(ctx, host.ID, commands[1], "snapshot-verify-b"))
	var activated bool
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &activated,
		`SELECT activated_at IS NOT NULL FROM upcoming_activities WHERE execution_id = ?`, commands[2]))
	require.False(t, activated)
	close(completed)
	require.NoError(t, <-done)
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &activated,
		`SELECT activated_at IS NOT NULL FROM upcoming_activities WHERE execution_id = ?`, commands[2]))
	require.True(t, activated)
	var unfinished []string
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &unfinished,
		`SELECT command_uuid FROM host_vpp_software_installs WHERE verification_at IS NULL AND verification_failed_at IS NULL`))
	require.Equal(t, []string{commands[2]}, unfinished)
	assertVPPAppInstallStateMatchesRequests(t, ds)
	assertVPPAppInstallReadsMatchRequests(t, ds)
}

func testVPPAppInstallDiscoveryFailureSnapshot(t *testing.T, ds *Datastore) {
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	test.CreateInsertGlobalVPPToken(t, ds)
	app := vppSplitApp(t, ds, "snapshot-failure-app", fleet.AndroidPlatform, nil)
	host, err := ds.NewAndroidHost(ctx, createAndroidHost("snapshot-failure-host"), false)
	require.NoError(t, err)
	require.NoError(t, ds.InsertAndroidSetupExperienceSoftwareInstall(ctx, &fleet.HostAndroidVPPSoftwareInstall{
		HostID: host.Host.ID, AdamID: app.AdamID, CommandUUID: "snapshot-failure-command",
		AssociatedEventID: "5", VPPAppTeamID: app.AppTeamID,
	}))
	discovered, completed := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	var users []*fleet.User
	var activities []fleet.ActivityDetails
	go func() {
		barrier := &vppAppInstallDiscoveryBarrier{
			ExtContext: ds.writer(ctx), discoveryQuery: pendingVPPAppInstallRequestsForHostQuery,
			afterDiscovery: func() error {
				close(discovered)
				select {
				case <-completed:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}
		keys, err := discoverVPPAppInstallKeys(ctx, barrier, pendingVPPAppInstallRequestsForHostQuery, host.Host.ID)
		if err != nil {
			done <- err
			return
		}
		done <- ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			var err error
			users, activities, err = ds.markAllPendingVPPInstallsAsFailedForHost(ctx, tx, host.Host.ID, "android", softwareTypeVPP, keys)
			return err
		})
	}()
	select {
	case <-discovered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	require.NoError(t, ds.BulkSetVPPInstallsAsVerified(ctx, host.Host.ID, []string{"snapshot-failure-command"}))
	close(completed)
	require.NoError(t, <-done)
	require.Empty(t, users)
	require.Empty(t, activities)
	var failed bool
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &failed,
		`SELECT verification_failed_at IS NOT NULL FROM host_vpp_software_installs WHERE command_uuid = 'snapshot-failure-command'`))
	require.False(t, failed)
	var outcomes []string
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &outcomes, `SELECT outcome FROM log_vpp_app_install_attempts`))
	require.Equal(t, []string{"installed"}, outcomes)
	assertVPPAppInstallStateMatchesRequests(t, ds)
	assertVPPAppInstallReadsMatchRequests(t, ds)
}

func testVPPAppInstallAttemptExpiry(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "expiry-host", "", "expiry-key", "expiry-uuid", time.Now())
	_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('expiry-app', 'android')`)
	require.NoError(t, err)
	require.NoError(t, ds.InsertAndroidSetupExperienceSoftwareInstall(ctx, &fleet.HostAndroidVPPSoftwareInstall{
		HostID: host.ID, AdamID: "expiry-app", CommandUUID: "expiry-command", AssociatedEventID: "5",
	}))
	for _, transition := range []struct {
		name  string
		apply func() error
	}{
		{"Android failed", func() error { return ds.BulkSetVPPInstallsAsFailed(ctx, host.ID, []string{"expiry-command"}) }},
		{"Android installed", func() error { return ds.BulkSetVPPInstallsAsVerified(ctx, host.ID, []string{"expiry-command"}) }},
		{"Apple failed", func() error { return ds.SetVPPInstallAsFailed(ctx, host.ID, "apple-expiry-command", "verify-failed") }},
		{"Apple installed", func() error {
			return ds.SetVPPInstallAsVerified(ctx, host.ID, "apple-expiry-command", "verify-installed")
		}},
	} {
		if transition.name == "Apple failed" {
			_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_vpp_software_installs
    (host_id, adam_id, platform, command_uuid) VALUES (?, 'expiry-app', 'android', 'apple-expiry-command')`, host.ID)
			require.NoError(t, err)
			refreshVPPAppInstallStateForTest(t, ds)
		}
		require.NoError(t, transition.apply(), transition.name)
		var count int
		require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &count, `SELECT COUNT(*) FROM log_vpp_app_install_attempts`))
		require.Equal(t, 1, count, transition.name)
		_, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM log_vpp_app_install_attempts`)
		require.NoError(t, err)
		require.NoError(t, transition.apply(), transition.name)
		require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &count, `SELECT COUNT(*) FROM log_vpp_app_install_attempts`))
		require.Zero(t, count, transition.name)
	}
	// A new configuration cycle remains eligible after retention.
	require.NoError(t, ds.SetAndroidAppInstallPendingApplyConfig(ctx, host.UUID, "expiry-app", 6))
	require.NoError(t, ds.BulkSetVPPInstallsAsVerified(ctx, host.ID, []string{"expiry-command"}))
	var events []string
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &events, `SELECT associated_event_id FROM log_vpp_app_install_attempts`))
	require.Equal(t, []string{"6"}, events)
}

func testVPPAppInstallCanceledMassFailure(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "canceled-host", "", "canceled-key", "canceled-uuid", time.Now())
	_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('canceled-app', 'android')`)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_vpp_software_installs
  (host_id, adam_id, platform, command_uuid, canceled) VALUES (?, 'canceled-app', 'android', 'canceled-command', 1)`, host.ID)
	require.NoError(t, err)
	refreshVPPAppInstallStateForTest(t, ds)
	_, _, err = ds.MarkAllPendingVPPInstallsAsFailedForAndroidHost(ctx, host.ID)
	require.NoError(t, err)
	var count int
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &count, `SELECT COUNT(*) FROM log_vpp_app_install_attempts`))
	require.Zero(t, count)
	var failed bool
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &failed, `SELECT verification_failed_at IS NOT NULL FROM host_vpp_software_installs`))
	require.True(t, failed)
}

func testVPPAppInstallAttemptLog(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "android-log", "", "android-log-key", "android-log-uuid", time.Now())
	_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('log-app', 'android')`)
	require.NoError(t, err)
	require.NoError(t, ds.InsertAndroidSetupExperienceSoftwareInstall(ctx, &fleet.HostAndroidVPPSoftwareInstall{
		HostID: host.ID, AdamID: "log-app", CommandUUID: "log-command", AssociatedEventID: "5",
	}))
	// The host argument doesn't restrict the legacy bulk UPDATE.
	require.NoError(t, ds.BulkSetVPPInstallsAsFailed(ctx, host.ID+1, []string{"log-command"}))
	require.NoError(t, ds.BulkSetVPPInstallsAsVerified(ctx, host.ID+1, []string{"log-command"}))
	type attempt struct {
		ID                uint      `db:"id"`
		CommandUUID       string    `db:"command_uuid"`
		Outcome           string    `db:"outcome"`
		AssociatedEventID string    `db:"associated_event_id"`
		FinishedAt        time.Time `db:"finished_at"`
	}
	readLog := func() []attempt {
		var rows []attempt
		require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &rows,
			`SELECT id, command_uuid, outcome, associated_event_id, finished_at FROM log_vpp_app_install_attempts ORDER BY id`))
		return rows
	}
	before := readLog()
	require.Len(t, before, 2)
	require.Equal(t, "failed", before[0].Outcome)
	require.Equal(t, "installed", before[1].Outcome)
	require.Equal(t, "5", before[1].AssociatedEventID)
	require.NoError(t, ds.BulkSetVPPInstallsAsVerified(ctx, host.ID, []string{"log-command"}))
	require.Equal(t, before, readLog())
	require.NoError(t, ds.SetAndroidAppInstallPendingApplyConfig(ctx, host.UUID, "log-app", 6))
	require.NoError(t, ds.BulkSetVPPInstallsAsVerified(ctx, host.ID, []string{"log-command"}))
	after := readLog()
	require.Len(t, after, 3)
	require.Equal(t, before, after[:2])
	require.Equal(t, "installed", after[2].Outcome)
	require.Equal(t, "6", after[2].AssociatedEventID)
	assertVPPAppInstallStateMatchesRequests(t, ds)
}

func assertVPPAppInstallReadsMatchRequests(t *testing.T, ds *Datastore) {
	t.Helper()
	ctx := t.Context()
	params := map[string]any{
		"software_status_installed": fleet.SoftwareInstalled,
		"software_status_failed":    fleet.SoftwareInstallFailed,
		"software_status_pending":   fleet.SoftwareInstallPending,
		"mdm_status_acknowledged":   fleet.MDMAppleStatusAcknowledged,
		"mdm_status_error":          fleet.MDMAppleStatusError,
		"mdm_status_format_error":   fleet.MDMAppleStatusCommandFormatError,
	}
	named := func(query string) (string, []any) {
		stmt, args, err := sqlx.Named(query, params)
		require.NoError(t, err)
		stmt, args, err = sqlx.In(stmt, args...)
		require.NoError(t, err)
		return stmt, args
	}
	compare := func(name, legacy, state string) {
		t.Helper()
		legacyStmt, legacyArgs := named(legacy)
		stateStmt, stateArgs := named(state)
		require.Equal(t, vppInstallReadRows(t, ds, legacyStmt, legacyArgs...), vppInstallReadRows(t, ds, stateStmt, stateArgs...), name)
	}
	var versions []struct {
		ID       uint   `db:"id"`
		AdamID   string `db:"adam_id"`
		Platform string `db:"platform"`
	}
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &versions, `SELECT id, adam_id, platform FROM vpp_apps_teams ORDER BY id`))
	var allVersions []uint
	byApp := make(map[fleet.VPPAppID][]uint)
	for _, v := range versions {
		allVersions = append(allVersions, v.ID)
		key := fleet.VPPAppID{AdamID: v.AdamID, Platform: fleet.InstallableDevicePlatform(v.Platform)}
		byApp[key] = append(byApp[key], v.ID)
	}
	versionSets := [][]uint{allVersions}
	for _, ids := range byApp {
		versionSets = append(versionSets, ids)
	}
	for _, v := range versions {
		versionSets = append(versionSets, []uint{v.ID})
	}
	for _, ids := range versionSets {
		if len(ids) == 0 {
			continue
		}
		params["version_ids"] = ids
		compare("summary and version filter", legacyVPPAppVersionHostStatusesQuery, vppAppVersionHostStatusesQuery)
	}

	// Correlating the preserved host query checks every host without one round trip per host.
	for _, filter := range []string{"", "(vat.self_service = 1) AND ", "FALSE AND "} {
		hostQuery := func(query string) string {
			query = fmt.Sprintf(query, filter, vppAppHostStatusNamedQuery("hvsi", "ncr", "status"), filter)
			query = strings.NewReplacer(":host_id", "oracle_host.host_id", ":global_or_team_id", "oracle_host.fleet_id").Replace(query)
			return `SELECT oracle_host.host_id, installs.* FROM (
				SELECT id AS host_id, COALESCE(team_id, 0) AS fleet_id FROM hosts
				UNION SELECT DISTINCT host_id, 0 FROM host_vpp_software_installs WHERE host_id NOT IN (SELECT id FROM hosts)
			) oracle_host JOIN LATERAL (` + query + `) installs ON TRUE`
		}
		compare("host software and self service", hostQuery(legacyHostVPPInstallsQuery), hostQuery(hostVPPInstallsQuery))
	}

	var apps []fleet.VPPAppID
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &apps, `SELECT adam_id, platform FROM vpp_apps`))
	for _, app := range apps {
		params["adam_id"], params["platform"] = app.AdamID, app.Platform
		for _, status := range []fleet.SoftwareInstallerStatus{fleet.SoftwareInstalled, fleet.SoftwareInstallFailed, fleet.SoftwareFailed} {
			params["status"] = status
			if status == fleet.SoftwareFailed {
				params["status"] = fleet.SoftwareInstallFailed
			}
			join, args, err := ds.vppAppJoin(ctx, app, nil, status)
			require.NoError(t, err)
			legacyStmt, legacyArgs := named(`SELECT h.id FROM hosts h ` + fmt.Sprintf(legacyVPPAppTerminalJoinQuery, vppAppHostStatusNamedQuery("ranked", "ncr", "")))
			require.Equal(t, vppInstallReadRows(t, ds, legacyStmt, legacyArgs...), vppInstallReadRows(t, ds, `SELECT h.id FROM hosts h `+join, args...), "title status filter %v %s", app, status)
		}
	}

	for _, version := range versions {
		params["vpp_app_team_id"] = version.ID
		versionIDs, err := ds.GetAppStoreAppVersionIDsFromSpecificVersion(ctx, version.ID)
		require.NoError(t, err)
		params["version_ids"] = versionIDs
		for _, status := range []fleet.SoftwareInstallerStatus{fleet.SoftwareInstalled, fleet.SoftwareInstallFailed, fleet.SoftwareInstallPending} {
			params["status"] = status
			join, args, err := ds.vppAppJoin(ctx, fleet.VPPAppID{AdamID: version.AdamID, Platform: fleet.InstallableDevicePlatform(version.Platform)}, new(version.ID), status)
			require.NoError(t, err)
			legacyStmt, legacyArgs := named(fmt.Sprintf(`SELECT h.id FROM hosts h JOIN (%s) hss ON hss.host_id = h.id AND hss.vpp_app_team_id = :vpp_app_team_id AND hss.status = :status`, legacyVPPAppVersionHostStatusesQuery))
			require.Equal(t, vppInstallReadRows(t, ds, legacyStmt, legacyArgs...), vppInstallReadRows(t, ds, `SELECT h.id FROM hosts h `+join, args...), "version status filter %d %s", version.ID, status)
		}
	}

	var hostIDs []uint
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &hostIDs, `SELECT id FROM hosts`))
	if len(hostIDs) == 0 {
		return
	}
	var fleetIDs []uint
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &fleetIDs, `SELECT DISTINCT global_or_team_id FROM vpp_apps_teams`))
	for _, fleetID := range fleetIDs {
		args := []any{fleetID, []string{"ios", "ipados", "darwin", "android"}, hostIDs, any(nil)}
		if fleetID != 0 {
			args[3] = fleetID
		}
		legacyStmt, legacyArgs, err := sqlx.In(legacyHostAppStoreAppInstallVersionsQuery, args...)
		require.NoError(t, err)
		stateStmt, stateArgs, err := sqlx.In(hostAppStoreAppInstallVersionsQuery, args...)
		require.NoError(t, err)
		require.Equal(t, vppInstallReadRows(t, ds, legacyStmt, legacyArgs...), vppInstallReadRows(t, ds, stateStmt, stateArgs...), "config reconcile fleet %d", fleetID)
	}
}

// These reads have no outer ORDER BY; sorting preserves duplicate rows while ignoring incidental query-plan order.
func vppInstallReadRows(t *testing.T, ds *Datastore, query string, args ...any) []string {
	t.Helper()
	rows, err := ds.writer(t.Context()).QueryxContext(t.Context(), query, args...)
	require.NoError(t, err)
	defer rows.Close()
	var result []string
	for rows.Next() {
		values, err := rows.SliceScan()
		require.NoError(t, err)
		for i, value := range values {
			if b, ok := value.([]byte); ok {
				values[i] = string(b)
			}
		}
		value, err := json.Marshal(values)
		require.NoError(t, err)
		result = append(result, string(value))
	}
	require.NoError(t, rows.Err())
	slices.Sort(result)
	return result
}

const legacyVPPAppVersionHostStatusesQuery = `
WITH

-- select the latest upcoming activity of each host and version
upcoming AS (
	SELECT
		ua.host_id,
		vaua.vpp_app_team_id,
		MAX(ua.id) AS id
	FROM
		upcoming_activities ua
		JOIN vpp_app_upcoming_activities vaua ON ua.id = vaua.upcoming_activity_id
		JOIN vpp_apps_teams vat ON vat.id = vaua.vpp_app_team_id
		JOIN hosts h ON ua.host_id = h.id
	WHERE
		ua.activity_type = 'vpp_app_install'
		AND vaua.vpp_app_team_id IN (:version_ids)
		AND COALESCE(h.team_id, 0) = vat.global_or_team_id
	GROUP BY ua.host_id, vaua.vpp_app_team_id
),

-- select the latest successful, failed and unfinished install of each host and version, reading only the index
-- NOTE if you change this logic make sure to change vppAppHostStatusNamedQuery accordingly
latest_installs AS (
	SELECT
		host_id,
		vpp_app_team_id,
		MAX(IF(verification_at IS NOT NULL, id, NULL)) AS installed_id,
		MAX(IF(verification_at IS NULL AND verification_failed_at IS NOT NULL, id, NULL)) AS failed_id,
		MAX(IF(verification_at IS NULL AND verification_failed_at IS NULL, id, NULL)) AS unfinished_id
	FROM host_vpp_software_installs
	WHERE vpp_app_team_id IN (:version_ids) AND canceled = 0 AND removed = 0
	GROUP BY vpp_app_team_id, host_id
),

-- select the installs of each host and version in the version's fleet, counting unfinished Android installs as pending because Android installs bypass upcoming activities
attempts AS (
	SELECT
		li.host_id,
		li.vpp_app_team_id,
		li.installed_id,
		li.failed_id,
		IF(vat.platform = 'android', li.unfinished_id, NULL) AS pending_id
	FROM latest_installs li
	JOIN hosts h ON h.id = li.host_id
	JOIN vpp_apps_teams vat ON vat.id = li.vpp_app_team_id
	WHERE COALESCE(h.team_id, 0) = vat.global_or_team_id

	UNION ALL

	-- deleting a version sets its installs' vpp_app_team_id to NULL, select them so they still supersede older installs of the remaining versions
	SELECT
		hvsi.host_id,
		NULL,
		MAX(IF(hvsi.verification_at IS NOT NULL, hvsi.id, NULL)),
		MAX(IF(hvsi.verification_at IS NULL AND hvsi.verification_failed_at IS NOT NULL, hvsi.id, NULL)),
		NULL
	FROM host_vpp_software_installs hvsi
	JOIN (SELECT DISTINCT adam_id, platform FROM vpp_apps_teams WHERE id IN (:version_ids)) app ON app.adam_id = hvsi.adam_id AND app.platform = hvsi.platform
	WHERE hvsi.vpp_app_team_id IS NULL AND hvsi.canceled = 0 AND hvsi.removed = 0
	GROUP BY hvsi.host_id
),

-- keep the version with the latest install of each status on each host
latest AS (
	SELECT
		host_id,
		vpp_app_team_id,
		status,
		install_id,
		ROW_NUMBER() OVER (
			PARTITION BY host_id, status
			ORDER BY install_id DESC
		) AS rn
	FROM (
		SELECT host_id, vpp_app_team_id, :software_status_installed AS status, installed_id AS install_id FROM attempts WHERE installed_id IS NOT NULL
		UNION ALL
		SELECT host_id, vpp_app_team_id, :software_status_failed, failed_id FROM attempts WHERE failed_id IS NOT NULL
		UNION ALL
		SELECT host_id, vpp_app_team_id, :software_status_pending, pending_id FROM attempts WHERE pending_id IS NOT NULL
		UNION ALL
		-- Use the upcoming activity id as the install id, it's only compared with other upcoming activities because Android installs bypass them and a title's versions are all on one platform
		SELECT host_id, vpp_app_team_id, :software_status_pending, id FROM upcoming
	) statuses
)

-- Drop failed installs that are older than the host's latest successful install
SELECT
	l.host_id,
	l.vpp_app_team_id,
	l.status
FROM latest l
LEFT JOIN latest i ON i.host_id = l.host_id AND i.status = :software_status_installed AND i.rn = 1
WHERE
	l.rn = 1
	AND l.vpp_app_team_id IS NOT NULL
	AND (l.status != :software_status_failed OR i.host_id IS NULL OR l.install_id > i.install_id)`

const legacyHostVPPInstallsQuery = `
	(   -- upcoming_vpp_install
			SELECT id, last_install_install_uuid, last_install_installed_at, vpp_app_adam_id, vpp_app_self_service, status FROM (
				SELECT
						vpp_apps.title_id AS id,
						ua.execution_id AS last_install_install_uuid,
						ua.created_at AS last_install_installed_at,
						vaua.adam_id AS vpp_app_adam_id,
						vat.self_service AS vpp_app_self_service,
						'pending_install' AS status,
						ROW_NUMBER() OVER (
							PARTITION BY vaua.adam_id, vaua.platform, ua.activity_type
							ORDER BY ua.priority ASC, ua.created_at DESC, ua.id DESC
						) AS rn
				FROM
						upcoming_activities ua
				INNER JOIN
						vpp_app_upcoming_activities vaua ON ua.id = vaua.upcoming_activity_id
				LEFT JOIN
					-- use the install's version when it's on the host's fleet, else the first-added version there, such as for installs of a deleted version or from another fleet
					vpp_apps_teams vat ON vat.id = (
						SELECT vat_fleet.id FROM vpp_apps_teams vat_fleet
						WHERE vat_fleet.adam_id = vaua.adam_id AND vat_fleet.platform = vaua.platform AND vat_fleet.global_or_team_id = :global_or_team_id
						ORDER BY vat_fleet.id <=> vaua.vpp_app_team_id DESC, vat_fleet.id
						LIMIT 1
					)
				INNER JOIN
					vpp_apps ON vaua.adam_id = vpp_apps.adam_id AND vaua.platform = vpp_apps.platform
				WHERE
					-- selfServiceFilter
					%s
				 ua.host_id = :host_id AND
				 ua.activity_type = 'vpp_app_install'
			) ranked
			WHERE rn = 1
) UNION (
			-- last_vpp_install
			SELECT
				vpp_apps.title_id AS id,
				hvsi.command_uuid AS last_install_install_uuid,
				hvsi.created_at AS last_install_installed_at,
				hvsi.adam_id AS vpp_app_adam_id,
				vat.self_service AS vpp_app_self_service,
				-- vppAppHostStatusNamedQuery(hvsi, ncr, status)
				%s
			FROM
				(
					SELECT ranked_hvsi.*, ROW_NUMBER() OVER (
						PARTITION BY ranked_hvsi.adam_id, ranked_hvsi.platform
						ORDER BY ranked_hvsi.created_at DESC, ranked_hvsi.id DESC
					) AS rn
					FROM host_vpp_software_installs ranked_hvsi
					WHERE
						ranked_hvsi.host_id = :host_id AND
						ranked_hvsi.removed = 0 AND
						ranked_hvsi.canceled = 0
				) hvsi
			LEFT JOIN
				nano_command_results ncr ON ncr.command_uuid = hvsi.command_uuid
			INNER JOIN
				-- use the install's version when it's on the host's fleet, else the first-added version there, such as for installs of a deleted version or from another fleet
				vpp_apps_teams vat ON vat.id = (
					SELECT vat_fleet.id FROM vpp_apps_teams vat_fleet
					WHERE vat_fleet.adam_id = hvsi.adam_id AND vat_fleet.platform = hvsi.platform AND vat_fleet.global_or_team_id = :global_or_team_id
					ORDER BY vat_fleet.id <=> hvsi.vpp_app_team_id DESC, vat_fleet.id
					LIMIT 1
				)
			INNER JOIN
				vpp_apps ON hvsi.adam_id = vpp_apps.adam_id AND hvsi.platform = vpp_apps.platform
			WHERE
				-- selfServiceFilter
				%s
				-- Android installs never produce nano_command_results (they use Google's
				-- Android Management API instead of nanoMDM), so ncr is always NULL for
				-- Android rows. No NCR filter is applied here — all statuses (pending,
				-- failed, installed) are shown, which is intentional for the host software
				-- list. Compare with vpp.go / software_installers.go which filter by NCR
				-- for per-app aggregate counts, different semantics.
				hvsi.rn = 1 AND
				-- a queued upcoming install supersedes history; checked on the winner
				-- only since it depends solely on the app, not the row
				NOT EXISTS (
					SELECT 1
					FROM
						upcoming_activities ua
					INNER JOIN
						vpp_app_upcoming_activities vaua ON ua.id = vaua.upcoming_activity_id
					WHERE
						ua.host_id = hvsi.host_id AND
						vaua.adam_id = hvsi.adam_id AND
						vaua.platform = hvsi.platform AND
						ua.activity_type = 'vpp_app_install'
				)
)
`

const legacyVPPAppTerminalJoinQuery = `JOIN (
SELECT
	ranked.host_id
FROM (
	SELECT
		hvsi.host_id,
		hvsi.command_uuid,
		hvsi.verification_at,
		hvsi.verification_failed_at,
		ROW_NUMBER() OVER (
			PARTITION BY hvsi.host_id
			ORDER BY hvsi.created_at DESC, hvsi.id DESC
		) AS rn
	FROM
		host_vpp_software_installs hvsi
	WHERE
		hvsi.adam_id = :adam_id
		AND hvsi.platform = :platform
		AND hvsi.canceled = 0
) ranked
LEFT JOIN
	nano_command_results ncr ON ncr.command_uuid = ranked.command_uuid
WHERE
	ranked.rn = 1
	-- Allow rows with no nano_command_results — Fleet-side pre-flight failures
	-- (unresolvable managed-config Fleet variable) record only the install row
	-- with verification_failed_at set, no MDM command. See same comment in
	-- vpp.go GetSummaryHostVPPAppInstalls. Checked after ranking, so a host whose
	-- most recent row lacks a command result is dropped rather than falling back.
	AND (ncr.id IS NOT NULL OR ranked.verification_at IS NOT NULL OR ranked.verification_failed_at IS NOT NULL OR (:platform = 'android' AND ncr.id IS NULL))
	AND (%s) = :status
	AND ranked.host_id NOT IN (
		SELECT
			ua.host_id
		FROM
			upcoming_activities ua
			JOIN vpp_app_upcoming_activities vaua ON ua.id = vaua.upcoming_activity_id
		WHERE
			vaua.adam_id = :adam_id
			AND vaua.platform = :platform
			AND ua.activity_type = 'vpp_app_install'
	)
) hss ON hss.host_id = h.id
`

const legacyHostAppStoreAppInstallVersionsQuery = `
SELECT
	app.title_id,
	h.id AS host_id,
	latest.vpp_app_team_id,
	latest.created_at,
	latest.needs_resend
FROM
	hosts h
	-- iOS and iPadOS apps are separate titles, so pair each host only with the fleet's apps of its own platform
	JOIN (
		SELECT DISTINCT va.title_id, va.adam_id, va.platform
		FROM
			vpp_apps_teams vat
			JOIN vpp_apps va ON va.adam_id = vat.adam_id AND va.platform = vat.platform
		WHERE vat.global_or_team_id = ? AND vat.platform IN (?)
	) app ON app.platform = h.platform
	-- canceled and failed installs didn't change the app on the host, so read the latest install that wasn't either
	JOIN LATERAL (
		SELECT hvsi.vpp_app_team_id, hvsi.created_at, hvsi.needs_resend
		FROM host_vpp_software_installs hvsi
		WHERE
			hvsi.host_id = h.id AND
			hvsi.adam_id = app.adam_id AND
			hvsi.platform = app.platform AND
			hvsi.canceled = 0 AND
			hvsi.verification_failed_at IS NULL
		ORDER BY hvsi.created_at DESC, hvsi.id DESC
		LIMIT 1
	) latest ON TRUE
WHERE
	h.id IN (?) AND
	h.team_id <=> ? AND
	EXISTS (
		SELECT 1
		FROM
			host_software hs
			JOIN software s ON s.id = hs.software_id
		WHERE
			hs.host_id = h.id AND
			s.title_id = app.title_id
	)
`
