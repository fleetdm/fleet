package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var enrollTests = []struct {
	uuid, hostname, platform, nodeKey string
}{
	0: {uuid: "6D14C88F-8ECF-48D5-9197-777647BF6B26",
		hostname: "web.fleet.co",
		platform: "linux",
		nodeKey:  "key0",
	},
	1: {uuid: "B998C0EB-38CE-43B1-A743-FBD7A5C9513B",
		hostname: "mail.fleet.co",
		platform: "linux",
		nodeKey:  "key1",
	},
	2: {uuid: "008F0688-5311-4C59-86EE-00C2D6FC3EC2",
		hostname: "home.fleet.co",
		platform: "darwin",
		nodeKey:  "key2",
	},
	3: {uuid: "uuid123",
		hostname: "fakehostname",
		platform: "darwin",
		nodeKey:  "key3",
	},
}

func TestHosts(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Save", testHostsSave},
		{"DeleteWithSoftware", testHostsDeleteWithSoftware},
		{"SavePackStats", testHostsSavePackStats},
		{"SavePackStatsOverwrites", testHostsSavePackStatsOverwrites},
		{"IgnoresTeamPackStats", testHostsIgnoresTeamPackStats},
		{"Delete", testHostsDelete},
		{"ListFilterAdditional", testHostsListFilterAdditional},
		{"ListStatus", testHostsListStatus},
		{"ListQuery", testHostsListQuery},
		{"Enroll", testHostsEnroll},
		{"Authenticate", testHostsAuthenticate},
		{"AuthenticateCaseSensitive", testHostsAuthenticateCaseSensitive},
		{"Search", testHostsSearch},
		{"SearchLimit", testHostsSearchLimit},
		{"GenerateStatusStatistics", testHostsGenerateStatusStatistics},
		{"MarkSeen", testHostsMarkSeen},
		{"MarkSeenMany", testHostsMarkSeenMany},
		{"CleanupIncoming", testHostsCleanupIncoming},
		{"IDsByName", testHostsIDsByName},
		{"Additional", testHostsAdditional},
		{"ByIdentifier", testHostsByIdentifier},
		{"AddToTeam", testHostsAddToTeam},
		{"SaveUsers", testHostsSaveUsers},
		{"SaveUsersWithoutUid", testHostsSaveUsersWithoutUid},
		{"TotalAndUnseenSince", testHostsTotalAndUnseenSince},
		{"ListByPolicy", testHostsListByPolicy},
		{"SaveTonsOfUsers", testHostsSaveTonsOfUsers},
		{"SavePackStatsConcurrent", testHostsSavePackStatsConcurrent},
		{"AuthenticateHostLoadsDisk", testAuthenticateHostLoadsDisk},
		{"HostsListBySoftware", testHostsListBySoftware},
		{"HostsListFailingPolicies", testHostsListFailingPolicies},
		{"HostsExpiration", testHostsExpiration},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testHostsSave(t *testing.T, ds *Datastore) {
	policyUpdatedAt := time.Now().UTC().Truncate(time.Second)
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: policyUpdatedAt,
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	host.Hostname = "bar.local"
	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	assert.Equal(t, "bar.local", host.Hostname)
	assert.Equal(t, "192.168.1.1", host.PrimaryIP)
	assert.Equal(t, "30-65-EC-6F-C4-58", host.PrimaryMac)
	assert.Equal(t, policyUpdatedAt.UTC(), host.PolicyUpdatedAt)

	additionalJSON := json.RawMessage(`{"foobar": "bim"}`)
	host.Additional = &additionalJSON

	require.NoError(t, ds.SaveHost(context.Background(), host))
	require.NoError(t, saveHostAdditionalDB(context.Background(), ds.writer, host))

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.NotNil(t, host)
	require.NotNil(t, host.Additional)
	assert.Equal(t, additionalJSON, *host.Additional)

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.NotNil(t, host)

	err = ds.DeleteHost(context.Background(), host.ID)
	assert.Nil(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	assert.NotNil(t, err)
	assert.Nil(t, host)
}

func testHostsDeleteWithSoftware(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	soft := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		},
	}
	host.HostSoftware = soft
	err = ds.SaveHostSoftware(context.Background(), host)
	require.NoError(t, err)

	err = ds.DeleteHost(context.Background(), host.ID)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	assert.NotNil(t, err)
	assert.Nil(t, host)
}

func testHostsSavePackStats(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	// Pack and query must exist for stats to save successfully
	pack1 := test.NewPack(t, ds, "test1")
	query1 := test.NewQuery(t, ds, "time", "select * from time", 0, true)
	squery1 := test.NewScheduledQuery(t, ds, pack1.ID, query1.ID, 30, true, true, "time-scheduled")
	stats1 := []fleet.ScheduledQueryStats{
		{
			ScheduledQueryName: squery1.Name,
			ScheduledQueryID:   squery1.ID,
			QueryName:          query1.Name,
			PackName:           pack1.Name,
			PackID:             pack1.ID,
			AverageMemory:      8000,
			Denylisted:         false,
			Executions:         164,
			Interval:           30,
			LastExecuted:       time.Unix(1620325191, 0).UTC(),
			OutputSize:         1337,
			SystemTime:         150,
			UserTime:           180,
			WallTime:           0,
		},
	}

	pack2 := test.NewPack(t, ds, "test2")
	squery2 := test.NewScheduledQuery(t, ds, pack2.ID, query1.ID, 30, true, true, "time-scheduled")
	query2 := test.NewQuery(t, ds, "processes", "select * from processes", 0, true)
	squery3 := test.NewScheduledQuery(t, ds, pack2.ID, query2.ID, 30, true, true, "processes")
	stats2 := []fleet.ScheduledQueryStats{
		{
			ScheduledQueryName: squery2.Name,
			ScheduledQueryID:   squery2.ID,
			QueryName:          query1.Name,
			PackName:           pack2.Name,
			PackID:             pack2.ID,
			AverageMemory:      431,
			Denylisted:         true,
			Executions:         1,
			Interval:           30,
			LastExecuted:       time.Unix(980943843, 0).UTC(),
			OutputSize:         134,
			SystemTime:         1656,
			UserTime:           18453,
			WallTime:           10,
		},
		{
			ScheduledQueryName: squery3.Name,
			ScheduledQueryID:   squery3.ID,
			QueryName:          query2.Name,
			PackName:           pack2.Name,
			PackID:             pack2.ID,
			AverageMemory:      8000,
			Denylisted:         false,
			Executions:         164,
			Interval:           30,
			LastExecuted:       time.Unix(1620325191, 0).UTC(),
			OutputSize:         1337,
			SystemTime:         150,
			UserTime:           180,
			WallTime:           0,
		},
	}

	host.PackStats = []fleet.PackStats{
		{
			PackName: "test1",
			// Append an additional entry to be sure that receiving stats for a
			// now-deleted query doesn't break saving. This extra entry should
			// not be returned on loading the host.
			QueryStats: append(stats1, fleet.ScheduledQueryStats{PackName: "foo", ScheduledQueryName: "bar"}),
		},
		{
			PackName:   "test2",
			QueryStats: stats2,
		},
	}

	require.NoError(t, ds.SaveHost(context.Background(), host))

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)

	require.Len(t, host.PackStats, 2)
	sort.Slice(host.PackStats, func(i, j int) bool {
		return host.PackStats[i].PackName < host.PackStats[j].PackName
	})
	assert.Equal(t, host.PackStats[0].PackName, "test1")
	assert.ElementsMatch(t, host.PackStats[0].QueryStats, stats1)
	assert.Equal(t, host.PackStats[1].PackName, "test2")
	assert.ElementsMatch(t, host.PackStats[1].QueryStats, stats2)

	// Set to nil should not overwrite
	host.PackStats = nil
	require.NoError(t, ds.SaveHost(context.Background(), host))
	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, host.PackStats, 2)
}

func testHostsSavePackStatsOverwrites(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	// Pack and query must exist for stats to save successfully
	pack1 := test.NewPack(t, ds, "test1")
	query1 := test.NewQuery(t, ds, "time", "select * from time", 0, true)
	squery1 := test.NewScheduledQuery(t, ds, pack1.ID, query1.ID, 30, true, true, "time-scheduled")
	pack2 := test.NewPack(t, ds, "test2")
	squery2 := test.NewScheduledQuery(t, ds, pack2.ID, query1.ID, 30, true, true, "time-scheduled")
	query2 := test.NewQuery(t, ds, "processes", "select * from processes", 0, true)

	execTime1 := time.Unix(1620325191, 0).UTC()

	host.PackStats = []fleet.PackStats{
		{
			PackName: "test1",
			QueryStats: []fleet.ScheduledQueryStats{
				{
					ScheduledQueryName: squery1.Name,
					ScheduledQueryID:   squery1.ID,
					QueryName:          query1.Name,
					PackName:           pack1.Name,
					PackID:             pack1.ID,
					AverageMemory:      8000,
					Denylisted:         false,
					Executions:         164,
					Interval:           30,
					LastExecuted:       execTime1,
					OutputSize:         1337,
					SystemTime:         150,
					UserTime:           180,
					WallTime:           0,
				},
			},
		},
		{
			PackName: "test2",
			QueryStats: []fleet.ScheduledQueryStats{
				{
					ScheduledQueryName: squery2.Name,
					ScheduledQueryID:   squery2.ID,
					QueryName:          query2.Name,
					PackName:           pack2.Name,
					PackID:             pack2.ID,
					AverageMemory:      431,
					Denylisted:         true,
					Executions:         1,
					Interval:           30,
					LastExecuted:       execTime1,
					OutputSize:         134,
					SystemTime:         1656,
					UserTime:           18453,
					WallTime:           10,
				},
			},
		},
	}

	require.NoError(t, ds.SaveHost(context.Background(), host))

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)

	sort.Slice(host.PackStats, func(i, j int) bool {
		return host.PackStats[i].PackName < host.PackStats[j].PackName
	})

	require.Len(t, host.PackStats, 2)
	assert.Equal(t, host.PackStats[0].PackName, "test1")
	assert.Equal(t, execTime1, host.PackStats[0].QueryStats[0].LastExecuted)

	execTime2 := execTime1.Add(24 * time.Hour)

	host.PackStats = []fleet.PackStats{
		{
			PackName: "test1",
			QueryStats: []fleet.ScheduledQueryStats{
				{
					ScheduledQueryName: squery1.Name,
					ScheduledQueryID:   squery1.ID,
					QueryName:          query1.Name,
					PackName:           pack1.Name,
					PackID:             pack1.ID,
					AverageMemory:      8000,
					Denylisted:         false,
					Executions:         164,
					Interval:           30,
					LastExecuted:       execTime2,
					OutputSize:         1337,
					SystemTime:         150,
					UserTime:           180,
					WallTime:           0,
				},
			},
		},
		{
			PackName: "test2",
			QueryStats: []fleet.ScheduledQueryStats{
				{
					ScheduledQueryName: squery2.Name,
					ScheduledQueryID:   squery2.ID,
					QueryName:          query2.Name,
					PackName:           pack2.Name,
					PackID:             pack2.ID,
					AverageMemory:      431,
					Denylisted:         true,
					Executions:         1,
					Interval:           30,
					LastExecuted:       execTime1,
					OutputSize:         134,
					SystemTime:         1656,
					UserTime:           18453,
					WallTime:           10,
				},
			},
		},
	}

	require.NoError(t, ds.SaveHost(context.Background(), host))

	gotHost, err := ds.Host(context.Background(), host.ID)
	require.NoError(t, err)

	sort.Slice(gotHost.PackStats, func(i, j int) bool {
		return gotHost.PackStats[i].PackName < gotHost.PackStats[j].PackName
	})

	require.Len(t, gotHost.PackStats, 2)
	assert.Equal(t, gotHost.PackStats[0].PackName, "test1")
	assert.Equal(t, execTime2, gotHost.PackStats[0].QueryStats[0].LastExecuted)
}

func testHostsIgnoresTeamPackStats(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	team, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID}))
	tp, err := ds.EnsureTeamPack(context.Background(), team.ID)
	require.NoError(t, err)

	tpQuery := test.NewQuery(t, ds, "tp-time", "select * from time", 0, true)
	tpSquery := test.NewScheduledQuery(t, ds, tp.ID, tpQuery.ID, 30, true, true, "time-scheduled")

	// Pack and query must exist for stats to save successfully
	pack1 := test.NewPack(t, ds, "test1")
	query1 := test.NewQuery(t, ds, "time", "select * from time", 0, true)
	squery1 := test.NewScheduledQuery(t, ds, pack1.ID, query1.ID, 30, true, true, "time-scheduled")
	stats1 := []fleet.ScheduledQueryStats{
		{
			ScheduledQueryName: squery1.Name,
			ScheduledQueryID:   squery1.ID,
			QueryName:          query1.Name,
			PackName:           pack1.Name,
			PackID:             pack1.ID,
			AverageMemory:      8000,
			Denylisted:         false,
			Executions:         164,
			Interval:           30,
			LastExecuted:       time.Unix(1620325191, 0).UTC(),
			OutputSize:         1337,
			SystemTime:         150,
			UserTime:           180,
			WallTime:           0,
		},
	}
	stats2 := []fleet.ScheduledQueryStats{
		{
			ScheduledQueryName: tpSquery.Name,
			ScheduledQueryID:   tpSquery.ID,
			QueryName:          tpQuery.Name,
			PackName:           tp.Name,
			PackID:             tp.ID,
			AverageMemory:      8000,
			Denylisted:         false,
			Executions:         164,
			Interval:           30,
			LastExecuted:       time.Unix(1620325191, 0).UTC(),
			OutputSize:         1337,
			SystemTime:         150,
			UserTime:           180,
			WallTime:           0,
		},
	}

	host.PackStats = []fleet.PackStats{
		{PackName: "test1", QueryStats: stats1},
		{PackName: teamScheduleName(team), QueryStats: stats2},
	}

	require.NoError(t, ds.SaveHost(context.Background(), host))

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)

	require.Len(t, host.PackStats, 1)
	assert.Equal(t, host.PackStats[0].PackName, "test1")
	assert.ElementsMatch(t, host.PackStats[0].QueryStats, stats1)
}

func testHostsDelete(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	err = ds.DeleteHost(context.Background(), host.ID)
	assert.Nil(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	assert.NotNil(t, err)
}

func listHostsCheckCount(t *testing.T, ds *Datastore, filter fleet.TeamFilter, opt fleet.HostListOptions, expectedCount int) []*fleet.Host {
	hosts, err := ds.ListHosts(context.Background(), filter, opt)
	require.NoError(t, err)
	count, err := ds.CountHosts(context.Background(), filter, opt)
	require.NoError(t, err)
	require.Equal(t, expectedCount, count)
	return hosts
}

func testHostsListFilterAdditional(t *testing.T, ds *Datastore) {
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   "foobar",
		NodeKey:         "nodekey",
		UUID:            "uuid",
		Hostname:        "foobar.local",
	})
	require.NoError(t, err)

	filter := fleet.TeamFilter{User: test.UserAdmin}

	// Add additional
	additional := json.RawMessage(`{"field1": "v1", "field2": "v2"}`)
	h.Additional = &additional
	require.NoError(t, saveHostAdditionalDB(context.Background(), ds.writer, h))

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 1)
	assert.Nil(t, hosts[0].Additional)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{AdditionalFilters: []string{"field1", "field2"}}, 1)
	require.Nil(t, err)
	assert.Equal(t, &additional, hosts[0].Additional)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{AdditionalFilters: []string{"*"}}, 1)
	require.Nil(t, err)
	assert.Equal(t, &additional, hosts[0].Additional)

	additional = json.RawMessage(`{"field1": "v1", "missing": null}`)
	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{AdditionalFilters: []string{"field1", "missing"}}, 1)
	assert.Equal(t, &additional, hosts[0].Additional)
}

func testHostsListStatus(t *testing.T, ds *Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   strconv.Itoa(i),
			NodeKey:         fmt.Sprintf("%d", i),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		assert.Nil(t, err)
		if err != nil {
			return
		}
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{StatusFilter: "online"}, 1)
	assert.Equal(t, 1, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{StatusFilter: "offline"}, 9)
	assert.Equal(t, 9, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{StatusFilter: "mia"}, 0)
	assert.Equal(t, 0, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{StatusFilter: "new"}, 10)
	assert.Equal(t, 10, len(hosts))
}

func testHostsListQuery(t *testing.T, ds *Datastore) {
	hosts := []*fleet.Host{}
	for i := 0; i < 10; i++ {
		host, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   strconv.Itoa(i),
			NodeKey:         fmt.Sprintf("%d", i),
			UUID:            fmt.Sprintf("uuid_00%d", i),
			Hostname:        fmt.Sprintf("hostname%%00%d", i),
			HardwareSerial:  fmt.Sprintf("serial00%d", i),
		})
		require.NoError(t, err)
		host.PrimaryIP = fmt.Sprintf("192.168.1.%d", i)
		require.NoError(t, ds.SaveHost(context.Background(), host))
		hosts = append(hosts, host)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	for _, host := range hosts {
		require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host.ID}))
	}

	gotHosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, len(hosts))
	assert.Equal(t, len(hosts), len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{TeamFilter: &team1.ID}, len(hosts))
	assert.Equal(t, len(hosts), len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{TeamFilter: &team2.ID}, 0)
	assert.Equal(t, 0, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{TeamFilter: nil}, len(hosts))
	assert.Equal(t, len(hosts), len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{ListOptions: fleet.ListOptions{MatchQuery: "00"}}, 10)
	assert.Equal(t, 10, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{ListOptions: fleet.ListOptions{MatchQuery: "000"}}, 1)
	assert.Equal(t, 1, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{ListOptions: fleet.ListOptions{MatchQuery: "192.168."}}, 10)
	assert.Equal(t, 10, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{ListOptions: fleet.ListOptions{MatchQuery: "192.168.1.1"}}, 1)
	assert.Equal(t, 1, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{ListOptions: fleet.ListOptions{MatchQuery: "hostname%00"}}, 10)
	assert.Equal(t, 10, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{ListOptions: fleet.ListOptions{MatchQuery: "hostname%003"}}, 1)
	assert.Equal(t, 1, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{ListOptions: fleet.ListOptions{MatchQuery: "uuid_"}}, 10)
	assert.Equal(t, 10, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{ListOptions: fleet.ListOptions{MatchQuery: "uuid_006"}}, 1)
	assert.Equal(t, 1, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{ListOptions: fleet.ListOptions{MatchQuery: "serial"}}, 10)
	assert.Equal(t, 10, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{ListOptions: fleet.ListOptions{MatchQuery: "serial009"}}, 1)
	assert.Equal(t, 1, len(gotHosts))
}

func testHostsEnroll(t *testing.T, ds *Datastore) {
	test.AddAllHostsLabel(t, ds)

	team, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	filter := fleet.TeamFilter{User: test.UserAdmin}
	hosts, err := ds.ListHosts(context.Background(), filter, fleet.HostListOptions{})
	require.NoError(t, err)
	for _, host := range hosts {
		assert.Zero(t, host.LastEnrolledAt)
	}

	for _, tt := range enrollTests {
		h, err := ds.EnrollHost(context.Background(), tt.uuid, tt.nodeKey, &team.ID, 0)
		require.NoError(t, err)

		assert.Equal(t, tt.uuid, h.OsqueryHostID)
		assert.Equal(t, tt.nodeKey, h.NodeKey)

		// This host should be allowed to re-enroll immediately if cooldown is disabled
		_, err = ds.EnrollHost(context.Background(), tt.uuid, tt.nodeKey+"new", nil, 0)
		require.NoError(t, err)

		// This host should not be allowed to re-enroll immediately if cooldown is enabled
		_, err = ds.EnrollHost(context.Background(), tt.uuid, tt.nodeKey+"new", nil, 10*time.Second)
		require.Error(t, err)
	}

	hosts, err = ds.ListHosts(context.Background(), filter, fleet.HostListOptions{})

	require.NoError(t, err)
	for _, host := range hosts {
		assert.NotZero(t, host.LastEnrolledAt)
	}
}

func testHostsAuthenticate(t *testing.T, ds *Datastore) {
	test.AddAllHostsLabel(t, ds)
	for _, tt := range enrollTests {
		h, err := ds.EnrollHost(context.Background(), tt.uuid, tt.nodeKey, nil, 0)
		require.NoError(t, err)

		returned, err := ds.AuthenticateHost(context.Background(), h.NodeKey)
		require.NoError(t, err)
		assert.Equal(t, h.NodeKey, returned.NodeKey)
	}

	_, err := ds.AuthenticateHost(context.Background(), "7B1A9DC9-B042-489F-8D5A-EEC2412C95AA")
	assert.Error(t, err)

	_, err = ds.AuthenticateHost(context.Background(), "")
	assert.Error(t, err)
}

func testHostsAuthenticateCaseSensitive(t *testing.T, ds *Datastore) {
	test.AddAllHostsLabel(t, ds)
	for _, tt := range enrollTests {
		h, err := ds.EnrollHost(context.Background(), tt.uuid, tt.nodeKey, nil, 0)
		require.NoError(t, err)

		_, err = ds.AuthenticateHost(context.Background(), strings.ToUpper(h.NodeKey))
		require.Error(t, err, "node key authentication should be case sensitive")
	}
}

func testHostsSearch(t *testing.T, ds *Datastore) {
	_, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   "1234",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)

	h2, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   "5679",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "bar.local",
	})
	require.NoError(t, err)

	h3, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   "99999",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "3",
		UUID:            "abc-def-ghi",
		Hostname:        "foo-bar.local",
	})
	require.NoError(t, err)

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	// We once threw errors when the search query was empty. Verify that we
	// don't error.
	_, err = ds.SearchHosts(context.Background(), filter, "")
	require.NoError(t, err)

	hosts, err := ds.SearchHosts(context.Background(), filter, "foo")
	assert.Nil(t, err)
	assert.Len(t, hosts, 2)

	host, err := ds.SearchHosts(context.Background(), filter, "foo", h3.ID)
	require.NoError(t, err)
	require.Len(t, host, 1)
	assert.Equal(t, "foo.local", host[0].Hostname)

	host, err = ds.SearchHosts(context.Background(), filter, "foo", h3.ID, h2.ID)
	require.NoError(t, err)
	require.Len(t, host, 1)
	assert.Equal(t, "foo.local", host[0].Hostname)

	host, err = ds.SearchHosts(context.Background(), filter, "abc")
	require.NoError(t, err)
	require.Len(t, host, 1)
	assert.Equal(t, "abc-def-ghi", host[0].UUID)

	none, err := ds.SearchHosts(context.Background(), filter, "xxx")
	assert.Nil(t, err)
	assert.Len(t, none, 0)

	// check to make sure search on ip address works
	h2.PrimaryIP = "99.100.101.103"
	err = ds.SaveHost(context.Background(), h2)
	require.NoError(t, err)

	hits, err := ds.SearchHosts(context.Background(), filter, "99.100.101")
	require.NoError(t, err)
	require.Equal(t, 1, len(hits))

	hits, err = ds.SearchHosts(context.Background(), filter, "99.100.111")
	require.NoError(t, err)
	assert.Equal(t, 0, len(hits))

	h3.PrimaryIP = "99.100.101.104"
	err = ds.SaveHost(context.Background(), h3)
	require.NoError(t, err)
	hits, err = ds.SearchHosts(context.Background(), filter, "99.100.101")
	require.NoError(t, err)
	assert.Equal(t, 2, len(hits))
	hits, err = ds.SearchHosts(context.Background(), filter, "99.100.101", h3.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(hits))
}

func testHostsSearchLimit(t *testing.T, ds *Datastore) {
	filter := fleet.TeamFilter{User: test.UserAdmin}

	for i := 0; i < 15; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   fmt.Sprintf("host%d", i),
			NodeKey:         fmt.Sprintf("%d", i),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.%d.local", i),
		})
		require.NoError(t, err)
	}

	hosts, err := ds.SearchHosts(context.Background(), filter, "foo")
	require.NoError(t, err)
	assert.Len(t, hosts, 10)
}

func testHostsGenerateStatusStatistics(t *testing.T, ds *Datastore) {
	filter := fleet.TeamFilter{User: test.UserAdmin}
	mockClock := clock.NewMockClock()

	online, offline, mia, new, err := ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now())
	assert.Nil(t, err)
	assert.Equal(t, uint(0), online)
	assert.Equal(t, uint(0), offline)
	assert.Equal(t, uint(0), mia)
	assert.Equal(t, uint(0), new)

	// Online
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              1,
		OsqueryHostID:   "1",
		NodeKey:         "1",
		DetailUpdatedAt: mockClock.Now().Add(-30 * time.Second),
		LabelUpdatedAt:  mockClock.Now().Add(-30 * time.Second),
		PolicyUpdatedAt: mockClock.Now().Add(-30 * time.Second),
		SeenTime:        mockClock.Now().Add(-30 * time.Second),
	})
	require.NoError(t, err)
	h.DistributedInterval = 15
	h.ConfigTLSRefresh = 30
	require.Nil(t, ds.SaveHost(context.Background(), h))

	// Online
	h, err = ds.NewHost(context.Background(), &fleet.Host{
		ID:              2,
		OsqueryHostID:   "2",
		NodeKey:         "2",
		DetailUpdatedAt: mockClock.Now().Add(-1 * time.Minute),
		LabelUpdatedAt:  mockClock.Now().Add(-1 * time.Minute),
		PolicyUpdatedAt: mockClock.Now().Add(-1 * time.Minute),
		SeenTime:        mockClock.Now().Add(-1 * time.Minute),
	})
	require.NoError(t, err)
	h.DistributedInterval = 60
	h.ConfigTLSRefresh = 3600
	require.Nil(t, ds.SaveHost(context.Background(), h))

	// Offline
	h, err = ds.NewHost(context.Background(), &fleet.Host{
		ID:              3,
		OsqueryHostID:   "3",
		NodeKey:         "3",
		DetailUpdatedAt: mockClock.Now().Add(-1 * time.Hour),
		LabelUpdatedAt:  mockClock.Now().Add(-1 * time.Hour),
		PolicyUpdatedAt: mockClock.Now().Add(-1 * time.Hour),
		SeenTime:        mockClock.Now().Add(-1 * time.Hour),
	})
	require.NoError(t, err)
	h.DistributedInterval = 300
	h.ConfigTLSRefresh = 300
	require.Nil(t, ds.SaveHost(context.Background(), h))

	// MIA
	h, err = ds.NewHost(context.Background(), &fleet.Host{
		ID:              4,
		OsqueryHostID:   "4",
		NodeKey:         "4",
		DetailUpdatedAt: mockClock.Now().Add(-35 * (24 * time.Hour)),
		LabelUpdatedAt:  mockClock.Now().Add(-35 * (24 * time.Hour)),
		PolicyUpdatedAt: mockClock.Now().Add(-35 * (24 * time.Hour)),
		SeenTime:        mockClock.Now().Add(-35 * (24 * time.Hour)),
	})
	require.NoError(t, err)

	online, offline, mia, new, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now())
	assert.Nil(t, err)
	assert.Equal(t, uint(2), online)
	assert.Equal(t, uint(1), offline)
	assert.Equal(t, uint(1), mia)
	assert.Equal(t, uint(4), new)

	online, offline, mia, new, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now().Add(1*time.Hour))
	assert.Nil(t, err)
	assert.Equal(t, uint(0), online)
	assert.Equal(t, uint(3), offline)
	assert.Equal(t, uint(1), mia)
	assert.Equal(t, uint(4), new)
}

func testHostsMarkSeen(t *testing.T, ds *Datastore) {
	mockClock := clock.NewMockClock()

	anHourAgo := mockClock.Now().Add(-1 * time.Hour).UTC()
	aDayAgo := mockClock.Now().Add(-24 * time.Hour).UTC()

	h1, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              1,
		OsqueryHostID:   "1",
		UUID:            "1",
		NodeKey:         "1",
		DetailUpdatedAt: aDayAgo,
		LabelUpdatedAt:  aDayAgo,
		PolicyUpdatedAt: aDayAgo,
		SeenTime:        aDayAgo,
	})
	assert.Nil(t, err)

	{
		h1Verify, err := ds.Host(context.Background(), 1)
		assert.Nil(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, aDayAgo, h1Verify.SeenTime, time.Second)
	}

	err = ds.MarkHostSeen(context.Background(), h1, anHourAgo)
	assert.Nil(t, err)

	{
		h1Verify, err := ds.Host(context.Background(), 1)
		assert.Nil(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, anHourAgo, h1Verify.SeenTime, time.Second)
	}
}

func testHostsMarkSeenMany(t *testing.T, ds *Datastore) {
	mockClock := clock.NewMockClock()

	aSecondAgo := mockClock.Now().Add(-1 * time.Second).UTC()
	anHourAgo := mockClock.Now().Add(-1 * time.Hour).UTC()
	aDayAgo := mockClock.Now().Add(-24 * time.Hour).UTC()

	h1, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              1,
		OsqueryHostID:   "1",
		UUID:            "1",
		NodeKey:         "1",
		DetailUpdatedAt: aDayAgo,
		LabelUpdatedAt:  aDayAgo,
		PolicyUpdatedAt: aDayAgo,
		SeenTime:        aDayAgo,
	})
	require.NoError(t, err)

	h2, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              2,
		OsqueryHostID:   "2",
		UUID:            "2",
		NodeKey:         "2",
		DetailUpdatedAt: aDayAgo,
		LabelUpdatedAt:  aDayAgo,
		PolicyUpdatedAt: aDayAgo,
		SeenTime:        aDayAgo,
	})
	require.NoError(t, err)

	err = ds.MarkHostsSeen(context.Background(), []uint{h1.ID}, anHourAgo)
	assert.Nil(t, err)

	{
		h1Verify, err := ds.Host(context.Background(), h1.ID)
		assert.Nil(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, anHourAgo, h1Verify.SeenTime, time.Second)

		h2Verify, err := ds.Host(context.Background(), h2.ID)
		assert.Nil(t, err)
		require.NotNil(t, h2Verify)
		assert.WithinDuration(t, aDayAgo, h2Verify.SeenTime, time.Second)
	}

	err = ds.MarkHostsSeen(context.Background(), []uint{h1.ID, h2.ID}, aSecondAgo)
	assert.Nil(t, err)

	{
		h1Verify, err := ds.Host(context.Background(), h1.ID)
		assert.Nil(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, aSecondAgo, h1Verify.SeenTime, time.Second)

		h2Verify, err := ds.Host(context.Background(), h2.ID)
		assert.Nil(t, err)
		require.NotNil(t, h2Verify)
		assert.WithinDuration(t, aSecondAgo, h2Verify.SeenTime, time.Second)
	}
}

func testHostsCleanupIncoming(t *testing.T, ds *Datastore) {
	mockClock := clock.NewMockClock()

	h1, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              1,
		OsqueryHostID:   "1",
		UUID:            "1",
		NodeKey:         "1",
		DetailUpdatedAt: mockClock.Now(),
		LabelUpdatedAt:  mockClock.Now(),
		PolicyUpdatedAt: mockClock.Now(),
		SeenTime:        mockClock.Now(),
	})
	require.NoError(t, err)

	h2, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              2,
		OsqueryHostID:   "2",
		UUID:            "2",
		NodeKey:         "2",
		Hostname:        "foobar",
		OsqueryVersion:  "3.2.3",
		DetailUpdatedAt: mockClock.Now(),
		LabelUpdatedAt:  mockClock.Now(),
		PolicyUpdatedAt: mockClock.Now(),
		SeenTime:        mockClock.Now(),
	})
	require.NoError(t, err)

	err = ds.CleanupIncomingHosts(context.Background(), mockClock.Now().UTC())
	assert.Nil(t, err)

	// Both hosts should still exist because they are new
	_, err = ds.Host(context.Background(), h1.ID)
	assert.Nil(t, err)
	_, err = ds.Host(context.Background(), h2.ID)
	assert.Nil(t, err)

	err = ds.CleanupIncomingHosts(context.Background(), mockClock.Now().Add(6*time.Minute).UTC())
	assert.Nil(t, err)

	// Now only the host with details should exist
	_, err = ds.Host(context.Background(), h1.ID)
	assert.NotNil(t, err)
	_, err = ds.Host(context.Background(), h2.ID)
	assert.Nil(t, err)
}

func testHostsIDsByName(t *testing.T, ds *Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   fmt.Sprintf("host%d", i),
			NodeKey:         fmt.Sprintf("%d", i),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.%d.local", i),
		})
		require.NoError(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}
	hosts, err := ds.HostIDsByName(context.Background(), filter, []string{"foo.2.local", "foo.1.local", "foo.5.local"})
	require.NoError(t, err)
	sort.Slice(hosts, func(i, j int) bool { return hosts[i] < hosts[j] })
	assert.Equal(t, hosts, []uint{2, 3, 6})
}

func testAuthenticateHostLoadsDisk(t *testing.T, ds *Datastore) {
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   "foobar",
		NodeKey:         "nodekey",
		UUID:            "uuid",
		Hostname:        "foobar.local",
	})
	require.NoError(t, err)

	h.GigsDiskSpaceAvailable = 1.24
	h.PercentDiskSpaceAvailable = 42.0
	require.NoError(t, ds.SaveHost(context.Background(), h))
	h, err = ds.Host(context.Background(), h.ID)
	require.NoError(t, err)

	h, err = ds.AuthenticateHost(context.Background(), "nodekey")
	require.NoError(t, err)
	assert.NotZero(t, h.GigsDiskSpaceAvailable)
	assert.NotZero(t, h.PercentDiskSpaceAvailable)
}

func testHostsAdditional(t *testing.T, ds *Datastore) {
	_, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   "foobar",
		NodeKey:         "nodekey",
		UUID:            "uuid",
		Hostname:        "foobar.local",
	})
	require.NoError(t, err)

	h, err := ds.AuthenticateHost(context.Background(), "nodekey")
	require.NoError(t, err)
	assert.Equal(t, "foobar.local", h.Hostname)
	assert.Nil(t, h.Additional)

	// Additional not yet set
	h, err = ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Nil(t, h.Additional)

	// Add additional
	additional := json.RawMessage(`{"additional": "result"}`)
	h.Additional = &additional
	require.NoError(t, saveHostAdditionalDB(context.Background(), ds.writer, h))

	// Additional should not be loaded for authenticatehost
	h, err = ds.AuthenticateHost(context.Background(), "nodekey")
	require.NoError(t, err)
	assert.Equal(t, "foobar.local", h.Hostname)
	assert.Nil(t, h.Additional)

	h, err = ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, &additional, h.Additional)

	// Update besides additional. Additional should be unchanged.
	h, err = ds.AuthenticateHost(context.Background(), "nodekey")
	require.NoError(t, err)
	h.Hostname = "baz.local"
	err = ds.SaveHost(context.Background(), h)
	require.NoError(t, err)

	h, err = ds.AuthenticateHost(context.Background(), "nodekey")
	require.NoError(t, err)
	assert.Equal(t, "baz.local", h.Hostname)
	assert.Nil(t, h.Additional)

	h, err = ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, &additional, h.Additional)

	// Update additional
	additional = json.RawMessage(`{"other": "additional"}`)
	h, err = ds.AuthenticateHost(context.Background(), "nodekey")
	require.NoError(t, err)
	h.Additional = &additional
	err = saveHostAdditionalDB(context.Background(), ds.writer, h)
	require.NoError(t, err)

	h, err = ds.AuthenticateHost(context.Background(), "nodekey")
	require.NoError(t, err)
	assert.Equal(t, "baz.local", h.Hostname)
	assert.Nil(t, h.Additional)

	h, err = ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, &additional, h.Additional)
}

func testHostsByIdentifier(t *testing.T, ds *Datastore) {
	for i := 1; i <= 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   fmt.Sprintf("osquery_host_id_%d", i),
			NodeKey:         fmt.Sprintf("node_key_%d", i),
			UUID:            fmt.Sprintf("uuid_%d", i),
			Hostname:        fmt.Sprintf("hostname_%d", i),
		})
		require.NoError(t, err)
	}

	var (
		h   *fleet.Host
		err error
	)
	h, err = ds.HostByIdentifier(context.Background(), "uuid_1")
	require.NoError(t, err)
	assert.Equal(t, uint(1), h.ID)

	h, err = ds.HostByIdentifier(context.Background(), "osquery_host_id_2")
	require.NoError(t, err)
	assert.Equal(t, uint(2), h.ID)

	h, err = ds.HostByIdentifier(context.Background(), "node_key_4")
	require.NoError(t, err)
	assert.Equal(t, uint(4), h.ID)

	h, err = ds.HostByIdentifier(context.Background(), "hostname_7")
	require.NoError(t, err)
	assert.Equal(t, uint(7), h.ID)

	h, err = ds.HostByIdentifier(context.Background(), "foobar")
	require.Error(t, err)
}

func testHostsAddToTeam(t *testing.T, ds *Datastore) {
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		test.NewHost(t, ds, fmt.Sprint(i), "", "key"+fmt.Sprint(i), "uuid"+fmt.Sprint(i), time.Now())
	}

	for i := 1; i <= 10; i++ {
		host, err := ds.Host(context.Background(), uint(i))
		require.NoError(t, err)
		assert.Nil(t, host.TeamID)
	}

	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{1, 2, 3}))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team2.ID, []uint{3, 4, 5}))

	for i := 1; i <= 10; i++ {
		host, err := ds.Host(context.Background(), uint(i))
		require.NoError(t, err)
		var expectedID *uint
		switch {
		case i <= 2:
			expectedID = &team1.ID
		case i <= 5:
			expectedID = &team2.ID
		}
		assert.Equal(t, expectedID, host.TeamID)
	}

	require.NoError(t, ds.AddHostsToTeam(context.Background(), nil, []uint{1, 2, 3, 4}))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{5, 6, 7, 8, 9, 10}))

	for i := 1; i <= 10; i++ {
		host, err := ds.Host(context.Background(), uint(i))
		require.NoError(t, err)
		var expectedID *uint
		switch {
		case i >= 5:
			expectedID = &team1.ID
		}
		assert.Equal(t, expectedID, host.TeamID)
	}
}

func testHostsSaveUsers(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	assert.Len(t, host.Users, 0)

	u1 := fleet.HostUser{
		Uid:       42,
		Username:  "user",
		Type:      "aaa",
		GroupName: "group",
	}
	u2 := fleet.HostUser{
		Uid:       43,
		Username:  "user2",
		Type:      "aaa",
		GroupName: "group",
	}
	host.Users = []fleet.HostUser{u1, u2}
	host.Modified = true

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, host.Users, 2)
	test.ElementsMatchSkipID(t, host.Users, []fleet.HostUser{u1, u2})

	// remove u1 user
	host.Users = []fleet.HostUser{u2}
	host.Modified = true

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, host.Users, 1)
	assert.Equal(t, host.Users[0].Uid, u2.Uid)

	// readd u1
	host.Users = []fleet.HostUser{u1, u2}
	host.Modified = true

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, host.Users, 2)
	test.ElementsMatchSkipID(t, host.Users, []fleet.HostUser{u1, u2})
}

func testHostsSaveUsersWithoutUid(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	assert.Len(t, host.Users, 0)

	u1 := fleet.HostUser{
		Username:  "user",
		Type:      "aaa",
		GroupName: "group",
	}
	u2 := fleet.HostUser{
		Username:  "user2",
		Type:      "aaa",
		GroupName: "group",
	}
	host.Users = []fleet.HostUser{u1, u2}
	host.Modified = true

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, host.Users, 2)
	test.ElementsMatchSkipID(t, host.Users, []fleet.HostUser{u1, u2})

	// remove u1 user
	host.Users = []fleet.HostUser{u2}
	host.Modified = true

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, host.Users, 1)
	assert.Equal(t, host.Users[0].Uid, u2.Uid)
}

func addHostSeenLast(t *testing.T, ds fleet.Datastore, i, days int) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Duration(days) * 24 * time.Hour),
		OsqueryHostID:   fmt.Sprintf("%d", i),
		NodeKey:         fmt.Sprintf("%d", i),
		UUID:            fmt.Sprintf("%d", i),
		Hostname:        fmt.Sprintf("foo.local%d", i),
		PrimaryIP:       fmt.Sprintf("192.168.1.%d", i),
		PrimaryMac:      fmt.Sprintf("30-65-EC-6F-C4-5%d", i),
	})
	require.NoError(t, err)
	require.NotNil(t, host)
}

func testHostsTotalAndUnseenSince(t *testing.T, ds *Datastore) {
	addHostSeenLast(t, ds, 1, 0)

	total, unseen, err := ds.TotalAndUnseenHostsSince(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, 0, unseen)

	addHostSeenLast(t, ds, 2, 2)
	addHostSeenLast(t, ds, 3, 4)

	total, unseen, err = ds.TotalAndUnseenHostsSince(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Equal(t, 2, unseen)
}

func testHostsListByPolicy(t *testing.T, ds *Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   strconv.Itoa(i),
			NodeKey:         fmt.Sprintf("%d", i),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	q := test.NewQuery(t, ds, "query1", "select 1", 0, true)
	p, err := ds.NewGlobalPolicy(context.Background(), q.ID, "")
	require.NoError(t, err)

	// When policy response is null, we list all hosts that haven't reported at all for the policy, or errored out
	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{PolicyIDFilter: &p.ID}, 10)
	require.Len(t, hosts, 10)

	h1 := hosts[0]
	h2 := hosts[1]

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{PolicyIDFilter: &p.ID, PolicyResponseFilter: ptr.Bool(true)}, 0)
	require.Len(t, hosts, 0)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{PolicyIDFilter: &p.ID, PolicyResponseFilter: ptr.Bool(false)}, 0)
	require.Len(t, hosts, 0)

	// Make one host pass the policy and another not pass
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h1, map[uint]*bool{1: ptr.Bool(true)}, time.Now()))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{1: ptr.Bool(false)}, time.Now()))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{PolicyIDFilter: &p.ID, PolicyResponseFilter: ptr.Bool(true)}, 1)
	require.Len(t, hosts, 1)
	assert.Equal(t, h1.ID, hosts[0].ID)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{PolicyIDFilter: &p.ID, PolicyResponseFilter: ptr.Bool(false)}, 1)
	require.Len(t, hosts, 1)
	assert.Equal(t, h2.ID, hosts[0].ID)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{PolicyIDFilter: &p.ID}, 8)
	require.Len(t, hosts, 8)
}

func testHostsListBySoftware(t *testing.T, ds *Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   strconv.Itoa(i),
			NodeKey:         fmt.Sprintf("%d", i),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 10)

	soft := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{Name: "bar", Version: "0.0.3", Source: "deb_packages", BundleIdentifier: "com.some.identifier"},
		},
	}
	host1 := hosts[0]
	host2 := hosts[1]
	host1.HostSoftware = soft
	host2.HostSoftware = soft

	require.NoError(t, ds.SaveHostSoftware(context.Background(), host1))
	require.NoError(t, ds.SaveHostSoftware(context.Background(), host2))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{SoftwareIDFilter: &host1.Software[0].ID}, 2)
	require.Len(t, hosts, 2)
}

func testHostsListFailingPolicies(t *testing.T, ds *Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   strconv.Itoa(i),
			NodeKey:         fmt.Sprintf("%d", i),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	q := test.NewQuery(t, ds, "query1", "select 1", 0, true)
	q2 := test.NewQuery(t, ds, "query2", "select 1", 0, true)
	p, err := ds.NewGlobalPolicy(context.Background(), q.ID, "")
	require.NoError(t, err)
	p2, err := ds.NewGlobalPolicy(context.Background(), q2.ID, "")
	require.NoError(t, err)

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 10)
	require.Len(t, hosts, 10)

	h1 := hosts[0]
	h2 := hosts[1]

	assert.Zero(t, h1.HostIssues.FailingPoliciesCount)
	assert.Zero(t, h1.HostIssues.TotalIssuesCount)
	assert.Zero(t, h2.HostIssues.FailingPoliciesCount)
	assert.Zero(t, h2.HostIssues.TotalIssuesCount)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h1, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now()))

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{p.ID: ptr.Bool(false), p2.ID: ptr.Bool(false)}, time.Now()))
	checkHostIssues(t, ds, hosts, filter, h2.ID, 2)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{p.ID: ptr.Bool(true), p2.ID: ptr.Bool(false)}, time.Now()))
	checkHostIssues(t, ds, hosts, filter, h2.ID, 1)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{p.ID: ptr.Bool(true), p2.ID: ptr.Bool(true)}, time.Now()))
	checkHostIssues(t, ds, hosts, filter, h2.ID, 0)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h1, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now()))
	checkHostIssues(t, ds, hosts, filter, h1.ID, 1)
}

func checkHostIssues(t *testing.T, ds *Datastore, hosts []*fleet.Host, filter fleet.TeamFilter, hid uint, expected int) {
	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 10)
	foundH2 := false
	var foundHost *fleet.Host
	for _, host := range hosts {
		if host.ID == hid {
			foundH2 = true
			foundHost = host
			break
		}
	}
	require.True(t, foundH2)
	assert.Equal(t, expected, foundHost.HostIssues.FailingPoliciesCount)
	assert.Equal(t, expected, foundHost.HostIssues.TotalIssuesCount)

	hostById, err := ds.Host(context.Background(), hid)
	require.NoError(t, err)
	assert.Equal(t, expected, hostById.HostIssues.FailingPoliciesCount)
	assert.Equal(t, expected, hostById.HostIssues.TotalIssuesCount)
}

func testHostsSaveTonsOfUsers(t *testing.T, ds *Datastore) {
	host1, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   "1",
	})
	require.NoError(t, err)
	require.NotNil(t, host1)

	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "foo2.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   "2",
	})
	require.NoError(t, err)
	require.NotNil(t, host2)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	errCh := make(chan error)
	var count1 int32
	var count2 int32

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		for {
			host1, err := ds.Host(context.Background(), host1.ID)
			if err != nil {
				errCh <- err
				return
			}

			u1 := fleet.HostUser{
				Uid:       42,
				Username:  "user",
				Type:      "aaa",
				GroupName: "group",
			}
			u2 := fleet.HostUser{
				Uid:       43,
				Username:  "user2",
				Type:      "aaa",
				GroupName: "group",
			}
			host1.Users = []fleet.HostUser{u1, u2}
			host1.SeenTime = time.Now()
			host1.Modified = true
			soft := fleet.HostSoftware{
				Modified: true,
				Software: []fleet.Software{
					{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
					{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
				},
			}
			host1.HostSoftware = soft
			additional := json.RawMessage(`{"some":"thing"}`)
			host1.Additional = &additional

			err = ds.SaveHost(context.Background(), host1)
			if err != nil {
				errCh <- err
				return
			}
			if atomic.AddInt32(&count1, 1) >= 100 {
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	go func() {
		defer wg.Done()

		for {
			host2, err := ds.Host(context.Background(), host2.ID)
			if err != nil {
				errCh <- err
				return
			}

			u1 := fleet.HostUser{
				Uid:       99,
				Username:  "user",
				Type:      "aaa",
				GroupName: "group",
			}
			u2 := fleet.HostUser{
				Uid:       98,
				Username:  "user2",
				Type:      "aaa",
				GroupName: "group",
			}
			host2.Users = []fleet.HostUser{u1, u2}
			host2.SeenTime = time.Now()
			host2.Modified = true
			soft := fleet.HostSoftware{
				Modified: true,
				Software: []fleet.Software{
					{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
					{Name: "foo4", Version: "0.0.3", Source: "chrome_extensions"},
				},
			}
			host2.HostSoftware = soft
			additional := json.RawMessage(`{"some":"thing"}`)
			host2.Additional = &additional

			err = ds.SaveHost(context.Background(), host2)
			if err != nil {
				errCh <- err
				return
			}
			if atomic.AddInt32(&count2, 1) >= 100 {
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		wg.Wait()
		cancelFunc()
	}()

	select {
	case err := <-errCh:
		cancelFunc()
		require.NoError(t, err)
	case <-ctx.Done():
	case <-ticker.C:
		require.Fail(t, "timed out")
	}
	t.Log("Count1", atomic.LoadInt32(&count1))
	t.Log("Count2", atomic.LoadInt32(&count2))
}

func testHostsSavePackStatsConcurrent(t *testing.T, ds *Datastore) {
	host1, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   "1",
	})
	require.NoError(t, err)
	require.NotNil(t, host1)

	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   "2",
	})
	require.NoError(t, err)
	require.NotNil(t, host2)

	pack1 := test.NewPack(t, ds, "test1")
	query1 := test.NewQuery(t, ds, "time", "select * from time", 0, true)
	squery1 := test.NewScheduledQuery(t, ds, pack1.ID, query1.ID, 30, true, true, "time-scheduled")

	pack2 := test.NewPack(t, ds, "test2")
	query2 := test.NewQuery(t, ds, "time2", "select * from time", 0, true)
	squery2 := test.NewScheduledQuery(t, ds, pack2.ID, query2.ID, 30, true, true, "time-scheduled")

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	saveHostRandomStats := func(host *fleet.Host) error {
		host.PackStats = []fleet.PackStats{
			{
				PackName: pack1.Name,
				QueryStats: []fleet.ScheduledQueryStats{
					{
						ScheduledQueryName: squery1.Name,
						ScheduledQueryID:   squery1.ID,
						QueryName:          query1.Name,
						PackName:           pack1.Name,
						PackID:             pack1.ID,
						AverageMemory:      8000,
						Denylisted:         false,
						Executions:         rand.Intn(1000),
						Interval:           30,
						LastExecuted:       time.Now().UTC(),
						OutputSize:         1337,
						SystemTime:         150,
						UserTime:           180,
						WallTime:           0,
					},
				},
			},
			{
				PackName: pack2.Name,
				QueryStats: []fleet.ScheduledQueryStats{
					{
						ScheduledQueryName: squery2.Name,
						ScheduledQueryID:   squery2.ID,
						QueryName:          query2.Name,
						PackName:           pack2.Name,
						PackID:             pack2.ID,
						AverageMemory:      8000,
						Denylisted:         false,
						Executions:         rand.Intn(1000),
						Interval:           30,
						LastExecuted:       time.Now().UTC(),
						OutputSize:         1337,
						SystemTime:         150,
						UserTime:           180,
						WallTime:           0,
					},
				},
			},
		}
		return ds.SaveHost(context.Background(), host)
	}

	errCh := make(chan error)
	var counter int32
	const total = int32(100)

	var wg sync.WaitGroup

	loopAndSaveHost := func(host *fleet.Host) {
		defer wg.Done()

		for {
			err := saveHostRandomStats(host)
			if err != nil {
				errCh <- err
				return
			}
			atomic.AddInt32(&counter, 1)
			select {
			case <-ctx.Done():
				return
			default:
				if atomic.LoadInt32(&counter) > total {
					cancelFunc()
					return
				}
			}
		}
	}

	wg.Add(3)
	go loopAndSaveHost(host1)
	go loopAndSaveHost(host2)

	go func() {
		defer wg.Done()

		for {
			specs := []*fleet.PackSpec{
				{
					Name: "test1",
					Queries: []fleet.PackSpecQuery{
						{
							QueryName: "time",
							Interval:  uint(rand.Intn(1000)),
						},
						{
							QueryName: "time2",
							Interval:  uint(rand.Intn(1000)),
						},
					},
				},
				{
					Name: "test2",
					Queries: []fleet.PackSpecQuery{
						{
							QueryName: "time",
							Interval:  uint(rand.Intn(1000)),
						},
						{
							QueryName: "time2",
							Interval:  uint(rand.Intn(1000)),
						},
					},
				},
			}
			err := ds.ApplyPackSpecs(context.Background(), specs)
			if err != nil {
				errCh <- err
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	select {
	case err := <-errCh:
		cancelFunc()
		require.NoError(t, err)
	case <-ctx.Done():
		wg.Wait()
	case <-ticker.C:
		require.Fail(t, "timed out")
	}
}

func testHostsExpiration(t *testing.T, ds *Datastore) {
	hostExpiryWindow := 70

	ac, err := ds.AppConfig(context.Background())
	require.NoError(t, err)

	ac.HostExpirySettings.HostExpiryWindow = hostExpiryWindow

	err = ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		seenTime := time.Now()
		if i >= 5 {
			seenTime = seenTime.Add(time.Duration(-1*(hostExpiryWindow+1)*24) * time.Hour)
		}
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        seenTime,
			OsqueryHostID:   strconv.Itoa(i),
			NodeKey:         fmt.Sprintf("%d", i),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 10)
	require.Len(t, hosts, 10)

	err = ds.CleanupExpiredHosts(context.Background())
	require.NoError(t, err)

	// host expiration is still disabled
	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 10)
	require.Len(t, hosts, 10)

	// once enabled, it works
	ac.HostExpirySettings.HostExpiryEnabled = true
	err = ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)

	err = ds.CleanupExpiredHosts(context.Background())
	require.NoError(t, err)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 5)
	require.Len(t, hosts, 5)

	// And it doesn't remove more than it should
	err = ds.CleanupExpiredHosts(context.Background())
	require.NoError(t, err)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 5)
	require.Len(t, hosts, 5)
}
