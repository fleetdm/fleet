package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var expLastExec = func() time.Time {
	t, _ := time.Parse(time.RFC3339, pastDate)
	return t
}()

var enrollTests = []struct {
	uuid, hostname, platform, nodeKey string
}{
	0: {
		uuid:     "6D14C88F-8ECF-48D5-9197-777647BF6B26",
		hostname: "web.fleet.co",
		platform: "linux",
		nodeKey:  "key0",
	},
	1: {
		uuid:     "B998C0EB-38CE-43B1-A743-FBD7A5C9513B",
		hostname: "mail.fleet.co",
		platform: "linux",
		nodeKey:  "key1",
	},
	2: {
		uuid:     "008F0688-5311-4C59-86EE-00C2D6FC3EC2",
		hostname: "home.fleet.co",
		platform: "darwin",
		nodeKey:  "key2",
	},
	3: {
		uuid:     "uuid123",
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
		{"WithTeamPackStats", testHostsWithTeamPackStats},
		{"Delete", testHostsDelete},
		{"ListFilterAdditional", testHostsListFilterAdditional},
		{"ListStatus", testHostsListStatus},
		{"ListQuery", testHostsListQuery},
		{"Enroll", testHostsEnroll},
		{"LoadHostByNodeKey", testHostsLoadHostByNodeKey},
		{"LoadHostByNodeKeyCaseSensitive", testHostsLoadHostByNodeKeyCaseSensitive},
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
		{"SaveHostUsers", testHostsSaveHostUsers},
		{"SaveUsersWithoutUid", testHostsSaveUsersWithoutUid},
		{"TotalAndUnseenSince", testHostsTotalAndUnseenSince},
		{"ListByPolicy", testHostsListByPolicy},
		{"SaveTonsOfUsers", testHostsSaveTonsOfUsers},
		{"SavePackStatsConcurrent", testHostsSavePackStatsConcurrent},
		{"LoadHostByNodeKeyLoadsDisk", testLoadHostByNodeKeyLoadsDisk},
		{"LoadHostByNodeKeyUsesStmt", testLoadHostByNodeKeyUsesStmt},
		{"HostsListBySoftware", testHostsListBySoftware},
		{"HostsListFailingPolicies", printReadsInTest(testHostsListFailingPolicies)},
		{"HostsExpiration", testHostsExpiration},
		{"HostsAllPackStats", testHostsAllPackStats},
		{"HostsPackStatsMultipleHosts", testHostsPackStatsMultipleHosts},
		{"HostsPackStatsForPlatform", testHostsPackStatsForPlatform},
		{"HostsReadsLessRows", testHostsReadsLessRows},
		{"HostsNoSeenTime", testHostsNoSeenTime},
		{"ListHostDeviceMapping", testHostsListHostDeviceMapping},
		{"ReplaceHostDeviceMapping", testHostsReplaceHostDeviceMapping},
		{"HostMDMAndMunki", testHostMDMAndMunki},
		{"AggregatedHostMDMAndMunki", testAggregatedHostMDMAndMunki},
		{"HostLite", testHostsLite},
		{"UpdateOsqueryIntervals", testUpdateOsqueryIntervals},
		{"UpdateRefetchRequested", testUpdateRefetchRequested},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testHostsSave(t *testing.T, ds *Datastore) {
	testSaveHost(t, ds, ds.SaveHost)
	testSaveHost(t, ds, ds.SerialUpdateHost)
}

func testSaveHost(t *testing.T, ds *Datastore, saveHostFunc func(context.Context, *fleet.Host) error) {
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
	err = saveHostFunc(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	assert.Equal(t, "bar.local", host.Hostname)
	assert.Equal(t, "192.168.1.1", host.PrimaryIP)
	assert.Equal(t, "30-65-EC-6F-C4-58", host.PrimaryMac)
	assert.Equal(t, policyUpdatedAt.UTC(), host.PolicyUpdatedAt)

	additionalJSON := json.RawMessage(`{"foobar": "bim"}`)
	host.Additional = &additionalJSON

	require.NoError(t, saveHostFunc(context.Background(), host))
	require.NoError(t, saveHostAdditionalDB(context.Background(), ds.writer, host.ID, host.Additional))

	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	require.NotNil(t, host)
	require.NotNil(t, host.Additional)
	assert.Equal(t, additionalJSON, *host.Additional)

	err = saveHostFunc(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	require.NotNil(t, host)

	p, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    t.Name(),
		HostIDs: []uint{host.ID},
	})
	require.NoError(t, err)

	err = ds.DeleteHost(context.Background(), host.ID)
	require.NoError(t, err)

	newP, err := ds.Pack(context.Background(), p.ID)
	require.NoError(t, err)
	require.Empty(t, newP.Hosts)

	host, err = ds.Host(context.Background(), host.ID, false)
	assert.NotNil(t, err)
	assert.Nil(t, host)

	require.NoError(t, ds.DeletePack(context.Background(), newP.Name))
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

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
	}
	err = ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)

	err = ds.DeleteHost(context.Background(), host.ID)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID, false)
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
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	// Pack and query must exist for stats to save successfully
	pack1, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test1",
		HostIDs: []uint{host.ID},
	})
	require.NoError(t, err)
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

	pack2, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test2",
		HostIDs: []uint{host.ID},
	})
	require.NoError(t, err)
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

	host, err = ds.Host(context.Background(), host.ID, false)
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
	host, err = ds.Host(context.Background(), host.ID, false)
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
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	// Pack and query must exist for stats to save successfully
	pack1, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test1",
		HostIDs: []uint{host.ID},
	})
	require.NoError(t, err)
	query1 := test.NewQuery(t, ds, "time", "select * from time", 0, true)
	squery1 := test.NewScheduledQuery(t, ds, pack1.ID, query1.ID, 30, true, true, "time-scheduled")
	pack2, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test2",
		HostIDs: []uint{host.ID},
	})
	require.NoError(t, err)
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

	host, err = ds.Host(context.Background(), host.ID, false)
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

	gotHost, err := ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)

	sort.Slice(gotHost.PackStats, func(i, j int) bool {
		return gotHost.PackStats[i].PackName < gotHost.PackStats[j].PackName
	})

	require.Len(t, gotHost.PackStats, 2)
	assert.Equal(t, gotHost.PackStats[0].PackName, "test1")
	assert.Equal(t, execTime2, gotHost.PackStats[0].QueryStats[0].LastExecuted)
}

func testHostsWithTeamPackStats(t *testing.T, ds *Datastore) {
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
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	team, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID}))
	tp, err := ds.EnsureTeamPack(context.Background(), team.ID)
	require.NoError(t, err)
	tpQuery := test.NewQuery(t, ds, "tp-time", "select * from time", 0, true)
	tpSquery := test.NewScheduledQuery(t, ds, tp.ID, tpQuery.ID, 30, true, true, "time-scheduled")

	// Create a new pack and target to the host.
	// Pack and query must exist for stats to save successfully
	pack1, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test1",
		HostIDs: []uint{host.ID},
	})
	require.NoError(t, err)
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

	// Reload the host and set the scheduled queries stats.
	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	host.PackStats = []fleet.PackStats{
		{PackID: pack1.ID, PackName: pack1.Name, QueryStats: stats1},
		{PackID: tp.ID, PackName: teamScheduleName(team), QueryStats: stats2},
	}
	require.NoError(t, ds.SaveHost(context.Background(), host))

	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)

	require.Len(t, host.PackStats, 2)
	sort.Sort(packStatsSlice(host.PackStats))

	assert.Equal(t, host.PackStats[0].PackName, teamScheduleName(team))
	assert.ElementsMatch(t, host.PackStats[0].QueryStats, stats2)

	assert.Equal(t, host.PackStats[1].PackName, pack1.Name)
	assert.ElementsMatch(t, host.PackStats[1].QueryStats, stats1)
}

type packStatsSlice []fleet.PackStats

func (p packStatsSlice) Len() int {
	return len(p)
}

func (p packStatsSlice) Less(i, j int) bool {
	return p[i].PackID < p[j].PackID
}

func (p packStatsSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
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
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID, false)
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
	require.NoError(t, saveHostAdditionalDB(context.Background(), ds.writer, h.ID, h.Additional))

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
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute * 2),
			OsqueryHostID:   strconv.Itoa(i),
			NodeKey:         fmt.Sprintf("%d", i),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
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

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{StatusFilter: "new", ListOptions: fleet.ListOptions{OrderKey: "h.id", After: fmt.Sprint(hosts[2].ID)}}, 7)
	assert.Equal(t, 7, len(hosts))
}

func testHostsListQuery(t *testing.T, ds *Datastore) {
	hosts := []*fleet.Host{}
	for i := 0; i < 10; i++ {
		hostname := fmt.Sprintf("hostname%%00%d", i)
		if i == 5 {
			hostname += "ba@b.ca"
		}
		host, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   strconv.Itoa(i),
			NodeKey:         fmt.Sprintf("%d", i),
			UUID:            fmt.Sprintf("uuid_00%d", i),
			Hostname:        hostname,
			HardwareSerial:  fmt.Sprintf("serial00%d", i),
		})
		require.NoError(t, err)
		host.PrimaryIP = fmt.Sprintf("192.168.1.%d", i)
		require.NoError(t, ds.SaveHost(context.Background(), host))
		hosts = append(hosts, host)
	}

	// add some device mapping for some hosts
	require.NoError(t, ds.ReplaceHostDeviceMapping(context.Background(), hosts[0].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[0].ID, Email: "a@b.c", Source: "src1"},
		{HostID: hosts[0].ID, Email: "b@b.c", Source: "src1"},
	}))
	require.NoError(t, ds.ReplaceHostDeviceMapping(context.Background(), hosts[1].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[1].ID, Email: "c@b.c", Source: "src1"},
	}))
	require.NoError(t, ds.ReplaceHostDeviceMapping(context.Background(), hosts[2].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[2].ID, Email: "dbca@b.cba", Source: "src1"},
	}))

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

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{ListOptions: fleet.ListOptions{MatchQuery: "a@b.c"}}, 3)
	require.Equal(t, 3, len(gotHosts))
	gotIDs := []uint{gotHosts[0].ID, gotHosts[1].ID, gotHosts[2].ID}
	wantIDs := []uint{hosts[0].ID, hosts[2].ID, hosts[5].ID}
	require.ElementsMatch(t, wantIDs, gotIDs)
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

func testHostsLoadHostByNodeKey(t *testing.T, ds *Datastore) {
	test.AddAllHostsLabel(t, ds)
	for _, tt := range enrollTests {
		h, err := ds.EnrollHost(context.Background(), tt.uuid, tt.nodeKey, nil, 0)
		require.NoError(t, err)

		returned, err := ds.LoadHostByNodeKey(context.Background(), h.NodeKey)
		require.NoError(t, err)
		assert.Equal(t, h, returned)
	}

	_, err := ds.LoadHostByNodeKey(context.Background(), "7B1A9DC9-B042-489F-8D5A-EEC2412C95AA")
	assert.Error(t, err)

	_, err = ds.LoadHostByNodeKey(context.Background(), "")
	assert.Error(t, err)
}

func testHostsLoadHostByNodeKeyCaseSensitive(t *testing.T, ds *Datastore) {
	test.AddAllHostsLabel(t, ds)
	for _, tt := range enrollTests {
		h, err := ds.EnrollHost(context.Background(), tt.uuid, tt.nodeKey, nil, 0)
		require.NoError(t, err)

		_, err = ds.LoadHostByNodeKey(context.Background(), strings.ToUpper(h.NodeKey))
		require.Error(t, err, "node key authentication should be case sensitive")
	}
}

func testHostsSearch(t *testing.T, ds *Datastore) {
	h1, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   "1234",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "fo.local",
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

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{h1.ID}))
	h1.TeamID = &team1.ID
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team2.ID, []uint{h2.ID}))
	h2.TeamID = &team2.ID

	userAdmin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: userAdmin}

	// We once threw errors when the search query was empty. Verify that we
	// don't error.
	_, err = ds.SearchHosts(context.Background(), filter, "")
	require.NoError(t, err)

	hosts, err := ds.SearchHosts(context.Background(), filter, "fo")
	require.NoError(t, err)
	assert.Len(t, hosts, 2)

	hosts, err = ds.SearchHosts(context.Background(), filter, "fo.")
	require.NoError(t, err)
	assert.Len(t, hosts, 1)

	host, err := ds.SearchHosts(context.Background(), filter, "fo", h3.ID)
	require.NoError(t, err)
	require.Len(t, host, 1)
	assert.Equal(t, "fo.local", host[0].Hostname)

	host, err = ds.SearchHosts(context.Background(), filter, "fo", h3.ID, h2.ID)
	require.NoError(t, err)
	require.Len(t, host, 1)
	assert.Equal(t, "fo.local", host[0].Hostname)

	host, err = ds.SearchHosts(context.Background(), filter, "abc")
	require.NoError(t, err)
	require.Len(t, host, 1)
	assert.Equal(t, "abc-def-ghi", host[0].UUID)

	none, err := ds.SearchHosts(context.Background(), filter, "xxx")
	require.NoError(t, err)
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

	hits, err = ds.SearchHosts(context.Background(), filter, "f")
	require.NoError(t, err)
	assert.Equal(t, 2, len(hits))

	hits, err = ds.SearchHosts(context.Background(), filter, "f", h3.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(hits))

	hits, err = ds.SearchHosts(context.Background(), filter, "fx")
	require.NoError(t, err)
	assert.Equal(t, 0, len(hits))

	hits, err = ds.SearchHosts(context.Background(), filter, "x")
	require.NoError(t, err)
	assert.Equal(t, 0, len(hits))

	hits, err = ds.SearchHosts(context.Background(), filter, "x", h3.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, len(hits))

	userObs := &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)}
	filter = fleet.TeamFilter{User: userObs}

	// observer not included
	hosts, err = ds.SearchHosts(context.Background(), filter, "local")
	require.NoError(t, err)
	assert.Len(t, hosts, 0)

	// observer included
	filter.IncludeObserver = true
	hosts, err = ds.SearchHosts(context.Background(), filter, "local")
	require.NoError(t, err)
	assert.Len(t, hosts, 3)

	userTeam1 := &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleAdmin}}}
	filter = fleet.TeamFilter{User: userTeam1}

	hosts, err = ds.SearchHosts(context.Background(), filter, "local")
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, hosts[0].ID, h1.ID)

	userTeam2 := &fleet.User{Teams: []fleet.UserTeam{{Team: *team2, Role: fleet.RoleObserver}}}
	filter = fleet.TeamFilter{User: userTeam2}

	// observer not included
	hosts, err = ds.SearchHosts(context.Background(), filter, "local")
	require.NoError(t, err)
	assert.Len(t, hosts, 0)

	// observer included
	filter.IncludeObserver = true
	hosts, err = ds.SearchHosts(context.Background(), filter, "local")
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, hosts[0].ID, h2.ID)

	// specific team id
	filter.TeamID = &team2.ID
	hosts, err = ds.SearchHosts(context.Background(), filter, "local")
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, hosts[0].ID, h2.ID)

	// sorted by ids desc
	filter = fleet.TeamFilter{User: userObs, IncludeObserver: true}
	hits, err = ds.SearchHosts(context.Background(), filter, "")
	require.NoError(t, err)
	assert.Len(t, hits, 3)
	assert.Equal(t, []uint{h3.ID, h2.ID, h1.ID}, []uint{hits[0].ID, hits[1].ID, hits[2].ID})
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

	summary, err := ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now(), nil)
	require.NoError(t, err)
	assert.Nil(t, summary.TeamID)
	assert.Equal(t, uint(0), summary.TotalsHostsCount)
	assert.Equal(t, uint(0), summary.OnlineCount)
	assert.Equal(t, uint(0), summary.OfflineCount)
	assert.Equal(t, uint(0), summary.MIACount)
	assert.Equal(t, uint(0), summary.NewCount)

	// Online
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              1,
		OsqueryHostID:   "1",
		NodeKey:         "1",
		DetailUpdatedAt: mockClock.Now().Add(-30 * time.Second),
		LabelUpdatedAt:  mockClock.Now().Add(-30 * time.Second),
		PolicyUpdatedAt: mockClock.Now().Add(-30 * time.Second),
		SeenTime:        mockClock.Now().Add(-30 * time.Second),
		Platform:        "debian",
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
		Platform:        "windows",
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
		Platform:        "darwin",
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
		Platform:        "rhel",
	})
	require.NoError(t, err)

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{h.ID}))

	wantPlatforms := []*fleet.HostSummaryPlatform{
		{Platform: "debian", HostsCount: 1},
		{Platform: "rhel", HostsCount: 1},
		{Platform: "windows", HostsCount: 1},
		{Platform: "darwin", HostsCount: 1},
	}

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now(), nil)
	require.NoError(t, err)
	assert.Equal(t, uint(4), summary.TotalsHostsCount)
	assert.Equal(t, uint(2), summary.OnlineCount)
	assert.Equal(t, uint(1), summary.OfflineCount)
	assert.Equal(t, uint(1), summary.MIACount)
	assert.Equal(t, uint(4), summary.NewCount)
	assert.ElementsMatch(t, summary.Platforms, wantPlatforms)

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now().Add(1*time.Hour), nil)
	require.NoError(t, err)
	assert.Equal(t, uint(4), summary.TotalsHostsCount)
	assert.Equal(t, uint(0), summary.OnlineCount)
	assert.Equal(t, uint(3), summary.OfflineCount)
	assert.Equal(t, uint(1), summary.MIACount)
	assert.Equal(t, uint(4), summary.NewCount)
	assert.ElementsMatch(t, summary.Platforms, wantPlatforms)

	userObs := &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)}
	filter = fleet.TeamFilter{User: userObs}

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now().Add(1*time.Hour), nil)
	require.NoError(t, err)
	assert.Equal(t, uint(0), summary.TotalsHostsCount)

	filter.IncludeObserver = true
	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now().Add(1*time.Hour), nil)
	require.NoError(t, err)
	assert.Equal(t, uint(4), summary.TotalsHostsCount)

	userTeam1 := &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleAdmin}}}
	filter = fleet.TeamFilter{User: userTeam1}
	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now().Add(1*time.Hour), nil)
	require.NoError(t, err)
	assert.Equal(t, uint(1), summary.TotalsHostsCount)
	assert.Equal(t, uint(1), summary.MIACount)

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), fleet.TeamFilter{User: test.UserAdmin}, mockClock.Now(), ptr.String("linux"))
	require.NoError(t, err)
	assert.Equal(t, uint(2), summary.TotalsHostsCount)

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now(), ptr.String("linux"))
	require.NoError(t, err)
	assert.Equal(t, uint(1), summary.TotalsHostsCount)

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), fleet.TeamFilter{User: test.UserAdmin}, mockClock.Now(), ptr.String("darwin"))
	require.NoError(t, err)
	assert.Equal(t, uint(1), summary.TotalsHostsCount)

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), fleet.TeamFilter{User: test.UserAdmin}, mockClock.Now(), ptr.String("windows"))
	require.NoError(t, err)
	assert.Equal(t, uint(1), summary.TotalsHostsCount)
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
	require.NoError(t, err)

	{
		h1Verify, err := ds.Host(context.Background(), 1, false)
		require.NoError(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, aDayAgo, h1Verify.SeenTime, time.Second)
	}

	err = ds.MarkHostsSeen(context.Background(), []uint{h1.ID}, anHourAgo)
	require.NoError(t, err)

	{
		h1Verify, err := ds.Host(context.Background(), 1, false)
		require.NoError(t, err)
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
	require.NoError(t, err)

	{
		h1Verify, err := ds.Host(context.Background(), h1.ID, false)
		require.NoError(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, anHourAgo, h1Verify.SeenTime, time.Second)

		h2Verify, err := ds.Host(context.Background(), h2.ID, false)
		require.NoError(t, err)
		require.NotNil(t, h2Verify)
		assert.WithinDuration(t, aDayAgo, h2Verify.SeenTime, time.Second)
	}

	err = ds.MarkHostsSeen(context.Background(), []uint{h1.ID, h2.ID}, aSecondAgo)
	require.NoError(t, err)

	{
		h1Verify, err := ds.Host(context.Background(), h1.ID, false)
		require.NoError(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, aSecondAgo, h1Verify.SeenTime, time.Second)

		h2Verify, err := ds.Host(context.Background(), h2.ID, false)
		require.NoError(t, err)
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
	require.NoError(t, err)

	// Both hosts should still exist because they are new
	_, err = ds.Host(context.Background(), h1.ID, false)
	require.NoError(t, err)
	_, err = ds.Host(context.Background(), h2.ID, false)
	require.NoError(t, err)

	err = ds.CleanupIncomingHosts(context.Background(), mockClock.Now().Add(6*time.Minute).UTC())
	require.NoError(t, err)

	// Now only the host with details should exist
	_, err = ds.Host(context.Background(), h1.ID, false)
	assert.NotNil(t, err)
	_, err = ds.Host(context.Background(), h2.ID, false)
	require.NoError(t, err)
}

func testHostsIDsByName(t *testing.T, ds *Datastore) {
	hosts := make([]*fleet.Host, 10)
	for i := range hosts {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
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
		hosts[i] = h
	}

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{hosts[0].ID}))

	filter := fleet.TeamFilter{User: test.UserAdmin}
	hostsByName, err := ds.HostIDsByName(context.Background(), filter, []string{"foo.2.local", "foo.1.local", "foo.5.local"})
	require.NoError(t, err)
	sort.Slice(hostsByName, func(i, j int) bool { return hostsByName[i] < hostsByName[j] })
	assert.Equal(t, hostsByName, []uint{2, 3, 6})

	userObs := &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)}
	filter = fleet.TeamFilter{User: userObs}

	hostsByName, err = ds.HostIDsByName(context.Background(), filter, []string{"foo.2.local", "foo.1.local", "foo.5.local"})
	require.NoError(t, err)
	assert.Len(t, hostsByName, 0)

	filter.IncludeObserver = true
	hostsByName, err = ds.HostIDsByName(context.Background(), filter, []string{"foo.2.local", "foo.1.local", "foo.5.local"})
	require.NoError(t, err)
	assert.Len(t, hostsByName, 3)

	userTeam1 := &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleAdmin}}}
	filter = fleet.TeamFilter{User: userTeam1}

	hostsByName, err = ds.HostIDsByName(context.Background(), filter, []string{"foo.2.local", "foo.1.local", "foo.5.local"})
	require.NoError(t, err)
	assert.Len(t, hostsByName, 0)

	hostsByName, err = ds.HostIDsByName(context.Background(), filter, []string{"foo.0.local", "foo.1.local", "foo.5.local"})
	require.NoError(t, err)
	require.Len(t, hostsByName, 1)
	assert.Equal(t, hostsByName[0], hosts[0].ID)
}

func testLoadHostByNodeKeyLoadsDisk(t *testing.T, ds *Datastore) {
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
	h, err = ds.LoadHostByNodeKey(context.Background(), "nodekey")
	require.NoError(t, err)
	assert.NotZero(t, h.GigsDiskSpaceAvailable)
	assert.NotZero(t, h.PercentDiskSpaceAvailable)
}

func testLoadHostByNodeKeyUsesStmt(t *testing.T, ds *Datastore) {
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
	_, err = ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   "foobar2",
		NodeKey:         "nodekey2",
		UUID:            "uuid2",
		Hostname:        "foobar2.local",
	})
	require.NoError(t, err)

	err = ds.closeStmts()
	require.NoError(t, err)

	ds.stmtCacheMu.Lock()
	require.Len(t, ds.stmtCache, 0)
	ds.stmtCacheMu.Unlock()

	h, err := ds.LoadHostByNodeKey(context.Background(), "nodekey")
	require.NoError(t, err)
	require.Equal(t, "foobar.local", h.Hostname)

	ds.stmtCacheMu.Lock()
	require.Len(t, ds.stmtCache, 1)
	ds.stmtCacheMu.Unlock()

	h, err = ds.LoadHostByNodeKey(context.Background(), "nodekey")
	require.NoError(t, err)
	require.Equal(t, "foobar.local", h.Hostname)

	ds.stmtCacheMu.Lock()
	require.Len(t, ds.stmtCache, 1)
	ds.stmtCacheMu.Unlock()

	h, err = ds.LoadHostByNodeKey(context.Background(), "nodekey2")
	require.NoError(t, err)
	require.Equal(t, "foobar2.local", h.Hostname)
}

func testHostsAdditional(t *testing.T, ds *Datastore) {
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

	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, "foobar.local", h.Hostname)
	assert.Nil(t, h.Additional)

	// Additional not yet set
	h, err = ds.Host(context.Background(), h.ID, false)
	require.NoError(t, err)
	assert.Nil(t, h.Additional)

	// Add additional
	additional := json.RawMessage(`{"additional": "result"}`)
	h.Additional = &additional
	require.NoError(t, saveHostAdditionalDB(context.Background(), ds.writer, h.ID, h.Additional))

	// Additional should not be loaded for HostLite
	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, "foobar.local", h.Hostname)
	assert.Nil(t, h.Additional)

	h, err = ds.Host(context.Background(), h.ID, false)
	require.NoError(t, err)
	assert.Equal(t, &additional, h.Additional)

	// Update besides additional. Additional should be unchanged.
	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	h.Hostname = "baz.local"
	err = ds.SaveHost(context.Background(), h)
	require.NoError(t, err)

	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, "baz.local", h.Hostname)
	assert.Nil(t, h.Additional)

	h, err = ds.Host(context.Background(), h.ID, false)
	require.NoError(t, err)
	assert.Equal(t, &additional, h.Additional)

	// Update additional
	additional = json.RawMessage(`{"other": "additional"}`)
	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	h.Additional = &additional
	err = saveHostAdditionalDB(context.Background(), ds.writer, h.ID, h.Additional)
	require.NoError(t, err)

	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, "baz.local", h.Hostname)
	assert.Nil(t, h.Additional)

	h, err = ds.Host(context.Background(), h.ID, false)
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
		host, err := ds.Host(context.Background(), uint(i), false)
		require.NoError(t, err)
		assert.Nil(t, host.TeamID)
	}

	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{1, 2, 3}))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team2.ID, []uint{3, 4, 5}))

	for i := 1; i <= 10; i++ {
		host, err := ds.Host(context.Background(), uint(i), false)
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
		host, err := ds.Host(context.Background(), uint(i), false)
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

	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	assert.Len(t, host.Users, 0)

	u1 := fleet.HostUser{
		Uid:       42,
		Username:  "user",
		Type:      "aaa",
		GroupName: "group",
		Shell:     "shell",
	}
	u2 := fleet.HostUser{
		Uid:       43,
		Username:  "user2",
		Type:      "aaa",
		GroupName: "group",
		Shell:     "shell",
	}
	host.Users = []fleet.HostUser{u1, u2}
	host.Modified = true

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	require.Len(t, host.Users, 2)
	test.ElementsMatchSkipID(t, host.Users, []fleet.HostUser{u1, u2})

	// remove u1 user
	host.Users = []fleet.HostUser{u2}
	host.Modified = true

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	require.Len(t, host.Users, 1)
	assert.Equal(t, host.Users[0].Uid, u2.Uid)

	// readd u1 but with a different shell
	u1.Shell = "/some/new/shell"
	host.Users = []fleet.HostUser{u1, u2}
	host.Modified = true

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID, false)
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

	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	assert.Len(t, host.Users, 0)

	u1 := fleet.HostUser{
		Username:  "user",
		Type:      "aaa",
		GroupName: "group",
		Shell:     "shell",
	}
	u2 := fleet.HostUser{
		Username:  "user2",
		Type:      "aaa",
		GroupName: "group",
		Shell:     "shell",
	}
	host.Users = []fleet.HostUser{u1, u2}
	host.Modified = true

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	require.Len(t, host.Users, 2)
	test.ElementsMatchSkipID(t, host.Users, []fleet.HostUser{u1, u2})

	// remove u1 user
	host.Users = []fleet.HostUser{u2}
	host.Modified = true

	err = ds.SaveHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID, false)
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
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
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
	p, err := ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		QueryID: &q.ID,
	})
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
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h1, map[uint]*bool{1: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{1: ptr.Bool(false)}, time.Now(), false))

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

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages", BundleIdentifier: "com.some.identifier"},
	}
	host1 := hosts[0]
	host2 := hosts[1]
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host1.ID, software))
	require.NoError(t, ds.UpdateHostSoftware(context.Background(), host2.ID, software))

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{SoftwareIDFilter: &host1.Software[0].ID}, 2)
	require.Len(t, hosts, 2)
}

func testHostsListFailingPolicies(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
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
	p, err := ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		QueryID: &q.ID,
	})
	require.NoError(t, err)
	p2, err := ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		QueryID: &q2.ID,
	})
	require.NoError(t, err)

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 10)
	require.Len(t, hosts, 10)

	h1 := hosts[0]
	h2 := hosts[1]

	assert.Zero(t, h1.HostIssues.FailingPoliciesCount)
	assert.Zero(t, h1.HostIssues.TotalIssuesCount)
	assert.Zero(t, h2.HostIssues.FailingPoliciesCount)
	assert.Zero(t, h2.HostIssues.TotalIssuesCount)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h1, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now(), false))

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{p.ID: ptr.Bool(false), p2.ID: ptr.Bool(false)}, time.Now(), false))
	checkHostIssues(t, ds, hosts, filter, h2.ID, 2)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{p.ID: ptr.Bool(true), p2.ID: ptr.Bool(false)}, time.Now(), false))
	checkHostIssues(t, ds, hosts, filter, h2.ID, 1)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{p.ID: ptr.Bool(true), p2.ID: ptr.Bool(true)}, time.Now(), false))
	checkHostIssues(t, ds, hosts, filter, h2.ID, 0)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h1, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now(), false))
	checkHostIssues(t, ds, hosts, filter, h1.ID, 1)

	checkHostIssuesWithOpts(t, ds, hosts, filter, h1.ID, fleet.HostListOptions{DisableFailingPolicies: true}, 0)
}

// This doesn't work when running the whole test suite, but helps inspect individual tests
func printReadsInTest(test func(t *testing.T, ds *Datastore)) func(t *testing.T, ds *Datastore) {
	return func(t *testing.T, ds *Datastore) {
		prevRead := getReads(t, ds)
		test(t, ds)
		newRead := getReads(t, ds)
		t.Log("Rows read in test:", newRead-prevRead)
	}
}

func getReads(t *testing.T, ds *Datastore) int {
	rows, err := ds.writer.Query("show engine innodb status")
	require.NoError(t, err)
	defer rows.Close()
	r := 0
	for rows.Next() {
		type_, name, status := "", "", ""
		require.NoError(t, rows.Scan(&type_, &name, &status))
		assert.Equal(t, type_, "InnoDB")
		m := regexp.MustCompile(`Number of rows inserted \d+, updated \d+, deleted \d+, read \d+`)
		rowsStr := m.FindString(status)
		nums := regexp.MustCompile(`\d+`)
		parts := nums.FindAllString(rowsStr, -1)
		require.Len(t, parts, 4)
		read, err := strconv.Atoi(parts[len(parts)-1])
		require.NoError(t, err)
		r = read
		break
	}
	require.NoError(t, rows.Err())
	return r
}

func testHostsReadsLessRows(t *testing.T, ds *Datastore) {
	t.Skip("flaky: https://github.com/fleetdm/fleet/issues/4270")

	user1 := test.NewUser(t, ds, "alice", "alice-123@example.com", true)
	var hosts []*fleet.Host
	for i := 0; i < 10; i++ {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
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
		hosts = append(hosts, h)
	}
	h1 := hosts[0]
	h2 := hosts[1]

	q := test.NewQuery(t, ds, "query1", "select 1", 0, true)
	p, err := ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		QueryID: &q.ID,
	})
	require.NoError(t, err)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h1, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now(), false))

	prevRead := getReads(t, ds)
	h1WithExtras, err := ds.Host(context.Background(), h1.ID, false)
	require.NoError(t, err)
	newRead := getReads(t, ds)
	withExtraRowReads := newRead - prevRead

	prevRead = getReads(t, ds)
	h1WithoutExtras, err := ds.Host(context.Background(), h1.ID, true)
	require.NoError(t, err)
	newRead = getReads(t, ds)
	withoutExtraRowReads := newRead - prevRead

	t.Log("withExtraRowReads", withExtraRowReads)
	t.Log("withoutExtraRowReads", withoutExtraRowReads)
	assert.Less(t, withoutExtraRowReads, withExtraRowReads)

	assert.Equal(t, h1WithExtras.ID, h1WithoutExtras.ID)
	assert.Equal(t, h1WithExtras.OsqueryHostID, h1WithoutExtras.OsqueryHostID)
	assert.Equal(t, h1WithExtras.NodeKey, h1WithoutExtras.NodeKey)
	assert.Equal(t, h1WithExtras.UUID, h1WithoutExtras.UUID)
	assert.Equal(t, h1WithExtras.Hostname, h1WithoutExtras.Hostname)
}

func checkHostIssues(t *testing.T, ds *Datastore, hosts []*fleet.Host, filter fleet.TeamFilter, hid uint, expected int) {
	checkHostIssuesWithOpts(t, ds, hosts, filter, hid, fleet.HostListOptions{}, expected)
}

func checkHostIssuesWithOpts(t *testing.T, ds *Datastore, hosts []*fleet.Host, filter fleet.TeamFilter, hid uint, opts fleet.HostListOptions, expected int) {
	hosts = listHostsCheckCount(t, ds, filter, opts, 10)
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

	if opts.DisableFailingPolicies {
		return
	}

	hostById, err := ds.Host(context.Background(), hid, false)
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
			host1, err := ds.Host(context.Background(), host1.ID, false)
			if err != nil {
				errCh <- err
				return
			}

			u1 := fleet.HostUser{
				Uid:       42,
				Username:  "user",
				Type:      "aaa",
				GroupName: "group",
				Shell:     "shell",
			}
			u2 := fleet.HostUser{
				Uid:       43,
				Username:  "user2",
				Type:      "aaa",
				GroupName: "group",
				Shell:     "shell",
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
			host2, err := ds.Host(context.Background(), host2.ID, false)
			if err != nil {
				errCh <- err
				return
			}

			u1 := fleet.HostUser{
				Uid:       99,
				Username:  "user",
				Type:      "aaa",
				GroupName: "group",
				Shell:     "shell",
			}
			u2 := fleet.HostUser{
				Uid:       98,
				Username:  "user2",
				Type:      "aaa",
				GroupName: "group",
				Shell:     "shell",
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

func testHostsAllPackStats(t *testing.T, ds *Datastore) {
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
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	// Create global pack (and one scheduled query in it).
	test.AddAllHostsLabel(t, ds) // the global pack needs the "All Hosts" label.
	labels, err := ds.ListLabels(context.Background(), fleet.TeamFilter{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, labels, 1)
	globalPack, err := ds.EnsureGlobalPack(context.Background())
	require.NoError(t, err)
	globalQuery := test.NewQuery(t, ds, "global-time", "select * from time", 0, true)
	globalSQuery := test.NewScheduledQuery(t, ds, globalPack.ID, globalQuery.ID, 30, true, true, "time-scheduled-global")
	err = ds.AsyncBatchInsertLabelMembership(context.Background(), [][2]uint{{labels[0].ID, host.ID}})
	require.NoError(t, err)

	// Create a team and its pack (and one scheduled query in it).
	team, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID}))
	teamPack, err := ds.EnsureTeamPack(context.Background(), team.ID)
	require.NoError(t, err)
	teamQuery := test.NewQuery(t, ds, "team-time", "select * from time", 0, true)
	teamSQuery := test.NewScheduledQuery(t, ds, teamPack.ID, teamQuery.ID, 31, true, true, "time-scheduled-team")

	// Create a "user created" pack (and one scheduled query in it).
	userPack, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test1",
		HostIDs: []uint{host.ID},
	})
	require.NoError(t, err)
	userQuery := test.NewQuery(t, ds, "user-time", "select * from time", 0, true)
	userSQuery := test.NewScheduledQuery(t, ds, userPack.ID, userQuery.ID, 30, true, true, "time-scheduled-user")

	// Even if the scheduled queries didn't run, we get their pack stats (with zero values).
	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	packStats := host.PackStats
	require.Len(t, packStats, 3)
	sort.Sort(packStatsSlice(packStats))
	for _, tc := range []struct {
		expectedPack   *fleet.Pack
		expectedQuery  *fleet.Query
		expectedSQuery *fleet.ScheduledQuery
		packStats      fleet.PackStats
	}{
		{
			expectedPack:   globalPack,
			expectedQuery:  globalQuery,
			expectedSQuery: globalSQuery,
			packStats:      packStats[0],
		},
		{
			expectedPack:   teamPack,
			expectedQuery:  teamQuery,
			expectedSQuery: teamSQuery,
			packStats:      packStats[1],
		},
		{
			expectedPack:   userPack,
			expectedQuery:  userQuery,
			expectedSQuery: userSQuery,
			packStats:      packStats[2],
		},
	} {
		require.Equal(t, tc.expectedPack.ID, tc.packStats.PackID)
		require.Equal(t, tc.expectedPack.Name, tc.packStats.PackName)
		require.Len(t, tc.packStats.QueryStats, 1)
		require.False(t, tc.packStats.QueryStats[0].Denylisted)
		require.Empty(t, tc.packStats.QueryStats[0].Description) // because test.NewQuery doesn't set a description.
		require.NotZero(t, tc.packStats.QueryStats[0].Interval)
		require.Equal(t, tc.packStats.PackID, tc.packStats.QueryStats[0].PackID)
		require.Equal(t, tc.packStats.PackName, tc.packStats.QueryStats[0].PackName)
		require.Equal(t, tc.expectedQuery.Name, tc.packStats.QueryStats[0].QueryName)
		require.Equal(t, tc.expectedSQuery.ID, tc.packStats.QueryStats[0].ScheduledQueryID)
		require.Equal(t, tc.expectedSQuery.Name, tc.packStats.QueryStats[0].ScheduledQueryName)

		require.Zero(t, tc.packStats.QueryStats[0].AverageMemory)
		require.Zero(t, tc.packStats.QueryStats[0].Executions)
		require.Equal(t, expLastExec, tc.packStats.QueryStats[0].LastExecuted)
		require.Zero(t, tc.packStats.QueryStats[0].OutputSize)
		require.Zero(t, tc.packStats.QueryStats[0].SystemTime)
		require.Zero(t, tc.packStats.QueryStats[0].UserTime)
		require.Zero(t, tc.packStats.QueryStats[0].WallTime)
	}

	globalPackSQueryStats := []fleet.ScheduledQueryStats{{
		ScheduledQueryName: globalSQuery.Name,
		ScheduledQueryID:   globalSQuery.ID,
		QueryName:          globalQuery.Name,
		PackName:           globalPack.Name,
		PackID:             globalPack.ID,
		AverageMemory:      8000,
		Denylisted:         false,
		Executions:         164,
		Interval:           30,
		LastExecuted:       time.Unix(1620325191, 0).UTC(),
		OutputSize:         1337,
		SystemTime:         150,
		UserTime:           180,
		WallTime:           0,
	}}
	teamPackSQueryStats := []fleet.ScheduledQueryStats{{
		ScheduledQueryName: teamSQuery.Name,
		ScheduledQueryID:   teamSQuery.ID,
		QueryName:          teamQuery.Name,
		PackName:           teamPack.Name,
		PackID:             teamPack.ID,
		AverageMemory:      8001,
		Denylisted:         true,
		Executions:         165,
		Interval:           31,
		LastExecuted:       time.Unix(1620325190, 0).UTC(),
		OutputSize:         1338,
		SystemTime:         151,
		UserTime:           181,
		WallTime:           1,
	}}
	userPackSQueryStats := []fleet.ScheduledQueryStats{{
		ScheduledQueryName: userSQuery.Name,
		ScheduledQueryID:   userSQuery.ID,
		QueryName:          userQuery.Name,
		PackName:           userPack.Name,
		PackID:             userPack.ID,
		AverageMemory:      0,
		Denylisted:         false,
		Executions:         0,
		Interval:           30,
		LastExecuted:       expLastExec,
		OutputSize:         0,
		SystemTime:         0,
		UserTime:           0,
		WallTime:           0,
	}}
	// Reload the host and set the scheduled queries stats.
	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	host.PackStats = []fleet.PackStats{
		{PackID: globalPack.ID, PackName: globalPack.Name, QueryStats: globalPackSQueryStats},
		{PackID: teamPack.ID, PackName: teamPack.Name, QueryStats: teamPackSQueryStats},
	}
	require.NoError(t, ds.SaveHost(context.Background(), host))

	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	packStats = host.PackStats
	require.Len(t, packStats, 3)
	sort.Sort(packStatsSlice(packStats))

	require.ElementsMatch(t, packStats[0].QueryStats, globalPackSQueryStats)
	require.ElementsMatch(t, packStats[1].QueryStats, teamPackSQueryStats)
	require.ElementsMatch(t, packStats[2].QueryStats, userPackSQueryStats)
}

// See #2965.
func testHostsPackStatsMultipleHosts(t *testing.T, ds *Datastore) {
	osqueryHostID1, _ := server.GenerateRandomText(10)
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
		Platform:        "darwin",
		OsqueryHostID:   osqueryHostID1,
	})
	require.NoError(t, err)
	require.NotNil(t, host1)
	osqueryHostID2, _ := server.GenerateRandomText(10)
	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "bar.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "darwin",
		OsqueryHostID:   osqueryHostID2,
	})
	require.NoError(t, err)
	require.NotNil(t, host2)

	// Create global pack (and one scheduled query in it).
	test.AddAllHostsLabel(t, ds) // the global pack needs the "All Hosts" label.
	labels, err := ds.ListLabels(context.Background(), fleet.TeamFilter{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, labels, 1)
	globalPack, err := ds.EnsureGlobalPack(context.Background())
	require.NoError(t, err)
	globalQuery := test.NewQuery(t, ds, "global-time", "select * from time", 0, true)
	globalSQuery := test.NewScheduledQuery(t, ds, globalPack.ID, globalQuery.ID, 30, true, true, "time-scheduled-global")
	err = ds.AsyncBatchInsertLabelMembership(context.Background(), [][2]uint{
		{labels[0].ID, host1.ID},
		{labels[0].ID, host2.ID},
	})
	require.NoError(t, err)

	globalStatsHost1 := []fleet.ScheduledQueryStats{{
		ScheduledQueryName: globalSQuery.Name,
		ScheduledQueryID:   globalSQuery.ID,
		QueryName:          globalQuery.Name,
		PackName:           globalPack.Name,
		PackID:             globalPack.ID,
		AverageMemory:      8000,
		Denylisted:         false,
		Executions:         164,
		Interval:           30,
		LastExecuted:       time.Unix(1620325191, 0).UTC(),
		OutputSize:         1337,
		SystemTime:         150,
		UserTime:           180,
		WallTime:           0,
	}}
	globalStatsHost2 := []fleet.ScheduledQueryStats{{
		ScheduledQueryName: globalSQuery.Name,
		ScheduledQueryID:   globalSQuery.ID,
		QueryName:          globalQuery.Name,
		PackName:           globalPack.Name,
		PackID:             globalPack.ID,
		AverageMemory:      9000,
		Denylisted:         false,
		Executions:         165,
		Interval:           30,
		LastExecuted:       time.Unix(1620325192, 0).UTC(),
		OutputSize:         1338,
		SystemTime:         151,
		UserTime:           181,
		WallTime:           1,
	}}

	// Reload the hosts and set the scheduled queries stats.
	for _, tc := range []struct {
		hostID      uint
		globalStats []fleet.ScheduledQueryStats
	}{
		{
			hostID:      host1.ID,
			globalStats: globalStatsHost1,
		},
		{
			hostID:      host2.ID,
			globalStats: globalStatsHost2,
		},
	} {
		host, err := ds.Host(context.Background(), tc.hostID, false)
		require.NoError(t, err)
		host.PackStats = []fleet.PackStats{
			{PackID: globalPack.ID, PackName: globalPack.Name, QueryStats: tc.globalStats},
		}
		require.NoError(t, ds.SaveHost(context.Background(), host))
	}

	// Both hosts should see just one stats entry on the one pack.
	for _, tc := range []struct {
		host          *fleet.Host
		expectedStats []fleet.ScheduledQueryStats
	}{
		{
			host:          host1,
			expectedStats: globalStatsHost1,
		},
		{
			host:          host2,
			expectedStats: globalStatsHost2,
		},
	} {
		host, err := ds.Host(context.Background(), tc.host.ID, false)
		require.NoError(t, err)
		packStats := host.PackStats
		require.Len(t, packStats, 1)
		require.Len(t, packStats[0].QueryStats, 1)
		require.ElementsMatch(t, packStats[0].QueryStats, tc.expectedStats)
	}
}

// See #2964.
func testHostsPackStatsForPlatform(t *testing.T, ds *Datastore) {
	osqueryHostID1, _ := server.GenerateRandomText(10)
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
		Platform:        "darwin",
		OsqueryHostID:   osqueryHostID1,
	})
	require.NoError(t, err)
	require.NotNil(t, host1)
	osqueryHostID2, _ := server.GenerateRandomText(10)
	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "foo.local.2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "rhel",
		OsqueryHostID:   osqueryHostID2,
	})
	require.NoError(t, err)
	require.NotNil(t, host2)

	// Create global pack (and one scheduled query in it).
	test.AddAllHostsLabel(t, ds) // the global pack needs the "All Hosts" label.
	labels, err := ds.ListLabels(context.Background(), fleet.TeamFilter{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, labels, 1)
	globalPack, err := ds.EnsureGlobalPack(context.Background())
	require.NoError(t, err)
	globalQuery := test.NewQuery(t, ds, "global-time", "select * from time", 0, true)
	globalSQuery1, err := ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "Scheduled Query For Linux only",
		PackID:   globalPack.ID,
		QueryID:  globalQuery.ID,
		Interval: 30,
		Snapshot: ptr.Bool(true),
		Removed:  ptr.Bool(true),
		Platform: ptr.String("linux"),
	})
	require.NoError(t, err)
	require.NotZero(t, globalSQuery1.ID)
	globalSQuery2, err := ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "Scheduled Query For Darwin only",
		PackID:   globalPack.ID,
		QueryID:  globalQuery.ID,
		Interval: 30,
		Snapshot: ptr.Bool(true),
		Removed:  ptr.Bool(true),
		Platform: ptr.String("darwin"),
	})
	require.NoError(t, err)
	require.NotZero(t, globalSQuery2.ID)
	globalSQuery3, err := ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "Scheduled Query For Darwin and Linux",
		PackID:   globalPack.ID,
		QueryID:  globalQuery.ID,
		Interval: 30,
		Snapshot: ptr.Bool(true),
		Removed:  ptr.Bool(true),
		Platform: ptr.String("darwin,linux"),
	})
	require.NoError(t, err)
	require.NotZero(t, globalSQuery3.ID)
	globalSQuery4, err := ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "Scheduled Query For All Platforms",
		PackID:   globalPack.ID,
		QueryID:  globalQuery.ID,
		Interval: 30,
		Snapshot: ptr.Bool(true),
		Removed:  ptr.Bool(true),
		Platform: ptr.String(""),
	})
	require.NoError(t, err)
	require.NotZero(t, globalSQuery4.ID)
	globalSQuery5, err := ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "Scheduled Query For All Platforms v2",
		PackID:   globalPack.ID,
		QueryID:  globalQuery.ID,
		Interval: 30,
		Snapshot: ptr.Bool(true),
		Removed:  ptr.Bool(true),
		Platform: nil,
	})
	require.NoError(t, err)
	require.NotZero(t, globalSQuery5.ID)

	err = ds.AsyncBatchInsertLabelMembership(context.Background(), [][2]uint{
		{labels[0].ID, host1.ID},
		{labels[0].ID, host2.ID},
	})
	require.NoError(t, err)

	globalStats := []fleet.ScheduledQueryStats{
		{
			ScheduledQueryName: globalSQuery2.Name,
			ScheduledQueryID:   globalSQuery2.ID,
			QueryName:          globalQuery.Name,
			PackName:           globalPack.Name,
			PackID:             globalPack.ID,
			AverageMemory:      8001,
			Denylisted:         false,
			Executions:         165,
			Interval:           30,
			LastExecuted:       time.Unix(1620325192, 0).UTC(),
			OutputSize:         1338,
			SystemTime:         151,
			UserTime:           181,
			WallTime:           1,
		},
		{
			ScheduledQueryName: globalSQuery3.Name,
			ScheduledQueryID:   globalSQuery3.ID,
			QueryName:          globalQuery.Name,
			PackName:           globalPack.Name,
			PackID:             globalPack.ID,
			AverageMemory:      8002,
			Denylisted:         false,
			Executions:         166,
			Interval:           30,
			LastExecuted:       time.Unix(1620325193, 0).UTC(),
			OutputSize:         1339,
			SystemTime:         152,
			UserTime:           182,
			WallTime:           2,
		},
		{
			ScheduledQueryName: globalSQuery4.Name,
			ScheduledQueryID:   globalSQuery4.ID,
			QueryName:          globalQuery.Name,
			PackName:           globalPack.Name,
			PackID:             globalPack.ID,
			AverageMemory:      8003,
			Denylisted:         false,
			Executions:         167,
			Interval:           30,
			LastExecuted:       time.Unix(1620325194, 0).UTC(),
			OutputSize:         1340,
			SystemTime:         153,
			UserTime:           183,
			WallTime:           3,
		},
		{
			ScheduledQueryName: globalSQuery5.Name,
			ScheduledQueryID:   globalSQuery5.ID,
			QueryName:          globalQuery.Name,
			PackName:           globalPack.Name,
			PackID:             globalPack.ID,
			AverageMemory:      8003,
			Denylisted:         false,
			Executions:         167,
			Interval:           30,
			LastExecuted:       time.Unix(1620325194, 0).UTC(),
			OutputSize:         1340,
			SystemTime:         153,
			UserTime:           183,
			WallTime:           3,
		},
	}

	// Reload the host and set the scheduled queries stats for the scheduled queries that apply.
	// Plus we set schedule query stats for a query that does not apply (globalSQuery1)
	// (This could happen if the target platform of a schedule query is changed after creation.)
	stats := make([]fleet.ScheduledQueryStats, len(globalStats))
	for i := range globalStats {
		stats[i] = globalStats[i]
	}
	stats = append(stats, fleet.ScheduledQueryStats{
		ScheduledQueryName: globalSQuery1.Name,
		ScheduledQueryID:   globalSQuery1.ID,
		QueryName:          globalQuery.Name,
		PackName:           globalPack.Name,
		PackID:             globalPack.ID,
		AverageMemory:      8003,
		Denylisted:         false,
		Executions:         167,
		Interval:           30,
		LastExecuted:       time.Unix(1620325194, 0).UTC(),
		OutputSize:         1340,
		SystemTime:         153,
		UserTime:           183,
		WallTime:           3,
	})
	host, err := ds.Host(context.Background(), host1.ID, false)
	require.NoError(t, err)
	host.PackStats = []fleet.PackStats{
		{PackID: globalPack.ID, PackName: globalPack.Name, QueryStats: stats},
	}
	require.NoError(t, ds.SaveHost(context.Background(), host))

	// host should only return scheduled query stats only for the scheduled queries
	// scheduled to run on "darwin".
	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	packStats := host.PackStats
	require.Len(t, packStats, 1)
	require.Len(t, packStats[0].QueryStats, 4)
	sort.Slice(packStats[0].QueryStats, func(i, j int) bool {
		return packStats[0].QueryStats[i].ScheduledQueryID < packStats[0].QueryStats[j].ScheduledQueryID
	})
	sort.Slice(globalStats, func(i, j int) bool {
		return globalStats[i].ScheduledQueryID < globalStats[j].ScheduledQueryID
	})
	require.ElementsMatch(t, packStats[0].QueryStats, globalStats)

	// host2 should only return scheduled query stats only for the scheduled queries
	// scheduled to run on "linux"
	host2, err = ds.Host(context.Background(), host2.ID, false)
	require.NoError(t, err)
	packStats2 := host2.PackStats
	require.Len(t, packStats2, 1)
	require.Len(t, packStats2[0].QueryStats, 4)
	zeroStats := []fleet.ScheduledQueryStats{
		{
			ScheduledQueryName: globalSQuery1.Name,
			ScheduledQueryID:   globalSQuery1.ID,
			QueryName:          globalQuery.Name,
			PackName:           globalPack.Name,
			PackID:             globalPack.ID,
			AverageMemory:      0,
			Denylisted:         false,
			Executions:         0,
			Interval:           30,
			LastExecuted:       expLastExec,
			OutputSize:         0,
			SystemTime:         0,
			UserTime:           0,
			WallTime:           0,
		},
		{
			ScheduledQueryName: globalSQuery3.Name,
			ScheduledQueryID:   globalSQuery3.ID,
			QueryName:          globalQuery.Name,
			PackName:           globalPack.Name,
			PackID:             globalPack.ID,
			AverageMemory:      0,
			Denylisted:         false,
			Executions:         0,
			Interval:           30,
			LastExecuted:       expLastExec,
			OutputSize:         0,
			SystemTime:         0,
			UserTime:           0,
			WallTime:           0,
		},
		{
			ScheduledQueryName: globalSQuery4.Name,
			ScheduledQueryID:   globalSQuery4.ID,
			QueryName:          globalQuery.Name,
			PackName:           globalPack.Name,
			PackID:             globalPack.ID,
			AverageMemory:      0,
			Denylisted:         false,
			Executions:         0,
			Interval:           30,
			LastExecuted:       expLastExec,
			OutputSize:         0,
			SystemTime:         0,
			UserTime:           0,
			WallTime:           0,
		},
		{
			ScheduledQueryName: globalSQuery5.Name,
			ScheduledQueryID:   globalSQuery5.ID,
			QueryName:          globalQuery.Name,
			PackName:           globalPack.Name,
			PackID:             globalPack.ID,
			AverageMemory:      0,
			Denylisted:         false,
			Executions:         0,
			Interval:           30,
			LastExecuted:       expLastExec,
			OutputSize:         0,
			SystemTime:         0,
			UserTime:           0,
			WallTime:           0,
		},
	}
	require.ElementsMatch(t, packStats2[0].QueryStats, zeroStats)
}

// testHostsNoSeenTime tests all changes around the seen_time issue #3095.
func testHostsNoSeenTime(t *testing.T, ds *Datastore) {
	h1, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              1,
		OsqueryHostID:   "1",
		NodeKey:         "1",
		Platform:        "linux",
		Hostname:        "host1",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	removeHostSeenTimes := func(hostID uint) {
		result, err := ds.writer.Exec("DELETE FROM host_seen_times WHERE host_id = ?", hostID)
		require.NoError(t, err)
		rowsAffected, err := result.RowsAffected()
		require.NoError(t, err)
		require.EqualValues(t, 1, rowsAffected)
	}
	removeHostSeenTimes(h1.ID)

	h1, err = ds.Host(context.Background(), h1.ID, true)
	require.NoError(t, err)
	require.Equal(t, h1.CreatedAt, h1.SeenTime)

	teamFilter := fleet.TeamFilter{User: test.UserAdmin}
	hosts, err := ds.ListHosts(context.Background(), teamFilter, fleet.HostListOptions{})
	require.NoError(t, err)
	hostsLen := len(hosts)
	require.Equal(t, hostsLen, 1)
	var foundHost *fleet.Host
	for _, host := range hosts {
		if host.ID == h1.ID {
			foundHost = host
			break
		}
	}
	require.NotNil(t, foundHost)
	require.Equal(t, foundHost.CreatedAt, foundHost.SeenTime)
	hostCount, err := ds.CountHosts(context.Background(), teamFilter, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Equal(t, hostsLen, hostCount)

	labelID := uint(1)
	l1 := &fleet.LabelSpec{
		ID:    labelID,
		Name:  "label foo",
		Query: "query1",
	}
	err = ds.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{l1})
	require.NoError(t, err)
	err = ds.RecordLabelQueryExecutions(context.Background(), h1, map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)
	listHostsInLabelCheckCount(t, ds, fleet.TeamFilter{
		User: test.UserAdmin,
	}, labelID, fleet.HostListOptions{}, 1)

	mockClock := clock.NewMockClock()
	summary, err := ds.GenerateHostStatusStatistics(context.Background(), teamFilter, mockClock.Now(), nil)
	assert.NoError(t, err)
	assert.Nil(t, summary.TeamID)
	assert.Equal(t, uint(1), summary.TotalsHostsCount)
	assert.Equal(t, uint(1), summary.OnlineCount)
	assert.Equal(t, uint(0), summary.OfflineCount)
	assert.Equal(t, uint(0), summary.MIACount)
	assert.Equal(t, uint(1), summary.NewCount)

	var count []int
	err = ds.writer.Select(&count, "SELECT COUNT(*) FROM host_seen_times")
	require.NoError(t, err)
	require.Len(t, count, 1)
	require.Zero(t, count[0])

	// Enroll existing host.
	_, err = ds.EnrollHost(context.Background(), "1", "1", nil, 0)
	require.NoError(t, err)

	var seenTime1 []time.Time
	err = ds.writer.Select(&seenTime1, "SELECT seen_time FROM host_seen_times WHERE host_id = ?", h1.ID)
	require.NoError(t, err)
	require.Len(t, seenTime1, 1)
	require.NotZero(t, seenTime1[0])

	time.Sleep(1 * time.Second)

	// Enroll again to trigger an update of host_seen_times.
	_, err = ds.EnrollHost(context.Background(), "1", "1", nil, 0)
	require.NoError(t, err)

	var seenTime2 []time.Time
	err = ds.writer.Select(&seenTime2, "SELECT seen_time FROM host_seen_times WHERE host_id = ?", h1.ID)
	require.NoError(t, err)
	require.Len(t, seenTime2, 1)
	require.NotZero(t, seenTime2[0])

	require.True(t, seenTime2[0].After(seenTime1[0]), "%s vs. %s", seenTime1[0], seenTime2[0])

	removeHostSeenTimes(h1.ID)

	h2, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              2,
		OsqueryHostID:   "2",
		NodeKey:         "2",
		Platform:        "windows",
		Hostname:        "host2",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	t1 := time.Now().UTC()
	// h1 has no host_seen_times entry, h2 does.
	err = ds.MarkHostsSeen(context.Background(), []uint{h1.ID, h2.ID}, t1)
	require.NoError(t, err)

	// Reload hosts.
	h1, err = ds.Host(context.Background(), h1.ID, true)
	require.NoError(t, err)
	h2, err = ds.Host(context.Background(), h2.ID, true)
	require.NoError(t, err)

	// Equal doesn't work, it looks like a time.Time scanned from
	// the database is different from the original in some fields
	// (wall and ext).
	require.WithinDuration(t, t1, h1.SeenTime, time.Second)
	require.WithinDuration(t, t1, h2.SeenTime, time.Second)

	removeHostSeenTimes(h1.ID)

	foundHosts, err := ds.SearchHosts(context.Background(), teamFilter, "")
	require.NoError(t, err)
	require.Len(t, foundHosts, 2)
	// SearchHosts orders by seen time.
	require.Equal(t, h2.ID, foundHosts[0].ID)
	require.WithinDuration(t, t1, foundHosts[0].SeenTime, time.Second)
	require.Equal(t, h1.ID, foundHosts[1].ID)
	require.Equal(t, foundHosts[1].SeenTime, foundHosts[1].CreatedAt)

	total, unseen, err := ds.TotalAndUnseenHostsSince(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, total, 2)
	require.Equal(t, unseen, 0)

	h3, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              3,
		OsqueryHostID:   "3",
		NodeKey:         "3",
		Platform:        "darwin",
		Hostname:        "host3",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	removeHostSeenTimes(h3.ID)

	err = ds.CleanupExpiredHosts(context.Background())
	require.NoError(t, err)

	hosts, err = ds.ListHosts(context.Background(), teamFilter, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 3)

	err = ds.RecordLabelQueryExecutions(context.Background(), h2, map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)
	err = ds.RecordLabelQueryExecutions(context.Background(), h3, map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)
	metrics, err := ds.CountHostsInTargets(context.Background(), teamFilter, fleet.HostTargets{
		LabelIDs: []uint{l1.ID},
	}, mockClock.Now())
	require.NoError(t, err)
	assert.Equal(t, uint(3), metrics.TotalHosts)
	assert.Equal(t, uint(0), metrics.OfflineHosts)
	assert.Equal(t, uint(3), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)
}

func testHostsListHostDeviceMapping(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	h, err := ds.NewHost(ctx, &fleet.Host{
		ID:              1,
		OsqueryHostID:   "1",
		NodeKey:         "1",
		Platform:        "linux",
		Hostname:        "host1",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	ds.writer.ExecContext(ctx, `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
		h.ID, "a@b.c", "src1")
	ds.writer.ExecContext(ctx, `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
		h.ID, "b@b.c", "src1")
	ds.writer.ExecContext(ctx, `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
		h.ID, "a@b.c", "src2")

	dms, err := ds.ListHostDeviceMapping(ctx, h.ID+1)
	require.NoError(t, err)
	require.Len(t, dms, 0)

	dms, err = ds.ListHostDeviceMapping(ctx, h.ID)
	require.NoError(t, err)
	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: "src1"},
		{Email: "a@b.c", Source: "src2"},
		{Email: "b@b.c", Source: "src1"},
	})
}

func testHostsReplaceHostDeviceMapping(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	h, err := ds.NewHost(ctx, &fleet.Host{
		ID:              1,
		OsqueryHostID:   "1",
		NodeKey:         "1",
		Platform:        "linux",
		Hostname:        "host1",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, nil)
	require.NoError(t, err)

	dms, err := ds.ListHostDeviceMapping(ctx, h.ID)
	require.NoError(t, err)
	require.Len(t, dms, 0)

	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, []*fleet.HostDeviceMapping{
		{HostID: h.ID, Email: "a@b.c", Source: "src1"},
		{HostID: h.ID + 1, Email: "a@b.c", Source: "src1"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("found %d", h.ID+1))

	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, []*fleet.HostDeviceMapping{
		{HostID: h.ID, Email: "a@b.c", Source: "src1"},
		{HostID: h.ID, Email: "b@b.c", Source: "src1"},
		{HostID: h.ID, Email: "c@b.c", Source: "src2"},
	})
	require.NoError(t, err)

	dms, err = ds.ListHostDeviceMapping(ctx, h.ID)
	require.NoError(t, err)
	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: "src1"},
		{Email: "b@b.c", Source: "src1"},
		{Email: "c@b.c", Source: "src2"},
	})

	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, []*fleet.HostDeviceMapping{
		{HostID: h.ID, Email: "a@b.c", Source: "src1"},
		{HostID: h.ID, Email: "d@b.c", Source: "src2"},
	})
	require.NoError(t, err)

	dms, err = ds.ListHostDeviceMapping(ctx, h.ID)
	require.NoError(t, err)
	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: "src1"},
		{Email: "d@b.c", Source: "src2"},
	})

	// delete only
	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, nil)
	require.NoError(t, err)

	dms, err = ds.ListHostDeviceMapping(ctx, h.ID)
	require.NoError(t, err)
	assertHostDeviceMapping(t, dms, nil)
}

func assertHostDeviceMapping(t *testing.T, got, want []*fleet.HostDeviceMapping) {
	t.Helper()

	// only the email and source are validated
	require.Len(t, got, len(want))

	for i, g := range got {
		w := want[i]
		g.ID, g.HostID = 0, 0
		assert.Equal(t, w, g, "index %d", i)
	}
}

func testHostMDMAndMunki(t *testing.T, ds *Datastore) {
	_, err := ds.GetMunkiVersion(context.Background(), 123)
	require.True(t, fleet.IsNotFound(err))

	require.NoError(t, ds.SetOrUpdateMunkiVersion(context.Background(), 123, "1.2.3"))
	require.NoError(t, ds.SetOrUpdateMunkiVersion(context.Background(), 999, "9.0"))
	require.NoError(t, ds.SetOrUpdateMunkiVersion(context.Background(), 123, "1.3.0"))

	version, err := ds.GetMunkiVersion(context.Background(), 123)
	require.NoError(t, err)
	require.Equal(t, "1.3.0", version)

	version, err = ds.GetMunkiVersion(context.Background(), 999)
	require.NoError(t, err)
	require.Equal(t, "9.0", version)

	// simulate uninstall
	require.NoError(t, ds.SetOrUpdateMunkiVersion(context.Background(), 123, ""))

	_, err = ds.GetMunkiVersion(context.Background(), 123)
	require.True(t, fleet.IsNotFound(err))

	_, _, _, err = ds.GetMDM(context.Background(), 432)
	require.True(t, fleet.IsNotFound(err), err)

	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 432, true, "url", false))

	enrolled, serverURL, installedFromDep, err := ds.GetMDM(context.Background(), 432)
	require.NoError(t, err)
	assert.True(t, enrolled)
	assert.Equal(t, "url", serverURL)
	assert.False(t, installedFromDep)

	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 455, true, "url2", true))
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 432, false, "url3", true))

	enrolled, serverURL, installedFromDep, err = ds.GetMDM(context.Background(), 432)
	require.NoError(t, err)
	assert.False(t, enrolled)
	assert.Equal(t, "url3", serverURL)
	assert.True(t, installedFromDep)

	enrolled, serverURL, installedFromDep, err = ds.GetMDM(context.Background(), 455)
	require.NoError(t, err)
	assert.True(t, enrolled)
	assert.Equal(t, "url2", serverURL)
	assert.True(t, installedFromDep)
}

func testAggregatedHostMDMAndMunki(t *testing.T, ds *Datastore) {
	// Make sure things work before data is generated
	versions, updatedAt, err := ds.AggregatedMunkiVersion(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, versions, 0)
	require.Zero(t, updatedAt)
	status, updatedAt, err := ds.AggregatedMDMStatus(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, status)
	require.Zero(t, updatedAt)

	// Make sure generation works when there's no mdm or munki data
	require.NoError(t, ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	// And after generating without any data, it all looks reasonable
	versions, updatedAt, err = ds.AggregatedMunkiVersion(context.Background(), nil)
	firstUpdatedAt := updatedAt

	require.NoError(t, err)
	require.Len(t, versions, 0)
	require.NotZero(t, updatedAt)
	status, updatedAt, err = ds.AggregatedMDMStatus(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, status)
	require.NotZero(t, updatedAt)

	// So now we try with data
	require.NoError(t, ds.SetOrUpdateMunkiVersion(context.Background(), 123, "1.2.3"))
	require.NoError(t, ds.SetOrUpdateMunkiVersion(context.Background(), 999, "9.0"))
	require.NoError(t, ds.SetOrUpdateMunkiVersion(context.Background(), 342, "1.2.3"))

	require.NoError(t, ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	versions, _, err = ds.AggregatedMunkiVersion(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.ElementsMatch(t, versions, []fleet.AggregatedMunkiVersion{
		{
			HostMunkiInfo: fleet.HostMunkiInfo{Version: "1.2.3"},
			HostsCount:    2,
		},
		{
			HostMunkiInfo: fleet.HostMunkiInfo{Version: "9.0"},
			HostsCount:    1,
		},
	})

	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 432, true, "url", false))
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 123, true, "url", false))
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 124, true, "url", false))
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 455, true, "url2", true))
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 999, false, "url3", true))
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 875, false, "url3", true))

	require.NoError(t, ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	status, _, err = ds.AggregatedMDMStatus(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 6, status.HostsCount)
	assert.Equal(t, 2, status.UnenrolledHostsCount)
	assert.Equal(t, 3, status.EnrolledManualHostsCount)
	assert.Equal(t, 1, status.EnrolledAutomatedHostsCount)

	// Team filters
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team1" + t.Name(),
		Description: "desc team1",
	})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team2" + t.Name(),
		Description: "desc team2",
	})
	require.NoError(t, err)

	h1 := test.NewHost(t, ds, "h1"+t.Name(), "192.168.1.10", "1", "1", time.Now())
	h2 := test.NewHost(t, ds, "h2"+t.Name(), "192.168.1.11", "2", "2", time.Now())
	h3 := test.NewHost(t, ds, "h3"+t.Name(), "192.168.1.11", "3", "3", time.Now())

	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{h1.ID}))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team2.ID, []uint{h2.ID}))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{h3.ID}))

	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), h1.ID, true, "url", false))
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), h2.ID, true, "url", false))
	require.NoError(t, ds.SetOrUpdateMunkiVersion(context.Background(), h1.ID, "1.2.3"))
	require.NoError(t, ds.SetOrUpdateMunkiVersion(context.Background(), h2.ID, "1.2.3"))

	// h3 adds it but then removes it
	require.NoError(t, ds.SetOrUpdateMunkiVersion(context.Background(), h3.ID, "1.2.3"))
	require.NoError(t, ds.SetOrUpdateMunkiVersion(context.Background(), h3.ID, ""))

	// Make the updated_at different enough
	time.Sleep(1 * time.Second)
	require.NoError(t, ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	versions, updatedAt, err = ds.AggregatedMunkiVersion(context.Background(), &team1.ID)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.ElementsMatch(t, versions, []fleet.AggregatedMunkiVersion{
		{
			HostMunkiInfo: fleet.HostMunkiInfo{Version: "1.2.3"},
			HostsCount:    1,
		},
	})
	require.True(t, updatedAt.After(firstUpdatedAt))
	status, _, err = ds.AggregatedMDMStatus(context.Background(), &team1.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, status.HostsCount)
	assert.Equal(t, 0, status.UnenrolledHostsCount)
	assert.Equal(t, 1, status.EnrolledManualHostsCount)
	assert.Equal(t, 0, status.EnrolledAutomatedHostsCount)
}

func testHostsLite(t *testing.T, ds *Datastore) {
	_, err := ds.HostLite(context.Background(), 1)
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.True(t, errors.As(err, &nfe))

	now := time.Now()
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:                  1,
		OsqueryHostID:       "foobar",
		NodeKey:             "nodekey",
		Hostname:            "foobar.local",
		UUID:                "uuid",
		Platform:            "darwin",
		DistributedInterval: 60,
		LoggerTLSPeriod:     50,
		ConfigTLSRefresh:    40,
		DetailUpdatedAt:     now,
		LabelUpdatedAt:      now,
		LastEnrolledAt:      now,
		PolicyUpdatedAt:     now,
		RefetchRequested:    true,

		SeenTime: now,

		CPUType: "cpuType",
	})
	require.NoError(t, err)

	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	// HostLite does not load host details.
	require.Empty(t, h.CPUType)
	// HostLite does not load host seen time.
	require.Empty(t, h.SeenTime)

	require.Equal(t, uint(1), h.ID)
	require.NotEmpty(t, h.CreatedAt)
	require.NotEmpty(t, h.UpdatedAt)
	require.Equal(t, "foobar", h.OsqueryHostID)
	require.Equal(t, "nodekey", h.NodeKey)
	require.Equal(t, "foobar.local", h.Hostname)
	require.Equal(t, "uuid", h.UUID)
	require.Equal(t, "darwin", h.Platform)
	require.Nil(t, h.TeamID)
	require.Equal(t, uint(60), h.DistributedInterval)
	require.Equal(t, uint(50), h.LoggerTLSPeriod)
	require.Equal(t, uint(40), h.ConfigTLSRefresh)
	require.WithinDuration(t, now.UTC(), h.DetailUpdatedAt, 1*time.Second)
	require.WithinDuration(t, now.UTC(), h.LabelUpdatedAt, 1*time.Second)
	require.WithinDuration(t, now.UTC(), h.PolicyUpdatedAt, 1*time.Second)
	require.WithinDuration(t, now.UTC(), h.LastEnrolledAt, 1*time.Second)
	require.True(t, h.RefetchRequested)
}

func testUpdateOsqueryIntervals(t *testing.T, ds *Datastore) {
	now := time.Now()
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:                  1,
		OsqueryHostID:       "foobar",
		NodeKey:             "nodekey",
		Hostname:            "foobar.local",
		UUID:                "uuid",
		Platform:            "darwin",
		DistributedInterval: 60,
		LoggerTLSPeriod:     50,
		ConfigTLSRefresh:    40,
		DetailUpdatedAt:     now,
		LabelUpdatedAt:      now,
		LastEnrolledAt:      now,
		PolicyUpdatedAt:     now,
		RefetchRequested:    true,
		SeenTime:            now,
	})
	require.NoError(t, err)

	err = ds.UpdateHostOsqueryIntervals(context.Background(), h.ID, fleet.HostOsqueryIntervals{
		DistributedInterval: 120,
		LoggerTLSPeriod:     110,
		ConfigTLSRefresh:    100,
	})
	require.NoError(t, err)

	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	require.Equal(t, uint(120), h.DistributedInterval)
	require.Equal(t, uint(110), h.LoggerTLSPeriod)
	require.Equal(t, uint(100), h.ConfigTLSRefresh)
}

func testUpdateRefetchRequested(t *testing.T, ds *Datastore) {
	now := time.Now()
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:                  1,
		OsqueryHostID:       "foobar",
		NodeKey:             "nodekey",
		Hostname:            "foobar.local",
		UUID:                "uuid",
		Platform:            "darwin",
		DistributedInterval: 60,
		LoggerTLSPeriod:     50,
		ConfigTLSRefresh:    40,
		DetailUpdatedAt:     now,
		LabelUpdatedAt:      now,
		LastEnrolledAt:      now,
		PolicyUpdatedAt:     now,
		RefetchRequested:    false,
		SeenTime:            now,
	})
	require.NoError(t, err)

	err = ds.UpdateHostRefetchRequested(context.Background(), h.ID, true)
	require.NoError(t, err)

	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	require.True(t, h.RefetchRequested)

	err = ds.UpdateHostRefetchRequested(context.Background(), h.ID, false)
	require.NoError(t, err)

	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	require.False(t, h.RefetchRequested)
}

func testHostsSaveHostUsers(t *testing.T, ds *Datastore) {
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

	users := []fleet.HostUser{
		{
			Uid:       42,
			Username:  "user",
			Type:      "aaa",
			GroupName: "group",
			Shell:     "shell",
		},
		{
			Uid:       43,
			Username:  "user2",
			Type:      "aaa",
			GroupName: "group",
			Shell:     "shell",
		},
	}

	err = ds.SaveHostUsers(context.Background(), host.ID, users)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	require.Len(t, host.Users, 2)
	test.ElementsMatchSkipID(t, users, host.Users)
}
