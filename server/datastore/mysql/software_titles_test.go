package mysql

import (
	"context"
	"testing"
	"time"

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

	cmpNameVersionCount := func(want, got []fleet.SoftwareTitle) {
		cmp := make([]fleet.SoftwareTitle, len(got))
		for i, sw := range got {
			cmp[i] = fleet.SoftwareTitle{Name: sw.Name, HostsCount: sw.HostsCount}
		}
		require.ElementsMatch(t, want, cmp)
	}

	// this check ensures that the total number of rows in
	// software_title_host_counts matches the expected value.
	checkTableTotalCount := func(want int) {
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
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	globalOpts := fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending}}
	globalCounts := listSoftwareTitlesCheckCount(t, ds, 2, 2, globalOpts, false)

	want := []fleet.SoftwareTitle{
		{Name: "foo", HostsCount: 2},
		{Name: "bar", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)
	checkTableTotalCount(2)

	// update host2, remove "bar" software
	software2 = []fleet.Software{
		{Name: "foo", Version: "v0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	globalCounts = listSoftwareTitlesCheckCount(t, ds, 1, 1, globalOpts, false)
	want = []fleet.SoftwareTitle{
		{Name: "foo", HostsCount: 2},
	}
	cmpNameVersionCount(want, globalCounts)
	checkTableTotalCount(1)

	// create a software title entry without any host and any counts
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO software_titles (name, source) VALUES ('baz', 'testing')`)
	require.NoError(t, err)

	// listing does not return the new software title entry
	allSw := listSoftwareTitlesCheckCount(t, ds, 1, 1, fleet.SoftwareTitleListOptions{}, false)
	want = []fleet.SoftwareTitle{
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
	globalCounts = listSoftwareTitlesCheckCount(t, ds, 1, 1, globalOpts, false)
	want = []fleet.SoftwareTitle{
		{Name: "foo", HostsCount: 2},
	}
	cmpNameVersionCount(want, globalCounts)
	checkTableTotalCount(1)

	team1Opts := fleet.SoftwareTitleListOptions{
		TeamID:      ptr.Uint(team1.ID),
		ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending},
	}
	team1Counts := listSoftwareTitlesCheckCount(t, ds, 0, 0, team1Opts, false)
	want = []fleet.SoftwareTitle{}
	cmpNameVersionCount(want, team1Counts)
	checkTableTotalCount(1)

	// after a call to Calculate, the global counts are updated and the team counts appear
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	globalCounts = listSoftwareTitlesCheckCount(t, ds, 2, 2, globalOpts, false)
	want = []fleet.SoftwareTitle{
		{Name: "foo", HostsCount: 4},
		{Name: "bar", HostsCount: 1},
	}
	cmpNameVersionCount(want, globalCounts)

	team1Counts = listSoftwareTitlesCheckCount(t, ds, 1, 1, team1Opts, false)
	want = []fleet.SoftwareTitle{
		{Name: "foo", HostsCount: 2},
	}
	cmpNameVersionCount(want, team1Counts)

	// composite pk (software_title_id, team_id), so we expect more rows
	checkTableTotalCount(5)

	team2Opts := fleet.SoftwareTitleListOptions{
		TeamID:      ptr.Uint(team2.ID),
		ListOptions: fleet.ListOptions{OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending},
	}
	team2Counts := listSoftwareTitlesCheckCount(t, ds, 2, 2, team2Opts, false)
	want = []fleet.SoftwareTitle{
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
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	globalCounts = listSoftwareTitlesCheckCount(t, ds, 1, 1, globalOpts, false)
	want = []fleet.SoftwareTitle{
		{Name: "foo", HostsCount: 4},
	}
	cmpNameVersionCount(want, globalCounts)

	team1Counts = listSoftwareTitlesCheckCount(t, ds, 1, 1, team1Opts, false)
	want = []fleet.SoftwareTitle{
		{Name: "foo", HostsCount: 2},
	}
	cmpNameVersionCount(want, team1Counts)

	team2Counts = listSoftwareTitlesCheckCount(t, ds, 1, 1, team2Opts, false)
	want = []fleet.SoftwareTitle{
		{Name: "foo", HostsCount: 1},
	}
	cmpNameVersionCount(want, team2Counts)

	checkTableTotalCount(3)

	// update host4 (team2), remove all software and delete team
	software4 = []fleet.Software{}
	_, err = ds.UpdateHostSoftware(ctx, host4.ID, software4)
	require.NoError(t, err)
	require.NoError(t, ds.DeleteTeam(ctx, team2.ID))

	// this call will remove team2 from the software host counts table
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	globalCounts = listSoftwareTitlesCheckCount(t, ds, 1, 1, globalOpts, false)
	want = []fleet.SoftwareTitle{
		{Name: "foo", HostsCount: 3},
	}
	cmpNameVersionCount(want, globalCounts)

	team1Counts = listSoftwareTitlesCheckCount(t, ds, 1, 1, team1Opts, false)
	want = []fleet.SoftwareTitle{
		{Name: "foo", HostsCount: 2},
	}
	cmpNameVersionCount(want, team1Counts)

	listSoftwareTitlesCheckCount(t, ds, 0, 0, team2Opts, false)
	checkTableTotalCount(2)
}

func testOrderSoftwareTitles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now())

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
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	// primary sort is "hosts_count DESC", followed by "name ASC, source ASC, browser ASC"
	titles, _, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{
		OrderKey:       "hosts_count",
		OrderDirection: fleet.OrderDescending,
	}})
	require.NoError(t, err)
	require.Len(t, titles, 7)
	require.Equal(t, "bar", titles[0].Name)
	require.Equal(t, "deb_packages", titles[0].Source)
	require.Equal(t, "foo", titles[1].Name)
	require.Equal(t, "chrome_extensions", titles[1].Source)
	require.Equal(t, "foo", titles[2].Name)
	require.Equal(t, "deb_packages", titles[2].Source)
	require.Equal(t, "bar", titles[3].Name)
	require.Equal(t, "apps", titles[3].Source)
	require.Equal(t, "baz", titles[4].Name)
	require.Equal(t, "chrome_extensions", titles[4].Source)
	require.Equal(t, "chrome", titles[4].Browser)
	require.Equal(t, "baz", titles[5].Name)
	require.Equal(t, "chrome_extensions", titles[5].Source)
	require.Equal(t, "edge", titles[5].Browser)
	require.Equal(t, "foo", titles[6].Name)
	require.Equal(t, "rpm_packages", titles[6].Source)

	// primary sort is "hosts_count ASC", followed by "name ASC, source ASC, browser ASC"
	titles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{
		OrderKey:       "hosts_count",
		OrderDirection: fleet.OrderAscending,
	}})
	require.NoError(t, err)
	require.Len(t, titles, 7)
	require.Equal(t, "bar", titles[0].Name)
	require.Equal(t, "apps", titles[0].Source)
	require.Equal(t, "baz", titles[1].Name)
	require.Equal(t, "chrome_extensions", titles[1].Source)
	require.Equal(t, "chrome", titles[1].Browser)
	require.Equal(t, "baz", titles[2].Name)
	require.Equal(t, "chrome_extensions", titles[2].Source)
	require.Equal(t, "edge", titles[2].Browser)
	require.Equal(t, "foo", titles[3].Name)
	require.Equal(t, "rpm_packages", titles[3].Source)
	require.Equal(t, "bar", titles[4].Name)
	require.Equal(t, "deb_packages", titles[4].Source)
	require.Equal(t, "foo", titles[5].Name)
	require.Equal(t, "chrome_extensions", titles[5].Source)
	require.Equal(t, "foo", titles[6].Name)
	require.Equal(t, "deb_packages", titles[6].Source)

	// primary sort is "name ASC", followed by "host_count DESC, source ASC, browser ASC"
	titles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{
		OrderKey:       "name",
		OrderDirection: fleet.OrderAscending,
	}})
	require.NoError(t, err)
	require.Len(t, titles, 7)
	require.Equal(t, "bar", titles[0].Name)
	require.Equal(t, "deb_packages", titles[0].Source)
	require.Equal(t, "bar", titles[1].Name)
	require.Equal(t, "apps", titles[1].Source)
	require.Equal(t, "baz", titles[2].Name)
	require.Equal(t, "chrome_extensions", titles[2].Source)
	require.Equal(t, "chrome", titles[2].Browser)
	require.Equal(t, "baz", titles[3].Name)
	require.Equal(t, "chrome_extensions", titles[3].Source)
	require.Equal(t, "edge", titles[3].Browser)
	require.Equal(t, "foo", titles[4].Name)
	require.Equal(t, "chrome_extensions", titles[4].Source)
	require.Equal(t, "foo", titles[5].Name)
	require.Equal(t, "deb_packages", titles[5].Source)
	require.Equal(t, "foo", titles[6].Name)
	require.Equal(t, "rpm_packages", titles[6].Source)

	// primary sort is "name DESC", followed by "host_count DESC, source ASC, browser ASC"
	titles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{
		OrderKey:       "name",
		OrderDirection: fleet.OrderDescending,
	}})
	require.NoError(t, err)
	require.Len(t, titles, 7)
	require.Equal(t, "foo", titles[0].Name)
	require.Equal(t, "chrome_extensions", titles[0].Source)
	require.Equal(t, "foo", titles[1].Name)
	require.Equal(t, "deb_packages", titles[1].Source)
	require.Equal(t, "foo", titles[2].Name)
	require.Equal(t, "rpm_packages", titles[2].Source)
	require.Equal(t, "baz", titles[3].Name)
	require.Equal(t, "chrome_extensions", titles[3].Source)
	require.Equal(t, "chrome", titles[3].Browser)
	require.Equal(t, "baz", titles[4].Name)
	require.Equal(t, "chrome_extensions", titles[4].Source)
	require.Equal(t, "edge", titles[4].Browser)
	require.Equal(t, "bar", titles[5].Name)
	require.Equal(t, "deb_packages", titles[5].Source)
	require.Equal(t, "bar", titles[6].Name)
	require.Equal(t, "apps", titles[6].Source)
}

func listSoftwareTitlesCheckCount(t *testing.T, ds *Datastore, expectedListCount int, expectedFullCount int, opts fleet.SoftwareTitleListOptions, returnSorted bool) []fleet.SoftwareTitle {
	titles, count, _, err := ds.ListSoftwareTitles(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, titles, expectedListCount)
	require.NoError(t, err)
	require.Equal(t, expectedFullCount, count)
	return titles
}
