package mysql

import (
	"context"
	"database/sql"
	"sort"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestSoftwareTitles(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"SyncHostsSoftwareTitles", testSoftwareSyncHostsSoftwareTitles},
		{"OrderSoftwareTitles", testOrderSoftwareTitles},
		{"TeamFilterSoftwareTitles", testTeamFilterSoftwareTitles},
		{"ListSoftwareTitlesInstallersOnly", testListSoftwareTitlesInstallersOnly},
		{"ListSoftwareTitlesAvailableForInstallFilter", testListSoftwareTitlesAvailableForInstallFilter},
		{"ListSoftwareTitlesAllTeams", testListSoftwareTitlesAllTeams},
		{"UploadedSoftwareExists", testUploadedSoftwareExists},
		{"ListSoftwareTitlesVulnerabilityFilters", testListSoftwareTitlesVulnerabilityFilters},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testSoftwareSyncHostsSoftwareTitles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	cmpNameVersionCount := func(want, got []fleet.SoftwareTitleListResult) {
		cmp := make([]fleet.SoftwareTitleListResult, len(got))
		for i, sw := range got {
			cmp[i] = fleet.SoftwareTitleListResult{Name: sw.Name, HostsCount: sw.HostsCount}
		}
		require.ElementsMatch(t, want, cmp)
	}

	// this check ensures that the total number of rows in
	// software_title_host_counts matches the expected value.
	checkTableTotalCount := func(want int) {
		t.Helper()
		var tableCount int
		err := ds.writer(context.Background()).Get(&tableCount, "SELECT COUNT(*) FROM software_titles_host_counts")
		require.NoError(t, err)
		require.Equal(t, want, tableCount)
	}

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())

	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	software2 := []fleet.Software{
		{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}

	_, err := ds.UpdateHostSoftware(ctx, host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	globalOpts := fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending}}
	globalCounts := listSoftwareTitlesCheckCount(t, ds, 2, 2, globalOpts)

	want := []fleet.SoftwareTitleListResult{
		{Name: "foo", HostsCount: 2},
		{Name: "bar", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)
	checkTableTotalCount(4)

	// update host2, remove "bar" software
	software2 = []fleet.Software{
		{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	globalCounts = listSoftwareTitlesCheckCount(t, ds, 1, 1, globalOpts)
	want = []fleet.SoftwareTitleListResult{
		{Name: "foo", HostsCount: 2},
	}
	cmpNameVersionCount(want, globalCounts)
	checkTableTotalCount(2)

	// create a software title entry without any host and any counts
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO software_titles (name, source) VALUES ('baz', 'testing')`)
	require.NoError(t, err)

	// listing does not return the new software title entry
	allSw := listSoftwareTitlesCheckCount(t, ds, 1, 1, fleet.SoftwareTitleListOptions{})
	want = []fleet.SoftwareTitleListResult{
		{Name: "foo", HostsCount: 2},
	}
	cmpNameVersionCount(want, allSw)

	// create 2 teams and assign a new host to each
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	host3 := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now())
	require.NoError(t, ds.AddHostsToTeam(ctx, &team1.ID, []uint{host3.ID}))
	host4 := test.NewHost(t, ds, "host4", "", "host4key", "host4uuid", time.Now())
	require.NoError(t, ds.AddHostsToTeam(ctx, &team2.ID, []uint{host4.ID}))

	// assign existing host1 to team1 too, so we have a team with multiple hosts
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))
	// use some software for host3 and host4
	software3 := []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	software4 := []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}

	_, err = ds.UpdateHostSoftware(ctx, host3.ID, software3)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host4.ID, software4)
	require.NoError(t, err)

	// at this point, there's no counts per team, only global counts
	globalCounts = listSoftwareTitlesCheckCount(t, ds, 1, 1, globalOpts)
	want = []fleet.SoftwareTitleListResult{
		{Name: "foo", HostsCount: 2},
	}
	cmpNameVersionCount(want, globalCounts)
	checkTableTotalCount(2)

	team1Opts := fleet.SoftwareTitleListOptions{
		TeamID:      ptr.Uint(team1.ID),
		ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending},
	}
	team1Counts := listSoftwareTitlesCheckCount(t, ds, 0, 0, team1Opts)
	want = []fleet.SoftwareTitleListResult{}
	cmpNameVersionCount(want, team1Counts)
	checkTableTotalCount(2)

	// after a call to Calculate, the global counts are updated and the team counts appear
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	globalCounts = listSoftwareTitlesCheckCount(t, ds, 2, 2, globalOpts)
	want = []fleet.SoftwareTitleListResult{
		{Name: "foo", HostsCount: 4},
		{Name: "bar", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)

	team1Counts = listSoftwareTitlesCheckCount(t, ds, 1, 1, team1Opts)
	want = []fleet.SoftwareTitleListResult{
		{Name: "foo", HostsCount: 2},
	}
	cmpNameVersionCount(want, team1Counts)

	// composite pk (software_title_id, team_id), so we expect more rows
	checkTableTotalCount(6)

	team2Opts := fleet.SoftwareTitleListOptions{
		TeamID:      ptr.Uint(team2.ID),
		ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending},
	}
	team2Counts := listSoftwareTitlesCheckCount(t, ds, 2, 2, team2Opts)
	want = []fleet.SoftwareTitleListResult{
		{Name: "foo", HostsCount: 1},
		{Name: "bar", HostsCount: 1},
	}
	cmpNameVersionCount(want, team2Counts)

	// update host4 (team2), remove "bar" software
	software4 = []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}

	_, err = ds.UpdateHostSoftware(ctx, host4.ID, software4)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	globalCounts = listSoftwareTitlesCheckCount(t, ds, 1, 1, globalOpts)
	want = []fleet.SoftwareTitleListResult{
		{Name: "foo", HostsCount: 4},
	}
	cmpNameVersionCount(want, globalCounts)

	team1Counts = listSoftwareTitlesCheckCount(t, ds, 1, 1, team1Opts)
	want = []fleet.SoftwareTitleListResult{
		{Name: "foo", HostsCount: 2},
	}
	cmpNameVersionCount(want, team1Counts)

	team2Counts = listSoftwareTitlesCheckCount(t, ds, 1, 1, team2Opts)
	want = []fleet.SoftwareTitleListResult{
		{Name: "foo", HostsCount: 1},
	}
	cmpNameVersionCount(want, team2Counts)

	checkTableTotalCount(4)

	// update host4 (team2), remove all software
	software4 = []fleet.Software{}
	_, err = ds.UpdateHostSoftware(ctx, host4.ID, software4)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))
	listSoftwareTitlesCheckCount(t, ds, 0, 0, team2Opts)

	// delete team
	require.NoError(t, ds.DeleteTeam(ctx, team2.ID))

	// this call will remove team2 from the software host counts table
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	globalCounts = listSoftwareTitlesCheckCount(t, ds, 1, 1, globalOpts)
	want = []fleet.SoftwareTitleListResult{
		{Name: "foo", HostsCount: 3},
	}
	cmpNameVersionCount(want, globalCounts)

	team1Counts = listSoftwareTitlesCheckCount(t, ds, 1, 1, team1Opts)
	want = []fleet.SoftwareTitleListResult{
		{Name: "foo", HostsCount: 2},
	}
	cmpNameVersionCount(want, team1Counts)

	listSoftwareTitlesCheckCount(t, ds, 0, 0, team2Opts)
	checkTableTotalCount(3)
}

func testOrderSoftwareTitles(t *testing.T, ds *Datastore) {
	//
	// All tests below are in hosts in "No team".
	//

	ctx := context.Background()

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now())

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions", Browser: "chrome"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions", Browser: "chrome"},
		{Name: "foo", Version: "0.0.3", Source: "deb_packages"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}
	software2 := []fleet.Software{
		{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions", Browser: "chrome"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions", Browser: "chrome"},
		{Name: "foo", Version: "0.0.3", Source: "deb_packages"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}
	software3 := []fleet.Software{
		{Name: "foo", Version: "v0.0.2", Source: "rpm_packages"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "baz", Version: "0.0.3", Source: "chrome_extensions", Browser: "edge"},
		{Name: "baz", Version: "0.0.3", Source: "chrome_extensions", Browser: "chrome"},
	}

	_, err := ds.UpdateHostSoftware(ctx, host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host3.ID, software3)
	require.NoError(t, err)

	// create a software installer not installed on any host
	installer1, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:         "installer1",
		Source:        "apps",
		InstallScript: "echo",
		Filename:      "installer1.pkg",
		UserID:        user1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, installer1)
	// make installer1 "self-service" available
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE software_installers SET self_service = 1 WHERE id = ?`, installer1)
		return err
	})
	// create a software installer with an install request on host1
	installer2, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:         "installer2",
		Source:        "apps",
		InstallScript: "echo",
		Filename:      "installer2.pkg",
		UserID:        user1.ID,
	})
	require.NoError(t, err)
	_, err = ds.InsertSoftwareInstallRequest(ctx, host1.ID, installer2, false, nil)
	require.NoError(t, err)

	test.CreateInsertGlobalVPPToken(t, ds)

	// create a VPP app not installed anywhere
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app.vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_1", Platform: fleet.IPadOSPlatform}},
	}, nil)
	require.NoError(t, err)

	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	// primary sort is "hosts_count DESC", followed by "name ASC, source ASC, browser ASC"
	titles, _, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey:       "hosts_count",
			OrderDirection: fleet.OrderDescending,
		},
		TeamID: ptr.Uint(0),
	}, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.NoError(t, err)
	require.Len(t, titles, 10)
	i := 0
	require.Equal(t, "bar", titles[i].Name)
	require.Equal(t, "deb_packages", titles[i].Source)
	require.Nil(t, titles[i].SoftwarePackage)
	require.Nil(t, titles[i].AppStoreApp)
	i++
	require.Equal(t, "foo", titles[i].Name)
	require.Equal(t, "chrome_extensions", titles[i].Source)
	require.Nil(t, titles[i].SoftwarePackage)
	require.Nil(t, titles[i].AppStoreApp)
	i++
	require.Equal(t, "foo", titles[i].Name)
	require.Equal(t, "deb_packages", titles[i].Source)
	require.Nil(t, titles[i].SoftwarePackage)
	require.Nil(t, titles[i].AppStoreApp)
	i++
	require.Equal(t, "bar", titles[i].Name)
	require.Equal(t, "apps", titles[i].Source)
	require.Nil(t, titles[i].SoftwarePackage)
	require.Nil(t, titles[i].AppStoreApp)
	i++
	require.Equal(t, "baz", titles[i].Name)
	require.Equal(t, "chrome_extensions", titles[i].Source)
	require.Equal(t, "chrome", titles[i].Browser)
	require.Nil(t, titles[i].SoftwarePackage)
	require.Nil(t, titles[i].AppStoreApp)
	i++
	require.Equal(t, "baz", titles[i].Name)
	require.Equal(t, "chrome_extensions", titles[i].Source)
	require.Equal(t, "edge", titles[i].Browser)
	require.Nil(t, titles[i].SoftwarePackage)
	require.Nil(t, titles[i].AppStoreApp)
	i++
	require.Equal(t, "foo", titles[i].Name)
	require.Equal(t, "rpm_packages", titles[i].Source)
	require.Nil(t, titles[i].SoftwarePackage)
	require.Nil(t, titles[i].AppStoreApp)
	i++
	require.Equal(t, "installer1", titles[i].Name)
	require.Equal(t, "apps", titles[i].Source)
	require.NotNil(t, titles[i].SoftwarePackage)
	require.Nil(t, titles[i].AppStoreApp)
	i++
	require.Equal(t, "installer2", titles[i].Name)
	require.Equal(t, "apps", titles[i].Source)
	require.NotNil(t, titles[i].SoftwarePackage)
	require.Nil(t, titles[i].AppStoreApp)
	i++
	require.Equal(t, "vpp1", titles[i].Name)
	assert.Equal(t, "ipados_apps", titles[i].Source)
	require.Nil(t, titles[i].SoftwarePackage)
	require.NotNil(t, titles[i].AppStoreApp)

	// primary sort is "hosts_count ASC", followed by "name ASC, source ASC, browser ASC"
	titles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey:       "hosts_count",
			OrderDirection: fleet.OrderAscending,
		},
		TeamID: ptr.Uint(0),
	}, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.NoError(t, err)
	require.Len(t, titles, 10)
	i = 0
	require.Equal(t, "installer1", titles[i].Name)
	require.Equal(t, "apps", titles[i].Source)
	i++
	require.Equal(t, "installer2", titles[i].Name)
	require.Equal(t, "apps", titles[i].Source)
	i++
	require.Equal(t, "vpp1", titles[i].Name)
	assert.Equal(t, "ipados_apps", titles[i].Source)
	i++
	require.Equal(t, "bar", titles[i].Name)
	require.Equal(t, "apps", titles[i].Source)
	i++
	require.Equal(t, "baz", titles[i].Name)
	require.Equal(t, "chrome_extensions", titles[i].Source)
	require.Equal(t, "chrome", titles[i].Browser)
	i++
	require.Equal(t, "baz", titles[i].Name)
	require.Equal(t, "chrome_extensions", titles[i].Source)
	require.Equal(t, "edge", titles[i].Browser)
	i++
	require.Equal(t, "foo", titles[i].Name)
	require.Equal(t, "rpm_packages", titles[i].Source)
	i++
	require.Equal(t, "bar", titles[i].Name)
	require.Equal(t, "deb_packages", titles[i].Source)
	i++
	require.Equal(t, "foo", titles[i].Name)
	require.Equal(t, "chrome_extensions", titles[i].Source)
	i++
	require.Equal(t, "foo", titles[i].Name)
	require.Equal(t, "deb_packages", titles[i].Source)

	// primary sort is "name ASC", followed by "host_count DESC, source ASC, browser ASC"
	titles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey:       "name",
			OrderDirection: fleet.OrderAscending,
		},
		TeamID: ptr.Uint(0),
	}, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.NoError(t, err)
	require.Len(t, titles, 10)
	i = 0
	require.Equal(t, "bar", titles[i].Name)
	require.Equal(t, "deb_packages", titles[i].Source)
	i++
	require.Equal(t, "bar", titles[i].Name)
	require.Equal(t, "apps", titles[i].Source)
	i++
	require.Equal(t, "baz", titles[i].Name)
	require.Equal(t, "chrome_extensions", titles[i].Source)
	require.Equal(t, "chrome", titles[i].Browser)
	i++
	require.Equal(t, "baz", titles[i].Name)
	require.Equal(t, "chrome_extensions", titles[i].Source)
	require.Equal(t, "edge", titles[i].Browser)
	i++
	require.Equal(t, "foo", titles[i].Name)
	require.Equal(t, "chrome_extensions", titles[i].Source)
	i++
	require.Equal(t, "foo", titles[i].Name)
	require.Equal(t, "deb_packages", titles[i].Source)
	i++
	require.Equal(t, "foo", titles[i].Name)
	require.Equal(t, "rpm_packages", titles[i].Source)
	i++
	require.Equal(t, "installer1", titles[i].Name)
	require.Equal(t, "apps", titles[i].Source)
	i++
	require.Equal(t, "installer2", titles[i].Name)
	require.Equal(t, "apps", titles[i].Source)
	i++
	require.Equal(t, "vpp1", titles[i].Name)
	assert.Equal(t, "ipados_apps", titles[i].Source)

	// primary sort is "name DESC", followed by "host_count DESC, source ASC, browser ASC"
	titles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey:       "name",
			OrderDirection: fleet.OrderDescending,
		},
		TeamID: ptr.Uint(0),
	}, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.NoError(t, err)
	require.Len(t, titles, 10)
	i = 0
	require.Equal(t, "vpp1", titles[i].Name)
	assert.Equal(t, "ipados_apps", titles[i].Source)
	i++
	require.Equal(t, "installer2", titles[i].Name)
	require.Equal(t, "apps", titles[i].Source)
	i++
	require.Equal(t, "installer1", titles[i].Name)
	require.Equal(t, "apps", titles[i].Source)
	i++
	require.Equal(t, "foo", titles[i].Name)
	require.Equal(t, "chrome_extensions", titles[i].Source)
	i++
	require.Equal(t, "foo", titles[i].Name)
	require.Equal(t, "deb_packages", titles[i].Source)
	i++
	require.Equal(t, "foo", titles[i].Name)
	require.Equal(t, "rpm_packages", titles[i].Source)
	i++
	require.Equal(t, "baz", titles[i].Name)
	require.Equal(t, "chrome_extensions", titles[i].Source)
	require.Equal(t, "chrome", titles[i].Browser)
	i++
	require.Equal(t, "baz", titles[i].Name)
	require.Equal(t, "chrome_extensions", titles[i].Source)
	require.Equal(t, "edge", titles[i].Browser)
	i++
	require.Equal(t, "bar", titles[i].Name)
	require.Equal(t, "deb_packages", titles[i].Source)
	i++
	require.Equal(t, "bar", titles[i].Name)
	require.Equal(t, "apps", titles[i].Source)

	// using a match query
	titles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey:       "name",
			OrderDirection: fleet.OrderDescending,
			MatchQuery:     "ba",
		},
		TeamID: ptr.Uint(0),
	}, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.NoError(t, err)
	require.Len(t, titles, 4)
	require.Equal(t, "baz", titles[0].Name)
	require.Equal(t, "chrome_extensions", titles[0].Source)
	require.Equal(t, "chrome", titles[0].Browser)
	require.Equal(t, "baz", titles[1].Name)
	require.Equal(t, "chrome_extensions", titles[1].Source)
	require.Equal(t, "edge", titles[1].Browser)
	require.Equal(t, "bar", titles[2].Name)
	require.Equal(t, "deb_packages", titles[2].Source)
	require.Equal(t, "bar", titles[3].Name)
	require.Equal(t, "apps", titles[3].Source)

	// using another (installer-only) match query
	titles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey:       "name",
			OrderDirection: fleet.OrderDescending,
			MatchQuery:     "insta",
		},
		TeamID: ptr.Uint(0),
	}, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.NoError(t, err)
	require.Len(t, titles, 2)
	require.Equal(t, "installer2", titles[0].Name)
	require.Equal(t, "apps", titles[0].Source)
	require.Equal(t, "installer1", titles[1].Name)
	require.Equal(t, "apps", titles[1].Source)

	// filter on self-service only
	titles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey:       "name",
			OrderDirection: fleet.OrderDescending,
		},
		TeamID:          ptr.Uint(0),
		SelfServiceOnly: true,
	}, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.NoError(t, err)
	require.Len(t, titles, 1)
	require.Equal(t, "installer1", titles[0].Name)
	require.Equal(t, "apps", titles[0].Source)
}

func listSoftwareTitlesCheckCount(t *testing.T, ds *Datastore, expectedListCount int, expectedFullCount int, opts fleet.SoftwareTitleListOptions) []fleet.SoftwareTitleListResult {
	titles, count, _, err := ds.ListSoftwareTitles(context.Background(), opts, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.NoError(t, err)
	require.Len(t, titles, expectedListCount)
	require.NoError(t, err)
	require.Equal(t, expectedFullCount, count)
	return titles
}

func testTeamFilterSoftwareTitles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	test.CreateInsertGlobalVPPToken(t, ds)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	require.NoError(t, ds.AddHostsToTeam(ctx, &team1.ID, []uint{host1.ID}))
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())
	require.NoError(t, ds.AddHostsToTeam(ctx, &team2.ID, []uint{host2.ID}))

	userGlobalAdmin, err := ds.NewUser(ctx, &fleet.User{Name: "user1", Password: []byte("test"), Email: "test1@email.com", GlobalRole: ptr.String(fleet.RoleAdmin)})
	require.NoError(t, err)
	userTeam1Admin, err := ds.NewUser(ctx, &fleet.User{Name: "user2", Password: []byte("test"), Email: "test2@email.com", Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleAdmin}}})
	require.NoError(t, err)
	userTeam2Admin, err := ds.NewUser(ctx, &fleet.User{Name: "user3", Password: []byte("test"), Email: "test3@email.com", Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team2.ID}, Role: fleet.RoleAdmin}}})
	require.NoError(t, err)

	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	software2 := []fleet.Software{
		{Name: "foo", Version: "0.0.4", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}

	_, err = ds.UpdateHostSoftware(ctx, host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)

	// create a software installer for team1
	installer1, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            "installer1",
		Source:           "apps",
		InstallScript:    "echo",
		Filename:         "installer1.pkg",
		BundleIdentifier: "foo.bar",
		TeamID:           &team1.ID,
		UserID:           user1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, installer1)
	// make installer1 "self-service" available
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE software_installers SET self_service = 1 WHERE id = ?`, installer1)
		return err
	})
	// create a software installer for team2
	installer2, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:         "installer2",
		Source:        "apps",
		InstallScript: "echo",
		Filename:      "installer2.pkg",
		TeamID:        &team2.ID,
		UserID:        user1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, installer2)

	// create a VPP app for team2
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp2", BundleIdentifier: "com.app.vpp2",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_2", Platform: fleet.IOSPlatform}},
	}, &team2.ID)
	require.NoError(t, err)

	// create a VPP app for "No team", allowing self-service
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp3", BundleIdentifier: "com.app.vpp3",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_3", Platform: fleet.MacOSPlatform}, SelfService: true},
	}, ptr.Uint(0))
	require.NoError(t, err)

	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	// Testing the global user (for "All teams")
	// Should not return VPP apps or software installers (because they are not installed yet).
	globalTeamFilter := fleet.TeamFilter{User: userGlobalAdmin, IncludeObserver: true}
	titles, count, _, err := ds.ListSoftwareTitles(
		context.Background(), fleet.SoftwareTitleListOptions{
			ListOptions: fleet.ListOptions{},
			TeamID:      nil,
		}, globalTeamFilter,
	)
	sortTitlesByName(titles)
	require.NoError(t, err)
	require.Len(t, titles, 2)
	require.Equal(t, 2, count)

	require.Equal(t, "bar", titles[0].Name)
	require.Equal(t, "deb_packages", titles[0].Source)
	require.Equal(t, "foo", titles[1].Name)
	require.Equal(t, "chrome_extensions", titles[1].Source)
	require.Equal(t, uint(1), titles[0].VersionsCount)
	assert.Equal(t, uint(1), titles[0].HostsCount)
	require.Nil(t, titles[0].SoftwarePackage)
	require.Nil(t, titles[0].AppStoreApp)
	require.Equal(t, uint(2), titles[1].VersionsCount)
	assert.Equal(t, uint(2), titles[1].HostsCount)
	require.Nil(t, titles[1].SoftwarePackage)
	require.Nil(t, titles[1].AppStoreApp)
	barTitle := titles[0]
	fooTitle := titles[1]

	// Testing the global user (for "No team")
	// should only return vpp3 because it's the only app in the "No team".
	titles, count, _, err = ds.ListSoftwareTitles(
		context.Background(), fleet.SoftwareTitleListOptions{
			ListOptions: fleet.ListOptions{},
			TeamID:      ptr.Uint(0),
		}, globalTeamFilter,
	)
	sortTitlesByName(titles)
	require.NoError(t, err)
	require.Len(t, titles, 1)
	require.Equal(t, 1, count)
	require.Equal(t, uint(0), titles[0].VersionsCount)
	require.Nil(t, titles[0].SoftwarePackage)
	require.Equal(t, "vpp3", titles[0].Name)
	require.NotNil(t, titles[0].AppStoreApp)
	require.NotNil(t, titles[0].AppStoreApp.SelfService)
	require.True(t, *titles[0].AppStoreApp.SelfService)

	// Get title of bar software.
	title, err := ds.SoftwareTitleByID(context.Background(), barTitle.ID, nil, globalTeamFilter)
	require.NoError(t, err)
	require.Zero(t, title.SoftwareInstallersCount)
	require.Zero(t, title.VPPAppsCount)
	require.NotNil(t, title.CountsUpdatedAt)

	// ListSoftwareTitles does not populate version host counts, so we do that manually
	barTitle.Versions[0].HostsCount = ptr.Uint(1)
	assert.Equal(
		t,
		barTitle,
		fleet.SoftwareTitleListResult{
			ID:              title.ID,
			Name:            title.Name,
			Source:          title.Source,
			Browser:         title.Browser,
			HostsCount:      title.HostsCount,
			VersionsCount:   title.VersionsCount,
			Versions:        title.Versions,
			CountsUpdatedAt: title.CountsUpdatedAt,
		},
	)

	// Testing with team filter -- this team does not contain this software title
	_, err = ds.SoftwareTitleByID(context.Background(), barTitle.ID, &team1.ID, globalTeamFilter)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	// Testing with team filter -- this team does contain this software title
	title, err = ds.SoftwareTitleByID(context.Background(), fooTitle.ID, &team1.ID, globalTeamFilter)
	require.NoError(t, err)
	require.Zero(t, title.SoftwareInstallersCount)
	require.Zero(t, title.VPPAppsCount)
	assert.Equal(t, uint(1), title.HostsCount)
	assert.Equal(t, uint(1), title.VersionsCount)
	require.Len(t, title.Versions, 1)
	assert.Equal(t, ptr.Uint(1), title.Versions[0].HostsCount)
	assert.Equal(t, "0.0.3", title.Versions[0].Version)

	// Testing the team 1 user
	team1TeamFilter := fleet.TeamFilter{User: userTeam1Admin, IncludeObserver: true}
	titles, count, _, err = ds.ListSoftwareTitles(
		context.Background(), fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{}, TeamID: &team1.ID}, team1TeamFilter,
	)
	// installer1 is associated with team 1
	require.NoError(t, err)
	require.Len(t, titles, 2)
	require.Equal(t, 2, count)
	require.Equal(t, "foo", titles[0].Name)
	require.Equal(t, "chrome_extensions", titles[0].Source)
	require.Equal(t, "installer1", titles[1].Name)
	require.Equal(t, "apps", titles[1].Source)
	require.NotNil(t, titles[1].BundleIdentifier)
	require.Equal(t, "foo.bar", *titles[1].BundleIdentifier)
	require.Equal(t, uint(1), titles[0].VersionsCount)
	require.Nil(t, titles[0].SoftwarePackage)
	require.Nil(t, titles[0].AppStoreApp)
	require.Equal(t, uint(0), titles[1].VersionsCount)
	require.NotNil(t, titles[1].SoftwarePackage)
	require.Nil(t, titles[1].AppStoreApp)

	title, err = ds.SoftwareTitleByID(context.Background(), titles[1].ID, &team1.ID, team1TeamFilter)
	require.NoError(t, err)
	require.Equal(t, "installer1", title.Name)
	require.Equal(t, "apps", title.Source)
	require.NotNil(t, title.BundleIdentifier)
	require.Equal(t, "foo.bar", *title.BundleIdentifier)

	// Testing with team filter -- this team does contain this software title
	title, err = ds.SoftwareTitleByID(context.Background(), titles[0].ID, &team1.ID, team1TeamFilter)
	require.NoError(t, err)
	// ListSoftwareTitles does not populate version host counts, so we do that manually
	titles[0].Versions[0].HostsCount = ptr.Uint(1)
	assert.Equal(t, titles[0], fleet.SoftwareTitleListResult{ID: title.ID, Name: title.Name, Source: title.Source, Browser: title.Browser, HostsCount: title.HostsCount, VersionsCount: title.VersionsCount, Versions: title.Versions, CountsUpdatedAt: title.CountsUpdatedAt})

	// Testing the team 2 user
	titles, count, _, err = ds.ListSoftwareTitles(context.Background(), fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{}, TeamID: &team2.ID}, fleet.TeamFilter{
		User:            userTeam2Admin,
		IncludeObserver: true,
	})
	// installer2 and vpp2 is associated with team 2
	require.NoError(t, err)
	require.Len(t, titles, 4)
	require.Equal(t, 4, count)
	require.Equal(t, "bar", titles[0].Name)
	require.Equal(t, "deb_packages", titles[0].Source)
	require.Equal(t, "foo", titles[1].Name)
	require.Equal(t, "chrome_extensions", titles[1].Source)
	require.Equal(t, "installer2", titles[2].Name)
	require.Equal(t, "apps", titles[2].Source)
	require.Equal(t, "vpp2", titles[3].Name)
	assert.Equal(t, "ios_apps", titles[3].Source)
	require.Equal(t, uint(1), titles[0].VersionsCount)
	require.Equal(t, uint(1), titles[1].VersionsCount)
	require.Equal(t, uint(0), titles[2].VersionsCount)
	require.Equal(t, uint(0), titles[3].VersionsCount)
	require.Nil(t, titles[0].SoftwarePackage)
	require.Nil(t, titles[0].AppStoreApp)
	require.Nil(t, titles[1].SoftwarePackage)
	require.Nil(t, titles[1].AppStoreApp)
	require.NotNil(t, titles[2].SoftwarePackage)
	require.Nil(t, titles[2].AppStoreApp)
	require.Nil(t, titles[3].SoftwarePackage)
	require.NotNil(t, titles[3].AppStoreApp)

	// Testing the team 1 user with self-service only
	titles, _, _, err = ds.ListSoftwareTitles(
		context.Background(), fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{}, SelfServiceOnly: true, TeamID: &team1.ID}, team1TeamFilter,
	)
	// installer1 is associated with team 1
	require.NoError(t, err)
	require.Len(t, titles, 1)
	require.Equal(t, "installer1", titles[0].Name)
	require.Equal(t, "apps", titles[0].Source)

	title, err = ds.SoftwareTitleByID(context.Background(), titles[0].ID, &team1.ID, team1TeamFilter)
	require.NoError(t, err)
	require.Equal(t, 1, title.SoftwareInstallersCount)
	require.Zero(t, title.VPPAppsCount)

	// Testing the team 2 user with self-service only
	titles, _, _, err = ds.ListSoftwareTitles(context.Background(), fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{}, SelfServiceOnly: true, TeamID: &team2.ID}, fleet.TeamFilter{
		User:            userTeam2Admin,
		IncludeObserver: true,
	})
	require.NoError(t, err)
	require.Len(t, titles, 0)

	// Testing the no-team filter with self-service only
	titles, _, _, err = ds.ListSoftwareTitles(context.Background(), fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{}, SelfServiceOnly: true, TeamID: ptr.Uint(0)}, fleet.TeamFilter{
		User:            userGlobalAdmin,
		IncludeObserver: true,
	})
	require.NoError(t, err)
	require.Len(t, titles, 1)
	require.Equal(t, "vpp3", titles[0].Name)
}

func sortTitlesByName(titles []fleet.SoftwareTitleListResult) {
	sort.Slice(titles, func(i, j int) bool { return titles[i].Name < titles[j].Name })
}

func testListSoftwareTitlesInstallersOnly(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	test.CreateInsertGlobalVPPToken(t, ds)

	// create a couple software installers not installed on any host
	installer1, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:         "installer1",
		Source:        "apps",
		InstallScript: "echo",
		Filename:      "installer1.pkg",
		UserID:        user1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, installer1)
	installer2, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:         "installer2",
		Source:        "apps",
		InstallScript: "echo",
		Filename:      "installer2.pkg",
		UserID:        user1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, installer2)
	// create a VPP app not installed on a host
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app,vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_1", Platform: fleet.MacOSPlatform}},
	}, nil)
	require.NoError(t, err)

	titles, counts, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey:       "name",
			OrderDirection: fleet.OrderAscending,
		},
		TeamID: ptr.Uint(0),
	}, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.NoError(t, err)
	require.EqualValues(t, 3, counts)
	require.Len(t, titles, 3)
	require.Equal(t, "installer1", titles[0].Name)
	require.Equal(t, "apps", titles[0].Source)
	require.Equal(t, "installer2", titles[1].Name)
	require.Equal(t, "apps", titles[1].Source)
	require.Equal(t, "vpp1", titles[2].Name)
	require.Equal(t, "apps", titles[2].Source)
	require.True(t, titles[0].CountsUpdatedAt.IsZero())
	require.True(t, titles[1].CountsUpdatedAt.IsZero())
	require.True(t, titles[2].CountsUpdatedAt.IsZero())

	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	// match installer1 name
	titles, counts, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey:       "name",
			OrderDirection: fleet.OrderAscending,
			MatchQuery:     "installer1",
		},
		TeamID: ptr.Uint(0),
	}, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.NoError(t, err)
	require.EqualValues(t, 1, counts)
	require.Len(t, titles, 1)
	require.Equal(t, "installer1", titles[0].Name)
	require.Equal(t, "apps", titles[0].Source)
	require.True(t, titles[0].CountsUpdatedAt.IsZero())

	// vulnerable only returns nothing
	titles, counts, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey:       "name",
			OrderDirection: fleet.OrderAscending,
			MatchQuery:     "installer1",
		},
		TeamID:         ptr.Uint(0),
		VulnerableOnly: true,
	}, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.NoError(t, err)
	require.EqualValues(t, 0, counts)
	require.Len(t, titles, 0)

	// using the available_for_install filter
	titles, counts, _, err = ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "name",
				OrderDirection: fleet.OrderAscending,
			},
			AvailableForInstall: true,
			TeamID:              ptr.Uint(0),
		},
		fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}},
	)
	require.NoError(t, err)
	require.EqualValues(t, 3, counts)
	require.Len(t, titles, 3)
	require.True(t, titles[0].CountsUpdatedAt.IsZero())
}

func testListSoftwareTitlesAvailableForInstallFilter(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	test.CreateInsertGlobalVPPToken(t, ds)

	// create 2 software installers
	installer1, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:         "installer1",
		Source:        "apps",
		InstallScript: "echo",
		Filename:      "installer1.pkg",
		UserID:        user1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, installer1)
	installer2, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:         "installer2",
		Source:        "apps",
		InstallScript: "echo",
		Filename:      "installer2.pkg",
		UserID:        user1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, installer2)

	// create a 4 VPP apps
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.example.vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_1", Platform: fleet.MacOSPlatform}},
	}, nil)
	require.NoError(t, err)
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp2", BundleIdentifier: "com.example.vpp2",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_2", Platform: fleet.IPadOSPlatform}},
	}, nil)
	require.NoError(t, err)
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp2", BundleIdentifier: "com.example.vpp2",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_2", Platform: fleet.MacOSPlatform}},
	}, nil)
	require.NoError(t, err)
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp2", BundleIdentifier: "com.example.vpp2",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_2", Platform: fleet.IOSPlatform}},
	}, nil)
	require.NoError(t, err)

	host := test.NewHost(t, ds, "host", "", "hostkey", "hostuuid", time.Now())
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
		{Name: "vpp1", Version: "0.0.1", Source: "apps"},
		{Name: "installer1", Version: "0.0.1", Source: "apps"},
	}
	_, err = ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	// without filter returns all software
	titles, counts, _, err := ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "name",
				OrderDirection: fleet.OrderAscending,
			},
			TeamID: ptr.Uint(0),
		},
		fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}},
	)
	require.NoError(t, err)
	assert.EqualValues(t, 8, counts)
	assert.Len(t, titles, 8)
	type nameSource struct {
		name   string
		source string
	}
	names := make([]nameSource, 0, len(titles))
	for _, title := range titles {
		names = append(names, nameSource{name: title.Name, source: title.Source})
	}
	assert.ElementsMatch(t, []nameSource{
		{name: "bar", source: "deb_packages"},
		{name: "foo", source: "chrome_extensions"},
		{name: "installer1", source: "apps"},
		{name: "installer2", source: "apps"},
		{name: "vpp1", source: "apps"},
		{name: "vpp2", source: "ios_apps"},
		{name: "vpp2", source: "ipados_apps"},
		{name: "vpp2", source: "apps"},
	}, names)

	var vppVersionID uint
	var installer1ID uint
	var fooID uint
	for _, title := range titles {
		switch title.Name {
		case "vpp1":
			vppVersionID = title.Versions[0].ID
		case "installer1":
			installer1ID = title.Versions[0].ID
		case "foo":
			fooID = title.Versions[0].ID
		}
	}

	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: vppVersionID,
		CVE:        "CVE-2021-1234",
	}, fleet.NVDSource)
	require.NoError(t, err)

	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: installer1ID,
		CVE:        "CVE-2021-1234",
	}, fleet.NVDSource)
	require.NoError(t, err)

	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: fooID,
		CVE:        "CVE-2021-1234",
	}, fleet.NVDSource)
	require.NoError(t, err)

	titles, counts, _, err = ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "name",
				OrderDirection: fleet.OrderAscending,
			},
			TeamID:              ptr.Uint(0),
			AvailableForInstall: true,
			VulnerableOnly:      true,
		},
		fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}},
	)
	require.NoError(t, err)
	require.EqualValues(t, 2, counts)
	require.Len(t, titles, 2)
	names = make([]nameSource, 0, len(titles))
	for _, title := range titles {
		names = append(names, nameSource{name: title.Name, source: title.Source})
	}
	assert.ElementsMatch(t, []nameSource{
		{name: "installer1", source: "apps"},
		{name: "vpp1", source: "apps"},
	}, names)

	// with filter returns only available for install
	titles, counts, _, err = ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "name",
				OrderDirection: fleet.OrderAscending,
			},
			AvailableForInstall: true,
			TeamID:              ptr.Uint(0),
		},
		fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}},
	)
	require.NoError(t, err)
	require.EqualValues(t, 6, counts)
	require.Len(t, titles, 6)

	names = make([]nameSource, 0, len(titles))
	for _, title := range titles {
		names = append(names, nameSource{name: title.Name, source: title.Source})
	}
	assert.ElementsMatch(t, []nameSource{
		{name: "installer1", source: "apps"},
		{name: "installer2", source: "apps"},
		{name: "vpp1", source: "apps"},
		{name: "vpp2", source: "ios_apps"},
		{name: "vpp2", source: "ipados_apps"},
		{name: "vpp2", source: "apps"},
	}, names)
}

func testListSoftwareTitlesAllTeams(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	test.CreateInsertGlobalVPPToken(t, ds)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// Create a macOS software foobar installer on "No team".
	macOSInstallerNoTeam, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            "foobar",
		BundleIdentifier: "com.foo.bar",
		Source:           "apps",
		InstallScript:    "echo",
		Filename:         "foobar.pkg",
		TeamID:           nil,
		UserID:           user1.ID,
	})
	require.NoError(t, err)

	// Create an iOS Canva installer on "team1".
	require.NotZero(t, macOSInstallerNoTeam)
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "Canva", BundleIdentifier: "com.example.canva",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_canva", Platform: fleet.IOSPlatform}},
	}, &team1.ID)
	require.NoError(t, err)

	// Create a macOS Canva installer on "team1".
	require.NotZero(t, macOSInstallerNoTeam)
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "Canva", BundleIdentifier: "com.example.canva",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_canva", Platform: fleet.MacOSPlatform}},
	}, &team1.ID)
	require.NoError(t, err)

	// Create an iPadOS Canva installer on "team2".
	require.NotZero(t, macOSInstallerNoTeam)
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "Canva", BundleIdentifier: "com.example.canva",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_canva", Platform: fleet.IPadOSPlatform}},
	}, &team2.ID)
	require.NoError(t, err)

	// Add a macOS host on "No team" with some software.
	host := test.NewHost(t, ds, "host", "", "hostkey", "hostuuid", time.Now())
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
	}
	_, err = ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	// Simulate vulnerabilities cron
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	// List software titles for "All teams", should only return the host software titles
	// and no installers/VPP-apps because none is installed yet.
	titles, counts, _, err := ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "name",
				OrderDirection: fleet.OrderAscending,
			},
			TeamID: nil,
		},
		fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}},
	)
	require.NoError(t, err)
	assert.EqualValues(t, 2, counts)
	assert.Len(t, titles, 2)
	type nameSource struct {
		name   string
		source string
	}
	names := make([]nameSource, 0, len(titles))
	for _, title := range titles {
		names = append(names, nameSource{name: title.Name, source: title.Source})
	}
	assert.ElementsMatch(t, []nameSource{
		{name: "bar", source: "deb_packages"},
		{name: "foo", source: "chrome_extensions"},
	}, names)

	// List software for "No team". Should list the host's software + the macOS installer.
	titles, counts, _, err = ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "name",
				OrderDirection: fleet.OrderAscending,
			},
			TeamID: ptr.Uint(0),
		},
		fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}},
	)
	require.NoError(t, err)
	assert.EqualValues(t, 3, counts)
	assert.Len(t, titles, 3)
	names = make([]nameSource, 0, len(titles))
	for _, title := range titles {
		names = append(names, nameSource{name: title.Name, source: title.Source})
	}
	assert.ElementsMatch(t, []nameSource{
		{name: "bar", source: "deb_packages"},
		{name: "foo", source: "chrome_extensions"},
		{name: "foobar", source: "apps"},
	}, names)

	// List software for "team1". Should list Canva for iOS and macOS.
	titles, counts, _, err = ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "name",
				OrderDirection: fleet.OrderAscending,
			},
			TeamID: &team1.ID,
		},
		fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}},
	)
	require.NoError(t, err)
	assert.EqualValues(t, 2, counts)
	assert.Len(t, titles, 2)
	names = make([]nameSource, 0, len(titles))
	for _, title := range titles {
		names = append(names, nameSource{name: title.Name, source: title.Source})
	}
	assert.ElementsMatch(t, []nameSource{
		{name: "Canva", source: "ios_apps"},
		{name: "Canva", source: "apps"},
	}, names)

	// List software for "team2". Should list Canva for iPadOS.
	titles, counts, _, err = ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "name",
				OrderDirection: fleet.OrderAscending,
			},
			TeamID: &team2.ID,
		},
		fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}},
	)
	require.NoError(t, err)
	assert.EqualValues(t, 1, counts)
	assert.Len(t, titles, 1)
	names = make([]nameSource, 0, len(titles))
	for _, title := range titles {
		names = append(names, nameSource{name: title.Name, source: title.Source})
	}
	assert.ElementsMatch(t, []nameSource{
		{name: "Canva", source: "ipados_apps"},
	}, names)

	// List software available for install on "No team". Should list "foobar" package only.
	titles, counts, _, err = ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "name",
				OrderDirection: fleet.OrderAscending,
			},
			AvailableForInstall: true,
			TeamID:              ptr.Uint(0),
		},
		fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}},
	)
	require.NoError(t, err)
	require.EqualValues(t, 1, counts)
	require.Len(t, titles, 1)

	names = make([]nameSource, 0, len(titles))
	for _, title := range titles {
		names = append(names, nameSource{name: title.Name, source: title.Source})
	}
	assert.ElementsMatch(t, []nameSource{
		{name: "foobar", source: "apps"},
	}, names)
}

func testUploadedSoftwareExists(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team Foo"})
	require.NoError(t, err)
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	installer1, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            "installer1",
		Source:           "apps",
		InstallScript:    "echo",
		Filename:         "installer1.pkg",
		BundleIdentifier: "com.foo.installer1",
		UserID:           user1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, installer1)
	installer2, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            "installer2",
		Source:           "apps",
		InstallScript:    "echo",
		Filename:         "installer2.pkg",
		TeamID:           &tm.ID,
		BundleIdentifier: "com.foo.installer2",
		UserID:           user1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, installer2)

	exists, err := ds.UploadedSoftwareExists(ctx, "com.foo.installer1", nil)
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = ds.UploadedSoftwareExists(ctx, "com.foo.installer2", nil)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = ds.UploadedSoftwareExists(ctx, "com.foo.installer2", &tm.ID)
	require.NoError(t, err)
	require.True(t, exists)
}

func testListSoftwareTitlesVulnerabilityFilters(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := test.NewHost(t, ds, "host", "", "hostkey", "hostuuid", time.Now())

	software := []fleet.Software{
		{Name: "chrome", Version: "0.0.1", Source: "apps"},
		{Name: "chrome", Version: "0.0.3", Source: "apps"},
		{Name: "safari", Version: "0.0.3", Source: "apps"},
		{Name: "safari", Version: "0.0.1", Source: "apps"},
		{Name: "firefox", Version: "0.0.3", Source: "apps"},
		{Name: "edge", Version: "0.0.3", Source: "apps"},
		{Name: "brave", Version: "0.0.3", Source: "apps"},
		{Name: "opera", Version: "0.0.3", Source: "apps"},
		{Name: "internet explorer", Version: "0.0.3", Source: "apps"},
		{Name: "netscape", Version: "0.0.3", Source: "apps"},
	}

	sw, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	var chrome001 uint
	var safari001 uint
	var firefox003 uint
	var edge003 uint
	var brave003 uint
	var opera003 uint
	var ie003 uint
	for s := range sw.Inserted {
		switch {
		case sw.Inserted[s].Name == "chrome" && sw.Inserted[s].Version == "0.0.1":
			chrome001 = sw.Inserted[s].ID
		case sw.Inserted[s].Name == "safari" && sw.Inserted[s].Version == "0.0.1":
			safari001 = sw.Inserted[s].ID
		case sw.Inserted[s].Name == "firefox" && sw.Inserted[s].Version == "0.0.3":
			firefox003 = sw.Inserted[s].ID
		case sw.Inserted[s].Name == "edge" && sw.Inserted[s].Version == "0.0.3":
			edge003 = sw.Inserted[s].ID
		case sw.Inserted[s].Name == "brave" && sw.Inserted[s].Version == "0.0.3":
			brave003 = sw.Inserted[s].ID
		case sw.Inserted[s].Name == "opera" && sw.Inserted[s].Version == "0.0.3":
			opera003 = sw.Inserted[s].ID
		case sw.Inserted[s].Name == "internet explorer" && sw.Inserted[s].Version == "0.0.3":
			ie003 = sw.Inserted[s].ID
		}
	}

	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: chrome001,
		CVE:        "CVE-2024-1234",
	}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: safari001,
		CVE:        "CVE-2024-1235",
	}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: firefox003,
		CVE:        "CVE-2024-1236",
	}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: edge003,
		CVE:        "CVE-2024-1237",
	}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: brave003,
		CVE:        "CVE-2024-1238",
	}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: opera003,
		CVE:        "CVE-2024-1239",
	}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: ie003,
		CVE:        "CVE-2024-1240",
	}, fleet.NVDSource)
	require.NoError(t, err)

	err = ds.InsertCVEMeta(ctx, []fleet.CVEMeta{
		{
			// chrome
			CVE:              "CVE-2024-1234",
			CVSSScore:        ptr.Float64(7.5),
			CISAKnownExploit: ptr.Bool(true),
		},
		{
			// safari
			CVE:              "CVE-2024-1235",
			CVSSScore:        ptr.Float64(7.5),
			CISAKnownExploit: ptr.Bool(false),
		},
		{
			// firefox
			CVE:              "CVE-2024-1236",
			CVSSScore:        ptr.Float64(8.0),
			CISAKnownExploit: ptr.Bool(true),
		},
		{
			// edge
			CVE:              "CVE-2024-1237",
			CVSSScore:        ptr.Float64(8.0),
			CISAKnownExploit: ptr.Bool(false),
		},
		{
			// brave
			CVE:              "CVE-2024-1238",
			CVSSScore:        ptr.Float64(9.0),
			CISAKnownExploit: ptr.Bool(true),
		},
		// CVE-2024-1239 for opera has no CVE Meta
		{
			// internet explorer
			CVE:              "CVE-2024-1240",
			CVSSScore:        nil,
			CISAKnownExploit: nil,
		},
	})
	require.NoError(t, err)

	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	globalUser := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}

	tc := []struct {
		name           string
		opts           fleet.SoftwareTitleListOptions
		expectedTitles []string
		err            error
	}{
		{
			name: "vulnerable only",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions:    fleet.ListOptions{},
				VulnerableOnly: true,
			},
			expectedTitles: []string{"chrome", "safari", "firefox", "edge", "brave", "opera", "internet explorer"},
		},
		{
			name: "known exploit true",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions:    fleet.ListOptions{},
				VulnerableOnly: true,
				KnownExploit:   true,
			},
			expectedTitles: []string{"chrome", "firefox", "brave"},
		},
		{
			name: "minimum cvss 8.0",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions:    fleet.ListOptions{},
				VulnerableOnly: true,
				MinimumCVSS:    8.0,
			},
			expectedTitles: []string{"edge", "firefox", "brave"},
		},
		{
			name: "minimum cvss 7.9",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions:    fleet.ListOptions{},
				VulnerableOnly: true,
				MinimumCVSS:    7.9,
			},
			expectedTitles: []string{"edge", "firefox", "brave"},
		},
		{
			name: "minimum cvss 8.0 and known exploit",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions:    fleet.ListOptions{},
				VulnerableOnly: true,
				MinimumCVSS:    8.0,
				KnownExploit:   true,
			},
			expectedTitles: []string{"firefox", "brave"},
		},
		{
			name: "minimum cvss 7.5 and known exploit",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions:    fleet.ListOptions{},
				VulnerableOnly: true,
				MinimumCVSS:    7.5,
				KnownExploit:   true,
			},
			expectedTitles: []string{"chrome", "firefox", "brave"},
		},
		{
			name: "maximum cvss 7.5",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions:    fleet.ListOptions{},
				VulnerableOnly: true,
				MaximumCVSS:    7.5,
			},
			expectedTitles: []string{"chrome", "safari"},
		},
		{
			name: "maximum cvss 7.6",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions:    fleet.ListOptions{},
				VulnerableOnly: true,
				MaximumCVSS:    7.6,
			},
			expectedTitles: []string{"chrome", "safari"},
		},
		{
			name: "maximum cvss 7.5 and known exploit",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions:    fleet.ListOptions{},
				VulnerableOnly: true,
				MaximumCVSS:    7.5,
				KnownExploit:   true,
			},
			expectedTitles: []string{"chrome"},
		},
		{
			name: "minimum cvss 7.5 and maximum cvss 8.0",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions:    fleet.ListOptions{},
				VulnerableOnly: true,
				MinimumCVSS:    7.5,
				MaximumCVSS:    8.0,
			},
			expectedTitles: []string{"chrome", "safari", "firefox", "edge"},
		},
		{
			name: "minimum cvss 7.5 and maximum cvss 8.0 and known exploit",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions:    fleet.ListOptions{},
				VulnerableOnly: true,
				MinimumCVSS:    7.5,
				MaximumCVSS:    8.0,
				KnownExploit:   true,
			},
			expectedTitles: []string{"chrome", "firefox"},
		},
		{
			name: "err if vulnerableOnly is not set with MinimumCVSS",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions: fleet.ListOptions{},
				MinimumCVSS: 7.5,
			},
			err: fleet.NewInvalidArgumentError("query", "min_cvss_score, max_cvss_score, and exploit can only be provided with vulnerable=true"),
		},
		{
			name: "err if vulnerableOnly is not set with MaximumCVSS",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions: fleet.ListOptions{},
				MaximumCVSS: 7.5,
			},
			err: fleet.NewInvalidArgumentError("query", "min_cvss_score, max_cvss_score, and exploit can only be provided with vulnerable=true"),
		},
		{
			name: "err if vulnerableOnly is not set with KnownExploit",
			opts: fleet.SoftwareTitleListOptions{
				ListOptions:  fleet.ListOptions{},
				KnownExploit: true,
			},
			err: fleet.NewInvalidArgumentError("query", "min_cvss_score, max_cvss_score, and exploit can only be provided with vulnerable=true"),
		},
	}

	assertTitles := func(t *testing.T, titles []fleet.SoftwareTitleListResult, expectedTitles []string) {
		t.Helper()
		require.Len(t, titles, len(expectedTitles))
		for _, title := range titles {
			require.Contains(t, expectedTitles, title.Name)
		}
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			titles, _, _, err := ds.ListSoftwareTitles(ctx, tt.opts, fleet.TeamFilter{User: globalUser})
			if tt.err != nil {
				require.Error(t, err)
				require.Equal(t, tt.err, err)
				return
			}
			assertTitles(t, titles, tt.expectedTitles)
		})
	}
}
