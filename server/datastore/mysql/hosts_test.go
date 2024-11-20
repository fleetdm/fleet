package mysql

import (
	"context"
	"crypto/sha256"
	"database/sql"
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
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
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
		{"Save", testHostsUpdate},
		{"DeleteWithSoftware", testHostsDeleteWithSoftware},
		{"SaveHostPackStatsDB", testSaveHostPackStatsDB},
		{"SavePackStatsOverwrites", testHostsSavePackStatsOverwrites},
		{"WithTeamPackStats", testHostsWithTeamPackStats},
		{"Delete", testHostsDelete},
		{"HostListOptionsTeamFilter", testHostListOptionsTeamFilter},
		{"ListFilterAdditional", testHostsListFilterAdditional},
		{"ListStatus", testHostsListStatus},
		{"ListQuery", testHostsListQuery},
		{"ListMDM", testHostsListMDM},
		{"SelectHostMDM", testHostMDMSelect},
		{"ListMunkiIssueID", testHostsListMunkiIssueID},
		{"Enroll", testHostsEnroll},
		{"LoadHostByNodeKey", testHostsLoadHostByNodeKey},
		{"LoadHostByNodeKeyCaseSensitive", testHostsLoadHostByNodeKeyCaseSensitive},
		{"Search", testHostsSearch},
		{"SearchWildCards", testSearchHostsWildCards},
		{"SearchLimit", testHostsSearchLimit},
		{"GenerateStatusStatistics", testHostsGenerateStatusStatistics},
		{"MarkSeen", testHostsMarkSeen},
		{"MarkSeenMany", testHostsMarkSeenMany},
		{"CleanupIncoming", testHostsCleanupIncoming},
		{"IDsByIdentifier", testHostIDsByIdentifier},
		{"Additional", testHostsAdditional},
		{"ByIdentifier", testHostsByIdentifier},
		{"HostLiteByIdentifierAndID", testHostLiteByIdentifierAndID},
		{"AddToTeam", testHostsAddToTeam},
		{"SaveUsers", testHostsSaveUsers},
		{"SaveHostUsers", testHostsSaveHostUsers},
		{"SaveUsersWithoutUid", testHostsSaveUsersWithoutUid},
		{"TotalAndUnseenSince", testHostsTotalAndUnseenSince},
		{"ListByPolicy", testHostsListByPolicy},
		{"SaveTonsOfUsers", testHostsUpdateTonsOfUsers},
		{"SavePackStatsConcurrent", testHostsSavePackStatsConcurrent},
		{"LoadHostByNodeKeyLoadsDisk", testLoadHostByNodeKeyLoadsDisk},
		{"LoadHostByNodeKeyUsesStmt", testLoadHostByNodeKeyUsesStmt},
		{"HostsListBySoftware", testHostsListBySoftware},
		{"HostsListBySoftwareChangedAt", testHostsListBySoftwareChangedAt},
		{"HostsListByOperatingSystemID", testHostsListByOperatingSystemID},
		{"HostsListByOSNameAndVersion", testHostsListByOSNameAndVersion},
		{"HostsListByVulnerability", testHostsListByVulnerability},
		{"HostsListByDiskEncryptionStatus", testHostsListMacOSSettingsDiskEncryptionStatus},
		{"HostsListFailingPolicies", printReadsInTest(testHostsListFailingPolicies)},
		{"HostsExpiration", testHostsExpiration},
		{"TeamHostsExpiration", testTeamHostsExpiration},
		{"HostsIncludesScheduledQueriesInPackStats", testHostsIncludesScheduledQueriesInPackStats},
		{"HostsAllPackStats", testHostsAllPackStats},
		{"HostsPackStatsMultipleHosts", testHostsPackStatsMultipleHosts},
		{"HostsPackStatsForPlatform", testHostsPackStatsForPlatform},
		{"HostsReadsLessRows", testHostsReadsLessRows},
		{"HostsNoSeenTime", testHostsNoSeenTime},
		{"HostDeviceMapping", testHostDeviceMapping},
		{"ReplaceHostDeviceMapping", testHostsReplaceHostDeviceMapping},
		{"CustomHostDeviceMapping", testHostsCustomHostDeviceMapping},
		{"HostMDMAndMunki", testHostMDMAndMunki},
		{"AggregatedHostMDMAndMunki", testAggregatedHostMDMAndMunki},
		{"MunkiIssuesBatchSize", testMunkiIssuesBatchSize},
		{"HostLite", testHostsLite},
		{"UpdateOsqueryIntervals", testUpdateOsqueryIntervals},
		{"UpdateRefetchRequested", testUpdateRefetchRequested},
		{"LoadHostByDeviceAuthToken", testHostsLoadHostByDeviceAuthToken},
		{"SetOrUpdateDeviceAuthToken", testHostsSetOrUpdateDeviceAuthToken},
		{"OSVersions", testOSVersions},
		{"DeleteHosts", testHostsDeleteHosts},
		{"HostIDsByOSVersion", testHostIDsByOSVersion},
		{"ReplaceHostBatteries", testHostsReplaceHostBatteries},
		{"ReplaceHostBatteriesDeadlock", testHostsReplaceHostBatteriesDeadlock},
		{"CountHostsNotResponding", testCountHostsNotResponding},
		{"FailingPoliciesCount", testFailingPoliciesCount},
		{"HostRecordNoPolicies", testHostsRecordNoPolicies},
		{"SetOrUpdateHostDisksSpace", testHostsSetOrUpdateHostDisksSpace},
		{"HostIDsByOSID", testHostIDsByOSID},
		{"SetOrUpdateHostDisksEncryption", testHostsSetOrUpdateHostDisksEncryption},
		{"HostOrder", testHostOrder},
		{"GetHostMDMCheckinInfo", testHostsGetHostMDMCheckinInfo},
		{"UnenrollFromMDM", testHostsUnenrollFromMDM},
		{"LoadHostByOrbitNodeKey", testHostsLoadHostByOrbitNodeKey},
		{"SetOrUpdateHostDiskEncryptionKeys", testHostsSetOrUpdateHostDisksEncryptionKey},
		{"SetHostsDiskEncryptionKeyStatus", testHostsSetDiskEncryptionKeyStatus},
		{"GetUnverifiedDiskEncryptionKeys", testHostsGetUnverifiedDiskEncryptionKeys},
		{"LUKS", testLUKSDatastoreFunctions},
		{"EnrollOrbit", testHostsEnrollOrbit},
		{"EnrollUpdatesMissingInfo", testHostsEnrollUpdatesMissingInfo},
		{"EncryptionKeyRawDecryption", testHostsEncryptionKeyRawDecryption},
		{"ListHostsLiteByUUIDs", testHostsListHostsLiteByUUIDs},
		{"GetMatchingHostSerials", testGetMatchingHostSerials},
		{"ListHostsLiteByIDs", testHostsListHostsLiteByIDs},
		{"ListHostsWithPagination", testListHostsWithPagination},
		{"LastRestarted", testLastRestarted},
		{"HostHealth", testHostHealth},
		{"GetHostOrbitInfo", testGetHostOrbitInfo},
		{"HostnamesByIdentifiers", testHostnamesByIdentifiers},
		{"HostsAddToTeamCleansUpTeamQueryResults", testHostsAddToTeamCleansUpTeamQueryResults},
		{"UpdateHostIssues", testUpdateHostIssues},
		{"ListUpcomingHostMaintenanceWindows", testListUpcomingHostMaintenanceWindows},
		{"GetHostEmails", testGetHostEmails},
		{"TestGetMatchingHostSerialsMarkedDeleted", testGetMatchingHostSerialsMarkedDeleted},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testHostsUpdate(t *testing.T, ds *Datastore) {
	testUpdateHost(t, ds, ds.UpdateHost)
	testUpdateHost(t, ds, ds.SerialUpdateHost)
}

func testUpdateHost(t *testing.T, ds *Datastore, updateHostFunc func(context.Context, *fleet.Host) error) {
	policyUpdatedAt := time.Now().UTC().Truncate(time.Second)
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: policyUpdatedAt,
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	host.Hostname = "bar.local"
	err = updateHostFunc(context.Background(), host)
	require.NoError(t, err)

	host.RefetchCriticalQueriesUntil = ptr.Time(time.Now().UTC().Add(time.Hour))
	err = updateHostFunc(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)

	assert.Equal(t, "bar.local", host.Hostname)
	assert.Equal(t, "192.168.1.1", host.PrimaryIP)
	assert.Equal(t, "30-65-EC-6F-C4-58", host.PrimaryMac)
	assert.Equal(t, policyUpdatedAt.UTC(), host.PolicyUpdatedAt)
	assert.NotNil(t, host.RefetchCriticalQueriesUntil)
	assert.True(t, time.Now().Before(*host.RefetchCriticalQueriesUntil))
	assert.Nil(t, host.OrbitVersion)
	assert.Nil(t, host.DesktopVersion)
	assert.Nil(t, host.ScriptsEnabled)

	additionalJSON := json.RawMessage(`{"foobar": "bim"}`)
	err = ds.SaveHostAdditional(context.Background(), host.ID, &additionalJSON)
	require.NoError(t, err)
	// set host orbit info
	var (
		orbitVersion   = "1.1.0"
		desktopVersion = "2.1.0"
	)
	err = ds.SetOrUpdateHostOrbitInfo(
		context.Background(), host.ID, orbitVersion, sql.NullString{String: desktopVersion, Valid: true},
		sql.NullBool{Bool: true, Valid: true},
	)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.NotNil(t, host)
	require.NotNil(t, host.Additional)
	assert.Equal(t, additionalJSON, *host.Additional)
	assert.Equal(t, orbitVersion, *host.OrbitVersion)
	assert.Equal(t, desktopVersion, *host.DesktopVersion)
	assert.True(t, *host.ScriptsEnabled)

	err = updateHostFunc(context.Background(), host)
	require.NoError(t, err)

	host.RefetchCriticalQueriesUntil = nil
	err = updateHostFunc(context.Background(), host)
	require.NoError(t, err)

	err = ds.SetOrUpdateHostOrbitInfo(
		context.Background(), host.ID, orbitVersion, sql.NullString{Valid: false}, sql.NullBool{Valid: false},
	)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.NotNil(t, host)
	require.Nil(t, host.RefetchCriticalQueriesUntil)
	assert.Equal(t, orbitVersion, *host.OrbitVersion)
	assert.Nil(t, host.DesktopVersion)
	assert.Nil(t, host.ScriptsEnabled)

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

	host, err = ds.Host(context.Background(), host.ID)
	assert.NotNil(t, err)
	assert.Nil(t, host)

	err = ds.DeletePack(context.Background(), newP.Name)
	require.NoError(t, err)
}

func testHostsDeleteWithSoftware(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
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
	_, err = ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)

	err = ds.DeleteHost(context.Background(), host.ID)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	assert.NotNil(t, err)
	assert.Nil(t, host)
}

func testSaveHostPackStatsDB(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
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
	query1 := test.NewQuery(t, ds, nil, "time", "select * from time", 0, true)
	squery1 := test.NewScheduledQuery(t, ds, pack1.ID, query1.ID, 30, true, true, "time-scheduled")
	stats1 := []fleet.ScheduledQueryStats{
		{
			PackName:           pack1.Name,
			ScheduledQueryName: squery1.Name,

			ScheduledQueryID: squery1.ID,
			QueryName:        query1.Name,
			PackID:           pack1.ID,
			AverageMemory:    8000,
			Denylisted:       false,
			Executions:       164,
			Interval:         30,
			LastExecuted:     time.Unix(1620325191, 0).UTC(),
			OutputSize:       1337,
			SystemTime:       150,
			UserTime:         180,
			WallTime:         0,
		},
	}

	pack2, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test2",
		HostIDs: []uint{host.ID},
	})
	require.NoError(t, err)
	squery2 := test.NewScheduledQuery(t, ds, pack2.ID, query1.ID, 30, true, true, "time-scheduled")
	query2 := test.NewQuery(t, ds, nil, "processes", "select * from processes", 0, true)
	squery3 := test.NewScheduledQuery(t, ds, pack2.ID, query2.ID, 30, true, true, "processes")
	stats2 := []fleet.ScheduledQueryStats{
		{
			PackName:           pack2.Name,
			ScheduledQueryName: squery2.Name,

			ScheduledQueryID: squery2.ID,
			QueryName:        query1.Name,
			PackID:           pack2.ID,
			AverageMemory:    431,
			Denylisted:       true,
			Executions:       1,
			Interval:         30,
			LastExecuted:     time.Unix(980943843, 0).UTC(),
			OutputSize:       134,
			SystemTime:       1656,
			UserTime:         18453,
			WallTime:         10,
		},
		{
			ScheduledQueryName: squery3.Name,
			PackName:           pack2.Name,

			ScheduledQueryID: squery3.ID,
			QueryName:        query2.Name,
			PackID:           pack2.ID,
			AverageMemory:    8000,
			Denylisted:       false,
			Executions:       164,
			Interval:         30,
			LastExecuted:     time.Unix(1620325191, 0).UTC(),
			OutputSize:       1337,
			SystemTime:       150,
			UserTime:         180,
			WallTime:         0,
		},
	}

	packStats := []fleet.PackStats{
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

	err = ds.SaveHostPackStats(context.Background(), host.TeamID, host.ID, packStats)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)

	require.Len(t, host.PackStats, 2)
	sort.Slice(host.PackStats, func(i, j int) bool {
		return host.PackStats[i].PackName < host.PackStats[j].PackName
	})

	assert.Equal(t, host.PackStats[0].PackName, "test1")
	// A new behavior is introduced with the new query model. If multiple scheduled queries
	// with the same referenced query_id are executed in user packs, then only one of the results
	// is gathered in Fleet.
	assert.ElementsMatch(t, host.PackStats[0].QueryStats, []fleet.ScheduledQueryStats{
		{
			PackName:           pack1.Name,
			ScheduledQueryName: squery1.Name,

			ScheduledQueryID: squery1.ID,
			QueryName:        query1.Name,
			PackID:           pack1.ID,
			//
			// These are the values for the same query1 in the second pack (it overrides the first schedule stats).
			//
			AverageMemory: 431,
			Denylisted:    true,
			Executions:    1,
			Interval:      30,
			LastExecuted:  time.Unix(980943843, 0).UTC(),
			OutputSize:    134,
			SystemTime:    1656,
			UserTime:      18453,
			WallTime:      10000,
		},
	})
	assert.Equal(t, host.PackStats[1].PackName, "test2")
	// Server calculates WallTimeMs if WallTimeMs==0 coming in. (osquery wall_time -> wall_time_ms -> DB wall_time)
	stats2[0].WallTime *= 1000
	assert.ElementsMatch(t, host.PackStats[1].QueryStats, stats2)
}

// testHostsSavePackStatsOverwrites now behaves in a way that if two scheduled queries in a pack
// reference the same query_id, then their stat values are overriden.
func testHostsSavePackStatsOverwrites(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
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
	query1 := test.NewQuery(t, ds, nil, "time", "select * from time", 0, true)
	squery1 := test.NewScheduledQuery(t, ds, pack1.ID, query1.ID, 30, true, true, "time-scheduled")
	pack2, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test2",
		HostIDs: []uint{host.ID},
	})
	require.NoError(t, err)
	squery2 := test.NewScheduledQuery(t, ds, pack2.ID, query1.ID, 30, true, true, "time-scheduled")
	query2 := test.NewQuery(t, ds, nil, "processes", "select * from processes", 0, true)

	execTime1 := time.Unix(1620325191, 0).UTC()

	packStats := []fleet.PackStats{
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

	err = ds.SaveHostPackStats(context.Background(), host.TeamID, host.ID, packStats)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)

	sort.Slice(host.PackStats, func(i, j int) bool {
		return host.PackStats[i].PackName < host.PackStats[j].PackName
	})

	require.Len(t, host.PackStats, 2)
	assert.Equal(t, host.PackStats[0].PackName, "test1")
	assert.Equal(t, execTime1, host.PackStats[0].QueryStats[0].LastExecuted)

	execTime2 := execTime1.Add(24 * time.Hour)

	packStats = []fleet.PackStats{
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
	err = ds.SaveHostPackStats(context.Background(), host.TeamID, host.ID, packStats)
	require.NoError(t, err)

	gotHost, err := ds.Host(context.Background(), host.ID)
	require.NoError(t, err)

	sort.Slice(gotHost.PackStats, func(i, j int) bool {
		return gotHost.PackStats[i].PackName < gotHost.PackStats[j].PackName
	})

	require.Len(t, gotHost.PackStats, 2)
	assert.Equal(t, gotHost.PackStats[0].PackName, "test1")
	assert.Equal(t, execTime1, gotHost.PackStats[0].QueryStats[0].LastExecuted)
}

func testHostsWithTeamPackStats(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
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
	host.TeamID = &team.ID
	tpQuery := test.NewQueryWithSchedule(t, ds, &team.ID, "tp-time", "select * from time", 0, true, 30, true)

	// Create a new pack and target to the host.
	// Pack and query must exist for stats to save successfully
	pack1, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test1",
		HostIDs: []uint{host.ID},
	})
	require.NoError(t, err)
	query1 := test.NewQuery(t, ds, nil, "time", "select * from time", 0, true)
	squery1 := test.NewScheduledQuery(t, ds, pack1.ID, query1.ID, 30, true, true, "time-scheduled")
	stats1 := []fleet.ScheduledQueryStats{
		{
			PackName:           pack1.Name,
			ScheduledQueryName: squery1.Name,

			QueryName:          query1.Name,
			PackID:             pack1.ID,
			DiscardData:        false,
			AutomationsEnabled: false,
			LastFetched:        nil,
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
			PackName:           fmt.Sprintf("team-%d", team.ID),
			ScheduledQueryName: tpQuery.Name,

			QueryName:          tpQuery.Name,
			PackID:             0, // pack_id will be 0 for stats of queries not in packs.
			LastFetched:        nil,
			DiscardData:        tpQuery.DiscardData,
			AutomationsEnabled: tpQuery.AutomationsEnabled,
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

	packStats := []fleet.PackStats{
		{PackName: pack1.Name, QueryStats: stats1},
		{PackName: fmt.Sprintf("team-%d", team.ID), QueryStats: stats2},
	}
	err = ds.SaveHostPackStats(context.Background(), host.TeamID, host.ID, packStats)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)

	require.Len(t, host.PackStats, 2)
	sort.Sort(packStatsSlice(host.PackStats))

	assert.Equal(t, host.PackStats[0].PackName, teamScheduleName(team))
	stats2[0].PackName = "Team: team1"
	stats2[0].ScheduledQueryID = tpQuery.ID
	assert.ElementsMatch(t, host.PackStats[0].QueryStats, stats2)
	assert.Equal(t, host.PackStats[1].PackName, pack1.Name)
	stats1[0].ScheduledQueryID = squery1.ID
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
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	err = ds.DeleteHost(context.Background(), host.ID)
	require.NoError(t, err)

	_, err = ds.Host(context.Background(), host.ID)
	assert.NotNil(t, err)

	originalHostDeleteBatchSize := hostsDeleteBatchSize
	hostsDeleteBatchSize = 2
	t.Cleanup(func() {
		hostsDeleteBatchSize = originalHostDeleteBatchSize
	})

	// Delete nothing -- no-op
	require.NoError(t, ds.DeleteHosts(context.Background(), nil))

	numHosts := 5
	hosts := make([]*fleet.Host, numHosts)
	for i := 0; i < numHosts; i++ {
		hosts[i], err = ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String(fmt.Sprint(i)),
			UUID:            fmt.Sprint(i),
			Hostname:        fmt.Sprintf("foo.local.%d", i),
		})
		require.NoError(t, err)
		require.NotNil(t, hosts[i])
	}
	var hostIDs []uint
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID)
	}

	// Delete all hosts
	require.NoError(t, ds.DeleteHosts(context.Background(), hostIDs))
	// Make sure each host is deleted
	for _, h := range hosts {
		_, err = ds.Host(context.Background(), h.ID)
		assert.NotNil(t, err)
	}
}

func listHostsCheckCount(t *testing.T, ds *Datastore, filter fleet.TeamFilter, opt fleet.HostListOptions, expectedCount int) []*fleet.Host {
	hosts, err := ds.ListHosts(context.Background(), filter, opt)
	require.NoError(t, err)
	count, err := ds.CountHosts(context.Background(), filter, opt)
	require.NoError(t, err)
	require.Equal(t, expectedCount, count)
	return hosts
}

func testHostListOptionsTeamFilter(t *testing.T, ds *Datastore) {
	var teamIDFilterNil *uint                // "All teams" option should include all hosts regardless of team assignment
	var teamIDFilterZero *uint = ptr.Uint(0) // "No team" option should include only hosts that are not assigned to any team
	teamIDFilterBad := ptr.Uint(9999)

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	var hosts []*fleet.Host
	for i := 0; i < 10; i++ {
		var opts []test.NewHostOption
		switch i {
		case 5, 6:
			opts = append(opts, test.WithPlatform("windows"))
		}
		h := test.NewHost(t, ds, fmt.Sprintf("foo.local.%d", i), "1.1.1.1",
			fmt.Sprintf("%d", i), fmt.Sprintf("%d", i), time.Now(), opts...)
		hosts = append(hosts, h)
		nanoEnrollAndSetHostMDMData(t, ds, h, false)
	}
	userFilter := fleet.TeamFilter{User: test.UserAdmin}

	// confirm intial state
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{}, len(hosts))
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterNil}, len(hosts))
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterZero}, len(hosts))
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team1.ID}, 0)
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team2.ID}, 0)

	// assign three hosts to team 1
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID}))
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{}, len(hosts))
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterNil}, len(hosts))
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterZero}, len(hosts)-3)
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team1.ID}, 3)
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team2.ID}, 0)

	// assign four hosts to team 2
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team2.ID, []uint{hosts[3].ID, hosts[4].ID, hosts[5].ID, hosts[6].ID}))
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{}, len(hosts))
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterNil}, len(hosts))
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterZero}, len(hosts)-7)
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team1.ID}, 3)
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team2.ID}, 4)

	// test team filter in combination with macos settings filter
	profUUID := "a" + uuid.NewString()
	require.NoError(t, ds.BulkUpsertMDMAppleHostProfiles(context.Background(), []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			ProfileUUID:       profUUID,
			ProfileIdentifier: "identifier",
			HostUUID:          hosts[0].UUID, // hosts[0] is assgined to team 1
			CommandUUID:       "command-uuid-1",
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryVerifying,
			Checksum:          []byte("csum"),
		},
	}))
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team1.ID, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 1) // hosts[0]
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team2.ID, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 0) // wrong team
	// macos settings filter does not support "all teams" so teamIDFilterNil acts the same as teamIDFilterZero
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterZero, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 0) // no team
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterNil, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 0)  // no team
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{MacOSSettingsFilter: fleet.OSSettingsVerifying}, 0)                               // no team

	require.NoError(t, ds.BulkUpsertMDMAppleHostProfiles(context.Background(), []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			ProfileUUID:       profUUID,
			ProfileIdentifier: "identifier",
			HostUUID:          hosts[9].UUID, // hosts[9] is assgined to no team
			CommandUUID:       "command-uuid-2",
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryVerifying,
			Checksum:          []byte("csum"),
		},
	}))
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team1.ID, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 1) // hosts[0]
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team2.ID, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 0) // wrong team
	// macos settings filter does not support "all teams" so both teamIDFilterNil acts the same as teamIDFilterZero
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterZero, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 1) // hosts[9]
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterNil, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 1)  // hosts[9]
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{MacOSSettingsFilter: fleet.OSSettingsVerifying}, 1)                               // hosts[9]

	// test team filter in combination with os settings filter
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team1.ID, OSSettingsFilter: fleet.OSSettingsVerifying}, 1) // hosts[0]
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team2.ID, OSSettingsFilter: fleet.OSSettingsVerifying}, 0) // wrong team
	// os settings filter does not support "all teams" so teamIDFilterNil acts the same as teamIDFilterZero
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterZero, OSSettingsFilter: fleet.OSSettingsVerifying}, 1) // hosts[9]
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterNil, OSSettingsFilter: fleet.OSSettingsVerifying}, 1)  // hosts[9]
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{OSSettingsFilter: fleet.OSSettingsVerifying}, 1)

	// test team filter in combination with os settings disk encryptionfilter
	require.NoError(t, ds.BulkUpsertMDMAppleHostProfiles(context.Background(), []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			ProfileUUID:       profUUID,
			ProfileIdentifier: mobileconfig.FleetFileVaultPayloadIdentifier,
			HostUUID:          hosts[8].UUID, // hosts[8] is assgined to no team
			CommandUUID:       "command-uuid-3",
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryPending,
			Checksum:          []byte("disk-encryption-csum"),
		},
	}))
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team1.ID, OSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 0) // hosts[0]
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: &team2.ID, OSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 0) // wrong team
	// os settings filter does not support "all teams" so teamIDFilterNil acts the same as teamIDFilterZero
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterZero, OSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 1) // hosts[8]
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterNil, OSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 1)  // hosts[8]
	listHostsCheckCount(t, ds, userFilter, fleet.HostListOptions{OSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 1)                               // hosts[8]

	// Bad team filter
	_, err = ds.ListHosts(context.Background(), userFilter, fleet.HostListOptions{TeamFilter: teamIDFilterBad})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "team is invalid"), err)
}

func testHostsListFilterAdditional(t *testing.T, ds *Datastore) {
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("foobar"),
		NodeKey:         ptr.String("nodekey"),
		UUID:            "uuid",
		Hostname:        "foobar.local",
	})
	require.NoError(t, err)

	filter := fleet.TeamFilter{User: test.UserAdmin}

	// Add additional
	additional := json.RawMessage(`{"field1": "v1", "field2": "v2"}`)
	err = ds.SaveHostAdditional(context.Background(), h.ID, &additional)
	require.NoError(t, err)

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
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute * 5),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{StatusFilter: "online"}, 1)
	assert.Equal(t, 1, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{StatusFilter: "offline"}, 9)
	assert.Equal(t, 9, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{StatusFilter: "mia"}, 0)
	assert.Equal(t, 0, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{StatusFilter: "missing"}, 0)
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
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("uuid_00%d", i),
			Hostname:        hostname,
			HardwareSerial:  fmt.Sprintf("serial00%d", i),
		})
		require.NoError(t, err)
		host.PrimaryIP = fmt.Sprintf("192.168.1.%d", i)
		err = ds.UpdateHost(context.Background(), host)
		require.NoError(t, err)
		hosts = append(hosts, host)
	}

	// add some device mapping for some hosts
	require.NoError(t, ds.ReplaceHostDeviceMapping(context.Background(), hosts[0].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[0].ID, Email: "a@b.c", Source: "src1"},
		{HostID: hosts[0].ID, Email: "b@b.c", Source: "src1"},
	}, "src1"))
	require.NoError(t, ds.ReplaceHostDeviceMapping(context.Background(), hosts[1].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[1].ID, Email: "c@b.c", Source: "src1"},
	}, "src1"))
	require.NoError(t, ds.ReplaceHostDeviceMapping(context.Background(), hosts[2].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[2].ID, Email: "dbca@b.cba", Source: "src1"},
	}, "src1"))

	// add some disks space info for some hosts
	require.NoError(t, ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[0].ID, 1.0, 2.0, 30.0))
	require.NoError(t, ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[1].ID, 3.0, 4.0, 50.0))
	require.NoError(t, ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[2].ID, 5.0, 6.0, 70.0))

	filter := fleet.TeamFilter{User: test.UserAdmin}

	var teamIDFilterNil *uint                // "All teams" filter
	var teamIDFilterZero *uint = ptr.Uint(0) // "No team" filter

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

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{TeamFilter: teamIDFilterNil}, len(hosts))
	assert.Equal(t, len(hosts), len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{TeamFilter: teamIDFilterZero}, 0)
	assert.Equal(t, 0, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{LowDiskSpaceFilter: ptr.Int(32)}, 3)
	assert.Equal(t, 3, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{LowDiskSpaceFilter: ptr.Int(5)}, 2) // less than 5GB, only 2 hosts
	assert.Equal(t, 2, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{TeamFilter: &team1.ID, LowDiskSpaceFilter: ptr.Int(5)}, 2)
	assert.Equal(t, 2, len(gotHosts))

	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{TeamFilter: &team2.ID, LowDiskSpaceFilter: ptr.Int(5)}, 0)
	assert.Equal(t, 0, len(gotHosts))

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
	for _, h := range gotHosts {
		switch h.ID {
		case hosts[0].ID:
			assert.Equal(t, h.GigsDiskSpaceAvailable, 1.0)
			assert.Equal(t, h.PercentDiskSpaceAvailable, 2.0)
		case hosts[1].ID:
			assert.Equal(t, h.GigsDiskSpaceAvailable, 3.0)
			assert.Equal(t, h.PercentDiskSpaceAvailable, 4.0)
		case hosts[2].ID:
			assert.Equal(t, h.GigsDiskSpaceAvailable, 5.0)
			assert.Equal(t, h.PercentDiskSpaceAvailable, 6.0)
		}
	}

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

	// device mapping not included because missing optional param
	require.Nil(t, gotHosts[0].DeviceMapping)
	require.Nil(t, gotHosts[1].DeviceMapping)
	require.Nil(t, gotHosts[2].DeviceMapping)

	// add optional param to include host device mapping
	gotHosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{ListOptions: fleet.ListOptions{MatchQuery: "a@b.c"}, DeviceMapping: true}, 3)
	require.NotNil(t, gotHosts[0].DeviceMapping)
	require.NotNil(t, gotHosts[1].DeviceMapping)
	require.NotNil(t, gotHosts[2].DeviceMapping) // json "null" rather than nil

	var dm []*fleet.HostDeviceMapping

	err = json.Unmarshal(*gotHosts[0].DeviceMapping, &dm)
	require.NoError(t, err)
	require.Len(t, dm, 2)

	err = json.Unmarshal(*gotHosts[1].DeviceMapping, &dm)
	require.NoError(t, err)
	require.Len(t, dm, 1)

	err = json.Unmarshal(*gotHosts[2].DeviceMapping, &dm)
	require.NoError(t, err)
	require.Nil(t, dm)
}

func testHostsUnenrollFromMDM(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	h, err := ds.NewHost(ctx, &fleet.Host{
		Platform:        "darwin",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("foo"),
		NodeKey:         ptr.String("foo"),
		UUID:            "foo",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)
	h2, err := ds.NewHost(ctx, &fleet.Host{
		Platform:        "darwin",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("foo2"),
		NodeKey:         ptr.String("foo2"),
		UUID:            "foo2",
		Hostname:        "foo2.local",
	})
	require.NoError(t, err)

	_, err = ds.GetHostMDM(ctx, h.ID)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	_, err = ds.GetHostMDM(ctx, h2.ID)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	// Set hosts to be enrolled to an MDM.
	const simpleMDM = "https://simplemdm.com"
	err = ds.SetOrUpdateMDMData(ctx, h.ID, false, true, simpleMDM, true, "", "")
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(ctx, h2.ID, false, true, simpleMDM, true, "", "")
	require.NoError(t, err)

	// force is_server to NULL for host 1
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_mdm SET is_server = NULL WHERE host_id = ?`, h.ID)
		return err
	})
	// GetHostMDM should still work and return false for is_server
	hmdm, err := ds.GetHostMDM(ctx, h.ID)
	require.NoError(t, err)
	require.Equal(t, h.ID, hmdm.HostID)
	require.False(t, hmdm.IsServer)

	for _, hi := range []*fleet.Host{h, h2} {
		hmdm, err := ds.GetHostMDM(ctx, hi.ID)
		require.NoError(t, err)
		require.Equal(t, hi.ID, hmdm.HostID)
		require.True(t, hmdm.Enrolled)
		require.True(t, hmdm.InstalledFromDep)
		require.NotNil(t, hmdm.MDMID)
		require.Equal(t, simpleMDM, hmdm.ServerURL)
	}

	err = ds.GenerateAggregatedMunkiAndMDM(ctx)
	require.NoError(t, err)

	// Check that both hosts are counted.
	solutions, _, err := ds.AggregatedMDMSolutions(ctx, nil, "darwin")
	require.NoError(t, err)
	require.Len(t, solutions, 1)
	require.Equal(t, 2, solutions[0].HostsCount)

	// Host `h` unenrolls from MDM, so MDM query returns empty server_url.
	err = ds.SetOrUpdateMDMData(ctx, h.ID, false, false, "", false, "", "")
	require.NoError(t, err)

	// host_mdm entry should still exist with empty values.
	hmdm, err = ds.GetHostMDM(ctx, h.ID)
	require.NoError(t, err)
	require.Equal(t, h.ID, hmdm.HostID)
	require.False(t, hmdm.Enrolled)
	require.False(t, hmdm.InstalledFromDep)
	require.Nil(t, hmdm.MDMID)
	require.Empty(t, hmdm.ServerURL)

	err = ds.GenerateAggregatedMunkiAndMDM(ctx)
	require.NoError(t, err)

	solutions, _, err = ds.AggregatedMDMSolutions(ctx, nil, "darwin")
	require.NoError(t, err)
	require.Len(t, solutions, 1)
	require.Equal(t, 1, solutions[0].HostsCount)

	// Host `h2` unenrolls from MDM, so MDM query returns empty server_url.
	err = ds.SetOrUpdateMDMData(ctx, h2.ID, false, false, "", false, "", "")
	require.NoError(t, err)

	// host_mdm entry should not exist anymore.
	_, err = ds.GetHostMDM(ctx, h2.ID)
	require.NoError(t, err)

	// host_mdm entry should still exist with empty values.
	hmdm, err = ds.GetHostMDM(ctx, h2.ID)
	require.NoError(t, err)
	require.Equal(t, h2.ID, hmdm.HostID)
	require.False(t, hmdm.Enrolled)
	require.False(t, hmdm.InstalledFromDep)
	require.Nil(t, hmdm.MDMID)
	require.Empty(t, hmdm.ServerURL)

	err = ds.GenerateAggregatedMunkiAndMDM(ctx)
	require.NoError(t, err)

	// No solutions should be listed now (both hosts are unenrolled).
	solutions, _, err = ds.AggregatedMDMSolutions(ctx, nil, "darwin")
	require.NoError(t, err)
	require.Len(t, solutions, 0)
}

func testHostsListMDM(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var hostIDs []uint
	for i := 0; i < 10; i++ {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute * 2),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
			Platform:        "darwin",
		})
		require.NoError(t, err)
		hostIDs = append(hostIDs, h.ID)
	}

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

	// enrollment: pending (with Fleet mdm)
	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{
		{SerialNumber: "532141num832", Model: "MacBook Pro", OS: "OSX", OpType: "added"},
	}, abmToken.ID, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	const simpleMDM, kandji, unknown = "https://simplemdm.com", "https://kandji.io", "https://url.com"
	err = ds.SetOrUpdateMDMData(ctx, hostIDs[0], false, true, simpleMDM, true, fleet.WellKnownMDMSimpleMDM, "") // enrollment: automatic
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(ctx, hostIDs[1], false, true, kandji, true, fleet.WellKnownMDMKandji, "") // enrollment: automatic
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(ctx, hostIDs[2], false, true, unknown, false, fleet.UnknownMDMName, "") // enrollment: manual
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(ctx, hostIDs[3], false, false, simpleMDM, false, fleet.WellKnownMDMSimpleMDM, "") // enrollment: unenrolled
	require.NoError(t, err)

	var simpleMDMID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &simpleMDMID, `SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`, fleet.WellKnownMDMSimpleMDM, simpleMDM)
	})
	var kandjiID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &kandjiID, `SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`, fleet.WellKnownMDMKandji, kandji)
	})

	filter := fleet.TeamFilter{User: test.UserAdmin}

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMIDFilter: &simpleMDMID}, 2)
	assert.Equal(t, 2, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMIDFilter: &kandjiID}, 1)
	assert.Equal(t, 1, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMEnrollmentStatusFilter: fleet.MDMEnrollStatusAutomatic}, 2)
	assert.Equal(t, 2, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMEnrollmentStatusFilter: fleet.MDMEnrollStatusManual}, 1)
	assert.Equal(t, 1, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMEnrollmentStatusFilter: fleet.MDMEnrollStatusUnenrolled}, 1)
	assert.Equal(t, 1, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMEnrollmentStatusFilter: fleet.MDMEnrollStatusEnrolled}, 3) // 2 auto, 1 manual
	assert.Equal(t, 3, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMEnrollmentStatusFilter: fleet.MDMEnrollStatusAutomatic, MDMIDFilter: &kandjiID}, 1)
	assert.Equal(t, 1, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMEnrollmentStatusFilter: fleet.MDMEnrollStatusEnrolled, MDMIDFilter: &kandjiID}, 1)
	assert.Equal(t, 1, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMEnrollmentStatusFilter: fleet.MDMEnrollStatusPending}, 1)
	assert.Equal(t, 1, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMNameFilter: ptr.String(fleet.WellKnownMDMSimpleMDM)}, 2)
	assert.Equal(t, 2, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMIDFilter: &simpleMDMID, MDMNameFilter: ptr.String(fleet.WellKnownMDMSimpleMDM), MDMEnrollmentStatusFilter: fleet.MDMEnrollStatusEnrolled}, 1)
	assert.Equal(t, 1, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMNameFilter: ptr.String(fleet.WellKnownMDMKandji)}, 1)
	assert.Equal(t, 1, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMNameFilter: ptr.String(fleet.WellKnownMDMFleet), MDMEnrollmentStatusFilter: fleet.MDMEnrollStatusPending}, 1)
	assert.Equal(t, 1, len(hosts))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMNameFilter: ptr.String(fleet.WellKnownMDMJamf)}, 0)
	assert.Equal(t, 0, len(hosts))

	// create a couple Windows host and ensure they are properly returned by that filter too
	for i := 10; i < 12; i++ {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute * 2),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
			Platform:        "windows",
		})
		require.NoError(t, err)
		hostIDs = append(hostIDs, h.ID)
	}
	err = ds.SetOrUpdateMDMData(ctx, hostIDs[10], false, true, "http://intuneexample.com", false, fleet.WellKnownMDMIntune, "") // enrolled in Intune
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(ctx, hostIDs[11], false, true, "http://example.com", false, fleet.WellKnownMDMFleet, "") // enrolled in Fleet
	require.NoError(t, err)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMNameFilter: ptr.String(fleet.WellKnownMDMIntune)}, 1)
	assert.Equal(t, 1, len(hosts))
	assert.Equal(t, hostIDs[10], hosts[0].ID)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMNameFilter: ptr.String(fleet.WellKnownMDMFleet), MDMEnrollmentStatusFilter: fleet.MDMEnrollStatusEnrolled}, 1)
	assert.Equal(t, 1, len(hosts))
	assert.Equal(t, hostIDs[11], hosts[0].ID)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MDMEnrollmentStatusFilter: fleet.MDMEnrollStatusEnrolled}, 5)
	// simplemdm, kandji, unknown, intune and fleet (both are Windows)
	assert.Equal(t, 5, len(hosts))
	gotIDs := make([]uint, 0, len(hosts))
	for _, h := range hosts {
		gotIDs = append(gotIDs, h.ID)
	}
	assert.ElementsMatch(t, []uint{hostIDs[0], hostIDs[1], hostIDs[2], hostIDs[10], hostIDs[11]}, gotIDs)
}

func testHostMDMSelect(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	mdmServerURL := "https://mdm.example.com"

	cases := []struct {
		host                 fleet.HostMDM
		expectedMDMStatus    *string
		expectedMDMServerURL *string
	}{
		{
			host: fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: false,
				Enrolled:         false,
			},
			expectedMDMStatus:    ptr.String("Off"),
			expectedMDMServerURL: ptr.String(mdmServerURL),
		},
		{
			host: fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         false,
			},
			expectedMDMStatus:    ptr.String("Pending"),
			expectedMDMServerURL: ptr.String(mdmServerURL),
		},
		{
			host: fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         true,
			},
			expectedMDMStatus:    ptr.String("On (automatic)"),
			expectedMDMServerURL: ptr.String(mdmServerURL),
		},
		{
			host: fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: false,
				Enrolled:         true,
			},
			expectedMDMStatus:    ptr.String("On (manual)"),
			expectedMDMServerURL: ptr.String(mdmServerURL),
		},
		{
			host: fleet.HostMDM{
				IsServer:         true,
				InstalledFromDep: false,
				Enrolled:         false,
			},
			expectedMDMStatus:    nil,
			expectedMDMServerURL: nil,
		},
		{
			host: fleet.HostMDM{
				IsServer:         true,
				InstalledFromDep: true,
				Enrolled:         false,
			},
			expectedMDMStatus:    nil,
			expectedMDMServerURL: nil,
		},
		{
			host: fleet.HostMDM{
				IsServer:         true,
				InstalledFromDep: true,
				Enrolled:         true,
			},
			expectedMDMStatus:    nil,
			expectedMDMServerURL: nil,
		},
		{
			host: fleet.HostMDM{
				IsServer:         true,
				InstalledFromDep: false,
				Enrolled:         true,
			},
			expectedMDMStatus:    nil,
			expectedMDMServerURL: nil,
		},
	}

	h, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("osquery-host-id"),
		NodeKey:         ptr.String("node-key"),
		UUID:            "uuid",
		Hostname:        "hostname",
	})
	require.NoError(t, err)

	for _, c := range cases {
		require.NoError(t, ds.SetOrUpdateMDMData(ctx, h.ID, c.host.IsServer, c.host.Enrolled, mdmServerURL, c.host.InstalledFromDep, "test", ""))

		hosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{})
		require.NoError(t, err)
		require.Len(t, hosts, 1)
		require.Equal(t, h.ID, hosts[0].ID)
		require.Equal(t, c.expectedMDMStatus, hosts[0].MDM.EnrollmentStatus)
		require.Equal(t, c.expectedMDMServerURL, hosts[0].MDM.ServerURL)
		require.Equal(t, "test", hosts[0].MDM.Name)

		hosts, err = ds.SearchHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, "")
		require.NoError(t, err)
		require.Len(t, hosts, 1)
		require.Equal(t, h.ID, hosts[0].ID)
		require.Equal(t, c.expectedMDMStatus, hosts[0].MDM.EnrollmentStatus)
		require.Equal(t, c.expectedMDMServerURL, hosts[0].MDM.ServerURL)
		require.Equal(t, "test", hosts[0].MDM.Name)

		host, err := ds.Host(ctx, h.ID)
		require.NoError(t, err)
		require.Equal(t, c.expectedMDMStatus, host.MDM.EnrollmentStatus)
		require.Equal(t, c.expectedMDMServerURL, host.MDM.ServerURL)
		require.Equal(t, "test", hosts[0].MDM.Name)

		host, err = ds.HostByIdentifier(ctx, h.UUID)
		require.NoError(t, err)
		require.Equal(t, c.expectedMDMStatus, host.MDM.EnrollmentStatus)
		require.Equal(t, c.expectedMDMServerURL, host.MDM.ServerURL)
		require.Equal(t, "test", hosts[0].MDM.Name)
	}
}

func testHostsListMunkiIssueID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var hostIDs []uint
	for i := 0; i < 3; i++ {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute * 2),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
		hostIDs = append(hostIDs, h.ID)
	}

	err := ds.SetOrUpdateMunkiInfo(ctx, hostIDs[0], "1.0.0", []string{"a", "b"}, []string{"c"})
	require.NoError(t, err)
	err = ds.SetOrUpdateMunkiInfo(ctx, hostIDs[1], "1.0.0", []string{"a"}, []string{"c"})
	require.NoError(t, err)
	err = ds.SetOrUpdateMunkiInfo(ctx, hostIDs[2], "1.0.0", []string{"a", "b"}, nil)
	require.NoError(t, err)
	err = ds.SetOrUpdateMunkiInfo(ctx, hostIDs[2], "1.0.0", []string{"a", "b"}, nil)
	require.NoError(t, err)

	var munkiIDs []uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &munkiIDs, `SELECT id FROM munki_issues WHERE name IN ('a', 'b', 'c') ORDER BY name`)
	})

	filter := fleet.TeamFilter{User: test.UserAdmin}

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MunkiIssueIDFilter: &munkiIDs[0]}, 3) // "a" error, all 3 hosts
	assert.Len(t, hosts, 3)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MunkiIssueIDFilter: &munkiIDs[1]}, 2) // "b" error, 2 hosts
	assert.Len(t, hosts, 2)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MunkiIssueIDFilter: &munkiIDs[2]}, 2) // "c" warning, 2 hosts
	assert.Len(t, hosts, 2)

	nonExisting := uint(123)
	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{MunkiIssueIDFilter: &nonExisting}, 0)
	assert.Len(t, hosts, 0)

	// insert issue names at the limit of what is allowed
	err = ds.SetOrUpdateMunkiInfo(ctx, hostIDs[0], "1.0.0", []string{strings.Repeat("Z", maxMunkiIssueNameLen)}, []string{strings.Repeat("💞", maxMunkiIssueNameLen)})
	require.NoError(t, err)

	issues, err := ds.GetHostMunkiIssues(ctx, hostIDs[0])
	require.NoError(t, err)
	require.Len(t, issues, 2)
	names := []string{issues[0].Name, issues[1].Name}
	require.ElementsMatch(t, []string{strings.Repeat("Z", maxMunkiIssueNameLen), strings.Repeat("💞", maxMunkiIssueNameLen)}, names)

	// test the truncation of overly long issue names, ascii and multi-byte utf8
	// Note that some unicode characters may not be supported properly by mysql
	// (e.g. 🐈 did fail even with truncation), but there's not much we can do
	// about it.
	err = ds.SetOrUpdateMunkiInfo(ctx, hostIDs[0], "1.0.0", []string{strings.Repeat("A", maxMunkiIssueNameLen+1)}, []string{strings.Repeat("☺", maxMunkiIssueNameLen+1)})
	require.NoError(t, err)

	issues, err = ds.GetHostMunkiIssues(ctx, hostIDs[0])
	require.NoError(t, err)
	require.Len(t, issues, 2)
	names = []string{issues[0].Name, issues[1].Name}
	require.ElementsMatch(t, []string{strings.Repeat("A", maxMunkiIssueNameLen), strings.Repeat("☺", maxMunkiIssueNameLen)}, names)
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
		h, err := ds.EnrollHost(context.Background(), false, tt.uuid, "", "", tt.nodeKey, &team.ID, 0)
		require.NoError(t, err)
		assert.NotZero(t, h.LastEnrolledAt)

		assert.Equal(t, tt.uuid, *h.OsqueryHostID)
		assert.Equal(t, tt.nodeKey, *h.NodeKey)

		// This host should be allowed to re-enroll immediately if cooldown is disabled
		_, err = ds.EnrollHost(context.Background(), false, tt.uuid, "", "", tt.nodeKey+"new", nil, 0)
		require.NoError(t, err)
		assert.NotZero(t, h.LastEnrolledAt)

		// This host should not be allowed to re-enroll immediately if cooldown is enabled
		_, err = ds.EnrollHost(context.Background(), false, tt.uuid, "", "", tt.nodeKey+"new", nil, 10*time.Second)
		require.Error(t, err)
		assert.NotZero(t, h.LastEnrolledAt)
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
		h, err := ds.EnrollHost(context.Background(), false, tt.uuid, "", "", tt.nodeKey, nil, 0)
		require.NoError(t, err)

		returned, err := ds.LoadHostByNodeKey(context.Background(), *h.NodeKey)
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
		h, err := ds.EnrollHost(context.Background(), false, tt.uuid, "", "", tt.nodeKey, nil, 0)
		require.NoError(t, err)

		_, err = ds.LoadHostByNodeKey(context.Background(), strings.ToUpper(*h.NodeKey))
		require.Error(t, err, "node key authentication should be case sensitive")
	}
}

func testHostsSearch(t *testing.T, ds *Datastore) {
	h1, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   ptr.String("1234"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "fo.local",
	})
	require.NoError(t, err)

	h2, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   ptr.String("5679"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "bar.local",
	})
	require.NoError(t, err)

	h3, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   ptr.String("99999"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("3"),
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
	err = ds.UpdateHost(context.Background(), h2)
	require.NoError(t, err)

	hits, err := ds.SearchHosts(context.Background(), filter, "99.100.101")
	require.NoError(t, err)
	require.Equal(t, 1, len(hits))

	hits, err = ds.SearchHosts(context.Background(), filter, "99.100.111")
	require.NoError(t, err)
	assert.Equal(t, 0, len(hits))

	h3.PrimaryIP = "99.100.101.104"
	err = ds.UpdateHost(context.Background(), h3)
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

	// Add email to mapping table
	_, err = ds.writer(context.Background()).ExecContext(context.Background(), `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
		hosts[0].ID, "a@b.c", "src1")
	require.NoError(t, err)
	// Verify search works
	hits, err = ds.SearchHosts(context.Background(), filter, "a@b.c")
	require.NoError(t, err)
	assert.Len(t, hits, 1)
}

func testSearchHostsWildCards(t *testing.T, ds *Datastore) {
	/*
		+------------------+
		|hostname          |
		+------------------+
		|Molly‘s MacbookPro|
		|Molly's MacbookPro|
		|Molly‘s MacbookPro|
		|Molly❛s MacbookPro|
		|Molly❜s MacbookPro|
		|Alex's MacbookPro |
		+------------------+

	*/
	hostnames := []string{
		"Molly‘s MacbookPro",
		"Molly's MacbookPro",
		"Molly‘s MacbookPro",
		"Molly❛s MacbookPro",
		"Molly❜s MacbookPro",
		"Alex's MacbookPro",
	}
	hostIDs := make([]uint, len(hostnames))
	for i, name := range hostnames {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String(strconv.Itoa(i)),
			UUID:            strconv.Itoa(i),
			Hostname:        name,
		})
		require.NoError(t, err)
		hostIDs[i] = h.ID
	}
	// hosts are returned in ORDER BY host.id DESC
	sort.Slice(hostIDs, func(i, j int) bool {
		return hostIDs[i] > hostIDs[j]
	})

	userAdmin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: userAdmin}

	type args struct {
		ctx        context.Context
		filter     fleet.TeamFilter
		matchQuery string
		omit       []uint
	}
	tests := []struct {
		name    string
		args    args
		want    []uint
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "empty match criteria should match everything",
			args: args{
				ctx:        context.Background(),
				matchQuery: "",
				filter:     filter,
				omit:       nil,
			},
			want:    hostIDs,
			wantErr: require.NoError,
		},
		{
			name: "searching for host with regular apostrophe should return just that result",
			args: args{
				ctx:        context.Background(),
				matchQuery: "Molly's",
				filter:     filter,
				omit:       nil,
			},
			want:    []uint{2}, // hosts.id autoincrement starts at 1
			wantErr: require.NoError,
		},
		{
			name: "excluding the host you are searching for should return an empty set",
			args: args{
				ctx:        context.Background(),
				matchQuery: "Molly's",
				filter:     filter,
				omit:       []uint{2},
			},
			want:    []uint{},
			wantErr: require.NoError,
		},
		{
			name: "searching for non-ascii characters should use wildcard searching",
			args: args{
				ctx:        context.Background(),
				matchQuery: "Molly‘s",
				filter:     filter,
				omit:       []uint{},
			},
			want:    []uint{5, 4, 3, 2, 1}, // all Molly_s endpoints should return
			wantErr: require.NoError,
		},
		{
			name: "searching for criteria that doesn't match anything should yield empty results",
			args: args{
				ctx:        context.Background(),
				matchQuery: "Foobar",
				filter:     filter,
				omit:       []uint{},
			},
			want:    []uint{},
			wantErr: require.NoError,
		},
		{
			name: "searching for criteria that doesn't match anything should yield empty results, omitting id that isn't in the potential result set shouldn't effect result",
			args: args{
				ctx:        context.Background(),
				matchQuery: "Foobar",
				filter:     filter,
				omit:       []uint{1},
			},
			want:    []uint{},
			wantErr: require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ds.SearchHosts(tt.args.ctx, tt.args.filter, tt.args.matchQuery, tt.args.omit...)
			tt.wantErr(t, err)
			resultHostIDs := make([]uint, len(got))
			for i, h := range got {
				resultHostIDs[i] = h.ID
			}
			assert.Equalf(t, tt.want, resultHostIDs, "SearchHosts(_, _, %v, %v)", tt.args.matchQuery, tt.args.omit)
		})
	}
}

func testHostsSearchLimit(t *testing.T, ds *Datastore) {
	filter := fleet.TeamFilter{User: test.UserAdmin}

	for i := 0; i < 15; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String(fmt.Sprintf("host%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
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

	summary, err := ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now(), nil, nil)
	require.NoError(t, err)
	assert.Nil(t, summary.TeamID)
	assert.Equal(t, uint(0), summary.TotalsHostsCount)
	assert.Equal(t, uint(0), summary.OnlineCount)
	assert.Equal(t, uint(0), summary.OfflineCount)
	assert.Equal(t, uint(0), summary.MIACount)
	assert.Equal(t, uint(0), summary.NewCount)
	assert.Nil(t, summary.LowDiskSpaceCount)

	// Online
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              1,
		OsqueryHostID:   ptr.String("1"),
		NodeKey:         ptr.String("1"),
		DetailUpdatedAt: mockClock.Now().Add(-30 * time.Second),
		LabelUpdatedAt:  mockClock.Now().Add(-30 * time.Second),
		PolicyUpdatedAt: mockClock.Now().Add(-30 * time.Second),
		SeenTime:        mockClock.Now().Add(-30 * time.Second),
		Platform:        "debian",
	})
	require.NoError(t, err)
	h.DistributedInterval = 15
	h.ConfigTLSRefresh = 30
	err = ds.UpdateHost(context.Background(), h)
	require.NoError(t, err)
	require.NoError(t, ds.SetOrUpdateHostDisksSpace(context.Background(), h.ID, 5, 5, 100.0))

	// Online
	h, err = ds.NewHost(context.Background(), &fleet.Host{
		ID:              2,
		OsqueryHostID:   ptr.String("2"),
		NodeKey:         ptr.String("2"),
		DetailUpdatedAt: mockClock.Now().Add(-1 * time.Minute),
		LabelUpdatedAt:  mockClock.Now().Add(-1 * time.Minute),
		PolicyUpdatedAt: mockClock.Now().Add(-1 * time.Minute),
		SeenTime:        mockClock.Now().Add(-1 * time.Minute),
		Platform:        "windows",
	})
	require.NoError(t, err)
	h.DistributedInterval = 60
	h.ConfigTLSRefresh = 3600
	err = ds.UpdateHost(context.Background(), h)
	require.NoError(t, err)
	require.NoError(t, ds.SetOrUpdateHostDisksSpace(context.Background(), h.ID, 50, 50, 100.0))

	// Offline
	h, err = ds.NewHost(context.Background(), &fleet.Host{
		ID:              3,
		OsqueryHostID:   ptr.String("3"),
		NodeKey:         ptr.String("3"),
		DetailUpdatedAt: mockClock.Now().Add(-1 * time.Hour),
		LabelUpdatedAt:  mockClock.Now().Add(-1 * time.Hour),
		PolicyUpdatedAt: mockClock.Now().Add(-1 * time.Hour),
		SeenTime:        mockClock.Now().Add(-1 * time.Hour),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	h.DistributedInterval = 300
	h.ConfigTLSRefresh = 300
	err = ds.UpdateHost(context.Background(), h)
	require.NoError(t, err)

	// MIA
	h, err = ds.NewHost(context.Background(), &fleet.Host{
		ID:              4,
		OsqueryHostID:   ptr.String("4"),
		NodeKey:         ptr.String("4"),
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

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, uint(4), summary.TotalsHostsCount)
	assert.Equal(t, uint(2), summary.OnlineCount)
	assert.Equal(t, uint(2), summary.OfflineCount)
	assert.Equal(t, uint(1), summary.MIACount)
	assert.Equal(t, uint(1), summary.Missing30DaysCount)
	assert.Equal(t, uint(4), summary.NewCount)
	assert.Nil(t, summary.LowDiskSpaceCount)
	assert.ElementsMatch(t, summary.Platforms, wantPlatforms)

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now().Add(1*time.Hour), nil, ptr.Int(10))
	require.NoError(t, err)
	assert.Equal(t, uint(4), summary.TotalsHostsCount)
	assert.Equal(t, uint(0), summary.OnlineCount)
	assert.Equal(t, uint(4), summary.OfflineCount) // offline count includes mia hosts as of Fleet 4.15
	assert.Equal(t, uint(1), summary.MIACount)
	assert.Equal(t, uint(1), summary.Missing30DaysCount)
	assert.Equal(t, uint(4), summary.NewCount)
	require.NotNil(t, summary.LowDiskSpaceCount)
	assert.Equal(t, uint(1), *summary.LowDiskSpaceCount)
	assert.ElementsMatch(t, summary.Platforms, wantPlatforms)

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now().Add(11*24*time.Hour), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, uint(1), summary.MIACount)
	assert.Equal(t, uint(1), summary.Missing30DaysCount)

	userObs := &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)}
	filter = fleet.TeamFilter{User: userObs}

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now().Add(1*time.Hour), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, uint(0), summary.TotalsHostsCount)

	filter.IncludeObserver = true
	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now().Add(1*time.Hour), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, uint(4), summary.TotalsHostsCount)

	userTeam1 := &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleAdmin}}}
	filter = fleet.TeamFilter{User: userTeam1}
	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now().Add(1*time.Hour), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, uint(1), summary.TotalsHostsCount)
	assert.Equal(t, uint(1), summary.MIACount)

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), fleet.TeamFilter{User: test.UserAdmin}, mockClock.Now(), ptr.String("linux"), nil)
	require.NoError(t, err)
	assert.Equal(t, uint(2), summary.TotalsHostsCount)

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), filter, mockClock.Now(), ptr.String("linux"), nil)
	require.NoError(t, err)
	assert.Equal(t, uint(1), summary.TotalsHostsCount)

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), fleet.TeamFilter{User: test.UserAdmin}, mockClock.Now(), ptr.String("darwin"), nil)
	require.NoError(t, err)
	assert.Equal(t, uint(1), summary.TotalsHostsCount)

	summary, err = ds.GenerateHostStatusStatistics(context.Background(), fleet.TeamFilter{User: test.UserAdmin}, mockClock.Now(), ptr.String("windows"), ptr.Int(60))
	require.NoError(t, err)
	assert.Equal(t, uint(1), summary.TotalsHostsCount)
	require.NotNil(t, summary.LowDiskSpaceCount)
	assert.Equal(t, uint(1), *summary.LowDiskSpaceCount)
}

func testHostsMarkSeen(t *testing.T, ds *Datastore) {
	mockClock := clock.NewMockClock()

	anHourAgo := mockClock.Now().Add(-1 * time.Hour).UTC()
	aDayAgo := mockClock.Now().Add(-24 * time.Hour).UTC()

	h1, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              1,
		OsqueryHostID:   ptr.String("1"),
		UUID:            "1",
		NodeKey:         ptr.String("1"),
		DetailUpdatedAt: aDayAgo,
		LabelUpdatedAt:  aDayAgo,
		PolicyUpdatedAt: aDayAgo,
		SeenTime:        aDayAgo,
	})
	require.NoError(t, err)

	{
		h1Verify, err := ds.Host(context.Background(), 1)
		require.NoError(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, aDayAgo, h1Verify.SeenTime, time.Second)
	}

	err = ds.MarkHostsSeen(context.Background(), []uint{h1.ID}, anHourAgo)
	require.NoError(t, err)

	{
		h1Verify, err := ds.Host(context.Background(), 1)
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
		OsqueryHostID:   ptr.String("1"),
		UUID:            "1",
		NodeKey:         ptr.String("1"),
		DetailUpdatedAt: aDayAgo,
		LabelUpdatedAt:  aDayAgo,
		PolicyUpdatedAt: aDayAgo,
		SeenTime:        aDayAgo,
	})
	require.NoError(t, err)

	h2, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              2,
		OsqueryHostID:   ptr.String("2"),
		UUID:            "2",
		NodeKey:         ptr.String("2"),
		DetailUpdatedAt: aDayAgo,
		LabelUpdatedAt:  aDayAgo,
		PolicyUpdatedAt: aDayAgo,
		SeenTime:        aDayAgo,
	})
	require.NoError(t, err)

	err = ds.MarkHostsSeen(context.Background(), []uint{h1.ID}, anHourAgo)
	require.NoError(t, err)

	{
		h1Verify, err := ds.Host(context.Background(), h1.ID)
		require.NoError(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, anHourAgo, h1Verify.SeenTime, time.Second)

		h2Verify, err := ds.Host(context.Background(), h2.ID)
		require.NoError(t, err)
		require.NotNil(t, h2Verify)
		assert.WithinDuration(t, aDayAgo, h2Verify.SeenTime, time.Second)
	}

	err = ds.MarkHostsSeen(context.Background(), []uint{h1.ID, h2.ID}, aSecondAgo)
	require.NoError(t, err)

	{
		h1Verify, err := ds.Host(context.Background(), h1.ID)
		require.NoError(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, aSecondAgo, h1Verify.SeenTime, time.Second)

		h2Verify, err := ds.Host(context.Background(), h2.ID)
		require.NoError(t, err)
		require.NotNil(t, h2Verify)
		assert.WithinDuration(t, aSecondAgo, h2Verify.SeenTime, time.Second)
	}
}

func testHostsCleanupIncoming(t *testing.T, ds *Datastore) {
	mockClock := clock.NewMockClock()

	h1, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              1,
		OsqueryHostID:   ptr.String("1"),
		UUID:            "1",
		NodeKey:         ptr.String("1"),
		DetailUpdatedAt: mockClock.Now(),
		LabelUpdatedAt:  mockClock.Now(),
		PolicyUpdatedAt: mockClock.Now(),
		SeenTime:        mockClock.Now(),
	})
	require.NoError(t, err)

	h2, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              2,
		OsqueryHostID:   ptr.String("2"),
		UUID:            "2",
		NodeKey:         ptr.String("2"),
		Hostname:        "foobar",
		OsqueryVersion:  "3.2.3",
		DetailUpdatedAt: mockClock.Now(),
		LabelUpdatedAt:  mockClock.Now(),
		PolicyUpdatedAt: mockClock.Now(),
		SeenTime:        mockClock.Now(),
	})
	require.NoError(t, err)

	_, err = ds.CleanupIncomingHosts(context.Background(), mockClock.Now().UTC())
	require.NoError(t, err)

	// Both hosts should still exist because they are new
	_, err = ds.Host(context.Background(), h1.ID)
	require.NoError(t, err)
	_, err = ds.Host(context.Background(), h2.ID)
	require.NoError(t, err)

	deleted, err := ds.CleanupIncomingHosts(context.Background(), mockClock.Now().Add(6*time.Minute).UTC())
	require.NoError(t, err)
	require.Equal(t, []uint{h1.ID}, deleted)

	// Now only the host with details should exist
	_, err = ds.Host(context.Background(), h1.ID)
	assert.NotNil(t, err)
	_, err = ds.Host(context.Background(), h2.ID)
	require.NoError(t, err)
}

func testHostIDsByIdentifier(t *testing.T, ds *Datastore) {
	hosts := make([]*fleet.Host, 10)
	for i := range hosts {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String(fmt.Sprintf("osq.host%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("nk.%d", i)),
			UUID:            fmt.Sprintf("uuid.%d", i),
			HardwareSerial:  fmt.Sprintf("hws.%d", i),
			Hostname:        fmt.Sprintf("foo.%d.local", i),
		})
		require.NoError(t, err)
		hosts[i] = h
	}

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{hosts[0].ID}))

	filter := fleet.TeamFilter{User: test.UserAdmin}
	hostsByIdentifier, err := ds.HostIDsByIdentifier(context.Background(), filter, []string{"foo.2.local", "foo.1.local", "foo.5.local"})
	require.NoError(t, err)
	sort.Slice(hostsByIdentifier, func(i, j int) bool { return hostsByIdentifier[i] < hostsByIdentifier[j] })
	assert.Equal(t, hostsByIdentifier, []uint{2, 3, 6})

	// by UUID
	hostsByIdentifier, err = ds.HostIDsByIdentifier(context.Background(), filter, []string{"uuid.0", "uuid.4"})
	require.NoError(t, err)
	require.Len(t, hostsByIdentifier, 2)
	assert.Equal(t, hostsByIdentifier[0], hosts[0].ID)
	assert.Equal(t, hostsByIdentifier[1], hosts[4].ID)

	// by HardwareSerial
	hostsByIdentifier, err = ds.HostIDsByIdentifier(context.Background(), filter, []string{"hws.2"})
	require.NoError(t, err)
	require.Len(t, hostsByIdentifier, 1)
	assert.Equal(t, hostsByIdentifier[0], hosts[2].ID)

	userObs := &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)}
	filter = fleet.TeamFilter{User: userObs}

	hostsByIdentifier, err = ds.HostIDsByIdentifier(context.Background(), filter, []string{"foo.2.local", "foo.1.local", "foo.5.local"})
	require.NoError(t, err)
	assert.Len(t, hostsByIdentifier, 0)

	filter.IncludeObserver = true
	hostsByIdentifier, err = ds.HostIDsByIdentifier(context.Background(), filter, []string{"foo.2.local", "foo.1.local", "foo.5.local"})
	require.NoError(t, err)
	assert.Len(t, hostsByIdentifier, 3)

	userTeam1 := &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleAdmin}}}
	filter = fleet.TeamFilter{User: userTeam1}

	hostsByIdentifier, err = ds.HostIDsByIdentifier(context.Background(), filter, []string{"foo.2.local", "foo.1.local", "foo.5.local"})
	require.NoError(t, err)
	assert.Len(t, hostsByIdentifier, 0)

	hostsByIdentifier, err = ds.HostIDsByIdentifier(context.Background(), filter, []string{"foo.0.local", "foo.1.local", "foo.5.local"})
	require.NoError(t, err)
	require.Len(t, hostsByIdentifier, 1)
	assert.Equal(t, hostsByIdentifier[0], hosts[0].ID)
}

func testLoadHostByNodeKeyLoadsDisk(t *testing.T, ds *Datastore) {
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("foobar"),
		NodeKey:         ptr.String("nodekey"),
		UUID:            "uuid",
		Hostname:        "foobar.local",
	})
	require.NoError(t, err)

	err = ds.UpdateHost(context.Background(), h)
	require.NoError(t, err)
	err = ds.SetOrUpdateHostDisksSpace(context.Background(), h.ID, 1.24, 42.0, 3.0)
	require.NoError(t, err)

	h, err = ds.LoadHostByNodeKey(context.Background(), "nodekey")
	require.NoError(t, err)
	assert.Equal(t, 1.24, h.GigsDiskSpaceAvailable)
	assert.Equal(t, 42.0, h.PercentDiskSpaceAvailable)
}

func testLoadHostByNodeKeyUsesStmt(t *testing.T, ds *Datastore) {
	_, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("foobar"),
		NodeKey:         ptr.String("nodekey"),
		UUID:            "uuid",
		Hostname:        "foobar.local",
	})
	require.NoError(t, err)
	_, err = ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("foobar2"),
		NodeKey:         ptr.String("nodekey2"),
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
		OsqueryHostID:   ptr.String("foobar"),
		NodeKey:         ptr.String("nodekey"),
		UUID:            "uuid",
		Hostname:        "foobar.local",
	})
	require.NoError(t, err)

	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, "foobar.local", h.Hostname)
	assert.Nil(t, h.Additional)

	// Additional not yet set
	h, err = ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Nil(t, h.Additional)

	// Add additional
	additional := json.RawMessage(`{"additional": "result"}`)
	require.NoError(t, ds.SaveHostAdditional(context.Background(), h.ID, &additional))

	// Additional should not be loaded for HostLite
	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, "foobar.local", h.Hostname)
	assert.Nil(t, h.Additional)

	h, err = ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, &additional, h.Additional)

	// Update besides additional. Additional should be unchanged.
	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	h.Hostname = "baz.local"
	err = ds.UpdateHost(context.Background(), h)
	require.NoError(t, err)

	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, "baz.local", h.Hostname)
	assert.Nil(t, h.Additional)

	h, err = ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, &additional, h.Additional)

	// Update additional
	additional = json.RawMessage(`{"other": "additional"}`)
	require.NoError(t, ds.SaveHostAdditional(context.Background(), h.ID, &additional))
	require.NoError(t, err)

	h, err = ds.HostLite(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, "baz.local", h.Hostname)
	assert.Nil(t, h.Additional)

	h, err = ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, &additional, h.Additional)
}

func testHostsByIdentifier(t *testing.T, ds *Datastore) {
	now := time.Now().UTC().Truncate(time.Second)
	for i := 1; i <= 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: now,
			LabelUpdatedAt:  now,
			PolicyUpdatedAt: now,
			SeenTime:        now,
			OsqueryHostID:   ptr.String(fmt.Sprintf("osquery_host_id_%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("node_key_%d", i)),
			UUID:            fmt.Sprintf("uuid_%d", i),
			Hostname:        fmt.Sprintf("hostname_%d", i),
			HardwareSerial:  fmt.Sprintf("serial_%d", i),
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
	assert.Equal(t, now.UTC(), h.SeenTime)

	h, err = ds.HostByIdentifier(context.Background(), "osquery_host_id_2")
	require.NoError(t, err)
	assert.Equal(t, uint(2), h.ID)
	assert.Equal(t, now.UTC(), h.SeenTime)

	h, err = ds.HostByIdentifier(context.Background(), "node_key_4")
	require.NoError(t, err)
	assert.Equal(t, uint(4), h.ID)
	assert.Equal(t, now.UTC(), h.SeenTime)

	h, err = ds.HostByIdentifier(context.Background(), "hostname_7")
	require.NoError(t, err)
	assert.Equal(t, uint(7), h.ID)
	assert.Equal(t, now.UTC(), h.SeenTime)

	h, err = ds.HostByIdentifier(context.Background(), "serial_9")
	require.NoError(t, err)
	assert.Equal(t, uint(9), h.ID)
	assert.Equal(t, now.UTC(), h.SeenTime)

	h, err = ds.HostByIdentifier(context.Background(), "foobar")
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, h)
}

func testHostLiteByIdentifierAndID(t *testing.T, ds *Datastore) {
	now := time.Now().UTC().Truncate(time.Second)
	for i := 1; i <= 10; i++ {
		_, err := ds.NewHost(
			context.Background(), &fleet.Host{
				DetailUpdatedAt: now,
				LabelUpdatedAt:  now,
				PolicyUpdatedAt: now,
				SeenTime:        now,
				OsqueryHostID:   ptr.String(fmt.Sprintf("osquery_host_id_%d", i)),
				NodeKey:         ptr.String(fmt.Sprintf("node_key_%d", i)),
				UUID:            fmt.Sprintf("uuid_%d", i),
				Hostname:        fmt.Sprintf("hostname_%d", i),
				HardwareSerial:  fmt.Sprintf("serial_%d", i),
			},
		)
		require.NoError(t, err)
	}

	var (
		h   *fleet.HostLite
		err error
	)
	identifier := "uuid_1"
	h, err = ds.HostLiteByIdentifier(context.Background(), identifier)
	require.NoError(t, err)
	assert.Equal(t, uint(1), h.ID)
	assert.Equal(t, now.UTC(), h.SeenTime)

	// Also test fetching host by ID
	h, err = ds.HostLiteByID(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, identifier, h.UUID)

	h, err = ds.HostLiteByIdentifier(context.Background(), "osquery_host_id_2")
	require.NoError(t, err)
	assert.Equal(t, uint(2), h.ID)
	assert.Equal(t, now.UTC(), h.SeenTime)

	h, err = ds.HostLiteByIdentifier(context.Background(), "node_key_4")
	require.NoError(t, err)
	assert.Equal(t, uint(4), h.ID)
	assert.Equal(t, now.UTC(), h.SeenTime)

	h, err = ds.HostLiteByIdentifier(context.Background(), "hostname_7")
	require.NoError(t, err)
	assert.Equal(t, uint(7), h.ID)
	assert.Equal(t, now.UTC(), h.SeenTime)

	h, err = ds.HostLiteByIdentifier(context.Background(), "serial_9")
	require.NoError(t, err)
	assert.Equal(t, uint(9), h.ID)
	assert.Equal(t, now.UTC(), h.SeenTime)

	h, err = ds.HostLiteByIdentifier(context.Background(), "foobar")
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, h)

	h, err = ds.HostLiteByIdentifier(context.Background(), "")
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, h)

	h, err = ds.HostLiteByID(context.Background(), 0)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, h)
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

	// Update batch size
	addHostsToTeamBatchSizeOrig := addHostsToTeamBatchSize
	t.Cleanup(
		func() {
			addHostsToTeamBatchSize = addHostsToTeamBatchSizeOrig
		},
	)
	addHostsToTeamBatchSize = 2

	require.NoError(t, ds.AddHostsToTeam(context.Background(), nil, []uint{1, 2, 3, 4}))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{5, 6, 7, 8, 9, 10}))

	for i := 1; i <= 10; i++ {
		host, err := ds.Host(context.Background(), uint(i))
		require.NoError(t, err)
		var expectedID *uint
		switch { //nolint:gocritic // ignore singleCaseSwitch
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
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	err = ds.UpdateHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
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
	hostUsers := []fleet.HostUser{u1, u2}
	err = ds.SaveHostUsers(context.Background(), host.ID, hostUsers)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, host.Users, 2)
	test.ElementsMatchSkipID(t, host.Users, []fleet.HostUser{u1, u2})

	// remove u1 user
	hostUsers = []fleet.HostUser{u2}
	err = ds.SaveHostUsers(context.Background(), host.ID, hostUsers)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, host.Users, 1)
	assert.Equal(t, host.Users[0].Uid, u2.Uid)

	// readd u1 but with a different shell
	u1.Shell = "/some/new/shell"
	hostUsers = []fleet.HostUser{u1, u2}
	err = ds.SaveHostUsers(context.Background(), host.ID, hostUsers)
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
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	err = ds.UpdateHost(context.Background(), host)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
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
	hostUsers := []fleet.HostUser{u1, u2}

	err = ds.SaveHostUsers(context.Background(), host.ID, hostUsers)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, host.Users, 2)
	test.ElementsMatchSkipID(t, host.Users, []fleet.HostUser{u1, u2})

	// remove u1 user
	hostUsers = []fleet.HostUser{u2}
	err = ds.SaveHostUsers(context.Background(), host.ID, hostUsers)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, host.Users, 1)
	assert.Equal(t, host.Users[0].Uid, u2.Uid)
}

func addHostSeenLast(t *testing.T, ds fleet.Datastore, i, days int) *fleet.Host {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Duration(days) * 24 * time.Hour),
		OsqueryHostID:   ptr.String(fmt.Sprintf("%d", i)),
		NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
		UUID:            fmt.Sprintf("%d", i),
		Hostname:        fmt.Sprintf("foo.local%d", i),
		PrimaryIP:       fmt.Sprintf("192.168.1.%d", i),
		PrimaryMac:      fmt.Sprintf("30-65-EC-6F-C4-5%d", i),
	})
	require.NoError(t, err)
	require.NotNil(t, host)
	return host
}

func testHostsTotalAndUnseenSince(t *testing.T, ds *Datastore) {
	host1 := addHostSeenLast(t, ds, 1, 0)

	total, unseen, err := ds.TotalAndUnseenHostsSince(context.Background(), nil, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, unseen, 0)

	host2 := addHostSeenLast(t, ds, 2, 2)
	host3 := addHostSeenLast(t, ds, 3, 4)

	total, unseen, err = ds.TotalAndUnseenHostsSince(context.Background(), nil, 1)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, unseen, 2)

	// host not counted as unseen if less than a full 24 hours has passed
	_, err = ds.writer(context.Background()).ExecContext(context.Background(), `UPDATE host_seen_times SET seen_time = ? WHERE host_id = 2`, time.Now().Add(-1*time.Duration(1)*86399*time.Second))
	require.NoError(t, err)

	total, unseen, err = ds.TotalAndUnseenHostsSince(context.Background(), nil, 1)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, unseen, 1)

	// host counted as unseen if more than 24 hours has passed
	_, err = ds.writer(context.Background()).ExecContext(context.Background(), `UPDATE host_seen_times SET seen_time = ? WHERE host_id = 2`, time.Now().Add(-1*time.Duration(1)*86401*time.Second))
	require.NoError(t, err)

	total, unseen, err = ds.TotalAndUnseenHostsSince(context.Background(), nil, 1)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, unseen, 2)
	assert.Equal(t, host2.ID, unseen[0])
	assert.Equal(t, host3.ID, unseen[1])

	// Test team hosts
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	total, unseen, err = ds.TotalAndUnseenHostsSince(context.Background(), &team1.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, unseen, 0)

	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID, host3.ID}))
	total, unseen, err = ds.TotalAndUnseenHostsSince(context.Background(), &team1.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, unseen, 1)
	assert.Equal(t, host3.ID, unseen[0])
}

func testHostsListByPolicy(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	q := test.NewQuery(t, ds, nil, "query1", "select 1", 0, true)
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
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
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
	host3 := hosts[2]
	_, err := ds.UpdateHostSoftware(context.Background(), host1.ID, software)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(context.Background(), host2.ID, software)
	require.NoError(t, err)
	// host 3 only has foo v0.0.3
	_, err = ds.UpdateHostSoftware(context.Background(), host3.ID, software[1:2])
	require.NoError(t, err)

	// reconcile software, will sync software titles
	err = ds.ReconcileSoftwareTitles(context.Background())
	require.NoError(t, err)

	var fooV002ID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &fooV002ID,
			"SELECT id FROM software WHERE name = ? AND source = ? AND version = ?", "foo", "chrome_extensions", "0.0.2")
	})

	var fooTitleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &fooTitleID,
			"SELECT id FROM software_titles WHERE name = ? AND source = ?", "foo", "chrome_extensions")
	})

	require.NoError(t, ds.LoadHostSoftware(context.Background(), host1, false))
	require.NoError(t, ds.LoadHostSoftware(context.Background(), host2, false))

	// software_id is foo v0.0.2
	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{SoftwareIDFilter: &fooV002ID}, 2)
	require.Len(t, hosts, 2)
	got := []uint{hosts[0].ID, hosts[1].ID}
	require.ElementsMatch(t, []uint{host1.ID, host2.ID}, got)

	// software_version_id is foo v0.0.2 (works exacty the same)
	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{SoftwareVersionIDFilter: &fooV002ID}, 2)
	require.Len(t, hosts, 2)
	got = []uint{hosts[0].ID, hosts[1].ID}
	require.ElementsMatch(t, []uint{host1.ID, host2.ID}, got)

	// unknown software_id
	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{SoftwareIDFilter: ptr.Uint(fooV002ID + 100)}, 0)
	require.Len(t, hosts, 0)

	// unknown software_version_id
	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{SoftwareVersionIDFilter: ptr.Uint(fooV002ID + 100)}, 0)
	require.Len(t, hosts, 0)

	// software_title_id is foo (any version)
	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{SoftwareTitleIDFilter: &fooTitleID}, 3)
	require.Len(t, hosts, 3)
	got = []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID}
	require.ElementsMatch(t, []uint{host1.ID, host2.ID, host3.ID}, got)

	// unknown software_title_id
	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{SoftwareTitleIDFilter: ptr.Uint(fooTitleID + 100)}, 0)
	require.Len(t, hosts, 0)
}

func testHostsListBySoftwareChangedAt(t *testing.T, ds *Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 10)
	require.Equal(t, hosts[0].SoftwareUpdatedAt, hosts[0].CreatedAt)

	host, err := ds.Host(context.Background(), hosts[0].ID)
	require.NoError(t, err)
	require.Equal(t, host.SoftwareUpdatedAt, host.CreatedAt)

	host, err = ds.HostByIdentifier(context.Background(), *hosts[0].OsqueryHostID)
	require.NoError(t, err)
	require.Equal(t, host.SoftwareUpdatedAt, host.CreatedAt)

	foundHosts, err := ds.SearchHosts(context.Background(), filter, "foo.local0")
	require.NoError(t, err)
	require.Len(t, foundHosts, 1)
	require.Equal(t, foundHosts[0].SoftwareUpdatedAt, foundHosts[0].CreatedAt)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "deb_packages", BundleIdentifier: "com.some.identifier"},
	}
	host1 := hosts[2]
	host2 := hosts[7]

	// need to sleep because timestamps have a 1 second resolution, otherwise it'll be a flaky test
	time.Sleep(1 * time.Second)
	_, err = ds.UpdateHostSoftware(context.Background(), host1.ID, software)
	require.NoError(t, err)
	time.Sleep(1 * time.Second)
	_, err = ds.UpdateHostSoftware(context.Background(), host2.ID, software)
	require.NoError(t, err)

	// if we update the host again with the same software, host2 will still be the one with the latest updated at
	// because nothing changed
	_, err = ds.UpdateHostSoftware(context.Background(), host1.ID, software)
	require.NoError(t, err)

	hosts, err = ds.ListHosts(context.Background(), filter, fleet.HostListOptions{
		ListOptions: fleet.ListOptions{OrderKey: "software_updated_at", OrderDirection: fleet.OrderDescending},
	})
	require.NoError(t, err)

	require.Len(t, hosts, 10)
	require.Equal(t, host2.ID, hosts[0].ID)
	require.Equal(t, host1.ID, hosts[1].ID)

	host, err = ds.Host(context.Background(), hosts[0].ID)
	require.NoError(t, err)
	require.Greater(t, host.SoftwareUpdatedAt, host.CreatedAt)

	host, err = ds.HostByIdentifier(context.Background(), *hosts[0].OsqueryHostID)
	require.NoError(t, err)
	require.Greater(t, host.SoftwareUpdatedAt, host.CreatedAt)

	foundHosts, err = ds.SearchHosts(context.Background(), filter, "foo.local2")
	require.NoError(t, err)
	require.Len(t, foundHosts, 1)
	require.Greater(t, foundHosts[0].SoftwareUpdatedAt, foundHosts[0].CreatedAt)
}

func testHostsListByOperatingSystemID(t *testing.T, ds *Datastore) {
	// seed hosts
	hostsByID := make(map[uint]fleet.Host)
	for i := 0; i < 9; i++ {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
		hostsByID[h.ID] = *h
	}

	// seed operating systems
	seeds := []fleet.OperatingSystem{
		{Name: "CentOS", Version: "8.0.0", Platform: "rhel", KernelVersion: "5.10.76-linuxkit"},
		{Name: "Ubuntu", Version: "20.4.0 LTS", Platform: "ubuntu", KernelVersion: "5.10.76-linuxkit"},
		{Name: "Ubuntu", Version: "20.5.0 LTS", Platform: "ubuntu", KernelVersion: "5.10.76-linuxkit"},
	}
	var hostIDsCentOS []uint
	var hostsIDsUbuntu20_4 []uint
	var hostsIDsUbuntu20_5 []uint
	for _, h := range hostsByID {
		r := h.ID % 3
		err := ds.UpdateHostOperatingSystem(context.Background(), h.ID, seeds[r])
		require.NoError(t, err)
		switch r {
		case 0:
			hostIDsCentOS = append(hostIDsCentOS, h.ID)
		case 1:
			hostsIDsUbuntu20_4 = append(hostsIDsUbuntu20_4, h.ID)
		case 2:
			hostsIDsUbuntu20_5 = append(hostsIDsUbuntu20_5, h.ID)
		}
	}

	storedOSs, err := ds.ListOperatingSystems(context.Background())
	require.NoError(t, err)
	require.Len(t, storedOSs, 3)
	storedOSByNameVers := make(map[string]fleet.OperatingSystem)
	for _, os := range storedOSs {
		storedOSByNameVers[fmt.Sprintf("%s %s", os.Name, os.Version)] = os
	}

	// filter by id of Ubuntu 20.4.0
	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{OSIDFilter: ptr.Uint(storedOSByNameVers["Ubuntu 20.4.0 LTS"].ID)}, 3)
	for _, h := range hosts {
		require.Contains(t, hostsIDsUbuntu20_4, h.ID)
	}

	// filter by id of Ubuntu 20.5.0
	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{OSIDFilter: ptr.Uint(storedOSByNameVers["Ubuntu 20.5.0 LTS"].ID)}, 3)
	for _, h := range hosts {
		require.Contains(t, hostsIDsUbuntu20_5, h.ID)
	}

	// filter by id of CentOS 8.0.0
	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{OSIDFilter: ptr.Uint(storedOSByNameVers["CentOS 8.0.0"].ID)}, 3)
	for _, h := range hosts {
		require.Contains(t, hostIDsCentOS, h.ID)
	}
}

func testHostsListByOSNameAndVersion(t *testing.T, ds *Datastore) {
	// seed hosts
	hostsByID := make(map[uint]fleet.Host)
	for i := 0; i < 9; i++ {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
		hostsByID[h.ID] = *h
	}

	// seed operating systems
	seeds := []fleet.OperatingSystem{
		{Name: "macOS", Version: "12.5.1", Arch: "x86_64", Platform: "darwin", KernelVersion: "21.4.0"},
		{Name: "macOS", Version: "12.5.1", Arch: "arm64", Platform: "darwin", KernelVersion: "21.4.0"},
		{Name: "macOS", Version: "12.5.2", Arch: "x86_64", Platform: "darwin", KernelVersion: "21.4.0"},
	}
	var hostIDs_12_5_1_X86 []uint
	var hostIDs_12_5_1_ARM []uint
	var hostIDs_12_5_2_X86 []uint
	for _, h := range hostsByID {
		r := h.ID % 3
		err := ds.UpdateHostOperatingSystem(context.Background(), h.ID, seeds[r])
		require.NoError(t, err)
		switch r {
		case 0:
			hostIDs_12_5_1_X86 = append(hostIDs_12_5_1_X86, h.ID)
		case 1:
			hostIDs_12_5_1_ARM = append(hostIDs_12_5_1_ARM, h.ID)
		case 2:
			hostIDs_12_5_2_X86 = append(hostIDs_12_5_2_X86, h.ID)
		}
	}

	storedOSs, err := ds.ListOperatingSystems(context.Background())
	require.NoError(t, err)
	require.Len(t, storedOSs, 3)
	storedOSByNameVersArch := make(map[string]fleet.OperatingSystem)
	for _, os := range storedOSs {
		storedOSByNameVersArch[fmt.Sprintf("%s %s %s", os.Name, os.Version, os.Arch)] = os
	}

	// filter by id of macOS 12.5.1 (x86_64)
	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{OSIDFilter: ptr.Uint(storedOSByNameVersArch["macOS 12.5.1 x86_64"].ID)}, 3)
	for _, h := range hosts {
		require.Contains(t, hostIDs_12_5_1_X86, h.ID)
	}

	// filter by id of macOS 12.5.1 (arm64)
	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{OSIDFilter: ptr.Uint(storedOSByNameVersArch["macOS 12.5.1 arm64"].ID)}, 3)
	for _, h := range hosts {
		require.Contains(t, hostIDs_12_5_1_ARM, h.ID)
	}

	// filter by name and version of macOS 12.5.1 includes both x86_64 and arm64 architectures
	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{OSNameFilter: ptr.String("macOS"), OSVersionFilter: ptr.String("12.5.1")}, 6)
	var testHostIDs []uint
	testHostIDs = append(testHostIDs, hostIDs_12_5_1_X86...)
	testHostIDs = append(testHostIDs, hostIDs_12_5_1_ARM...)
	for _, h := range hosts {
		require.Contains(t, testHostIDs, h.ID)
	}

	// filter by id of macOS 12.5.2 (x86_64)
	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{OSIDFilter: ptr.Uint(storedOSByNameVersArch["macOS 12.5.2 x86_64"].ID)}, 3)
	for _, h := range hosts {
		require.Contains(t, hostIDs_12_5_2_X86, h.ID)
	}

	// filter by name and version of macOS 12.5.2
	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{OSNameFilter: ptr.String("macOS"), OSVersionFilter: ptr.String("12.5.2")}, 3)
	for _, h := range hosts {
		require.Contains(t, hostIDs_12_5_2_X86, h.ID)
	}
}

func testHostsListByVulnerability(t *testing.T, ds *Datastore) {
	// seed hosts
	var hosts []*fleet.Host
	for i := 0; i < 9; i++ {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
		hosts = append(hosts, h)
	}

	// seed software
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
	}

	// add software to 5 hosts
	var swVulnHostIDs []uint
	for i := 0; i < 5; i++ {
		_, err := ds.UpdateHostSoftware(context.Background(), hosts[i].ID, software)
		require.NoError(t, err)
		swVulnHostIDs = append(swVulnHostIDs, hosts[i].ID)
	}

	// seed software vulnerabilities
	vuln := fleet.SoftwareVulnerability{
		CVE:        "CVE-2021-1234",
		SoftwareID: 1,
	}

	_, err := ds.InsertSoftwareVulnerability(context.Background(), vuln, fleet.NVDSource)
	require.NoError(t, err)

	list, err := ds.ListHosts(context.Background(), fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{VulnerabilityFilter: ptr.String("CVE-2021-1234")})
	require.NoError(t, err)
	require.Len(t, list, 5)
	for _, h := range list {
		require.Contains(t, swVulnHostIDs, h.ID)
	}

	// update 2 host operating system
	os := fleet.OperatingSystem{
		Name:          "Ubuntu",
		Version:       "20.4.0 LTS",
		Arch:          "x86_64",
		Platform:      "ubuntu",
		KernelVersion: "5.10.76-linuxkit",
	}
	err = ds.UpdateHostOperatingSystem(context.Background(), hosts[0].ID, os)
	require.NoError(t, err)
	err = ds.UpdateHostOperatingSystem(context.Background(), hosts[1].ID, os)
	require.NoError(t, err)

	// seed os vulnerability
	osVulns := []fleet.OSVulnerability{
		{
			OSID: 1,
			CVE:  "CVE-2021-1235",
		},
	}
	_, err = ds.InsertOSVulnerabilities(context.Background(), osVulns, fleet.NVDSource)
	require.NoError(t, err)

	list, err = ds.ListHosts(context.Background(), fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{VulnerabilityFilter: ptr.String("CVE-2021-1235")})
	require.NoError(t, err)
	require.Len(t, list, 2)
	for _, h := range list {
		require.Contains(t, []uint{hosts[0].ID, hosts[1].ID}, h.ID)
	}
}

func testHostsListMacOSSettingsDiskEncryptionStatus(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// seed hosts
	var hosts []*fleet.Host
	for i := 0; i < 10; i++ {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
		hosts = append(hosts, h)
		nanoEnrollAndSetHostMDMData(t, ds, h, false)
	}

	// set up data
	noTeamFVProfile, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("filevault-1", "com.fleetdm.fleet.mdm.filevault", 0))
	require.NoError(t, err)

	// verifying status
	upsertHostCPs([]*fleet.Host{hosts[0], hosts[1]}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	oneMinuteAfterThreshold := time.Now().Add(+1 * time.Minute)
	createDiskEncryptionRecord(ctx, ds, t, hosts[0].ID, "key-1", true, oneMinuteAfterThreshold)
	createDiskEncryptionRecord(ctx, ds, t, hosts[1].ID, "key-1", true, oneMinuteAfterThreshold)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 0)

	// action required status
	upsertHostCPs(
		[]*fleet.Host{hosts[2], hosts[3]},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		&fleet.MDMDeliveryVerifying, ctx, ds, t,
	)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[2].ID}, false, oneMinuteAfterThreshold)
	require.NoError(t, err)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[3].ID}, false, oneMinuteAfterThreshold)
	require.NoError(t, err)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 0)

	// enforcing status

	// host profile status is `pending`
	upsertHostCPs(
		[]*fleet.Host{hosts[4]},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		&fleet.MDMDeliveryPending, ctx, ds, t,
	)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 1)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 0)

	// host profile status does not exist
	upsertHostCPs(
		[]*fleet.Host{hosts[5]},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		nil, ctx, ds, t,
	)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 0)

	// host profile status is verifying but decryptable key field does not exist
	upsertHostCPs(
		[]*fleet.Host{hosts[6]},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		&fleet.MDMDeliveryPending, ctx, ds, t,
	)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[6].ID}, false, oneMinuteAfterThreshold)
	require.NoError(t, err)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 3)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 0)

	// failed status
	upsertHostCPs([]*fleet.Host{hosts[7], hosts[8]}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryFailed, ctx, ds, t)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 3)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 0)

	// removing enforcement status
	upsertHostCPs([]*fleet.Host{hosts[9]}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMOperationTypeRemove, &fleet.MDMDeliveryPending, ctx, ds, t)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 3)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 1)

	// verified status
	upsertHostCPs([]*fleet.Host{hosts[0]}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerified, ctx, ds, t)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 1)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 1)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 3)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 1)
}

func testHostsListFailingPolicies(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	q := test.NewQuery(t, ds, nil, "query1", "select 1", 0, true)
	q2 := test.NewQuery(t, ds, nil, "query2", "select 1", 0, true)
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
	assert.Zero(t, *h1.HostIssues.CriticalVulnerabilitiesCount)
	assert.Zero(t, h1.HostIssues.TotalIssuesCount)
	assert.Zero(t, h2.HostIssues.FailingPoliciesCount)
	assert.Zero(t, *h2.HostIssues.CriticalVulnerabilitiesCount)
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

	checkHostIssuesWithOpts(t, ds, filter, h1.ID, fleet.HostListOptions{DisableIssues: true}, 0)
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
	rows, err := ds.writer(context.Background()).Query("show engine innodb status")
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
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
		hosts = append(hosts, h)
	}
	h1 := hosts[0]
	h2 := hosts[1]

	q := test.NewQuery(t, ds, nil, "query1", "select 1", 0, true)
	p, err := ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		QueryID: &q.ID,
	})
	require.NoError(t, err)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h1, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now(), false))

	prevRead := getReads(t, ds)
	h1WithExtras, err := ds.Host(context.Background(), h1.ID)
	require.NoError(t, err)
	newRead := getReads(t, ds)
	withExtraRowReads := newRead - prevRead

	prevRead = getReads(t, ds)
	h1WithoutExtras, err := ds.Host(context.Background(), h1.ID)
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

func checkHostIssues(t *testing.T, ds *Datastore, hosts []*fleet.Host, filter fleet.TeamFilter, hid uint, expected uint64) {
	checkHostIssuesWithOpts(t, ds, filter, hid, fleet.HostListOptions{}, expected)
}

func checkHostIssuesWithOpts(
	t *testing.T, ds *Datastore, filter fleet.TeamFilter, hid uint, opts fleet.HostListOptions, expected uint64,
) {
	hosts := listHostsCheckCount(t, ds, filter, opts, 10)
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

	if opts.DisableIssues {
		return
	}

	hostById, err := ds.Host(context.Background(), hid)
	require.NoError(t, err)
	assert.Equal(t, expected, hostById.HostIssues.FailingPoliciesCount)
	assert.Equal(t, expected, hostById.HostIssues.TotalIssuesCount)
}

func testHostsUpdateTonsOfUsers(t *testing.T, ds *Datastore) {
	host1, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   ptr.String("1"),
	})
	require.NoError(t, err)
	require.NotNil(t, host1)

	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "foo2.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   ptr.String("2"),
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
				Shell:     "shell",
			}
			u2 := fleet.HostUser{
				Uid:       43,
				Username:  "user2",
				Type:      "aaa",
				GroupName: "group",
				Shell:     "shell",
			}
			host1Users := []fleet.HostUser{u1, u2}
			host1.SeenTime = time.Now()
			host1Software := []fleet.Software{
				{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
				{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			}
			host1Additional := json.RawMessage(`{"some":"thing"}`)

			if err = ds.UpdateHost(context.Background(), host1); err != nil {
				errCh <- err
				return
			}
			if err = ds.SaveHostUsers(context.Background(), host1.ID, host1Users); err != nil {
				errCh <- err
				return
			}
			if _, err = ds.UpdateHostSoftware(context.Background(), host1.ID, host1Software); err != nil {
				errCh <- err
				return
			}
			if err = ds.SaveHostAdditional(context.Background(), host1.ID, &host1Additional); err != nil {
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
				Shell:     "shell",
			}
			u2 := fleet.HostUser{
				Uid:       98,
				Username:  "user2",
				Type:      "aaa",
				GroupName: "group",
				Shell:     "shell",
			}
			host2Users := []fleet.HostUser{u1, u2}
			host2.SeenTime = time.Now()
			host2Software := []fleet.Software{
				{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
				{Name: "foo4", Version: "0.0.3", Source: "chrome_extensions"},
			}
			host2Additional := json.RawMessage(`{"some":"thing"}`)

			if err = ds.UpdateHost(context.Background(), host2); err != nil {
				errCh <- err
				return
			}
			if err = ds.SaveHostUsers(context.Background(), host2.ID, host2Users); err != nil {
				errCh <- err
				return
			}
			if _, err = ds.UpdateHostSoftware(context.Background(), host2.ID, host2Software); err != nil {
				errCh <- err
				return
			}
			if err = ds.SaveHostAdditional(context.Background(), host2.ID, &host2Additional); err != nil {
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

	ticker := time.NewTicker(30 * time.Second)
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
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   ptr.String("1"),
	})
	require.NoError(t, err)
	require.NotNil(t, host1)

	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   ptr.String("2"),
	})
	require.NoError(t, err)
	require.NotNil(t, host2)

	pack1 := test.NewPack(t, ds, "test1")
	query1 := test.NewQuery(t, ds, nil, "time", "select * from time", 0, true)
	squery1 := test.NewScheduledQuery(t, ds, pack1.ID, query1.ID, 30, true, true, "time-scheduled")

	pack2 := test.NewPack(t, ds, "test2")
	query2 := test.NewQuery(t, ds, nil, "time2", "select * from time", 0, true)
	squery2 := test.NewScheduledQuery(t, ds, pack2.ID, query2.ID, 30, true, true, "time-scheduled")

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	saveHostRandomStats := func(host *fleet.Host) error {
		packStats := []fleet.PackStats{
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
						Executions:         uint64(rand.Intn(1000)),
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
						Executions:         uint64(rand.Intn(1000)),
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
		return ds.SaveHostPackStats(context.Background(), host.TeamID, host.ID, packStats)
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

	ac.HostExpirySettings.HostExpiryEnabled = false
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
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 10)
	require.Len(t, hosts, 10)

	_, err = ds.CleanupExpiredHosts(context.Background())
	require.NoError(t, err)

	// host expiration is still disabled
	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 10)
	require.Len(t, hosts, 10)

	// once enabled, it works
	ac.HostExpirySettings.HostExpiryEnabled = true
	err = ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)

	deleted, err := ds.CleanupExpiredHosts(context.Background())
	require.NoError(t, err)
	require.Len(t, deleted, 5)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 5)
	require.Len(t, hosts, 5)

	// And it doesn't remove more than it should
	deleted, err = ds.CleanupExpiredHosts(context.Background())
	require.NoError(t, err)
	require.Len(t, deleted, 0)

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 5)
	require.Len(t, hosts, 5)
}

func testTeamHostsExpiration(t *testing.T, ds *Datastore) {
	// Set global host expiry windows
	const hostExpiryWindow = 70
	const team1HostExpiryWindow = 30
	const team2HostExpiryWindow = 170
	ac, err := ds.AppConfig(context.Background())
	require.NoError(t, err)
	ac.HostExpirySettings.HostExpiryEnabled = false
	ac.HostExpirySettings.HostExpiryWindow = hostExpiryWindow
	err = ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)

	createHost := func(id int, seenTime time.Time) {
		_, err := ds.NewHost(
			context.Background(), &fleet.Host{
				DetailUpdatedAt: time.Now(),
				LabelUpdatedAt:  time.Now(),
				PolicyUpdatedAt: time.Now(),
				SeenTime:        seenTime,
				OsqueryHostID:   ptr.String(strconv.Itoa(id)),
				NodeKey:         ptr.String(fmt.Sprintf("%d", id)),
				UUID:            fmt.Sprintf("%d", id),
				Hostname:        fmt.Sprintf("foo.local%d", id),
			},
		)
		require.NoError(t, err)
	}

	// Team 1 hosts (1, 2, 3)
	seenTime := time.Now().Add(time.Duration(-1*(team1HostExpiryWindow)*24)*time.Hour - time.Hour)         // 1 hour over expiry window
	seenRecentlyTime := time.Now().Add(time.Duration(-1*(team1HostExpiryWindow)*24)*time.Hour + time.Hour) // 1 hour under expiry window
	createHost(1, seenTime)
	createHost(2, seenTime)
	createHost(3, seenRecentlyTime)
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{1, 2, 3}))

	// Team 2 hosts (4, 5, 6)
	seenTime = time.Now().Add(time.Duration(-1*(team2HostExpiryWindow+1)*24) * time.Hour)
	seenRecentlyTime = time.Now().Add(time.Duration(-1*(team2HostExpiryWindow-1)*24) * time.Hour)
	createHost(4, seenRecentlyTime)
	createHost(5, time.Now())
	createHost(6, seenTime)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team2.ID, []uint{4, 5, 6}))

	// Team 3 hosts (7, 8, 9)
	seenTime = time.Now().Add(time.Duration(-1*(hostExpiryWindow+1)*24) * time.Hour)
	seenRecentlyTime = time.Now().Add(time.Duration(-1*(hostExpiryWindow-1)*24) * time.Hour)
	createHost(7, time.Now())
	createHost(8, seenTime)
	createHost(9, seenTime)
	team3, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team3"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team3.ID, []uint{7, 8, 9}))

	// Global hosts (10, 11)
	createHost(10, seenRecentlyTime)
	createHost(11, seenTime)

	filter := fleet.TeamFilter{User: test.UserAdmin}
	_ = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 11)
	_, err = ds.CleanupExpiredHosts(context.Background())
	require.NoError(t, err)
	// host expiration is still disabled
	_ = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 11)
	var count []int
	err = ds.writer(context.Background()).Select(&count, "SELECT COUNT(*) FROM host_seen_times")
	require.NoError(t, err)
	require.Len(t, count, 1)
	assert.Equal(t, 11, count[0])

	// once enabled, it works
	ac.HostExpirySettings.HostExpiryEnabled = true
	err = ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)

	team1.Config.HostExpirySettings.HostExpiryEnabled = true
	team1.Config.HostExpirySettings.HostExpiryWindow = team1HostExpiryWindow
	team1, err = ds.SaveTeam(context.Background(), team1)
	assert.Equal(t, team1HostExpiryWindow, team1.Config.HostExpirySettings.HostExpiryWindow)
	require.NoError(t, err)

	team2.Config.HostExpirySettings.HostExpiryEnabled = true
	team2.Config.HostExpirySettings.HostExpiryWindow = team2HostExpiryWindow
	team2, err = ds.SaveTeam(context.Background(), team2)
	assert.Equal(t, team2HostExpiryWindow, team2.Config.HostExpirySettings.HostExpiryWindow)
	require.NoError(t, err)

	deleted, err := ds.CleanupExpiredHosts(context.Background())
	require.NoError(t, err)
	assert.Len(t, deleted, 6)
	assert.ElementsMatch(t, []uint{1, 2, 6, 8, 9, 11}, deleted)
	_ = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 5)
	count = nil
	err = ds.writer(context.Background()).Select(&count, "SELECT COUNT(*) FROM host_seen_times WHERE host_id IN (1, 2, 6, 8, 9, 11)")
	require.NoError(t, err)
	require.Len(t, count, 1)
	assert.Zero(t, count[0])
	count = nil
	err = ds.writer(context.Background()).Select(&count, "SELECT COUNT(*) FROM host_seen_times")
	require.NoError(t, err)
	require.Len(t, count, 1)
	assert.Equal(t, 5, count[0])

	// And it doesn't remove more than it should
	deleted, err = ds.CleanupExpiredHosts(context.Background())
	require.NoError(t, err)
	assert.Len(t, deleted, 0)

	_ = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 5)
}

func testHostsIncludesScheduledQueriesInPackStats(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	team, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID})
	require.NoError(t, err)

	query1 := &fleet.Query{
		Name:               "Only Logged in Query Report",
		Query:              "select * from time",
		AuthorID:           nil,
		Platform:           "darwin",
		Saved:              true,
		TeamID:             nil,
		Interval:           60,
		Logging:            fleet.LoggingSnapshot,
		DiscardData:        false,
		AutomationsEnabled: false,
	}

	_, err = ds.NewQuery(context.Background(), query1)
	require.NoError(t, err)

	query2 := &fleet.Query{
		Name:               "Logged In Report and Log Destination",
		Query:              "select * from time",
		AuthorID:           nil,
		Platform:           "darwin",
		Saved:              true,
		TeamID:             nil,
		Interval:           60,
		Logging:            fleet.LoggingSnapshot,
		DiscardData:        false,
		AutomationsEnabled: true,
	}
	_, err = ds.NewQuery(context.Background(), query2)
	require.NoError(t, err)

	// This query should not be included in the pack stats
	query3 := &fleet.Query{
		Name:               "Not LoggingSnapshot",
		Query:              "select * from time",
		AuthorID:           nil,
		Platform:           "darwin",
		Saved:              true,
		TeamID:             nil,
		Interval:           60,
		Logging:            fleet.LoggingDifferential,
		DiscardData:        false,
		AutomationsEnabled: false, // automations not on
	}
	_, err = ds.NewQuery(context.Background(), query3)
	require.NoError(t, err)

	// This query should not be included in the pack stats
	query4 := &fleet.Query{
		Name:               "Query Report No Interval",
		Query:              "select * from time",
		AuthorID:           nil,
		Platform:           "darwin",
		Saved:              true,
		TeamID:             nil,
		Interval:           0,
		Logging:            fleet.LoggingSnapshot,
		DiscardData:        false,
		AutomationsEnabled: false,
	}
	_, err = ds.NewQuery(context.Background(), query4)
	require.NoError(t, err)

	// this query should not be included in the pack stats
	query5 := &fleet.Query{
		Name:               "Automations No Interval",
		Query:              "select * from time",
		AuthorID:           nil,
		Platform:           "darwin",
		Saved:              true,
		TeamID:             nil,
		Interval:           0,
		Logging:            fleet.LoggingSnapshot,
		DiscardData:        true,
		AutomationsEnabled: true,
	}
	_, err = ds.NewQuery(context.Background(), query5)
	require.NoError(t, err)

	query6 := &fleet.Query{
		Name:               "Team Query",
		Query:              "select * from time",
		AuthorID:           nil,
		Platform:           "darwin",
		Saved:              true,
		TeamID:             &team.ID,
		Interval:           60,
		Logging:            fleet.LoggingSnapshot,
		DiscardData:        false,
		AutomationsEnabled: true,
	}
	_, err = ds.NewQuery(context.Background(), query6)
	require.NoError(t, err)

	hostResult, err := ds.Host(context.Background(), host.ID)
	require.NoError(t, err)

	globalQueryStats := hostResult.PackStats[0].QueryStats
	require.NotNil(t, hostResult)
	require.Equal(t, 2, len(globalQueryStats))
	require.Equal(t, query1.Name, globalQueryStats[0].ScheduledQueryName)
	require.Equal(t, query2.Name, globalQueryStats[1].ScheduledQueryName)

	teamQueryStats := hostResult.PackStats[1].QueryStats
	require.Equal(t, query6.Name, teamQueryStats[0].ScheduledQueryName)

	// Queries with Query Results should be included in the pack stats
	// regardless of the query interval
	queryResultRow := []*fleet.ScheduledQueryResultRow{
		{
			QueryID: query4.ID, // no interval
			HostID:  host.ID,
			Data:    ptr.RawMessage(json.RawMessage(`{"foo": "bar"}`)),
		},
		{
			QueryID: query4.ID, // no interval
			HostID:  host.ID,
			Data:    ptr.RawMessage(json.RawMessage(`{"foo": "baz"}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), queryResultRow, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	hostResult, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.NotNil(t, hostResult)

	assertContains := func(stats []fleet.ScheduledQueryStats, name string) {
		t.Helper()
		for _, stat := range stats {
			if stat.ScheduledQueryName == name {
				return
			}
		}
		t.Errorf("expected to find %s in stats", name)
	}

	globalQueryStats = hostResult.PackStats[0].QueryStats
	require.Equal(t, 3, len(globalQueryStats))
	assertContains(globalQueryStats, query1.Name)
	assertContains(globalQueryStats, query2.Name)
	assertContains(globalQueryStats, query4.Name) // no interval, but has a query result
}

func testHostsAllPackStats(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	// Create a "user created" pack (and one scheduled query in it).
	userPack, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test1",
		HostIDs: []uint{host.ID},
	})
	require.NoError(t, err)
	userQuery := test.NewQuery(t, ds, nil, "user-time", "select * from time", 0, true)
	userSQuery := test.NewScheduledQuery(t, ds, userPack.ID, userQuery.ID, 30, true, true, "time-scheduled-user")

	// Even if the scheduled queries didn't run, we get their pack stats (with zero values).
	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	packStats := host.PackStats
	require.Len(t, packStats, 1)
	sort.Sort(packStatsSlice(packStats))
	for _, tc := range []struct {
		expectedPack   *fleet.Pack
		expectedQuery  *fleet.Query
		expectedSQuery *fleet.ScheduledQuery
		packStats      fleet.PackStats
	}{
		{
			expectedPack:   userPack,
			expectedQuery:  userQuery,
			expectedSQuery: userSQuery,
			packStats:      packStats[0],
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
	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	packStats = host.PackStats
	require.Len(t, packStats, 1)
	sort.Sort(packStatsSlice(packStats))

	require.ElementsMatch(t, packStats[0].QueryStats, userPackSQueryStats)
}

// See #2965.
func testHostsPackStatsMultipleHosts(t *testing.T, ds *Datastore) {
	osqueryHostID1, _ := server.GenerateRandomText(10)
	host1, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		Platform:        "darwin",
		OsqueryHostID:   &osqueryHostID1,
	})
	require.NoError(t, err)
	require.NotNil(t, host1)

	osqueryHostID2, _ := server.GenerateRandomText(10)
	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "bar.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "darwin",
		OsqueryHostID:   &osqueryHostID2,
	})
	require.NoError(t, err)
	require.NotNil(t, host2)

	// Create global pack (and one scheduled query in it).
	test.AddAllHostsLabel(t, ds) // the global pack needs the "All Hosts" label.
	labels, err := ds.ListLabels(context.Background(), fleet.TeamFilter{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, labels, 1)

	userPack, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test1",
		HostIDs: []uint{host1.ID, host2.ID},
	})
	require.NoError(t, err)

	userQuery := test.NewQuery(t, ds, nil, "global-time", "select * from time", 0, true)
	userSQuery := test.NewScheduledQuery(t, ds, userPack.ID, userQuery.ID, 30, true, true, "time-scheduled-global")
	err = ds.AsyncBatchInsertLabelMembership(context.Background(), [][2]uint{
		{labels[0].ID, host1.ID},
		{labels[0].ID, host2.ID},
	})
	require.NoError(t, err)

	globalStatsHost1 := []fleet.ScheduledQueryStats{{
		ScheduledQueryName: userSQuery.Name,
		ScheduledQueryID:   userSQuery.ID,
		QueryName:          userQuery.Name,
		PackName:           userPack.Name,
		PackID:             userPack.ID,
		AverageMemory:      8000,
		Denylisted:         false,
		Executions:         164,
		Interval:           30,
		LastExecuted:       time.Unix(1620325191, 0).UTC(),
		OutputSize:         1337,
		SystemTime:         150,
		UserTime:           180,
		WallTimeMs:         0,
	}}
	globalStatsHost2 := []fleet.ScheduledQueryStats{{
		ScheduledQueryName: userSQuery.Name,
		ScheduledQueryID:   userSQuery.ID,
		QueryName:          userQuery.Name,
		PackName:           userPack.Name,
		PackID:             userPack.ID,
		AverageMemory:      9000,
		Denylisted:         false,
		Executions:         165,
		Interval:           30,
		LastExecuted:       time.Unix(1620325192, 0).UTC(),
		OutputSize:         1338,
		SystemTime:         151,
		UserTime:           181,
		WallTimeMs:         1,
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
		host, err := ds.Host(context.Background(), tc.hostID)
		require.NoError(t, err)
		hostPackStats := []fleet.PackStats{
			{PackID: userPack.ID, PackName: userPack.Name, QueryStats: tc.globalStats},
		}
		err = ds.SaveHostPackStats(context.Background(), host.TeamID, host.ID, hostPackStats)
		require.NoError(t, err)
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
		host, err := ds.Host(context.Background(), tc.host.ID)
		require.NoError(t, err)
		packStats := host.PackStats
		require.Len(t, packStats, 1)
		require.Len(t, packStats[0].QueryStats, 1)
		// Update wall time.
		tc.expectedStats[0].WallTime = tc.expectedStats[0].WallTimeMs
		tc.expectedStats[0].WallTimeMs = 0
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
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		Platform:        "darwin",
		OsqueryHostID:   &osqueryHostID1,
	})
	require.NoError(t, err)
	require.NotNil(t, host1)

	osqueryHostID2, _ := server.GenerateRandomText(10)
	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "foo.local.2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "rhel",
		OsqueryHostID:   &osqueryHostID2,
	})
	require.NoError(t, err)
	require.NotNil(t, host2)

	test.AddAllHostsLabel(t, ds)
	labels, err := ds.ListLabels(context.Background(), fleet.TeamFilter{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, labels, 1)

	userPack, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test1",
		HostIDs: []uint{host1.ID, host2.ID},
	})
	require.NoError(t, err)
	userQuery1 := test.NewQuery(t, ds, nil, "global-time", "select * from time", 0, true)
	userQuery2 := test.NewQuery(t, ds, nil, "global-time-2", "select * from time", 0, true)
	userQuery3 := test.NewQuery(t, ds, nil, "global-time-3", "select * from time", 0, true)
	userQuery4 := test.NewQuery(t, ds, nil, "global-time-4", "select * from time", 0, true)
	userQuery5 := test.NewQuery(t, ds, nil, "global-time-5", "select * from time", 0, true)
	userSQuery1, err := ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "Scheduled Query For Linux only",
		PackID:   userPack.ID,
		QueryID:  userQuery1.ID,
		Interval: 30,
		Snapshot: ptr.Bool(true),
		Removed:  ptr.Bool(true),
		Platform: ptr.String("linux"),
	})
	require.NoError(t, err)
	require.NotZero(t, userSQuery1.ID)

	userSQuery2, err := ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "Scheduled Query For Darwin only",
		PackID:   userPack.ID,
		QueryID:  userQuery2.ID,
		Interval: 30,
		Snapshot: ptr.Bool(true),
		Removed:  ptr.Bool(true),
		Platform: ptr.String("darwin"),
	})
	require.NoError(t, err)
	require.NotZero(t, userSQuery2.ID)

	userSQuery3, err := ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "Scheduled Query For Darwin and Linux",
		PackID:   userPack.ID,
		QueryID:  userQuery3.ID,
		Interval: 30,
		Snapshot: ptr.Bool(true),
		Removed:  ptr.Bool(true),
		Platform: ptr.String("darwin,linux"),
	})
	require.NoError(t, err)
	require.NotZero(t, userSQuery3.ID)

	userSQuery4, err := ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "Scheduled Query For All Platforms",
		PackID:   userPack.ID,
		QueryID:  userQuery4.ID,
		Interval: 30,
		Snapshot: ptr.Bool(true),
		Removed:  ptr.Bool(true),
		Platform: ptr.String(""),
	})
	require.NoError(t, err)
	require.NotZero(t, userSQuery4.ID)

	userSQuery5, err := ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "Scheduled Query For All Platforms v2",
		PackID:   userPack.ID,
		QueryID:  userQuery5.ID,
		Interval: 30,
		Snapshot: ptr.Bool(true),
		Removed:  ptr.Bool(true),
		Platform: nil,
	})
	require.NoError(t, err)
	require.NotZero(t, userSQuery5.ID)

	err = ds.AsyncBatchInsertLabelMembership(context.Background(), [][2]uint{
		{labels[0].ID, host1.ID},
		{labels[0].ID, host2.ID},
	})
	require.NoError(t, err)

	globalStats := []fleet.ScheduledQueryStats{
		{
			ScheduledQueryName: userSQuery2.Name,
			ScheduledQueryID:   userSQuery2.ID,
			QueryName:          userQuery2.Name,
			PackName:           userPack.Name,
			PackID:             userPack.ID,
			AverageMemory:      8001,
			Denylisted:         false,
			Executions:         165,
			Interval:           30,
			LastExecuted:       time.Unix(1620325192, 0).UTC(),
			OutputSize:         1338,
			SystemTime:         151,
			UserTime:           181,
			WallTimeMs:         1,
		},
		{
			ScheduledQueryName: userSQuery3.Name,
			ScheduledQueryID:   userSQuery3.ID,
			QueryName:          userQuery3.Name,
			PackName:           userPack.Name,
			PackID:             userPack.ID,
			AverageMemory:      8002,
			Denylisted:         false,
			Executions:         166,
			Interval:           30,
			LastExecuted:       time.Unix(1620325193, 0).UTC(),
			OutputSize:         1339,
			SystemTime:         152,
			UserTime:           182,
			WallTimeMs:         2,
		},
		{
			ScheduledQueryName: userSQuery4.Name,
			ScheduledQueryID:   userSQuery4.ID,
			QueryName:          userQuery4.Name,
			PackName:           userPack.Name,
			PackID:             userPack.ID,
			AverageMemory:      8003,
			Denylisted:         false,
			Executions:         167,
			Interval:           30,
			LastExecuted:       time.Unix(1620325194, 0).UTC(),
			OutputSize:         1340,
			SystemTime:         153,
			UserTime:           183,
			WallTimeMs:         3,
		},
		{
			ScheduledQueryName: userSQuery5.Name,
			ScheduledQueryID:   userSQuery5.ID,
			QueryName:          userQuery5.Name,
			PackName:           userPack.Name,
			PackID:             userPack.ID,
			AverageMemory:      8003,
			Denylisted:         false,
			Executions:         167,
			Interval:           30,
			LastExecuted:       time.Unix(1620325194, 0).UTC(),
			OutputSize:         1340,
			SystemTime:         153,
			UserTime:           183,
			WallTimeMs:         3,
		},
	}

	// Reload the host and set the scheduled queries stats for the scheduled queries that apply.
	// Plus we set schedule query stats for a query that does not apply (globalSQuery1)
	// (This could happen if the target platform of a schedule query is changed after creation.)
	stats := make([]fleet.ScheduledQueryStats, len(globalStats))
	copy(stats, globalStats)
	stats = append(stats, fleet.ScheduledQueryStats{
		ScheduledQueryName: userSQuery1.Name,
		ScheduledQueryID:   userSQuery1.ID,
		QueryName:          userQuery1.Name,
		PackName:           userPack.Name,
		PackID:             userPack.ID,
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
	host, err := ds.Host(context.Background(), host1.ID)
	require.NoError(t, err)
	hostPackStats := []fleet.PackStats{
		{PackID: userPack.ID, PackName: userPack.Name, QueryStats: stats},
	}
	err = ds.SaveHostPackStats(context.Background(), host.TeamID, host.ID, hostPackStats)
	require.NoError(t, err)

	// host should only return scheduled query stats only for the scheduled queries
	// scheduled to run on "darwin".
	host, err = ds.Host(context.Background(), host.ID)
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
	// Update wall time
	for i := range globalStats {
		globalStats[i].WallTime = globalStats[i].WallTimeMs
		globalStats[i].WallTimeMs = 0
	}
	require.ElementsMatch(t, packStats[0].QueryStats, globalStats)

	// host2 should only return scheduled query stats only for the scheduled queries
	// scheduled to run on "linux"
	host2, err = ds.Host(context.Background(), host2.ID)
	require.NoError(t, err)
	packStats2 := host2.PackStats
	require.Len(t, packStats2, 1)
	require.Len(t, packStats2[0].QueryStats, 4)
	zeroStats := []fleet.ScheduledQueryStats{
		{
			ScheduledQueryName: userSQuery1.Name,
			ScheduledQueryID:   userSQuery1.ID,
			QueryName:          userQuery1.Name,
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
		},
		{
			ScheduledQueryName: userSQuery3.Name,
			ScheduledQueryID:   userSQuery3.ID,
			QueryName:          userQuery3.Name,
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
		},
		{
			ScheduledQueryName: userSQuery4.Name,
			ScheduledQueryID:   userSQuery4.ID,
			QueryName:          userQuery4.Name,
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
		},
		{
			ScheduledQueryName: userSQuery5.Name,
			ScheduledQueryID:   userSQuery5.ID,
			QueryName:          userQuery5.Name,
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
		},
	}
	require.ElementsMatch(t, packStats2[0].QueryStats, zeroStats)
}

// testHostsNoSeenTime tests all changes around the seen_time issue #3095.
func testHostsNoSeenTime(t *testing.T, ds *Datastore) {
	h1, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              1,
		OsqueryHostID:   ptr.String("1"),
		NodeKey:         ptr.String("1"),
		Platform:        "linux",
		Hostname:        "host1",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	removeHostSeenTimes := func(hostID uint) {
		result, err := ds.writer(context.Background()).Exec("DELETE FROM host_seen_times WHERE host_id = ?", hostID)
		require.NoError(t, err)
		rowsAffected, err := result.RowsAffected()
		require.NoError(t, err)
		require.EqualValues(t, 1, rowsAffected)
	}
	removeHostSeenTimes(h1.ID)

	h1, err = ds.Host(context.Background(), h1.ID)
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
	summary, err := ds.GenerateHostStatusStatistics(context.Background(), teamFilter, mockClock.Now(), nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, summary.TeamID)
	assert.Equal(t, uint(1), summary.TotalsHostsCount)
	assert.Equal(t, uint(1), summary.OnlineCount)
	assert.Equal(t, uint(0), summary.OfflineCount)
	assert.Equal(t, uint(0), summary.MIACount)
	assert.Equal(t, uint(1), summary.NewCount)

	var count []int
	err = ds.writer(context.Background()).Select(&count, "SELECT COUNT(*) FROM host_seen_times")
	require.NoError(t, err)
	require.Len(t, count, 1)
	require.Zero(t, count[0])

	// Enroll existing host.
	_, err = ds.EnrollHost(context.Background(), false, "1", "", "", "1", nil, 0)
	require.NoError(t, err)

	var seenTime1 []time.Time
	err = ds.writer(context.Background()).Select(&seenTime1, "SELECT seen_time FROM host_seen_times WHERE host_id = ?", h1.ID)
	require.NoError(t, err)
	require.Len(t, seenTime1, 1)
	require.NotZero(t, seenTime1[0])

	time.Sleep(1 * time.Second)

	// Enroll again to trigger an update of host_seen_times.
	_, err = ds.EnrollHost(context.Background(), false, "1", "", "", "1", nil, 0)
	require.NoError(t, err)

	var seenTime2 []time.Time
	err = ds.writer(context.Background()).Select(&seenTime2, "SELECT seen_time FROM host_seen_times WHERE host_id = ?", h1.ID)
	require.NoError(t, err)
	require.Len(t, seenTime2, 1)
	require.NotZero(t, seenTime2[0])

	require.True(t, seenTime2[0].After(seenTime1[0]), "%s vs. %s", seenTime1[0], seenTime2[0])

	removeHostSeenTimes(h1.ID)

	h2, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              2,
		OsqueryHostID:   ptr.String("2"),
		NodeKey:         ptr.String("2"),
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
	h1, err = ds.Host(context.Background(), h1.ID)
	require.NoError(t, err)
	h2, err = ds.Host(context.Background(), h2.ID)
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

	total, unseen, err := ds.TotalAndUnseenHostsSince(context.Background(), nil, 1)
	require.NoError(t, err)
	require.Equal(t, total, 2)
	require.Len(t, unseen, 0)

	h3, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:              3,
		OsqueryHostID:   ptr.String("3"),
		NodeKey:         ptr.String("3"),
		Platform:        "darwin",
		Hostname:        "host3",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	removeHostSeenTimes(h3.ID)

	_, err = ds.CleanupExpiredHosts(context.Background())
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

func testHostDeviceMapping(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	h, err := ds.NewHost(ctx, &fleet.Host{
		ID:              1,
		OsqueryHostID:   ptr.String("1"),
		NodeKey:         ptr.String("1"),
		Platform:        "linux",
		Hostname:        "host1",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	// add device mapping for host
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
		h.ID, "a@b.c", "src1")
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
		h.ID, "b@b.c", "src1")
	require.NoError(t, err)

	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
		h.ID, "a@b.c", "src2")
	require.NoError(t, err)

	// non-existent host should have empty device mapping
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

	// device mapping is not included in basic method for host by id
	host, err := ds.Host(ctx, h.ID)
	require.NoError(t, err)
	require.Nil(t, host.DeviceMapping)

	// create additional hosts to test device mapping of multiple hosts in ListHosts results
	h2, err := ds.NewHost(ctx, &fleet.Host{
		ID:              2,
		OsqueryHostID:   ptr.String("2"),
		NodeKey:         ptr.String("2"),
		Platform:        "linux",
		Hostname:        "host2",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	// add device mapping for second host
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
		h2.ID, "a@b.c", "src2")
	require.NoError(t, err)

	// create third host with no device mapping
	_, err = ds.NewHost(ctx, &fleet.Host{
		ID:              3,
		OsqueryHostID:   ptr.String("3"),
		NodeKey:         ptr.String("3"),
		Platform:        "linux",
		Hostname:        "host3",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	// device mapping not included in list hosts unless optional param is set to true
	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 3)
	require.Nil(t, hosts[0].DeviceMapping)
	require.Nil(t, hosts[1].DeviceMapping)
	require.Nil(t, hosts[2].DeviceMapping)

	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{DeviceMapping: true}, 3)

	hostsByID := make(map[uint]*fleet.Host)
	for _, hst := range hosts {
		hostsByID[hst.ID] = hst
	}

	var dm []*fleet.HostDeviceMapping

	// device mapping for host 1
	require.NotNil(t, hostsByID[1].DeviceMapping)
	err = json.Unmarshal(*hostsByID[1].DeviceMapping, &dm)
	require.NoError(t, err)
	var emails []string
	var sources []string
	for _, e := range dm {
		emails = append(emails, e.Email)
		sources = append(sources, e.Source)
	}
	assert.ElementsMatch(t, []string{"a@b.c", "b@b.c", "a@b.c"}, emails)
	assert.ElementsMatch(t, []string{"src1", "src1", "src2"}, sources)

	// device mapping for host 2
	require.NotNil(t, *hostsByID[2].DeviceMapping)
	err = json.Unmarshal(*hostsByID[2].DeviceMapping, &dm)
	require.NoError(t, err)
	assert.Len(t, dm, 1)
	assert.Equal(t, "a@b.c", dm[0].Email)
	assert.Equal(t, "src2", dm[0].Source)

	// no device mapping for host 3
	require.NotNil(t, hostsByID[3].DeviceMapping) // json "null" rather than nil
	err = json.Unmarshal(*hostsByID[3].DeviceMapping, &dm)
	require.NoError(t, err)
	assert.Nil(t, dm)
}

func testHostsReplaceHostDeviceMapping(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	h, err := ds.NewHost(ctx, &fleet.Host{
		ID:              1,
		OsqueryHostID:   ptr.String("1"),
		NodeKey:         ptr.String("1"),
		Platform:        "linux",
		Hostname:        "host1",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, nil, "src1")
	require.NoError(t, err)

	dms, err := ds.ListHostDeviceMapping(ctx, h.ID)
	require.NoError(t, err)
	require.Len(t, dms, 0)

	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, []*fleet.HostDeviceMapping{
		{HostID: h.ID, Email: "a@b.c", Source: "src1"},
		{HostID: h.ID + 1, Email: "a@b.c", Source: "src1"},
	}, "src1")
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("found %d", h.ID+1))

	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, []*fleet.HostDeviceMapping{
		{HostID: h.ID, Email: "a@b.c", Source: "src1"},
		{HostID: h.ID, Email: "b@b.c", Source: "src1"},
		{HostID: h.ID, Email: "c@b.c", Source: "src2"},
	}, "src1")
	require.ErrorContains(t, err, "host device mapping are not all for the provided source")

	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, []*fleet.HostDeviceMapping{
		{HostID: h.ID, Email: "c@b.c", Source: "src2"},
	}, "src2")
	require.NoError(t, err)

	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, []*fleet.HostDeviceMapping{
		{HostID: h.ID, Email: "a@b.c", Source: "src1"},
		{HostID: h.ID, Email: "b@b.c", Source: "src1"},
	}, "src1")
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
	}, "src2")
	require.ErrorContains(t, err, "host device mapping are not all for the provided source")

	// omit b@b.c from src1
	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, []*fleet.HostDeviceMapping{
		{HostID: h.ID, Email: "a@b.c", Source: "src1"},
	}, "src1")
	require.NoError(t, err)

	// add d@b to src2, omit c@b.c from src2
	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, []*fleet.HostDeviceMapping{
		{HostID: h.ID, Email: "d@b.c", Source: "src2"},
	}, "src2")
	require.NoError(t, err)

	dms, err = ds.ListHostDeviceMapping(ctx, h.ID)
	require.NoError(t, err)
	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: "src1"},
		{Email: "d@b.c", Source: "src2"},
	})

	// delete only
	err = ds.ReplaceHostDeviceMapping(ctx, h.ID, nil, "src1")
	require.NoError(t, err)

	dms, err = ds.ListHostDeviceMapping(ctx, h.ID)
	require.NoError(t, err)
	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{
		{Email: "d@b.c", Source: "src2"},
	})
}

func testHostsCustomHostDeviceMapping(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	h1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("1"),
		NodeKey:         ptr.String("1"),
		Platform:        "linux",
		Hostname:        "host1",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	h2, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("2"),
		NodeKey:         ptr.String("2"),
		Platform:        "linux",
		Hostname:        "host2",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	// create a custom installer email for h1
	dms, err := ds.SetOrUpdateCustomHostDeviceMapping(ctx, h1.ID, "a@b.c", fleet.DeviceMappingCustomInstaller)
	require.NoError(t, err)
	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{{Email: "a@b.c", Source: fleet.DeviceMappingCustomReplacement}})

	// custom installer can be updated
	dms, err = ds.SetOrUpdateCustomHostDeviceMapping(ctx, h1.ID, "b@b.c", fleet.DeviceMappingCustomInstaller)
	require.NoError(t, err)
	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{{Email: "b@b.c", Source: fleet.DeviceMappingCustomReplacement}})

	// set a custom override, custom installer is removed
	dms, err = ds.SetOrUpdateCustomHostDeviceMapping(ctx, h1.ID, "c@b.c", fleet.DeviceMappingCustomOverride)
	require.NoError(t, err)
	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{{Email: "c@b.c", Source: fleet.DeviceMappingCustomReplacement}})

	// updating the custom installer is now ignored
	dms, err = ds.SetOrUpdateCustomHostDeviceMapping(ctx, h1.ID, "d@b.c", fleet.DeviceMappingCustomInstaller)
	require.NoError(t, err)
	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{{Email: "c@b.c", Source: fleet.DeviceMappingCustomReplacement}})

	// updating the custom override works
	dms, err = ds.SetOrUpdateCustomHostDeviceMapping(ctx, h1.ID, "e@b.c", fleet.DeviceMappingCustomOverride)
	require.NoError(t, err)
	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{{Email: "e@b.c", Source: fleet.DeviceMappingCustomReplacement}})

	// set some unrelated emails for h2
	err = ds.ReplaceHostDeviceMapping(ctx, h2.ID, []*fleet.HostDeviceMapping{
		{HostID: h2.ID, Email: "a@c.d", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{HostID: h2.ID, Email: "b@c.d", Source: fleet.DeviceMappingGoogleChromeProfiles},
	}, fleet.DeviceMappingGoogleChromeProfiles)
	require.NoError(t, err)

	// create a custom override immediately, without a custom installer
	_, err = ds.SetOrUpdateCustomHostDeviceMapping(ctx, h2.ID, "c@c.d", fleet.DeviceMappingCustomOverride)
	require.NoError(t, err)

	// adding a custom installer is ignored
	dms, err = ds.SetOrUpdateCustomHostDeviceMapping(ctx, h2.ID, "d@c.d", fleet.DeviceMappingCustomInstaller)
	require.NoError(t, err)

	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{
		{Email: "a@c.d", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "b@c.d", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "c@c.d", Source: fleet.DeviceMappingCustomReplacement},
	})

	// updating the custom override works
	dms, err = ds.SetOrUpdateCustomHostDeviceMapping(ctx, h2.ID, "e@c.d", fleet.DeviceMappingCustomOverride)
	require.NoError(t, err)

	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{
		{Email: "a@c.d", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "b@c.d", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "e@c.d", Source: fleet.DeviceMappingCustomReplacement},
	})

	// deleting the host deletes the mappings
	err = ds.DeleteHost(ctx, h2.ID)
	require.NoError(t, err)
	dms, err = ds.ListHostDeviceMapping(ctx, h2.ID)
	require.NoError(t, err)
	require.Empty(t, dms)

	// other host was left untouched
	dms, err = ds.ListHostDeviceMapping(ctx, h1.ID)
	require.NoError(t, err)
	assertHostDeviceMapping(t, dms, []*fleet.HostDeviceMapping{{Email: "e@b.c", Source: fleet.DeviceMappingCustomReplacement}})
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
	_, err := ds.GetHostMunkiVersion(context.Background(), 123)
	require.True(t, fleet.IsNotFound(err))

	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), 123, "1.2.3", nil, nil))
	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), 999, "9.0", nil, nil))
	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), 123, "1.3.0", []string{"a", "b"}, []string{"c"}))

	version, err := ds.GetHostMunkiVersion(context.Background(), 123)
	require.NoError(t, err)
	require.Equal(t, "1.3.0", version)

	issues, err := ds.GetHostMunkiIssues(context.Background(), 123)
	require.NoError(t, err)
	require.Len(t, issues, 3)

	var aMunkiIssueID uint
	for _, iss := range issues {
		assert.NotZero(t, iss.MunkiIssueID)
		if iss.Name == "a" {
			aMunkiIssueID = iss.MunkiIssueID
		}
		assert.False(t, iss.HostIssueCreatedAt.IsZero())
	}

	// get a Munki Issue
	miss, err := ds.GetMunkiIssue(context.Background(), aMunkiIssueID)
	require.NoError(t, err)
	require.Equal(t, "a", miss.Name)

	// get an invalid munki issue
	_, err = ds.GetMunkiIssue(context.Background(), aMunkiIssueID+1000)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// ignore IDs and timestamps in slice comparison
	issues[0].MunkiIssueID, issues[0].HostIssueCreatedAt = 0, time.Time{}
	issues[1].MunkiIssueID, issues[1].HostIssueCreatedAt = 0, time.Time{}
	issues[2].MunkiIssueID, issues[2].HostIssueCreatedAt = 0, time.Time{}
	assert.ElementsMatch(t, []*fleet.HostMunkiIssue{
		{Name: "a", IssueType: "error"},
		{Name: "b", IssueType: "error"},
		{Name: "c", IssueType: "warning"},
	}, issues)

	version, err = ds.GetHostMunkiVersion(context.Background(), 999)
	require.NoError(t, err)
	require.Equal(t, "9.0", version)

	issues, err = ds.GetHostMunkiIssues(context.Background(), 999)
	require.NoError(t, err)
	require.Len(t, issues, 0)

	// simulate uninstall
	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), 123, "", nil, nil))

	_, err = ds.GetHostMunkiVersion(context.Background(), 123)
	require.True(t, fleet.IsNotFound(err))
	issues, err = ds.GetHostMunkiIssues(context.Background(), 123)
	require.NoError(t, err)
	require.Len(t, issues, 0)

	_, err = ds.GetHostMDM(context.Background(), 432)
	require.True(t, fleet.IsNotFound(err), err)

	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 432, false, true, "url", false, "", ""))

	hmdm, err := ds.GetHostMDM(context.Background(), 432)
	require.NoError(t, err)
	assert.True(t, hmdm.Enrolled)
	assert.Equal(t, "url", hmdm.ServerURL)
	assert.False(t, hmdm.InstalledFromDep)
	require.NotNil(t, hmdm.MDMID)
	assert.NotZero(t, *hmdm.MDMID)
	urlMDMID := *hmdm.MDMID
	assert.Equal(t, fleet.UnknownMDMName, hmdm.Name)

	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 455, false, true, "https://kandji.io", true, fleet.WellKnownMDMKandji, "")) // kandji mdm name
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 432, false, false, "url3", true, "", ""))

	hmdm, err = ds.GetHostMDM(context.Background(), 432)
	require.NoError(t, err)
	assert.False(t, hmdm.Enrolled)
	assert.Equal(t, "url3", hmdm.ServerURL)
	assert.True(t, hmdm.InstalledFromDep)
	require.NotNil(t, hmdm.MDMID)
	assert.NotZero(t, *hmdm.MDMID)
	assert.NotEqual(t, urlMDMID, *hmdm.MDMID)
	assert.Equal(t, fleet.UnknownMDMName, hmdm.Name)

	hmdm, err = ds.GetHostMDM(context.Background(), 455)
	require.NoError(t, err)
	assert.True(t, hmdm.Enrolled)
	assert.Equal(t, "https://kandji.io", hmdm.ServerURL)
	assert.True(t, hmdm.InstalledFromDep)
	require.NotNil(t, hmdm.MDMID)
	assert.NotZero(t, *hmdm.MDMID)
	kandjiID1 := *hmdm.MDMID
	assert.Equal(t, fleet.WellKnownMDMKandji, hmdm.Name)

	// get mdm solution
	mdmSol, err := ds.GetMDMSolution(context.Background(), kandjiID1)
	require.NoError(t, err)
	require.Equal(t, "https://kandji.io", mdmSol.ServerURL)
	require.Equal(t, fleet.WellKnownMDMKandji, mdmSol.Name)

	// get unknown mdm solution
	_, err = ds.GetMDMSolution(context.Background(), kandjiID1+1000)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// switch to simplemdm in an update
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 455, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "")) // now simplemdm name

	hmdm, err = ds.GetHostMDM(context.Background(), 455)
	require.NoError(t, err)
	assert.True(t, hmdm.Enrolled)
	assert.Equal(t, "https://simplemdm.com", hmdm.ServerURL)
	assert.False(t, hmdm.InstalledFromDep)
	require.NotNil(t, hmdm.MDMID)
	assert.NotZero(t, *hmdm.MDMID)
	assert.Equal(t, fleet.WellKnownMDMSimpleMDM, hmdm.Name)

	// switch back to "url"
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 455, false, false, "url", false, "", ""))

	hmdm, err = ds.GetHostMDM(context.Background(), 455)
	require.NoError(t, err)
	assert.False(t, hmdm.Enrolled)
	assert.Equal(t, "url", hmdm.ServerURL)
	assert.False(t, hmdm.InstalledFromDep)
	require.NotNil(t, hmdm.MDMID)
	assert.Equal(t, urlMDMID, *hmdm.MDMID) // id is the same as created previously for that url
	assert.Equal(t, fleet.UnknownMDMName, hmdm.Name)

	// switch to a different Kandji server URL, will have a different MDM ID as
	// even though this is another Kandji, the URL is different.
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 455, false, true, "https://kandji.io/2", false, fleet.WellKnownMDMKandji, ""))

	hmdm, err = ds.GetHostMDM(context.Background(), 455)
	require.NoError(t, err)
	assert.True(t, hmdm.Enrolled)
	assert.Equal(t, "https://kandji.io/2", hmdm.ServerURL)
	assert.False(t, hmdm.InstalledFromDep)
	require.NotNil(t, hmdm.MDMID)
	assert.NotZero(t, *hmdm.MDMID)
	assert.NotEqual(t, kandjiID1, *hmdm.MDMID)
	assert.Equal(t, fleet.WellKnownMDMKandji, hmdm.Name)
}

func testMunkiIssuesBatchSize(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	allIDs := make(map[string]uint)
	storeIDs := func(msgToID map[[2]string]uint) {
		for k, v := range msgToID {
			assert.NotZero(t, v)
			allIDs[k[0]] = v
		}
	}

	cases := []struct {
		errors   []string
		warnings []string
	}{
		{nil, nil},

		{[]string{"a"}, nil},
		{[]string{"b", "c"}, nil},
		{[]string{"d", "e", "f"}, nil},
		{[]string{"g", "h", "i", "j"}, nil},
		{[]string{"k", "l", "m", "n", "o"}, nil},

		{nil, []string{"A"}},
		{nil, []string{"B", "C"}},
		{nil, []string{"D", "E", "F"}},
		{nil, []string{"G", "H", "I", "J"}},
		{nil, []string{"K", "L", "M", "N", "O"}},

		{[]string{"a", "p", "q"}, []string{"A", "B", "P"}},
	}
	for _, c := range cases {
		t.Run(strings.Join(c.errors, ",")+","+strings.Join(c.warnings, ","), func(t *testing.T) {
			msgToID, err := ds.getOrInsertMunkiIssues(ctx, c.errors, c.warnings, 2)
			require.NoError(t, err)
			require.Len(t, msgToID, len(c.errors)+len(c.warnings))
			storeIDs(msgToID)
		})
	}

	// try those errors/warning with some hosts
	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), 123, "1.2.3", []string{"a", "b"}, []string{"C"}))
	issues, err := ds.GetHostMunkiIssues(ctx, 123)
	require.NoError(t, err)
	require.Len(t, issues, 3)
	for _, iss := range issues {
		assert.Equal(t, allIDs[iss.Name], iss.MunkiIssueID)
	}

	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), 123, "1.2.3", []string{"c", "z"}, []string{"D", "E", "Z"}))
	issues, err = ds.GetHostMunkiIssues(ctx, 123)
	require.NoError(t, err)
	require.Len(t, issues, 5)
	for _, iss := range issues {
		if iss.Name == "z" || iss.Name == "Z" {
			// z/Z do not exist in allIDs, by checking not equal it ensures it is not 0
			assert.NotEqual(t, allIDs[iss.Name], iss.MunkiIssueID)
		} else {
			assert.Equal(t, allIDs[iss.Name], iss.MunkiIssueID)
		}
	}
}

func testAggregatedHostMDMAndMunki(t *testing.T, ds *Datastore) {
	// Make sure things work before data is generated
	versions, updatedAt, err := ds.AggregatedMunkiVersion(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, versions, 0)
	require.Zero(t, updatedAt)
	issues, updatedAt, err := ds.AggregatedMunkiIssues(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, issues, 0)
	require.Zero(t, updatedAt)
	status, updatedAt, err := ds.AggregatedMDMStatus(context.Background(), nil, "")
	require.NoError(t, err)
	require.Empty(t, status)
	require.Zero(t, updatedAt)
	solutions, updatedAt, err := ds.AggregatedMDMSolutions(context.Background(), nil, "")
	require.NoError(t, err)
	require.Len(t, solutions, 0)
	require.Zero(t, updatedAt)
	status, updatedAt, err = ds.AggregatedMDMStatus(context.Background(), nil, "windows")
	require.NoError(t, err)
	require.Empty(t, status)
	require.Zero(t, updatedAt)
	solutions, updatedAt, err = ds.AggregatedMDMSolutions(context.Background(), nil, "windows")
	require.NoError(t, err)
	require.Len(t, solutions, 0)
	require.Zero(t, updatedAt)

	// Make sure generation works when there's no mdm or munki data
	require.NoError(t, ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	// And after generating without any data, it all looks reasonable
	versions, updatedAt, err = ds.AggregatedMunkiVersion(context.Background(), nil)
	firstUpdatedAt := updatedAt

	require.NoError(t, err)
	require.Len(t, versions, 0)
	require.NotZero(t, updatedAt)
	issues, updatedAt, err = ds.AggregatedMunkiIssues(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, issues)
	require.NotZero(t, updatedAt)
	status, updatedAt, err = ds.AggregatedMDMStatus(context.Background(), nil, "")
	require.NoError(t, err)
	require.Empty(t, status)
	require.NotZero(t, updatedAt)
	solutions, updatedAt, err = ds.AggregatedMDMSolutions(context.Background(), nil, "")
	require.NoError(t, err)
	require.Len(t, solutions, 0)
	require.NotZero(t, updatedAt)
	status, updatedAt, err = ds.AggregatedMDMStatus(context.Background(), nil, "windows")
	require.NoError(t, err)
	require.Empty(t, status)
	require.NotZero(t, updatedAt)
	solutions, updatedAt, err = ds.AggregatedMDMSolutions(context.Background(), nil, "windows")
	require.NoError(t, err)
	require.Len(t, solutions, 0)
	require.NotZero(t, updatedAt)

	// So now we try with data
	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), 123, "1.2.3", []string{"a", "b"}, []string{"c"}))
	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), 999, "9.0", []string{"a"}, nil))
	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), 342, "1.2.3", nil, []string{"c"}))

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

	issues, _, err = ds.AggregatedMunkiIssues(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, issues, 3)
	// ignore the ids
	issues[0].ID = 0
	issues[1].ID = 0
	issues[2].ID = 0
	assert.ElementsMatch(t, issues, []fleet.AggregatedMunkiIssue{
		{
			MunkiIssue: fleet.MunkiIssue{
				Name:      "a",
				IssueType: "error",
			},
			HostsCount: 2,
		},
		{
			MunkiIssue: fleet.MunkiIssue{
				Name:      "b",
				IssueType: "error",
			},
			HostsCount: 1,
		},
		{
			MunkiIssue: fleet.MunkiIssue{
				Name:      "c",
				IssueType: "warning",
			},
			HostsCount: 2,
		},
	})

	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 432, false, true, "url", false, "", ""))                                           // manual enrollment
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 123, false, true, "url", false, "", ""))                                           // manual enrollment
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 124, false, true, "url", false, "", ""))                                           // manual enrollment
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 455, false, true, "https://simplemdm.com", true, fleet.WellKnownMDMSimpleMDM, "")) // automatic enrollment
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 999, false, false, "https://kandji.io", false, fleet.WellKnownMDMKandji, ""))      // unenrolled
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 875, false, false, "https://kandji.io", true, fleet.WellKnownMDMKandji, ""))       // pending enrollment
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), 1337, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, ""))     // pending enrollment

	require.NoError(t, ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	status, _, err = ds.AggregatedMDMStatus(context.Background(), nil, "")
	require.NoError(t, err)
	assert.Equal(t, 7, status.HostsCount)
	assert.Equal(t, 1, status.UnenrolledHostsCount)
	assert.Equal(t, 2, status.PendingHostsCount)
	assert.Equal(t, 3, status.EnrolledManualHostsCount)
	assert.Equal(t, 1, status.EnrolledAutomatedHostsCount)

	solutions, _, err = ds.AggregatedMDMSolutions(context.Background(), nil, "")
	require.NoError(t, err)
	require.Len(t, solutions, 4) // 4 different urls
	for _, sol := range solutions {
		switch sol.ServerURL {
		case "url":
			assert.Equal(t, 3, sol.HostsCount)
			assert.Equal(t, fleet.UnknownMDMName, sol.Name)
		case "https://simplemdm.com":
			assert.Equal(t, 1, sol.HostsCount)
			assert.Equal(t, fleet.WellKnownMDMSimpleMDM, sol.Name)
		case "https://kandji.io":
			assert.Equal(t, 2, sol.HostsCount)
			assert.Equal(t, fleet.WellKnownMDMKandji, sol.Name)
		case "https://fleetdm.com":
			assert.Equal(t, 1, sol.HostsCount)
			assert.Equal(t, fleet.WellKnownMDMFleet, sol.Name)
		default:
			require.Fail(t, fmt.Sprintf("unknown MDM solutions URL: %s", sol.ServerURL))
		}
	}

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

	h1 := test.NewHost(t, ds, "h1"+t.Name(), "192.168.1.10", "1", "1", time.Now(), test.WithPlatform("windows"))
	h2 := test.NewHost(t, ds, "h2"+t.Name(), "192.168.1.11", "2", "2", time.Now(), test.WithPlatform("darwin"))
	h3 := test.NewHost(t, ds, "h3"+t.Name(), "192.168.1.11", "3", "3", time.Now(), test.WithPlatform("darwin"))
	h4 := test.NewHost(t, ds, "h4"+t.Name(), "192.168.1.11", "4", "4", time.Now(), test.WithPlatform("windows"))
	h5 := test.NewHost(t, ds, "h5"+t.Name(), "192.168.1.12", "5", "5", time.Now(), test.WithPlatform("ios"))
	h6 := test.NewHost(t, ds, "h6"+t.Name(), "192.168.1.12", "6", "6", time.Now(), test.WithPlatform("ipados"))

	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{h1.ID}))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team2.ID, []uint{h2.ID}))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{h3.ID}))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{h4.ID}))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{h6.ID}))

	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), h1.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, ""))
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), h2.ID, false, true, "url", false, "", ""))

	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), h5.ID, false, true, "https://fleet.example.com", true, fleet.WellKnownMDMFleet, ""))
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), h6.ID, false, true, "https://fleet.example.com", true, fleet.WellKnownMDMFleet, ""))

	// Add a server, this will be ignored in lists and aggregated data.
	require.NoError(t, ds.SetOrUpdateMDMData(context.Background(), h4.ID, true, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, ""))

	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), h1.ID, "1.2.3", []string{"d"}, nil))
	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), h2.ID, "1.2.3", []string{"d"}, []string{"e"}))

	// h3 adds the version but then removes it
	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), h3.ID, "1.2.3", []string{"f"}, nil))
	require.NoError(t, ds.SetOrUpdateMunkiInfo(context.Background(), h3.ID, "", []string{"d"}, []string{"f"}))

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

	issues, updatedAt, err = ds.AggregatedMunkiIssues(context.Background(), &team1.ID)
	require.NoError(t, err)
	require.Len(t, issues, 2)
	// ignore IDs
	issues[0].ID = 0
	issues[1].ID = 0
	assert.ElementsMatch(t, issues, []fleet.AggregatedMunkiIssue{
		{
			MunkiIssue: fleet.MunkiIssue{
				Name:      "d",
				IssueType: "error",
			},
			HostsCount: 2,
		},
		{
			MunkiIssue: fleet.MunkiIssue{
				Name:      "f",
				IssueType: "warning",
			},
			HostsCount: 1,
		},
	})
	require.True(t, updatedAt.After(firstUpdatedAt))

	status, _, err = ds.AggregatedMDMStatus(context.Background(), nil, "")
	require.NoError(t, err)
	assert.Equal(t, 11, status.HostsCount)
	assert.Equal(t, 1, status.UnenrolledHostsCount)
	assert.Equal(t, 5, status.EnrolledManualHostsCount)
	assert.Equal(t, 3, status.EnrolledAutomatedHostsCount)

	status, _, err = ds.AggregatedMDMStatus(context.Background(), &team1.ID, "")
	require.NoError(t, err)
	assert.Equal(t, 2, status.HostsCount)
	assert.Equal(t, 0, status.UnenrolledHostsCount)
	assert.Equal(t, 1, status.EnrolledManualHostsCount)
	assert.Equal(t, 1, status.EnrolledAutomatedHostsCount)

	solutions, updatedAt, err = ds.AggregatedMDMSolutions(context.Background(), nil, "")
	require.True(t, updatedAt.After(firstUpdatedAt))
	require.NoError(t, err)
	require.Len(t, solutions, 5)
	// Check the new MDM solution used by the iOS/iPadOS
	assert.Equal(t, "https://fleet.example.com", solutions[4].ServerURL)
	assert.Equal(t, fleet.WellKnownMDMFleet, solutions[4].Name)
	assert.Equal(t, 2, solutions[4].HostsCount)

	solutions, updatedAt, err = ds.AggregatedMDMSolutions(context.Background(), &team1.ID, "")
	require.True(t, updatedAt.After(firstUpdatedAt))
	require.NoError(t, err)
	require.Len(t, solutions, 2)
	assert.Equal(t, "https://simplemdm.com", solutions[0].ServerURL)
	assert.Equal(t, fleet.WellKnownMDMSimpleMDM, solutions[0].Name)
	assert.Equal(t, 1, solutions[0].HostsCount)
	assert.Equal(t, "https://fleet.example.com", solutions[1].ServerURL)
	assert.Equal(t, fleet.WellKnownMDMFleet, solutions[1].Name)
	assert.Equal(t, 1, solutions[1].HostsCount)

	status, _, err = ds.AggregatedMDMStatus(context.Background(), &team1.ID, "darwin")
	require.NoError(t, err)
	assert.Equal(t, 0, status.HostsCount)
	assert.Equal(t, 0, status.UnenrolledHostsCount)
	assert.Equal(t, 0, status.EnrolledManualHostsCount)
	assert.Equal(t, 0, status.EnrolledAutomatedHostsCount)

	solutions, updatedAt, err = ds.AggregatedMDMSolutions(context.Background(), &team1.ID, "darwin")
	require.True(t, updatedAt.After(firstUpdatedAt))
	require.NoError(t, err)
	require.Len(t, solutions, 0)

	status, _, err = ds.AggregatedMDMStatus(context.Background(), &team1.ID, "windows")
	require.NoError(t, err)
	assert.Equal(t, 1, status.HostsCount)
	assert.Equal(t, 0, status.UnenrolledHostsCount)
	assert.Equal(t, 1, status.EnrolledManualHostsCount)
	assert.Equal(t, 0, status.EnrolledAutomatedHostsCount)

	status, _, err = ds.AggregatedMDMStatus(context.Background(), &team1.ID, "ios")
	require.NoError(t, err)
	assert.Equal(t, 0, status.HostsCount)
	assert.Equal(t, 0, status.UnenrolledHostsCount)
	assert.Equal(t, 0, status.EnrolledManualHostsCount)
	assert.Equal(t, 0, status.EnrolledAutomatedHostsCount)

	status, _, err = ds.AggregatedMDMStatus(context.Background(), nil, "ios")
	require.NoError(t, err)
	assert.Equal(t, 1, status.HostsCount)
	assert.Equal(t, 0, status.UnenrolledHostsCount)
	assert.Equal(t, 0, status.EnrolledManualHostsCount)
	assert.Equal(t, 1, status.EnrolledAutomatedHostsCount)

	status, _, err = ds.AggregatedMDMStatus(context.Background(), &team1.ID, "ipados")
	require.NoError(t, err)
	assert.Equal(t, 1, status.HostsCount)
	assert.Equal(t, 0, status.UnenrolledHostsCount)
	assert.Equal(t, 0, status.EnrolledManualHostsCount)
	assert.Equal(t, 1, status.EnrolledAutomatedHostsCount)

	status, _, err = ds.AggregatedMDMStatus(context.Background(), nil, "ipados")
	require.NoError(t, err)
	assert.Equal(t, 1, status.HostsCount)
	assert.Equal(t, 0, status.UnenrolledHostsCount)
	assert.Equal(t, 0, status.EnrolledManualHostsCount)
	assert.Equal(t, 1, status.EnrolledAutomatedHostsCount)

	solutions, updatedAt, err = ds.AggregatedMDMSolutions(context.Background(), &team1.ID, "windows")
	require.True(t, updatedAt.After(firstUpdatedAt))
	require.NoError(t, err)
	require.Len(t, solutions, 1)
	assert.Equal(t, "https://simplemdm.com", solutions[0].ServerURL)
	assert.Equal(t, fleet.WellKnownMDMSimpleMDM, solutions[0].Name)
	assert.Equal(t, 1, solutions[0].HostsCount)

	solutions, updatedAt, err = ds.AggregatedMDMSolutions(context.Background(), nil, "ios")
	require.True(t, updatedAt.After(firstUpdatedAt))
	require.NoError(t, err)
	require.Len(t, solutions, 1)
	assert.Equal(t, "https://fleet.example.com", solutions[0].ServerURL)
	assert.Equal(t, fleet.WellKnownMDMFleet, solutions[0].Name)
	assert.Equal(t, 1, solutions[0].HostsCount)

	solutions, updatedAt, err = ds.AggregatedMDMSolutions(context.Background(), &team1.ID, "ios")
	require.True(t, updatedAt.After(firstUpdatedAt))
	require.NoError(t, err)
	require.Len(t, solutions, 0)

	solutions, updatedAt, err = ds.AggregatedMDMSolutions(context.Background(), nil, "ipados")
	require.True(t, updatedAt.After(firstUpdatedAt))
	require.NoError(t, err)
	require.Len(t, solutions, 1)
	assert.Equal(t, "https://fleet.example.com", solutions[0].ServerURL)
	assert.Equal(t, fleet.WellKnownMDMFleet, solutions[0].Name)
	assert.Equal(t, 1, solutions[0].HostsCount)

	solutions, updatedAt, err = ds.AggregatedMDMSolutions(context.Background(), &team1.ID, "ipados")
	require.True(t, updatedAt.After(firstUpdatedAt))
	require.NoError(t, err)
	require.Len(t, solutions, 1)
	assert.Equal(t, "https://fleet.example.com", solutions[0].ServerURL)
	assert.Equal(t, fleet.WellKnownMDMFleet, solutions[0].Name)
	assert.Equal(t, 1, solutions[0].HostsCount)
}

func testHostsLite(t *testing.T, ds *Datastore) {
	_, err := ds.HostLite(context.Background(), 1)
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.True(t, errors.As(err, &nfe))

	now := time.Now()
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		ID:                  1,
		OsqueryHostID:       ptr.String("foobar"),
		NodeKey:             ptr.String("nodekey"),
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
	require.Equal(t, "foobar", *h.OsqueryHostID)
	require.Equal(t, "nodekey", *h.NodeKey)
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
		OsqueryHostID:       ptr.String("foobar"),
		NodeKey:             ptr.String("nodekey"),
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
		OsqueryHostID:       ptr.String("foobar"),
		NodeKey:             ptr.String("nodekey"),
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
		NodeKey:         ptr.String("1"),
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

	host, err = ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, host.Users, 2)
	test.ElementsMatchSkipID(t, users, host.Users)
}

func testHostsLoadHostByDeviceAuthToken(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	validToken := "abcd"
	err = ds.SetOrUpdateDeviceAuthToken(context.Background(), host.ID, validToken)
	require.NoError(t, err)

	_, err = ds.LoadHostByDeviceAuthToken(context.Background(), "nosuchtoken", time.Hour)
	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	h, err := ds.LoadHostByDeviceAuthToken(context.Background(), validToken, time.Hour)
	require.NoError(t, err)
	require.Equal(t, host.ID, h.ID)

	time.Sleep(2 * time.Second) // make sure the token expires

	_, err = ds.LoadHostByDeviceAuthToken(context.Background(), validToken, time.Second) // 1s TTL
	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	createHostWithDeviceToken := func(tag string) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			Platform:        tag,
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String(tag),
			NodeKey:         ptr.String(tag),
			UUID:            tag,
			Hostname:        tag + ".local",
		})
		require.NoError(t, err)

		err = ds.SetOrUpdateDeviceAuthToken(context.Background(), h.ID, tag)
		require.NoError(t, err)

		return h
	}

	// create a host enrolled in Simple MDM
	hSimple := createHostWithDeviceToken("simple")
	err = ds.SetOrUpdateMDMData(ctx, hSimple.ID, false, true, "https://simplemdm.com", true, fleet.WellKnownMDMSimpleMDM, "")
	require.NoError(t, err)

	loadSimple, err := ds.LoadHostByDeviceAuthToken(ctx, "simple", time.Second)
	require.NoError(t, err)

	require.Equal(t, hSimple.ID, loadSimple.ID)
	require.True(t, loadSimple.IsOsqueryEnrolled())

	// create a host that will be pending enrollment in Fleet MDM
	hFleet := createHostWithDeviceToken("fleet")
	err = ds.SetOrUpdateMDMData(ctx, hFleet.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)

	loadFleet, err := ds.LoadHostByDeviceAuthToken(ctx, "fleet", time.Second)
	require.NoError(t, err)

	require.Equal(t, hFleet.ID, loadFleet.ID)
	require.True(t, loadFleet.IsOsqueryEnrolled())

	// force its is_server mdm field to NULL, should be same as false
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_mdm SET is_server = NULL WHERE host_id = ?`, hFleet.ID)
		return err
	})
	loadFleet, err = ds.LoadHostByDeviceAuthToken(ctx, "fleet", time.Second)
	require.NoError(t, err)

	require.Equal(t, hFleet.ID, loadFleet.ID)
}

func testHostsSetOrUpdateDeviceAuthToken(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		OsqueryHostID:   ptr.String("1"),
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		OsqueryHostID:   ptr.String("2"),
		Hostname:        "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
	})
	require.NoError(t, err)

	loadUpdatedAt := func(hostID uint) time.Time {
		var ts time.Time
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &ts, `SELECT updated_at FROM host_device_auth WHERE host_id = ?`, hostID)
		})
		return ts
	}

	token1 := "token1"
	err = ds.SetOrUpdateDeviceAuthToken(context.Background(), host.ID, token1)
	require.NoError(t, err)

	token2 := "token2"
	err = ds.SetOrUpdateDeviceAuthToken(context.Background(), host2.ID, token2)
	require.NoError(t, err)
	h2T1 := loadUpdatedAt(host2.ID)

	h, err := ds.LoadHostByDeviceAuthToken(context.Background(), token1, time.Hour)
	require.NoError(t, err)
	require.Equal(t, host.ID, h.ID)

	h, err = ds.LoadHostByDeviceAuthToken(context.Background(), token2, time.Hour)
	require.NoError(t, err)
	require.Equal(t, host2.ID, h.ID)

	time.Sleep(time.Second) // ensure the mysql timestamp is different

	token2Updated := "token2_updated"
	err = ds.SetOrUpdateDeviceAuthToken(context.Background(), host2.ID, token2Updated)
	require.NoError(t, err)
	h2T2 := loadUpdatedAt(host2.ID)
	require.True(t, h2T2.After(h2T1))

	h, err = ds.LoadHostByDeviceAuthToken(context.Background(), token1, time.Hour)
	require.NoError(t, err)
	require.Equal(t, host.ID, h.ID)

	h, err = ds.LoadHostByDeviceAuthToken(context.Background(), token2Updated, time.Hour)
	require.NoError(t, err)
	require.Equal(t, host2.ID, h.ID)

	_, err = ds.LoadHostByDeviceAuthToken(context.Background(), token2, time.Hour)
	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	time.Sleep(time.Second) // ensure the mysql timestamp is different

	// update with the same token, should not change the updated_at timestamp
	err = ds.SetOrUpdateDeviceAuthToken(context.Background(), host2.ID, token2Updated)
	require.NoError(t, err)
	h2T3 := loadUpdatedAt(host2.ID)
	require.True(t, h2T2.Equal(h2T3))
}

func testOSVersions(t *testing.T, ds *Datastore) {
	// empty tables
	err := ds.UpdateOSVersions(context.Background())
	require.NoError(t, err)

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	team2, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team2",
	})
	require.NoError(t, err)

	team3, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team3",
	})
	require.NoError(t, err)

	// create some hosts for testing
	hosts := []*fleet.Host{
		{
			Platform:  "darwin",
			OSVersion: "macOS 12.1.0", // os_version_id = 1
		},
		{
			Platform:  "darwin",
			OSVersion: "macOS 12.2.1", // os_version_id = 2
			TeamID:    &team1.ID,
		},
		{
			Platform:  "darwin",
			OSVersion: "macOS 12.2.1", // os_version_id = 2
			TeamID:    &team1.ID,
		},
		{
			Platform:  "darwin",
			OSVersion: "macOS 12.2.1", // os_version_id = 2
			TeamID:    &team2.ID,
		},
		{
			Platform:  "darwin",
			OSVersion: "macOS 12.3.0", // os_version_id = 3
			TeamID:    &team2.ID,
		},
		{
			Platform:  "darwin",
			OSVersion: "macOS 12.3.0", // os_version_id = 3
			TeamID:    &team2.ID,
		},
		{
			Platform:  "darwin",
			OSVersion: "macOS 12.3.0", // os_version_id = 3
			TeamID:    &team2.ID,
		},
		{
			Platform:  "rhel",
			OSVersion: "CentOS 8.0.0", // os_version_id = 4
		},
		{
			Platform:  "ubuntu",
			OSVersion: "Ubuntu 20.4.0", // os_version_id = 5
			TeamID:    &team1.ID,
		},
		{
			Platform:  "ubuntu",
			OSVersion: "Ubuntu 20.4.0", // os_version_id = 5
			TeamID:    &team1.ID,
		},
	}

	for i, host := range hosts {
		host.DetailUpdatedAt = time.Now()
		host.LabelUpdatedAt = time.Now()
		host.PolicyUpdatedAt = time.Now()
		host.SeenTime = time.Now()
		host.OsqueryHostID = ptr.String(strconv.Itoa(i))
		host.NodeKey = ptr.String(strconv.Itoa(i))
		host.UUID = strconv.Itoa(i)
		host.Hostname = fmt.Sprintf("%d.localdomain", i)

		_, err := ds.NewHost(context.Background(), host)
		require.NoError(t, err)
	}

	ctx := context.Background()

	// add host operating system records
	for _, h := range hosts {
		nv := strings.Split(h.OSVersion, " ")
		err := ds.UpdateHostOperatingSystem(ctx, h.ID, fleet.OperatingSystem{Name: nv[0], Version: nv[1], Platform: h.Platform, Arch: "x86_64"})
		require.NoError(t, err)
	}
	osList, err := ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, osList, 5)
	osByNameVers := make(map[string]fleet.OperatingSystem)
	for _, os := range osList {
		osByNameVers[fmt.Sprintf("%s %s", os.Name, os.Version)] = os
	}

	err = ds.UpdateOSVersions(ctx)
	require.NoError(t, err)

	// all hosts
	osVersions, err := ds.OSVersions(ctx, nil, nil, nil, nil)
	require.NoError(t, err)

	require.True(t, time.Now().After(osVersions.CountsUpdatedAt))
	expected := []fleet.OSVersion{
		{HostsCount: 1, Name: "CentOS 8.0.0", NameOnly: "CentOS", Version: "8.0.0", Platform: "rhel", OSVersionID: 4},
		{HostsCount: 2, Name: "Ubuntu 20.4.0", NameOnly: "Ubuntu", Version: "20.4.0", Platform: "ubuntu", OSVersionID: 5},
		{HostsCount: 1, Name: "macOS 12.1.0", NameOnly: "macOS", Version: "12.1.0", Platform: "darwin", OSVersionID: 1},
		{HostsCount: 3, Name: "macOS 12.2.1", NameOnly: "macOS", Version: "12.2.1", Platform: "darwin", OSVersionID: 2},
		{HostsCount: 3, Name: "macOS 12.3.0", NameOnly: "macOS", Version: "12.3.0", Platform: "darwin", OSVersionID: 3},
	}
	require.Equal(t, expected, osVersions.OSVersions)

	// filter by platform
	platform := "darwin"
	osVersions, err = ds.OSVersions(ctx, nil, &platform, nil, nil)
	require.NoError(t, err)

	expected = []fleet.OSVersion{
		{HostsCount: 1, Name: "macOS 12.1.0", NameOnly: "macOS", Version: "12.1.0", Platform: "darwin", OSVersionID: 1},
		{HostsCount: 3, Name: "macOS 12.2.1", NameOnly: "macOS", Version: "12.2.1", Platform: "darwin", OSVersionID: 2},
		{HostsCount: 3, Name: "macOS 12.3.0", NameOnly: "macOS", Version: "12.3.0", Platform: "darwin", OSVersionID: 3},
	}
	require.Equal(t, expected, osVersions.OSVersions)

	// filter by Linux pseudo-platform
	platform = "linux"
	osVersions, err = ds.OSVersions(ctx, nil, &platform, nil, nil)
	require.NoError(t, err)

	expected = []fleet.OSVersion{
		{HostsCount: 1, Name: "CentOS 8.0.0", NameOnly: "CentOS", Version: "8.0.0", Platform: "rhel", OSVersionID: 4},
		{HostsCount: 2, Name: "Ubuntu 20.4.0", NameOnly: "Ubuntu", Version: "20.4.0", Platform: "ubuntu", OSVersionID: 5},
	}
	require.Equal(t, expected, osVersions.OSVersions)

	// filter by operating system name and version
	osVersions, err = ds.OSVersions(ctx, nil, nil, ptr.String("Ubuntu"), ptr.String("20.4.0"))
	require.NoError(t, err)

	expected = []fleet.OSVersion{
		{HostsCount: 2, Name: "Ubuntu 20.4.0", NameOnly: "Ubuntu", Version: "20.4.0", Platform: "ubuntu", OSVersionID: 5},
	}
	require.Equal(t, expected, osVersions.OSVersions)

	// filter by operating system that has multiple versions
	expected = []fleet.OSVersion{
		{HostsCount: 3, Name: "macOS 12.3.0", NameOnly: "macOS", Version: "12.3.0", Platform: "darwin", OSVersionID: 3},
	}
	osVersions, err = ds.OSVersions(ctx, nil, nil, ptr.String("macOS"), ptr.String("12.3.0"))
	require.NoError(t, err)
	require.Equal(t, expected, osVersions.OSVersions)

	osVersion, _, err := ds.OSVersion(ctx, 3, nil)
	require.NoError(t, err)
	require.Equal(t, &expected[0], osVersion)

	// team 1
	userAdmin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	osVersions, err = ds.OSVersions(ctx, &fleet.TeamFilter{TeamID: &team1.ID, User: userAdmin}, nil, nil, nil)
	require.NoError(t, err)

	expected = []fleet.OSVersion{
		{HostsCount: 2, Name: "Ubuntu 20.4.0", NameOnly: "Ubuntu", Version: "20.4.0", Platform: "ubuntu", OSVersionID: 5},
		{HostsCount: 2, Name: "macOS 12.2.1", NameOnly: "macOS", Version: "12.2.1", Platform: "darwin", OSVersionID: 2},
	}
	require.Equal(t, expected, osVersions.OSVersions)

	osVersion, _, err = ds.OSVersion(ctx, 5, &fleet.TeamFilter{TeamID: &team1.ID})
	require.NoError(t, err)
	require.Equal(t, &expected[0], osVersion)

	osVersion, _, err = ds.OSVersion(ctx, 2, &fleet.TeamFilter{TeamID: &team1.ID, User: userAdmin})
	require.NoError(t, err)
	require.Equal(t, &expected[1], osVersion)

	userTeam1 := &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleAdmin}}}
	osVersions, err = ds.OSVersions(ctx, &fleet.TeamFilter{User: userTeam1}, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, expected, osVersions.OSVersions)

	osVersion, _, err = ds.OSVersion(ctx, 2, &fleet.TeamFilter{User: userTeam1})
	require.NoError(t, err)
	require.Equal(t, &expected[1], osVersion)

	// team 2
	osVersions, err = ds.OSVersions(ctx, &fleet.TeamFilter{TeamID: &team2.ID}, nil, nil, nil)
	require.NoError(t, err)

	expected = []fleet.OSVersion{
		{HostsCount: 1, Name: "macOS 12.2.1", NameOnly: "macOS", Version: "12.2.1", Platform: "darwin", OSVersionID: 2},
		{HostsCount: 3, Name: "macOS 12.3.0", NameOnly: "macOS", Version: "12.3.0", Platform: "darwin", OSVersionID: 3},
	}
	require.Equal(t, expected, osVersions.OSVersions)

	osVersion, _, err = ds.OSVersion(ctx, 2, &fleet.TeamFilter{TeamID: &team2.ID})
	require.NoError(t, err)
	require.Equal(t, &expected[0], osVersion)

	osVersion, _, err = ds.OSVersion(ctx, 3, &fleet.TeamFilter{TeamID: &team2.ID})
	require.NoError(t, err)
	require.Equal(t, &expected[1], osVersion)

	// Wrong team
	_, _, err = ds.OSVersion(ctx, 3, &fleet.TeamFilter{User: userTeam1})
	require.True(t, fleet.IsNotFound(err))

	// team 3 (no hosts assigned to team)
	osVersions, err = ds.OSVersions(ctx, &fleet.TeamFilter{TeamID: &team3.ID}, nil, nil, nil)
	require.NoError(t, err)
	expected = []fleet.OSVersion{}
	require.Equal(t, expected, osVersions.OSVersions)

	osVersion, _, err = ds.OSVersion(ctx, 2, &fleet.TeamFilter{TeamID: &team3.ID})
	require.Error(t, err)
	require.Nil(t, osVersion)

	// non-existent team
	_, err = ds.OSVersions(ctx, &fleet.TeamFilter{TeamID: ptr.Uint(404)}, nil, nil, nil)
	require.Error(t, err)

	// new host with arm64
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("666"),
		NodeKey:         ptr.String("666"),
		UUID:            "666",
		Hostname:        fmt.Sprintf("%s.localdomain", "666"),
	})
	require.NoError(t, err)

	err = ds.UpdateHostOperatingSystem(ctx, h.ID, fleet.OperatingSystem{
		Name:     "macOS",
		Version:  "12.2.1",
		Platform: "darwin",
		Arch:     "arm64",
	})
	require.NoError(t, err)

	// different architecture is considered a unique operating system
	newOSList, err := ds.ListOperatingSystems(ctx)
	require.NoError(t, err)
	require.Len(t, newOSList, len(osList)+1)

	// but aggregate stats should group architectures together
	err = ds.UpdateOSVersions(ctx)
	require.NoError(t, err)

	osVersions, err = ds.OSVersions(ctx, nil, nil, nil, nil)
	require.NoError(t, err)

	expected = []fleet.OSVersion{
		{HostsCount: 1, Name: "CentOS 8.0.0", NameOnly: "CentOS", Version: "8.0.0", Platform: "rhel", OSVersionID: 4},
		{HostsCount: 2, Name: "Ubuntu 20.4.0", NameOnly: "Ubuntu", Version: "20.4.0", Platform: "ubuntu", OSVersionID: 5},
		{HostsCount: 1, Name: "macOS 12.1.0", NameOnly: "macOS", Version: "12.1.0", Platform: "darwin", OSVersionID: 1},
		{HostsCount: 4, Name: "macOS 12.2.1", NameOnly: "macOS", Version: "12.2.1", Platform: "darwin", OSVersionID: 2}, // includes new arm64 host
		{HostsCount: 3, Name: "macOS 12.3.0", NameOnly: "macOS", Version: "12.3.0", Platform: "darwin", OSVersionID: 3},
	}
	require.Equal(t, expected, osVersions.OSVersions)
}

func testHostsDeleteHosts(t *testing.T, ds *Datastore) {
	// Updates hosts and host_seen_times.
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	// enroll in Fleet MDM
	nanoEnroll(t, ds, host, false)

	// Updates host_software.
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "1.0.0", Source: "deb_packages"},
	}
	_, err = ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	// Updates host_users.
	users := []fleet.HostUser{
		{
			Uid:       42,
			Username:  "user1",
			Type:      "aaa",
			GroupName: "group",
			Shell:     "shell",
		},
		{
			Uid:       43,
			Username:  "user2",
			Type:      "bbb",
			GroupName: "group2",
			Shell:     "bash",
		},
	}
	err = ds.SaveHostUsers(context.Background(), host.ID, users)
	require.NoError(t, err)
	// Updates host_emails.
	err = ds.ReplaceHostDeviceMapping(context.Background(), host.ID, []*fleet.HostDeviceMapping{
		{HostID: host.ID, Email: "a@b.c", Source: "src"},
	}, "src")
	require.NoError(t, err)

	// Updates host_additional.
	additional := json.RawMessage(`{"additional": "result"}`)
	err = ds.SaveHostAdditional(context.Background(), host.ID, &additional)
	require.NoError(t, err)

	// Updates scheduled_query_stats.
	pack, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "test1",
		HostIDs: []uint{host.ID},
	})
	require.NoError(t, err)
	query := test.NewQuery(t, ds, nil, "time", "select * from time", 0, true)
	squery := test.NewScheduledQuery(t, ds, pack.ID, query.ID, 30, true, true, "time-scheduled")
	stats := []fleet.ScheduledQueryStats{
		{
			ScheduledQueryName: squery.Name,
			ScheduledQueryID:   squery.ID,
			QueryName:          query.Name,
			PackName:           pack.Name,
			PackID:             pack.ID,
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
	hostPackStats := []fleet.PackStats{
		{
			PackName:   "test1",
			QueryStats: stats,
		},
	}
	err = ds.SaveHostPackStats(context.Background(), host.TeamID, host.ID, hostPackStats)
	require.NoError(t, err)

	// Updates label_membership.
	labelID := uint(1)
	label := &fleet.LabelSpec{
		ID:    labelID,
		Name:  "label foo",
		Query: "select * from time;",
	}
	err = ds.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{label})
	require.NoError(t, err)
	err = ds.RecordLabelQueryExecutions(context.Background(), host, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)
	// Update policy_membership.
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	policy, err := ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		Name:  "policy foo",
		Query: "select * from time",
	})
	require.NoError(t, err)

	// update policy_results
	_, err = ds.writer(context.Background()).Exec(`INSERT INTO query_results (host_id, query_id, last_fetched, data) VALUES (?, ?, ?, ?)`, host.ID, policy.ID, time.Now(), `{"foo": "bar"}`)
	require.NoError(t, err)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{policy.ID: ptr.Bool(true)}, time.Now(), false))
	// Update host_mdm.
	err = ds.SetOrUpdateMDMData(context.Background(), host.ID, false, true, "foo.mdm.example.com", false, "", "")
	require.NoError(t, err)
	// Update host_munki_info.
	err = ds.SetOrUpdateMunkiInfo(context.Background(), host.ID, "42", []string{"a"}, []string{"b"})
	require.NoError(t, err)
	// Update device_auth_token.
	err = ds.SetOrUpdateDeviceAuthToken(context.Background(), host.ID, "foo")
	require.NoError(t, err)
	// Update host_batteries
	err = ds.ReplaceHostBatteries(context.Background(), host.ID, []*fleet.HostBattery{{HostID: host.ID, SerialNumber: "a"}})
	require.NoError(t, err)
	// Update host_operating_system
	err = ds.UpdateHostOperatingSystem(context.Background(), host.ID, fleet.OperatingSystem{Name: "foo", Version: "bar"})
	require.NoError(t, err)
	// Insert a windows update for the host
	stmt := `INSERT INTO windows_updates (host_id, date_epoch, kb_id) VALUES (?, ?, ?)`
	_, err = ds.writer(context.Background()).Exec(stmt, host.ID, 1, 123)
	require.NoError(t, err)
	// set host' disk space
	err = ds.SetOrUpdateHostDisksSpace(context.Background(), host.ID, 12, 25, 40.0)
	require.NoError(t, err)
	// set host orbit info
	err = ds.SetOrUpdateHostOrbitInfo(
		context.Background(), host.ID, "1.1.0", sql.NullString{String: "2.1.0", Valid: true}, sql.NullBool{Bool: true, Valid: true},
	)
	require.NoError(t, err)
	// set an encryption key
	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host.ID, "TESTKEY", "", nil)
	require.NoError(t, err)
	// set an mdm profile
	prof, err := ds.NewMDMAppleConfigProfile(context.Background(), *configProfileForTest(t, "N1", "I1", "U1"))
	require.NoError(t, err)
	err = ds.BulkUpsertMDMAppleHostProfiles(context.Background(), []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{ProfileUUID: prof.ProfileUUID, ProfileIdentifier: prof.Identifier, ProfileName: prof.Name, HostUUID: host.UUID, OperationType: fleet.MDMOperationTypeInstall, Checksum: []byte("csum")},
	})
	require.NoError(t, err)

	_, err = ds.writer(context.Background()).Exec(`INSERT INTO host_software_installed_paths (host_id, software_id, installed_path) VALUES (?, ?, ?)`, host.ID, 1, "some_path")
	require.NoError(t, err)

	_, err = ds.writer(context.Background()).Exec(`INSERT INTO host_dep_assignments (host_id) VALUES (?)`, host.ID)
	require.NoError(t, err)

	_, err = ds.writer(context.Background()).Exec(`
          INSERT INTO nano_commands (command_uuid, request_type, command)
          VALUES ('command-uuid', 'foo', '<?xml')
	`)
	require.NoError(t, err)
	err = ds.InsertMDMAppleBootstrapPackage(context.Background(), &fleet.MDMAppleBootstrapPackage{
		TeamID: uint(0),
		Name:   t.Name(),
		Sha256: sha256.New().Sum(nil),
		Bytes:  []byte("content"),
		Token:  uuid.New().String(),
	}, nil)
	require.NoError(t, err)
	err = ds.RecordHostBootstrapPackage(context.Background(), "command-uuid", host.UUID)
	require.NoError(t, err)
	_, err = ds.NewHostScriptExecutionRequest(context.Background(), &fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "foo"})
	require.NoError(t, err)

	_, err = ds.writer(context.Background()).Exec(`
          INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, command_uuid)
          VALUES (?, uuid(), uuid())
	`, host.UUID)
	require.NoError(t, err)

	_, err = ds.writer(context.Background()).Exec(`
          INSERT INTO host_mdm_apple_declarations (host_uuid, declaration_uuid)
          VALUES (?, uuid())
	`, host.UUID)
	require.NoError(t, err)

	var activity fleet.ActivityDetails = fleet.ActivityTypeRanScript{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
	}
	detailsBytes, err := json.Marshal(activity)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), fleet.ActivityWebhookContextKey, true)
	err = ds.NewActivity( // automatically creates the host_activities entry
		ctx,
		user1,
		activity,
		detailsBytes,
		time.Now(),
	)
	require.NoError(t, err)

	// Update the host_mdm_actions table
	_, err = ds.writer(context.Background()).Exec(`
          INSERT INTO host_mdm_actions (host_id, lock_ref, wipe_ref)
          VALUES (?, uuid(), uuid())
	`, host.ID)
	require.NoError(t, err)

	// Add a calendar event for the host.
	_, err = ds.writer(context.Background()).Exec(`
		          INSERT INTO calendar_events (email, start_time, end_time, event)
		          VALUES ('foobar@example.com', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, '{}');
			`)
	require.NoError(t, err)
	var calendarEventID int
	err = ds.writer(context.Background()).Get(&calendarEventID, `
		          SELECT id FROM calendar_events WHERE email = 'foobar@example.com';
			`)
	require.NoError(t, err)
	_, err = ds.writer(context.Background()).Exec(`
		          INSERT INTO host_calendar_events (host_id, calendar_event_id, webhook_status)
		          VALUES (?, ?, 1);
			`, host.ID, calendarEventID)
	require.NoError(t, err)

	softwareInstaller, err := ds.MatchOrCreateSoftwareInstaller(context.Background(), &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "",
		PreInstallQuery: "",
		Title:           "ChocolateRain",
		UserID:          user1.ID,
	})
	require.NoError(t, err)
	_, err = ds.InsertSoftwareInstallRequest(context.Background(), host.ID, softwareInstaller, false, nil)
	require.NoError(t, err)

	// Add an awaiting configuration entry
	err = ds.SetHostAwaitingConfiguration(ctx, host.UUID, false)
	require.NoError(t, err)

	// Add a setup experience status result
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{Name: "test.sh", ScriptContents: "echo foo"})
	require.NoError(t, err)

	added, err := ds.EnqueueSetupExperienceItems(ctx, host.UUID, 0)
	require.NoError(t, err)
	require.True(t, added)

	// Check there's an entry for the host in all the associated tables.
	for _, hostRef := range hostRefs {
		var ok bool
		err = ds.writer(context.Background()).Get(&ok, fmt.Sprintf("SELECT 1 FROM %s WHERE host_id = ?", hostRef), host.ID)
		require.NoError(t, err, hostRef)
		require.True(t, ok, "table: %s", hostRef)
	}
	for tbl, col := range additionalHostRefsByUUID {
		var ok bool
		err = ds.writer(context.Background()).Get(&ok, fmt.Sprintf("SELECT 1 FROM %s WHERE %s = ?", tbl, col), host.UUID)
		require.NoError(t, err, tbl)
		require.True(t, ok, "table: %s", tbl)
	}
	for tbl, col := range additionalHostRefsSoftDelete {
		var ok bool
		err = ds.writer(context.Background()).Get(&ok, fmt.Sprintf("SELECT 1 FROM %s WHERE host_id = ? AND %s IS NULL", tbl, col), host.ID)
		require.NoError(t, err, tbl)
		require.True(t, ok, "table: %s", tbl)
	}

	err = ds.DeleteHosts(context.Background(), []uint{host.ID})
	require.NoError(t, err)

	// Check that all the associated tables were cleaned up.
	for _, hostRef := range hostRefs {
		var ok bool
		err = ds.writer(context.Background()).Get(&ok, fmt.Sprintf("SELECT 1 FROM %s WHERE host_id = ?", hostRef), host.ID)
		require.True(t, err == nil || errors.Is(err, sql.ErrNoRows), "table: %s", hostRef)
		require.False(t, ok, "table: %s", hostRef)
	}
	for tbl, col := range additionalHostRefsByUUID {
		var ok bool
		err = ds.writer(context.Background()).Get(&ok, fmt.Sprintf("SELECT 1 FROM %s WHERE %s = ?", tbl, col), host.UUID)
		require.True(t, err == nil || errors.Is(err, sql.ErrNoRows), "table: %s", tbl)
		require.False(t, ok, "table: %s", tbl)
	}
	for tbl, col := range additionalHostRefsSoftDelete {
		var ok bool
		err = ds.writer(context.Background()).Get(&ok, fmt.Sprintf("SELECT 1 FROM %s WHERE host_id = ? AND %s IS NULL", tbl, col), host.ID)
		require.True(t, err == nil || errors.Is(err, sql.ErrNoRows), "table: %s", tbl)
		require.False(t, ok, "table: %s", tbl) // the soft-delete column is not null anymore, so no row is found
	}
}

func testHostIDsByOSVersion(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	hosts := make([]*fleet.Host, 10)
	getPlatform := func(i int) string {
		if i < 5 {
			return "ubuntu"
		}
		return "centos"
	}
	for i := range hosts {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String(fmt.Sprintf("host%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.%d.local", i),
			Platform:        getPlatform(i),
			OSVersion:       fmt.Sprintf("20.4.%d", i),
		})
		require.NoError(t, err)
		hosts[i] = h
	}

	t.Run("no match", func(t *testing.T) {
		osVersion := fleet.OSVersion{Platform: "ubuntu", Name: "sdfasw"}
		none, err := ds.HostIDsByOSVersion(ctx, osVersion, 0, 1)
		require.NoError(t, err)
		require.Len(t, none, 0)
	})

	t.Run("filtering by os version", func(t *testing.T) {
		osVersion := fleet.OSVersion{Platform: "ubuntu", Name: "20.4.0"}
		result, err := ds.HostIDsByOSVersion(ctx, osVersion, 0, 1)
		require.NoError(t, err)
		require.Len(t, result, 1)
		for _, id := range result {
			r, err := ds.Host(ctx, id)
			require.NoError(t, err)
			require.Equal(t, r.Platform, "ubuntu")
			require.Equal(t, r.OSVersion, "20.4.0")
		}
	})
}

func testHostsReplaceHostBatteries(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	h1, err := ds.NewHost(ctx, &fleet.Host{
		ID:              1,
		OsqueryHostID:   ptr.String("1"),
		NodeKey:         ptr.String("1"),
		Platform:        "linux",
		Hostname:        "host1",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)
	h2, err := ds.NewHost(ctx, &fleet.Host{
		ID:              2,
		OsqueryHostID:   ptr.String("2"),
		NodeKey:         ptr.String("2"),
		Platform:        "linux",
		Hostname:        "host2",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	err = ds.ReplaceHostBatteries(ctx, h1.ID, nil)
	require.NoError(t, err)

	bat1, err := ds.ListHostBatteries(ctx, h1.ID)
	require.NoError(t, err)
	require.Len(t, bat1, 0)

	h1Bat := []*fleet.HostBattery{
		{HostID: h1.ID, SerialNumber: "a", CycleCount: 1, Health: "Good"},
		{HostID: h1.ID, SerialNumber: "b", CycleCount: 2, Health: "Check Battery"},
	}
	err = ds.ReplaceHostBatteries(ctx, h1.ID, h1Bat)
	require.NoError(t, err)

	bat1, err = ds.ListHostBatteries(ctx, h1.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, h1Bat, bat1)

	type timestamp struct {
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	var timestamps1 []timestamp
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &timestamps1, `SELECT created_at, updated_at FROM host_batteries WHERE host_id = ?`, h1.ID)
	})

	// Insert the same battery data again.
	err = ds.ReplaceHostBatteries(ctx, h1.ID, h1Bat)
	require.NoError(t, err)

	var timestamps2 []timestamp
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &timestamps2, `SELECT created_at, updated_at FROM host_batteries WHERE host_id = ?`, h1.ID)
	})

	// Verify that there were no inserts/updates (because reported data hasn't changed).
	require.ElementsMatch(t, timestamps1, timestamps2)

	bat1, err = ds.ListHostBatteries(ctx, h1.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, h1Bat, bat1)

	bat2, err := ds.ListHostBatteries(ctx, h2.ID)
	require.NoError(t, err)
	require.Len(t, bat2, 0)

	// update "a", remove "b", add "c"
	h1Bat = []*fleet.HostBattery{
		{HostID: h1.ID, SerialNumber: "a", CycleCount: 2, Health: "Good"},
		{HostID: h1.ID, SerialNumber: "c", CycleCount: 3, Health: "Bad"},
	}

	err = ds.ReplaceHostBatteries(ctx, h1.ID, h1Bat)
	require.NoError(t, err)

	bat1, err = ds.ListHostBatteries(ctx, h1.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, h1Bat, bat1)

	// add "d" to h2
	h2Bat := []*fleet.HostBattery{
		{HostID: h2.ID, SerialNumber: "d", CycleCount: 1, Health: "Good"},
	}

	err = ds.ReplaceHostBatteries(ctx, h2.ID, h2Bat)
	require.NoError(t, err)

	bat2, err = ds.ListHostBatteries(ctx, h2.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, h2Bat, bat2)

	// remove all from h1
	h1Bat = []*fleet.HostBattery{}

	err = ds.ReplaceHostBatteries(ctx, h1.ID, h1Bat)
	require.NoError(t, err)

	bat1, err = ds.ListHostBatteries(ctx, h1.ID)
	require.NoError(t, err)
	require.Len(t, bat1, 0)

	// h2 unchanged
	bat2, err = ds.ListHostBatteries(ctx, h2.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, h2Bat, bat2)
}

func testHostsReplaceHostBatteriesDeadlock(t *testing.T, ds *Datastore) {
	// To increase chance of deadlock increase these numbers.
	// We are keeping them low to not cause CI issues ("too many connections" errors
	// due to concurrent tests).
	const (
		hostCount    = 10
		replaceCount = 10
	)
	ctx := context.Background()
	var hosts []*fleet.Host
	for i := 1; i <= hostCount; i++ {
		h, err := ds.NewHost(ctx, &fleet.Host{
			ID:              uint(i),
			OsqueryHostID:   ptr.String(fmt.Sprintf("id-%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("key-%d", i)),
			Platform:        "linux",
			Hostname:        fmt.Sprintf("host-%d", i),
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
		})
		require.NoError(t, err)
		hosts = append(hosts, h)
	}

	var g errgroup.Group
	for _, h := range hosts {
		hostID := h.ID
		g.Go(func() error {
			for i := 0; i < replaceCount; i++ {
				if err := ds.ReplaceHostBatteries(ctx, hostID, []*fleet.HostBattery{
					{HostID: hostID, SerialNumber: fmt.Sprintf("%d-0000", hostID), CycleCount: 1, Health: "Good"},
					{HostID: hostID, SerialNumber: fmt.Sprintf("%d-0001", hostID), CycleCount: 2, Health: "Fair"},
				}); err != nil {
					return err
				}
				time.Sleep(10 * time.Millisecond)
			}
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
}

func testCountHostsNotResponding(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	config := config.FleetConfig{Osquery: config.OsqueryConfig{DetailUpdateInterval: 1 * time.Hour}}

	// responsive
	_, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:       ptr.String("1"),
		NodeKey:             ptr.String("1"),
		Platform:            "linux",
		Hostname:            "host1",
		DistributedInterval: 10,
		DetailUpdatedAt:     time.Now().Add(-1 * time.Hour),
		LabelUpdatedAt:      time.Now(),
		PolicyUpdatedAt:     time.Now(),
		SeenTime:            time.Now(),
	})
	require.NoError(t, err)

	count, err := countHostsNotRespondingDB(ctx, ds.writer(ctx), ds.logger, config)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// not responsive
	_, err = ds.NewHost(ctx, &fleet.Host{
		ID:                  2,
		OsqueryHostID:       ptr.String("2"),
		NodeKey:             ptr.String("2"),
		Platform:            "linux",
		Hostname:            "host2",
		DistributedInterval: 10,
		DetailUpdatedAt:     time.Now().Add(-3 * time.Hour),
		LabelUpdatedAt:      time.Now().Add(-3 * time.Hour),
		PolicyUpdatedAt:     time.Now().Add(-3 * time.Hour),
		SeenTime:            time.Now(),
	})
	require.NoError(t, err)

	count, err = countHostsNotRespondingDB(ctx, ds.writer(ctx), ds.logger, config)
	require.NoError(t, err)
	require.Equal(t, 1, count) // count increased by 1

	// responsive
	_, err = ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:       ptr.String("3"),
		NodeKey:             ptr.String("3"),
		Platform:            "linux",
		Hostname:            "host3",
		DistributedInterval: 10,
		DetailUpdatedAt:     time.Now().Add(-49 * time.Hour),
		LabelUpdatedAt:      time.Now().Add(-48 * time.Hour),
		PolicyUpdatedAt:     time.Now().Add(-48 * time.Hour),
		SeenTime:            time.Now().Add(-48 * time.Hour),
	})
	require.NoError(t, err)

	count, err = countHostsNotRespondingDB(ctx, ds.writer(ctx), ds.logger, config)
	require.NoError(t, err)
	require.Equal(t, 1, count) // count unchanged

	// not responsive
	_, err = ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:       ptr.String("4"),
		NodeKey:             ptr.String("4"),
		Platform:            "linux",
		Hostname:            "host4",
		DistributedInterval: 10,
		DetailUpdatedAt:     time.Now().Add(-51 * time.Hour),
		LabelUpdatedAt:      time.Now().Add(-48 * time.Hour),
		PolicyUpdatedAt:     time.Now().Add(-48 * time.Hour),
		SeenTime:            time.Now().Add(-48 * time.Hour),
	})
	require.NoError(t, err)

	count, err = countHostsNotRespondingDB(ctx, ds.writer(ctx), ds.logger, config)
	require.NoError(t, err)
	require.Equal(t, 2, count) // count increased by 1

	// was responsive but hasn't been seen in past 7 days so it is not counted
	_, err = ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:       ptr.String("5"),
		NodeKey:             ptr.String("5"),
		Platform:            "linux",
		Hostname:            "host5",
		DistributedInterval: 10,
		DetailUpdatedAt:     time.Now().Add(-8 * 24 * time.Hour).Add(-1 * time.Hour),
		LabelUpdatedAt:      time.Now().Add(-8 * 24 * time.Hour),
		PolicyUpdatedAt:     time.Now().Add(-8 * 24 * time.Hour),
		SeenTime:            time.Now().Add(-8 * 24 * time.Hour),
	})
	require.NoError(t, err)

	count, err = countHostsNotRespondingDB(ctx, ds.writer(ctx), ds.logger, config)
	require.NoError(t, err)
	require.Equal(t, 2, count) // count unchanged

	// distributed interval (1h1m) is greater than osquery detail interval (1h)
	// so measurement period for non-responsiveness is 2h2m
	_, err = ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:       ptr.String("6"),
		NodeKey:             ptr.String("6"),
		Platform:            "linux",
		Hostname:            "host6",
		DistributedInterval: uint((1*time.Hour + 1*time.Minute).Seconds()),        // 1h1m
		DetailUpdatedAt:     time.Now().Add(-2 * time.Hour).Add(-1 * time.Minute), // 2h1m
		LabelUpdatedAt:      time.Now().Add(-2 * time.Hour).Add(-1 * time.Minute),
		PolicyUpdatedAt:     time.Now().Add(-2 * time.Hour).Add(-1 * time.Minute),
		SeenTime:            time.Now(),
	})
	require.NoError(t, err)

	count, err = countHostsNotRespondingDB(ctx, ds.writer(ctx), ds.logger, config)
	require.NoError(t, err)
	require.Equal(t, 2, count) // count unchanged
}

func testFailingPoliciesCount(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var hosts []*fleet.Host
	for i := 0; i < 10; i++ {
		h := test.NewHost(t, ds, fmt.Sprintf("foo.local.%d", i), "1.1.1.1",
			fmt.Sprintf("%d", i), fmt.Sprintf("%d", i), time.Now())
		hosts = append(hosts, h)
	}

	t.Run("no policies", func(t *testing.T) {
		for _, h := range hosts {
			actual, err := ds.FailingPoliciesCount(ctx, h)
			require.NoError(t, err)
			require.Equal(t, actual, uint(0))
		}
	})

	t.Run("with policies and memberships", func(t *testing.T) {
		u := test.NewUser(t, ds, "Bob", "bob@example.com", true)

		var policies []*fleet.Policy
		for i := 0; i < 10; i++ {
			q := test.NewQuery(t, ds, nil, fmt.Sprintf("query%d", i), "select 1", 0, true)
			p, err := ds.NewGlobalPolicy(ctx, &u.ID, fleet.PolicyPayload{QueryID: &q.ID})
			require.NoError(t, err)
			policies = append(policies, p)
		}

		testCases := []struct {
			host     *fleet.Host
			policyEx map[uint]*bool
			expected uint
		}{
			{
				host: hosts[0],
				policyEx: map[uint]*bool{
					policies[0].ID: ptr.Bool(true),
					policies[1].ID: ptr.Bool(true),
					policies[2].ID: ptr.Bool(false),
					policies[3].ID: ptr.Bool(true),
					policies[4].ID: nil,
					policies[5].ID: nil,
				},
				expected: 1,
			},
			{
				host: hosts[1],
				policyEx: map[uint]*bool{
					policies[0].ID: ptr.Bool(true),
					policies[1].ID: ptr.Bool(true),
					policies[2].ID: ptr.Bool(true),
					policies[3].ID: ptr.Bool(true),
					policies[4].ID: ptr.Bool(true),
					policies[5].ID: ptr.Bool(true),
					policies[6].ID: ptr.Bool(true),
					policies[7].ID: ptr.Bool(true),
					policies[8].ID: ptr.Bool(true),
					policies[9].ID: ptr.Bool(true),
				},
				expected: 0,
			},
			{
				host: hosts[2],
				policyEx: map[uint]*bool{
					policies[0].ID: ptr.Bool(true),
					policies[1].ID: ptr.Bool(true),
					policies[2].ID: ptr.Bool(true),
					policies[3].ID: ptr.Bool(true),
					policies[4].ID: ptr.Bool(true),
					policies[5].ID: ptr.Bool(false),
					policies[6].ID: ptr.Bool(false),
					policies[7].ID: ptr.Bool(false),
					policies[8].ID: ptr.Bool(false),
					policies[9].ID: ptr.Bool(false),
				},
				expected: 5,
			},
			{
				host:     hosts[3],
				policyEx: map[uint]*bool{},
				expected: 0,
			},
		}

		for _, tc := range testCases {
			if len(tc.policyEx) != 0 {
				require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, tc.host, tc.policyEx, time.Now(), false))
			}
			actual, err := ds.FailingPoliciesCount(ctx, tc.host)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		}
	})
}

func testHostsRecordNoPolicies(t *testing.T, ds *Datastore) {
	initialTime := time.Now()

	for i := 0; i < 2; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: initialTime,
			LabelUpdatedAt:  initialTime,
			PolicyUpdatedAt: initialTime,
			SeenTime:        initialTime.Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 2)
	require.Len(t, hosts, 2)

	h1 := hosts[0]
	h2 := hosts[1]

	assert.WithinDuration(t, initialTime, h1.PolicyUpdatedAt, 1*time.Second)
	assert.Zero(t, h1.HostIssues.FailingPoliciesCount)
	assert.Zero(t, h1.HostIssues.TotalIssuesCount)
	assert.WithinDuration(t, initialTime, h2.PolicyUpdatedAt, 1*time.Second)
	assert.Zero(t, h2.HostIssues.FailingPoliciesCount)
	assert.Zero(t, h2.HostIssues.TotalIssuesCount)

	policyUpdatedAt := initialTime.Add(1 * time.Hour)
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h1, nil, policyUpdatedAt, false))

	hosts = listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 2)
	require.Len(t, hosts, 2)

	h1 = hosts[0]
	h2 = hosts[1]

	assert.WithinDuration(t, policyUpdatedAt, h1.PolicyUpdatedAt, 1*time.Second)
	assert.Zero(t, h1.HostIssues.FailingPoliciesCount)
	assert.Zero(t, h1.HostIssues.TotalIssuesCount)
	assert.WithinDuration(t, initialTime, h2.PolicyUpdatedAt, 1*time.Second)
	assert.Zero(t, h2.HostIssues.FailingPoliciesCount)
	assert.Zero(t, h2.HostIssues.TotalIssuesCount)
}

func testHostsSetOrUpdateHostDisksSpace(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		OsqueryHostID:   ptr.String("1"),
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		OsqueryHostID:   ptr.String("2"),
		Hostname:        "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
	})
	require.NoError(t, err)

	// set a device host token for host 1, to test loading disk space by device token
	token1 := "token1"
	err = ds.SetOrUpdateDeviceAuthToken(context.Background(), host.ID, token1)
	require.NoError(t, err)

	err = ds.SetOrUpdateHostDisksSpace(context.Background(), host.ID, 1, 2, 50.0)
	require.NoError(t, err)

	err = ds.SetOrUpdateHostDisksSpace(context.Background(), host2.ID, 3, 4, 90.0)
	require.NoError(t, err)

	h, err := ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.Equal(t, 1.0, h.GigsDiskSpaceAvailable)
	require.Equal(t, 2.0, h.PercentDiskSpaceAvailable)

	h, err = ds.LoadHostByNodeKey(context.Background(), *host2.NodeKey)
	require.NoError(t, err)
	require.Equal(t, 3.0, h.GigsDiskSpaceAvailable)
	require.Equal(t, 4.0, h.PercentDiskSpaceAvailable)

	err = ds.SetOrUpdateHostDisksSpace(context.Background(), host.ID, 5, 6, 80.0)
	require.NoError(t, err)

	h, err = ds.LoadHostByDeviceAuthToken(context.Background(), token1, time.Hour)
	require.NoError(t, err)
	require.Equal(t, 5.0, h.GigsDiskSpaceAvailable)
	require.Equal(t, 6.0, h.PercentDiskSpaceAvailable)
}

// testHostOrder tests listing a host sorted by different keys.
func testHostOrder(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	createdHosts := make([]*fleet.Host, 3)
	var err error
	createdHosts[0], err = ds.NewHost(ctx, &fleet.Host{ID: 1, OsqueryHostID: ptr.String("1"), Hostname: "0001", NodeKey: ptr.String("1")})
	require.NoError(t, err)
	createdHosts[1], err = ds.NewHost(
		ctx, &fleet.Host{ID: 2, OsqueryHostID: ptr.String("2"), Hostname: "0002", ComputerName: "0004", NodeKey: ptr.String("2")},
	)
	require.NoError(t, err)
	createdHosts[2], err = ds.NewHost(ctx, &fleet.Host{ID: 3, OsqueryHostID: ptr.String("3"), Hostname: "0003", NodeKey: ptr.String("3")})
	require.NoError(t, err)
	chk := func(hosts []*fleet.Host, expect ...string) {
		require.Len(t, hosts, len(expect))
		for i, h := range hosts {
			assert.Equal(t, expect[i], h.DisplayName())
		}
	}
	hosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey: "display_name",
		},
	})
	require.NoError(t, err)
	chk(hosts, "0001", "0003", "0004")

	_, err = ds.writer(ctx).Exec(`UPDATE hosts SET created_at = DATE_ADD(created_at, INTERVAL id DAY)`)
	require.NoError(t, err)

	hosts, err = ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey:       "created_at",
			After:          "2010-10-22T20:22:03Z",
			OrderDirection: fleet.OrderAscending,
		},
	})
	require.NoError(t, err)
	chk(hosts, "0001", "0004", "0003")

	hosts, err = ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{
		ListOptions: fleet.ListOptions{
			OrderKey:       "created_at",
			After:          "2180-10-22T20:22:03Z",
			OrderDirection: fleet.OrderDescending,
		},
	})
	require.NoError(t, err)
	chk(hosts, "0003", "0004", "0001")

	// Test sorting by issues
	policies := make([]*fleet.Policy, 0, 3)
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	for i := 0; i < 6; i++ {
		q := test.NewQuery(t, ds, nil, fmt.Sprintf("query%d", i), "select 1", 0, true)
		p, err := ds.NewGlobalPolicy(
			context.Background(), &user1.ID, fleet.PolicyPayload{
				QueryID: &q.ID,
			},
		)
		require.NoError(t, err)
		policies = append(policies, p)
	}
	for i := 0; i < 3; i++ {
		results := make(map[uint]*bool, 3)
		for j := 0; j <= i; j++ {
			results[policies[j].ID] = ptr.Bool(false) // fail
		}
		for j := i + 1; j < 3; j++ {
			results[policies[j].ID] = ptr.Bool(true) // pass
		}
		require.NoError(
			t, ds.RecordPolicyQueryExecutions(
				context.Background(), createdHosts[i], results, time.Now(), false,
			),
		)
	}
	hostIDs := make([]uint, len(createdHosts))
	for i, host := range createdHosts {
		hostIDs[i] = host.ID
	}
	assert.NoError(t, ds.UpdateHostIssuesFailingPolicies(ctx, hostIDs))
	hosts, err = ds.ListHosts(
		ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "issues",
				OrderDirection: fleet.OrderDescending,
			},
		},
	)
	require.NoError(t, err)
	chk(hosts, "0003", "0004", "0001")
}

func testHostIDsByOSID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	t.Run("no OS", func(t *testing.T) {
		actual, err := ds.HostIDsByOSID(ctx, 1, 0, 100)
		require.NoError(t, err)
		require.Empty(t, actual)
	})

	t.Run("returns empty if no more pages", func(t *testing.T) {
		for i := 1; i <= 510; i++ {
			os := fleet.OperatingSystem{
				Name:          "Microsoft Windows 11 Enterprise Evaluation II",
				Version:       "21H2",
				Arch:          "64-bit",
				KernelVersion: "10.0.22000.795",
				Platform:      "windows",
			}

			require.NoError(t, ds.UpdateHostOperatingSystem(ctx, uint(i+100), os))
		}

		storedOS, err := ds.ListOperatingSystems(ctx)
		require.NoError(t, err)
		for _, sOS := range storedOS {
			if sOS.Name == "Microsoft Windows 11 Enterprise Evaluation II" {

				actual, err := ds.HostIDsByOSID(ctx, sOS.ID, 0, 500)
				require.NoError(t, err)
				require.Len(t, actual, 500)

				actual, err = ds.HostIDsByOSID(ctx, sOS.ID, 500, 500)
				require.NoError(t, err)
				require.Len(t, actual, 10)

				actual, err = ds.HostIDsByOSID(ctx, sOS.ID, 510, 500)
				require.NoError(t, err)
				require.Empty(t, actual)
				break
			}
		}
	})

	t.Run("returns matching entries", func(t *testing.T) {
		os := []fleet.OperatingSystem{
			{
				Name:          "Microsoft Windows 11 Enterprise Evaluation",
				Version:       "21H2",
				Arch:          "64-bit",
				KernelVersion: "10.0.22000.795",
				Platform:      "windows",
			},
			{
				Name:          "macOS",
				Version:       "12.3.1",
				Arch:          "x86_64",
				KernelVersion: "21.4.0",
				Platform:      "darwin",
			},
		}

		require.NoError(t, ds.UpdateHostOperatingSystem(ctx, 1, os[0]))
		require.NoError(t, ds.UpdateHostOperatingSystem(ctx, 2, os[1]))

		storedOS, err := ds.ListOperatingSystems(ctx)
		require.NoError(t, err)

		for _, sOS := range storedOS {
			actual, err := ds.HostIDsByOSID(ctx, sOS.ID, 0, 100)
			require.NoError(t, err)
			if sOS.Name == "Microsoft Windows 11 Enterprise Evaluation" {
				require.Equal(t, []uint{1}, actual)
			}

			if sOS.Name == "macOS" {
				require.Equal(t, []uint{2}, actual)
			}
		}
	})
}

func testHostsSetOrUpdateHostDisksEncryption(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		OsqueryHostID:   ptr.String("1"),
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		OsqueryHostID:   ptr.String("2"),
		Hostname:        "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
	})
	require.NoError(t, err)

	err = ds.SetOrUpdateHostDisksEncryption(context.Background(), host.ID, true)
	require.NoError(t, err)

	err = ds.SetOrUpdateHostDisksEncryption(context.Background(), host2.ID, false)
	require.NoError(t, err)

	h, err := ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	require.True(t, *h.DiskEncryptionEnabled)

	h, err = ds.Host(context.Background(), host2.ID)
	require.NoError(t, err)
	require.False(t, *h.DiskEncryptionEnabled)

	err = ds.SetOrUpdateHostDisksEncryption(context.Background(), host2.ID, true)
	require.NoError(t, err)

	h, err = ds.Host(context.Background(), host2.ID)
	require.NoError(t, err)
	require.True(t, *h.DiskEncryptionEnabled)
}

func testHostsGetHostMDMCheckinInfo(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	tm, err := ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		OsqueryHostID:   ptr.String("1"),
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		HardwareSerial:  "123456789",
		TeamID:          &tm.ID,
		Platform:        "darwin",
	})
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(ctx, host.ID, false, true, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)

	info, err := ds.GetHostMDMCheckinInfo(ctx, host.UUID)
	require.NoError(t, err)
	require.Equal(t, host.HardwareSerial, info.HardwareSerial)
	require.Equal(t, true, info.InstalledFromDEP)
	require.EqualValues(t, tm.ID, info.TeamID)
	require.False(t, info.DEPAssignedToFleet)
	require.True(t, info.OsqueryEnrolled)
	require.Equal(t, "darwin", info.Platform)

	err = ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*host}, abmToken.ID)
	require.NoError(t, err)
	info, err = ds.GetHostMDMCheckinInfo(ctx, host.UUID)
	require.NoError(t, err)
	require.True(t, info.DEPAssignedToFleet)
	require.True(t, info.OsqueryEnrolled)

	err = ds.DeleteHostDEPAssignments(ctx, abmToken.ID, []string{host.HardwareSerial})
	require.NoError(t, err)
	info, err = ds.GetHostMDMCheckinInfo(ctx, host.UUID)
	require.NoError(t, err)
	require.False(t, info.DEPAssignedToFleet)
	require.True(t, info.OsqueryEnrolled)

	// host with an empty node key
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE hosts SET node_key = NULL WHERE uuid = ?`, host.UUID)
		return err
	})
	info, err = ds.GetHostMDMCheckinInfo(ctx, host.UUID)
	require.NoError(t, err)
	require.False(t, info.OsqueryEnrolled)
}

func testHostsLoadHostByOrbitNodeKey(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

	for _, tt := range enrollTests {
		h, err := ds.EnrollHost(ctx, false, tt.uuid, tt.uuid, "", tt.nodeKey, nil, 0)
		require.NoError(t, err)

		orbitKey := uuid.New().String()
		// on orbit enrollment, the "hardware UUID" is matched with the osquery
		// host ID to identify the host being enrolled
		_, err = ds.EnrollOrbit(ctx, false, fleet.OrbitHostInfo{
			HardwareUUID:   *h.OsqueryHostID,
			HardwareSerial: h.HardwareSerial,
		}, orbitKey, nil)
		require.NoError(t, err)

		// the returned host by LoadHostByOrbitNodeKey will have the orbit key stored
		h.OrbitNodeKey = &orbitKey
		returned, err := ds.LoadHostByOrbitNodeKey(ctx, orbitKey)
		require.NoError(t, err)

		// compare only the fields we care about
		h.CreatedAt = returned.CreatedAt
		h.UpdatedAt = returned.UpdatedAt
		h.DEPAssignedToFleet = ptr.Bool(false)
		assert.Equal(t, h, returned)
	}

	// test loading an unknown orbit key
	_, err = ds.LoadHostByOrbitNodeKey(ctx, uuid.New().String())
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	createOrbitHost := func(tag string) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			Platform:           tag,
			DetailUpdatedAt:    time.Now(),
			LabelUpdatedAt:     time.Now(),
			PolicyUpdatedAt:    time.Now(),
			SeenTime:           time.Now(),
			OsqueryHostID:      ptr.String(tag),
			NodeKey:            ptr.String(tag),
			UUID:               tag,
			Hostname:           tag + ".local",
			DEPAssignedToFleet: ptr.Bool(false),
		})
		require.NoError(t, err)

		orbitKey := uuid.New().String()
		_, err = ds.EnrollOrbit(ctx, false, fleet.OrbitHostInfo{
			HardwareUUID:   *h.OsqueryHostID,
			HardwareSerial: h.HardwareSerial,
		}, orbitKey, nil)
		require.NoError(t, err)
		h.OrbitNodeKey = &orbitKey
		return h
	}

	// create a host enrolled in Simple MDM
	hSimple := createOrbitHost("simple")
	err = ds.SetOrUpdateMDMData(ctx, hSimple.ID, false, true, "https://simplemdm.com", true, fleet.WellKnownMDMSimpleMDM, "")
	require.NoError(t, err)

	loadSimple, err := ds.LoadHostByOrbitNodeKey(ctx, *hSimple.OrbitNodeKey)
	require.NoError(t, err)

	require.Equal(t, hSimple.ID, loadSimple.ID)
	require.True(t, loadSimple.IsOsqueryEnrolled())

	// create a host that will be pending enrollment in Fleet MDM
	hFleet := createOrbitHost("fleet")
	err = ds.SetOrUpdateMDMData(ctx, hFleet.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)

	loadFleet, err := ds.LoadHostByOrbitNodeKey(ctx, *hFleet.OrbitNodeKey)
	require.NoError(t, err)

	require.Equal(t, hFleet.ID, loadFleet.ID)
	require.True(t, loadFleet.IsOsqueryEnrolled())

	// force its is_server mdm field to NULL, should be same as false
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_mdm SET is_server = NULL WHERE host_id = ?`, hFleet.ID)
		return err
	})
	loadFleet, err = ds.LoadHostByOrbitNodeKey(ctx, *hFleet.OrbitNodeKey)
	require.NoError(t, err)
	require.Equal(t, hFleet.ID, loadFleet.ID)

	// fill in disk encryption information
	require.NoError(t, ds.SetOrUpdateHostDisksEncryption(context.Background(), hFleet.ID, true))
	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hFleet.ID, "test-key", "", nil)
	require.NoError(t, err)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hFleet.ID}, true, time.Now())
	require.NoError(t, err)
	loadFleet, err = ds.LoadHostByOrbitNodeKey(ctx, *hFleet.OrbitNodeKey)
	require.NoError(t, err)
	require.NoError(t, err)
	require.True(t, loadFleet.MDM.EncryptionKeyAvailable)
	require.NotNil(t, loadFleet.DiskEncryptionEnabled)
	require.True(t, *loadFleet.DiskEncryptionEnabled)

	// simulate the device being assigned to Fleet in ABM
	err = ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*hFleet}, abmToken.ID)
	require.NoError(t, err)
	_, err = ds.LoadHostByOrbitNodeKey(ctx, *hFleet.OrbitNodeKey)
	require.NoError(t, err)

	// simulate a failed JSON profile assignment
	err = updateHostDEPAssignProfileResponses(
		ctx, ds.writer(ctx), ds.logger,
		"foo", []string{hFleet.HardwareSerial}, string(fleet.DEPAssignProfileResponseFailed), &abmToken.ID,
	)
	require.NoError(t, err)
	_, err = ds.LoadHostByOrbitNodeKey(ctx, *hFleet.OrbitNodeKey)
	require.NoError(t, err)
}

func checkEncryptionKeyStatus(t *testing.T, ds *Datastore, hostID uint, expectedKey string, expectedDecryptable *bool) {
	got, err := ds.GetHostDiskEncryptionKey(context.Background(), hostID)
	require.NoError(t, err)
	require.Equal(t, expectedKey, got.Base64Encrypted)
	require.Equal(t, expectedDecryptable, got.Decryptable)
}

func testLUKSDatastoreFunctions(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host1, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		OsqueryHostID:   ptr.String("1"),
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	host2, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		OsqueryHostID:   ptr.String("2"),
		Hostname:        "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
	})
	require.NoError(t, err)
	host3, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("3"),
		UUID:            "3",
		OsqueryHostID:   ptr.String("3"),
		Hostname:        "foo.local3",
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-60",
	})
	require.NoError(t, err)

	// queue shows as pending
	require.False(t, ds.IsHostPendingEscrow(ctx, host1.ID))
	err = ds.QueueEscrow(ctx, host1.ID)
	require.NoError(t, err)
	require.False(t, ds.IsHostPendingEscrow(ctx, host2.ID))
	require.True(t, ds.IsHostPendingEscrow(ctx, host1.ID))

	// clear removes pending
	err = ds.QueueEscrow(ctx, host2.ID)
	require.NoError(t, err)
	err = ds.ClearPendingEscrow(ctx, host1.ID)
	require.NoError(t, err)
	require.False(t, ds.IsHostPendingEscrow(ctx, host1.ID))
	require.True(t, ds.IsHostPendingEscrow(ctx, host2.ID))

	// report escrow error does not remove pending
	err = ds.ReportEscrowError(ctx, host2.ID, "this broke")
	require.NoError(t, err)
	require.True(t, ds.IsHostPendingEscrow(ctx, host2.ID))
	// TODO confirm error was persisted

	// assert no key stored on hosts with varying no-key-stored states
	require.NoError(t, ds.AssertHasNoEncryptionKeyStored(ctx, host1.ID))
	require.NoError(t, ds.AssertHasNoEncryptionKeyStored(ctx, host2.ID))
	require.NoError(t, ds.AssertHasNoEncryptionKeyStored(ctx, host3.ID))

	// no change when blank key or salt attempted to save
	err = ds.SaveLUKSData(ctx, host1.ID, "", "", 0)
	require.Error(t, err)
	require.NoError(t, ds.AssertHasNoEncryptionKeyStored(ctx, host1.ID))
	err = ds.SaveLUKSData(ctx, host1.ID, "foo", "", 0)
	require.Error(t, err)
	require.NoError(t, ds.AssertHasNoEncryptionKeyStored(ctx, host1.ID))

	// persists with passphrase and salt set
	err = ds.SaveLUKSData(ctx, host2.ID, "bazqux", "fuzzmuffin", 0)
	require.NoError(t, err)
	require.NoError(t, ds.AssertHasNoEncryptionKeyStored(ctx, host1.ID))
	require.Error(t, ds.AssertHasNoEncryptionKeyStored(ctx, host2.ID))
	key, err := ds.GetHostDiskEncryptionKey(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, "bazqux", key.Base64Encrypted)

	// persists when host hasn't had anything queued
	err = ds.SaveLUKSData(ctx, host3.ID, "newstuff", "fuzzball", 1)
	require.NoError(t, err)
	require.Error(t, ds.AssertHasNoEncryptionKeyStored(ctx, host3.ID))
	key, err = ds.GetHostDiskEncryptionKey(ctx, host3.ID)
	require.NoError(t, err)
	require.Equal(t, "newstuff", key.Base64Encrypted)
}

func testHostsSetOrUpdateHostDisksEncryptionKey(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		OsqueryHostID:   ptr.String("1"),
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		OsqueryHostID:   ptr.String("2"),
		Hostname:        "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
	})
	require.NoError(t, err)
	host3, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("3"),
		UUID:            "3",
		OsqueryHostID:   ptr.String("3"),
		Hostname:        "foo.local3",
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-60",
	})
	require.NoError(t, err)

	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host.ID, "AAA", "", nil)
	require.NoError(t, err)

	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host2.ID, "BBB", "", nil)
	require.NoError(t, err)

	h, err := ds.Host(context.Background(), host.ID)
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, h.ID, "AAA", nil)

	h, err = ds.Host(context.Background(), host2.ID)
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, h.ID, "BBB", nil)

	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host2.ID, "CCC", "", nil)
	require.NoError(t, err)

	h, err = ds.Host(context.Background(), host2.ID)
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, h.ID, "CCC", nil)

	// setting the encryption key to an existing value doesn't change its
	// encryption status
	err = ds.SetHostsDiskEncryptionKeyStatus(context.Background(), []uint{host.ID}, true, time.Now().Add(time.Hour))
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host.ID, "AAA", ptr.Bool(true))

	// same key doesn't change encryption status
	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host.ID, "AAA", "", nil)
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host.ID, "AAA", ptr.Bool(true))

	// different key resets encryption status
	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host.ID, "XZY", "", nil)
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host.ID, "XZY", nil)

	// set the key with an initial decrypted status of true
	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host3.ID, "abc", "", ptr.Bool(true))
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host3.ID, "abc", ptr.Bool(true))

	// same key, provided decrypted status is ignored (stored one is kept)
	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host3.ID, "abc", "", ptr.Bool(false))
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host3.ID, "abc", ptr.Bool(true))

	// client error, key is removed and decrypted status is nulled
	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host3.ID, "", "fail", nil)
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host3.ID, "", nil)

	// new key, provided decrypted status is applied
	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host3.ID, "def", "", ptr.Bool(true))
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host3.ID, "def", ptr.Bool(true))

	// different key, provided decrypted status is applied
	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host3.ID, "ghi", "", ptr.Bool(false))
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host3.ID, "ghi", ptr.Bool(false))

	// set an empty key (backfill for issue #15068)
	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host3.ID, "", "", nil)
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host3.ID, "", nil)

	// setting the decryptable value works even if the key is still empty
	err = ds.SetOrUpdateHostDiskEncryptionKey(context.Background(), host3.ID, "", "", ptr.Bool(false))
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host3.ID, "", ptr.Bool(false))
}

func testHostsSetDiskEncryptionKeyStatus(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		OsqueryHostID:   ptr.String("1"),
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, "TESTKEY", "", nil)
	require.NoError(t, err)

	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		OsqueryHostID:   ptr.String("2"),
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, host2.ID, "TESTKEY", "", nil)
	require.NoError(t, err)

	threshold := time.Now().Add(time.Hour)

	// empty set
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{}, false, threshold)
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host.ID, "TESTKEY", nil)
	checkEncryptionKeyStatus(t, ds, host2.ID, "TESTKEY", nil)

	// keys that changed after the provided threshold are not updated
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID, host2.ID}, true, threshold.Add(-24*time.Hour))
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host.ID, "TESTKEY", nil)
	checkEncryptionKeyStatus(t, ds, host2.ID, "TESTKEY", nil)

	// single host
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, true, threshold)
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host.ID, "TESTKEY", ptr.Bool(true))
	checkEncryptionKeyStatus(t, ds, host2.ID, "TESTKEY", nil)

	// multiple hosts
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID, host2.ID}, true, threshold)
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host.ID, "TESTKEY", ptr.Bool(true))
	checkEncryptionKeyStatus(t, ds, host2.ID, "TESTKEY", ptr.Bool(true))

	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID, host2.ID}, false, threshold)
	require.NoError(t, err)
	checkEncryptionKeyStatus(t, ds, host.ID, "TESTKEY", ptr.Bool(false))
	checkEncryptionKeyStatus(t, ds, host2.ID, "TESTKEY", ptr.Bool(false))
}

func testHostsGetUnverifiedDiskEncryptionKeys(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		OsqueryHostID:   ptr.String("1"),
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		OsqueryHostID:   ptr.String("2"),
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, "TESTKEY", "", nil)
	require.NoError(t, err)
	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, host2.ID, "TESTKEY", "", nil)
	require.NoError(t, err)

	keys, err := ds.GetUnverifiedDiskEncryptionKeys(ctx)
	require.NoError(t, err)
	require.Len(t, keys, 2)
	// ensure the updated_at value is grabbed from the database
	for _, k := range keys {
		require.NotZero(t, k.UpdatedAt)
	}

	threshold := time.Now().Add(time.Hour)

	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, false, threshold)
	require.NoError(t, err)

	keys, err = ds.GetUnverifiedDiskEncryptionKeys(ctx)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, host2.ID, keys[0].HostID)

	// update key of host 1 to empty with a client error, should not be reported
	// by GetUnverifiedDiskEncryptionKeys
	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, "", "failed", nil)
	require.NoError(t, err)

	keys, err = ds.GetUnverifiedDiskEncryptionKeys(ctx)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, host2.ID, keys[0].HostID)

	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID, host2.ID}, false, threshold)
	require.NoError(t, err)

	keys, err = ds.GetUnverifiedDiskEncryptionKeys(ctx)
	require.NoError(t, err)
	require.Empty(t, keys)
}

func testHostsEnrollOrbit(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	const (
		computerName  = "My computer"
		hardwareModel = "CMP-1000"
	)

	createHost := func(osqueryID, serial string) *fleet.Host {
		dbZeroTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		var osqueryIDPtr *string
		if osqueryID != "" {
			osqueryIDPtr = &osqueryID
		}
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:         "foo",
			HardwareSerial:   serial,
			Platform:         "darwin",
			LastEnrolledAt:   dbZeroTime,
			DetailUpdatedAt:  dbZeroTime,
			OsqueryHostID:    osqueryIDPtr,
			RefetchRequested: true,
			ComputerName:     computerName,
			HardwareModel:    hardwareModel,
		})
		require.NoError(t, err)
		return h
	}

	// create and enroll a host with just an osquery ID, no serial
	hOsqueryNoSerial := createHost(uuid.New().String(), "")
	h, err := ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{
		HardwareUUID:   *hOsqueryNoSerial.OsqueryHostID,
		HardwareSerial: hOsqueryNoSerial.HardwareSerial,
	}, uuid.New().String(), nil)
	require.NoError(t, err)
	require.Equal(t, hOsqueryNoSerial.ID, h.ID)
	require.Empty(t, h.HardwareSerial)
	// Hostname and platform values should not be overriden by the orbit enroll.
	h, err = ds.Host(ctx, h.ID)
	require.NoError(t, err)
	require.Equal(t, "foo", h.Hostname)
	require.Equal(t, "darwin", h.Platform)

	// create and enroll a host with just a serial, no osquery ID (that is, it
	// got created this way, but when enrolling in orbit it does have an osquery
	// ID)
	hSerialNoOsquery := createHost("", uuid.New().String())
	h, err = ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{
		HardwareUUID:   uuid.New().String(),
		HardwareSerial: hSerialNoOsquery.HardwareSerial,
	}, uuid.New().String(), nil)
	require.NoError(t, err)
	require.Equal(t, hSerialNoOsquery.ID, h.ID)
	require.Empty(t, h.OsqueryHostID)

	// create and enroll a host with both
	hBoth := createHost(uuid.New().String(), uuid.New().String())
	h, err = ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{
		HardwareUUID:   *hBoth.OsqueryHostID,
		HardwareSerial: hBoth.HardwareSerial,
		ComputerName:   hBoth.ComputerName,
		HardwareModel:  hBoth.HardwareModel,
	}, uuid.New().String(), nil)
	require.NoError(t, err)
	require.Equal(t, hBoth.ID, h.ID)
	assert.Equal(t, hBoth.HardwareSerial, h.HardwareSerial)
	assert.Equal(t, hBoth.ComputerName, h.ComputerName)
	assert.Equal(t, hBoth.HardwareModel, h.HardwareModel)
	h, err = ds.Host(ctx, h.ID)
	require.NoError(t, err)
	assert.Equal(t, hBoth.HardwareSerial, h.HardwareSerial)
	assert.Equal(t, hBoth.ComputerName, h.ComputerName)
	assert.Equal(t, hBoth.HardwareModel, h.HardwareModel)

	// enroll with osquery id from hBoth and serial from hSerialNoOsquery (should
	// use the osquery match)
	h, err = ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{
		HardwareUUID:   *hBoth.OsqueryHostID,
		HardwareSerial: hSerialNoOsquery.HardwareSerial,
	}, uuid.New().String(), nil)
	require.NoError(t, err)
	require.Equal(t, hBoth.ID, h.ID)
	assert.Equal(t, hSerialNoOsquery.HardwareSerial, h.HardwareSerial)

	// enroll with no match, will create a new one
	newSerial := uuid.NewString()
	h, err = ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{
		HardwareUUID:   uuid.New().String(),
		HardwareSerial: newSerial,
		Hostname:       "foo2",
		Platform:       "darwin",
		ComputerName:   "New computer",
		HardwareModel:  "ABC-3000",
	}, uuid.New().String(), nil)
	require.NoError(t, err)
	require.Greater(t, h.ID, hBoth.ID)
	// Hostname and platform values should be set by the Orbit enroll.
	h, err = ds.Host(ctx, h.ID)
	require.NoError(t, err)
	require.Equal(t, "foo2", h.Hostname)
	require.Equal(t, "darwin", h.Platform)
	assert.Equal(t, "New computer", h.ComputerName)
	assert.Equal(t, "ABC-3000", h.HardwareModel)
	assert.Equal(t, newSerial, h.HardwareSerial)

	// simulate a "corrupt database" where two hosts have the same serial and
	// enroll by serial should always use the same (the smaller ID)
	hDupSerial1 := createHost("", uuid.New().String())
	hDupSerial2 := createHost("", hDupSerial1.HardwareSerial)
	require.Greater(t, hDupSerial2.ID, hDupSerial1.ID)
	h, err = ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{
		HardwareUUID:   uuid.New().String(),
		HardwareSerial: hDupSerial1.HardwareSerial,
	}, uuid.New().String(), nil)
	require.NoError(t, err)
	require.Equal(t, hDupSerial1.ID, h.ID)

	// enroll with osquery ID from hOsqueryNoSerial and the duplicate serial,
	// will always match osquery ID
	h, err = ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{
		HardwareUUID:   *hOsqueryNoSerial.OsqueryHostID,
		HardwareSerial: hDupSerial1.HardwareSerial,
	}, uuid.New().String(), nil)
	require.NoError(t, err)
	require.Equal(t, hOsqueryNoSerial.ID, h.ID)

	// Scenario A:
	//	- Fleet with MDM disabled.
	// 	- two linux|darwin|windows hosts with the same hardware identifiers (e.g. two cloned VMs).
	//	- fleetd running with host identifier set to instance.
	//	- orbit enrolls first, then osquery
	// Expected output: The two fleetd instances should be enrolled as two hosts.
	scenarioA := func(platform string) {
		dupUUID := uuid.New().String()
		dupHWSerial := uuid.New().String()
		randomIdentifierH1 := uuid.New().String()

		h1Orbit, err := ds.EnrollOrbit(ctx, false, fleet.OrbitHostInfo{
			HardwareUUID:      dupUUID,
			HardwareSerial:    dupHWSerial,
			OsqueryIdentifier: randomIdentifierH1,
			Platform:          platform,
		}, uuid.New().String(), nil)
		require.NoError(t, err)
		h1Osquery, err := ds.EnrollHost(ctx, false, randomIdentifierH1, dupUUID, dupHWSerial, uuid.New().String(), nil, 0)
		require.NoError(t, err)
		require.Equal(t, h1Orbit.ID, h1Osquery.ID)
		randomIdentifierH2 := uuid.New().String()
		h2Orbit, err := ds.EnrollOrbit(ctx, false, fleet.OrbitHostInfo{
			HardwareUUID:      dupUUID,
			HardwareSerial:    dupHWSerial,
			OsqueryIdentifier: randomIdentifierH2,
			Platform:          platform,
		}, uuid.New().String(), nil)
		require.NoError(t, err)
		h2Osquery, err := ds.EnrollHost(ctx, false, randomIdentifierH2, dupUUID, dupHWSerial, uuid.New().String(), nil, 0)
		require.NoError(t, err)
		require.Equal(t, h2Orbit.ID, h2Osquery.ID)

		require.NotEqual(t, h1Orbit.ID, h2Orbit.ID) // the hosts are enrolled as two separate hosts
	}
	for _, platform := range []string{"ubuntu", "windows", "darwin"} {
		platform := platform
		t.Run("scenarioA_"+platform, func(t *testing.T) {
			scenarioA(platform)
		})
	}

	// Scenario B:
	//	- Fleet with MDM disabled.
	// 	- Two linux|darwin|windows hosts with the same hardware identifiers (e.g. two cloned VMs).
	//	- fleetd running with host identifier set to instance.
	//	- orbit and osquery of the two hosts enroll in mixed order.
	// Expected output: The two fleetd instances should be each its own host.
	scenarioB := func(platform string) {
		dupUUID := uuid.New().String()
		dupHWSerial := uuid.New().String()
		randomIdentifierH1 := uuid.New().String()

		// First osquery of the first host enrolls.
		h1Osquery, err := ds.EnrollHost(ctx, false, randomIdentifierH1, dupUUID, dupHWSerial, uuid.New().String(), nil, 0)
		require.NoError(t, err)
		randomIdentifierH2 := uuid.New().String()
		// Then orbit of the second host enrolls.
		h2Orbit, err := ds.EnrollOrbit(ctx, false, fleet.OrbitHostInfo{
			HardwareUUID:      dupUUID,
			HardwareSerial:    dupHWSerial,
			OsqueryIdentifier: randomIdentifierH2,
			Platform:          platform,
		}, uuid.New().String(), nil)
		require.NoError(t, err)
		// Then orbit of the first host enrolls.
		h1Orbit, err := ds.EnrollOrbit(ctx, false, fleet.OrbitHostInfo{
			HardwareUUID:      dupUUID,
			HardwareSerial:    dupHWSerial,
			OsqueryIdentifier: randomIdentifierH1,
			Platform:          platform,
		}, uuid.New().String(), nil)
		require.NoError(t, err)
		require.Equal(t, h1Orbit.ID, h1Osquery.ID)
		// Lastly osquery of the second host enrolls.
		h2Osquery, err := ds.EnrollHost(ctx, false, randomIdentifierH2, dupUUID, dupHWSerial, uuid.New().String(), nil, 0)
		require.NoError(t, err)
		require.Equal(t, h2Orbit.ID, h2Osquery.ID)

		require.NotEqual(t, h1Orbit.ID, h2Orbit.ID) // the hosts are enrolled as two separate hosts
	}
	for _, platform := range []string{"ubuntu", "windows", "darwin"} {
		platform := platform
		t.Run("scenarioB_"+platform, func(t *testing.T) {
			scenarioB(platform)
		})
	}

	// Scenario C:
	//	- Fleet with MDM enabled.
	// 	- Two linux|darwin|windows hosts with the same hardware identifiers (e.g. two cloned VMs).
	//	- fleetd running with host identifier set to instance.
	//	- orbit and osquery of the two hosts enroll in mixed order.
	//
	// For Linux and Windows this scenario behaves as expected. The two hosts are enrolled separately.
	//
	// For macOS:
	// Somewhat unexpected output of this scenario is that two hosts are enrolled as one
	// because MDM makes the effort to match by hardware serial.
	// Using fleetd's `--host-identifier=instance` with Fleet's MDM enabled is not compatible on macOS.
	scenarioC := func(platform string) {
		dupUUID := uuid.New().String()
		dupHWSerial := uuid.New().String()
		randomIdentifierH1 := uuid.New().String()
		randomIdentifierH2 := uuid.New().String()

		h1Orbit, err := ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{
			HardwareUUID:      dupUUID,
			HardwareSerial:    dupHWSerial,
			OsqueryIdentifier: randomIdentifierH1,
			Platform:          platform,
		}, uuid.New().String(), nil)
		require.NoError(t, err)
		h1Osquery, err := ds.EnrollHost(ctx, true, randomIdentifierH1, dupUUID, dupHWSerial, uuid.New().String(), nil, 0)
		require.NoError(t, err)
		require.Equal(t, h1Orbit.ID, h1Osquery.ID)

		// Second host enrolls osquery first, then orbit.
		h2Osquery, err := ds.EnrollHost(ctx, true, randomIdentifierH2, dupUUID, dupHWSerial, uuid.New().String(), nil, 0)
		require.NoError(t, err)
		h2Orbit, err := ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{
			HardwareUUID:      dupUUID,
			HardwareSerial:    dupHWSerial,
			OsqueryIdentifier: randomIdentifierH2,
			Platform:          platform,
		}, uuid.New().String(), nil)
		require.NoError(t, err)
		require.Equal(t, h2Orbit.ID, h2Osquery.ID)

		if platform == "darwin" {
			// This is a expected output of this scenario because MDM makes
			// the effort to match by hardware serial.
			require.Equal(t, h1Orbit.ID, h2Orbit.ID)
		} else {
			require.NotEqual(t, h1Orbit.ID, h2Orbit.ID)
		}
	}
	for _, platform := range []string{"ubuntu", "windows", "darwin"} {
		platform := platform
		t.Run("scenarioC_"+platform, func(t *testing.T) {
			scenarioC(platform)
		})
	}

	// Scenario D:
	//	- Fleet with MDM disabled.
	// 	- two linux|darwin|windows hosts with the same hardware identifiers (e.g. two cloned VMs).
	//	- fleetd running with host identifier set to uuid (default).
	//	- orbit enrolls first, then osquery
	// Expected output: The two fleetd instances should be enrolled as one host.
	scenarioD := func(platform string) {
		dupUUID := uuid.New().String()
		dupHWSerial := uuid.New().String()

		h1Orbit, err := ds.EnrollOrbit(ctx, false, fleet.OrbitHostInfo{
			HardwareUUID:   dupUUID,
			HardwareSerial: dupHWSerial,
			Platform:       platform,
		}, uuid.New().String(), nil)
		require.NoError(t, err)
		h1OrbitFetched, err := ds.Host(ctx, h1Orbit.ID)
		require.NoError(t, err)
		time.Sleep(1 * time.Second) // to test the update of last_enrolled_at
		h1Osquery, err := ds.EnrollHost(ctx, false, dupUUID, dupUUID, dupHWSerial, uuid.New().String(), nil, 0)
		require.NoError(t, err)
		h1OsqueryFetched, err := ds.Host(ctx, h1Osquery.ID)
		require.NoError(t, err)
		require.NotEqual(t, h1OrbitFetched.LastEnrolledAt, h1OsqueryFetched.LastEnrolledAt)
		require.Equal(t, h1Orbit.ID, h1Osquery.ID)
		time.Sleep(1 * time.Second) // to test the update of last_enrolled_at
		h2Orbit, err := ds.EnrollOrbit(ctx, false, fleet.OrbitHostInfo{
			HardwareUUID:   dupUUID,
			HardwareSerial: dupHWSerial,
			Platform:       platform,
		}, uuid.New().String(), nil)
		require.NoError(t, err)
		h2OrbitFetched, err := ds.Host(ctx, h2Orbit.ID)
		require.NoError(t, err)
		// orbit should not update last_enrolled_at if re-enrolling (because last_enrolled_at
		// is to be set by osquery only).
		require.Equal(t, h1OsqueryFetched.LastEnrolledAt, h2OrbitFetched.LastEnrolledAt)
		time.Sleep(1 * time.Second) // to test the update of last_enrolled_at
		h2Osquery, err := ds.EnrollHost(ctx, false, dupUUID, dupUUID, dupHWSerial, uuid.New().String(), nil, 0)
		require.NoError(t, err)
		require.Equal(t, h2Orbit.ID, h2Osquery.ID)
		h2OsqueryFetched, err := ds.Host(ctx, h2Osquery.ID)
		require.NoError(t, err)
		require.NotEqual(t, h2OrbitFetched.LastEnrolledAt, h2OsqueryFetched.LastEnrolledAt)

		// the hosts compete for the host entry (all have same row id)
		require.Equal(t, h1Orbit.ID, h2Orbit.ID)
		require.Equal(t, h1Orbit.ID, h1Osquery.ID)
		require.Equal(t, h2Orbit.ID, h2Osquery.ID)
	}
	for _, platform := range []string{"ubuntu", "windows", "darwin"} {
		platform := platform
		t.Run("scenarioD_"+platform, func(t *testing.T) {
			t.Parallel()
			scenarioD(platform)
		})
	}
}

func testHostsEnrollUpdatesMissingInfo(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a bare minimal host (as if created via DEP enrollment)
	// no team, osquery id, uuid.
	dbZeroTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	h, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:         "foobar",
		HardwareSerial:   "serial",
		Platform:         "darwin",
		LastEnrolledAt:   dbZeroTime,
		DetailUpdatedAt:  dbZeroTime,
		RefetchRequested: true,
	})
	require.NoError(t, err)

	tm, err := ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	// enroll with orbit and a uuid (will match on serial)
	_, err = ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{
		HardwareUUID:   "uuid",
		HardwareSerial: "serial",
	}, "orbit", nil)
	require.NoError(t, err)
	got, err := ds.LoadHostByOrbitNodeKey(ctx, "orbit")
	require.NoError(t, err)
	require.Equal(t, h.ID, got.ID)
	require.Equal(t, "serial", got.HardwareSerial)
	require.Equal(t, "uuid", got.UUID)
	require.NotNil(t, got.OsqueryHostID)
	require.Equal(t, "uuid", *got.OsqueryHostID)
	require.Nil(t, got.TeamID)
	require.Nil(t, got.NodeKey)
	// Verify that the orbit enroll didn't override these values set by a previous osquery enroll.
	require.Equal(t, "foobar", got.Hostname)
	require.Equal(t, "darwin", got.Platform)

	// enroll with osquery using uuid identifier, team
	_, err = ds.EnrollHost(ctx, true, "uuid", "uuid", "different-serial", "osquery", &tm.ID, 0)
	require.NoError(t, err)
	got, err = ds.LoadHostByOrbitNodeKey(ctx, "orbit")
	require.NoError(t, err)
	require.Equal(t, h.ID, got.ID)
	require.Equal(t, "serial", got.HardwareSerial) // unchanged as it was already filled
	require.Equal(t, "uuid", got.UUID)
	require.NotNil(t, got.OsqueryHostID)
	require.Equal(t, "uuid", *got.OsqueryHostID)
	require.NotNil(t, got.NodeKey)
	require.Equal(t, "osquery", *got.NodeKey)
	require.NotNil(t, got.TeamID)
	require.Equal(t, tm.ID, *got.TeamID)
	// Verify that the orbit enroll didn't override these values set by a previous osquery enroll.
	require.Equal(t, "foobar", got.Hostname)
	require.Equal(t, "darwin", got.Platform)
}

func testHostsEncryptionKeyRawDecryption(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		OsqueryHostID:   ptr.String("1"),
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	// no disk encryption key information
	got, err := ds.Host(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, got.MDM.TestGetRawDecryptable())
	require.False(t, got.MDM.EncryptionKeyAvailable)
	require.Equal(t, -1, *got.MDM.TestGetRawDecryptable())

	// create the encryption key row, but unknown decryptable
	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, "abc", "", nil)
	require.NoError(t, err)

	got, err = ds.Host(ctx, host.ID)
	require.NoError(t, err)
	require.False(t, got.MDM.EncryptionKeyAvailable)
	require.Nil(t, got.MDM.TestGetRawDecryptable())

	// mark the key as non-decryptable
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, false, time.Now())
	require.NoError(t, err)

	got, err = ds.Host(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, got.MDM.TestGetRawDecryptable())
	require.False(t, got.MDM.EncryptionKeyAvailable)
	require.Equal(t, 0, *got.MDM.TestGetRawDecryptable())

	// mark the key as decryptable
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, true, time.Now())
	require.NoError(t, err)

	got, err = ds.Host(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, got.MDM.TestGetRawDecryptable())
	require.True(t, got.MDM.EncryptionKeyAvailable)
	require.Equal(t, 1, *got.MDM.TestGetRawDecryptable())
}

func testHostsListHostsLiteByUUIDs(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create hosts, UUID is the `i` index
	hosts := make([]*fleet.Host, 10)
	for i := range hosts {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String(fmt.Sprintf("host%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.%d.local", i),
		})
		require.NoError(t, err)
		hosts[i] = h
	}

	// move hosts 0, 1, 2 to team 1
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(ctx, &team1.ID, []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID}))

	// move hosts 3, 4, 5 to team 2
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(ctx, &team2.ID, []uint{hosts[3].ID, hosts[4].ID, hosts[5].ID}))

	// create a team 3 without any host
	team3, err := ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	tm1Admin := &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleAdmin}}}
	tm1Maintainer := &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleMaintainer}}}
	tm1Observer := &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleObserver}}}
	tm2Admin := &fleet.User{Teams: []fleet.UserTeam{{Team: *team2, Role: fleet.RoleAdmin}}}
	tm2Maintainer := &fleet.User{Teams: []fleet.UserTeam{{Team: *team2, Role: fleet.RoleMaintainer}}}
	tm2Observer := &fleet.User{Teams: []fleet.UserTeam{{Team: *team2, Role: fleet.RoleObserver}}}
	tm3Admin := &fleet.User{Teams: []fleet.UserTeam{{Team: *team3, Role: fleet.RoleAdmin}}}
	tm1MaintainerTm2Observer := &fleet.User{Teams: []fleet.UserTeam{
		{Team: *team1, Role: fleet.RoleMaintainer},
		{Team: *team2, Role: fleet.RoleObserver},
	}}

	cases := []struct {
		desc    string
		filter  fleet.TeamFilter
		uuids   []string
		wantIDs []uint
	}{
		{
			"no user sees nothing",
			fleet.TeamFilter{},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			nil,
		},
		{
			"global admin no uuid provided",
			fleet.TeamFilter{User: test.UserAdmin},
			[]string{},
			nil,
		},
		{
			"global admin sees everything",
			fleet.TeamFilter{User: test.UserAdmin},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			[]uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID, hosts[5].ID, hosts[6].ID, hosts[7].ID, hosts[8].ID, hosts[9].ID},
		},
		{
			"global maintainer sees everything",
			fleet.TeamFilter{User: test.UserMaintainer},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			[]uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID, hosts[5].ID, hosts[6].ID, hosts[7].ID, hosts[8].ID, hosts[9].ID},
		},
		{
			"global observer sees nothing",
			fleet.TeamFilter{User: test.UserObserver},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			nil,
		},
		{
			"global observer sees everything with observer allowed",
			fleet.TeamFilter{User: test.UserObserver, IncludeObserver: true},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			[]uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID, hosts[5].ID, hosts[6].ID, hosts[7].ID, hosts[8].ID, hosts[9].ID},
		},
		{
			"team 1 admin sees team 1 hosts",
			fleet.TeamFilter{User: tm1Admin},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			[]uint{hosts[0].ID, hosts[1].ID, hosts[2].ID},
		},
		{
			"team 1 maintainer sees team 1 hosts",
			fleet.TeamFilter{User: tm1Maintainer},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			[]uint{hosts[0].ID, hosts[1].ID, hosts[2].ID},
		},
		{
			"team 1 observer sees nothing",
			fleet.TeamFilter{User: tm1Observer},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			nil,
		},
		{
			"team 1 observer sees team 1 hosts with observer allowed",
			fleet.TeamFilter{User: tm1Observer, IncludeObserver: true},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			[]uint{hosts[0].ID, hosts[1].ID, hosts[2].ID},
		},
		{
			"team 2 admin sees team 2 hosts",
			fleet.TeamFilter{User: tm2Admin},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			[]uint{hosts[3].ID, hosts[4].ID, hosts[5].ID},
		},
		{
			"team 2 maintainer sees team 2 hosts",
			fleet.TeamFilter{User: tm2Maintainer},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			[]uint{hosts[3].ID, hosts[4].ID, hosts[5].ID},
		},
		{
			"team 2 observer sees nothing",
			fleet.TeamFilter{User: tm2Observer},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			nil,
		},
		{
			"team 2 observer sees team 2 hosts with observer allowed",
			fleet.TeamFilter{User: tm2Observer, IncludeObserver: true},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			[]uint{hosts[3].ID, hosts[4].ID, hosts[5].ID},
		},
		{
			"team 3 admin sees nothing even with observer",
			fleet.TeamFilter{User: tm3Admin, IncludeObserver: true},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			nil,
		},
		{
			"filtering on a specific team ID returns only those hosts",
			fleet.TeamFilter{User: test.UserAdmin, TeamID: &team1.ID},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			[]uint{hosts[0].ID, hosts[1].ID, hosts[2].ID},
		},
		{
			"team 1 maintainer team 2 observer sees team 1",
			fleet.TeamFilter{User: tm1MaintainerTm2Observer},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			[]uint{hosts[0].ID, hosts[1].ID, hosts[2].ID},
		},
		{
			"team 1 maintainer team 2 observer sees team 1 and 2 with observer",
			fleet.TeamFilter{User: tm1MaintainerTm2Observer, IncludeObserver: true},
			[]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			[]uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID, hosts[5].ID},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			hosts, err := ds.ListHostsLiteByUUIDs(ctx, c.filter, c.uuids)
			require.NoError(t, err)

			gotIDs := make([]uint, len(hosts))
			for i, h := range hosts {
				gotIDs[i] = h.ID
			}
			require.ElementsMatch(t, c.wantIDs, gotIDs)
		})
	}
}

func testGetMatchingHostSerials(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	serials := []string{"foo", "bar", "baz"}
	team, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)
	for i, serial := range serials {
		var tmID *uint
		if serial == "bar" {
			tmID = &team.ID
		}
		_, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String(fmt.Sprint(i)),
			UUID:            fmt.Sprint(i),
			OsqueryHostID:   ptr.String(fmt.Sprint(i)),
			Hostname:        "foo.local",
			PrimaryIP:       "192.168.1.1",
			PrimaryMac:      "30-65-EC-6F-C4-58",
			HardwareSerial:  serial,
			TeamID:          tmID,
			ID:              uint(i),
		})
		require.NoError(t, err)
	}

	cases := []struct {
		name string
		in   []string
		want map[string]*fleet.Host
		err  string
	}{
		{"no serials provided", []string{}, map[string]*fleet.Host{}, ""},
		{"no matching serials", []string{"oof", "rab"}, map[string]*fleet.Host{}, ""},
		{
			"partial matches",
			[]string{"foo", "rab"},
			map[string]*fleet.Host{
				"foo": {HardwareSerial: "foo", TeamID: nil, ID: 1},
			},
			"",
		},
		{
			"all matching",
			[]string{"foo", "bar", "baz"},
			map[string]*fleet.Host{
				"foo": {HardwareSerial: "foo", TeamID: nil, ID: 1},
				"bar": {HardwareSerial: "bar", TeamID: &team.ID, ID: 2},
				"baz": {HardwareSerial: "baz", TeamID: nil, ID: 3},
			},
			"",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ds.GetMatchingHostSerials(ctx, tt.in)
			if tt.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.err)
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func testHostsListHostsLiteByIDs(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	hosts := make([]*fleet.Host, 3)
	for i := range hosts {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String(fmt.Sprintf("host%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.%d.local", i),
		})
		require.NoError(t, err)
		hosts[i] = h
	}

	cases := []struct {
		desc    string
		ids     []uint
		wantIDs []uint
	}{
		{
			"empty list",
			nil,
			nil,
		},
		{
			"invalid ids",
			[]uint{hosts[2].ID + 1000, hosts[2].ID + 1001},
			nil,
		},
		{
			"single valid id",
			[]uint{hosts[0].ID},
			[]uint{hosts[0].ID},
		},
		{
			"multiple valid ids",
			[]uint{hosts[0].ID, hosts[1].ID, hosts[2].ID},
			[]uint{hosts[0].ID, hosts[1].ID, hosts[2].ID},
		},
		{
			"valid and invalid ids",
			[]uint{hosts[0].ID, hosts[1].ID, hosts[2].ID + 1000},
			[]uint{hosts[0].ID, hosts[1].ID},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			hosts, err := ds.ListHostsLiteByIDs(ctx, c.ids)
			require.NoError(t, err)

			gotIDs := make([]uint, len(hosts))
			for i, h := range hosts {
				gotIDs[i] = h.ID
			}
			require.ElementsMatch(t, c.wantIDs, gotIDs)
		})
	}
}

func testListHostsWithPagination(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	newHostFunc := func(name string) *fleet.Host {
		host, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String(name),
			UUID:            name,
			Hostname:        "foo.local." + name,
		})
		require.NoError(t, err)
		require.NotNil(t, host)
		return host
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	hostCount := 150
	hosts := make([]*fleet.Host, 0, hostCount)
	for i := 0; i < hostCount; i++ {
		hosts = append(hosts, newHostFunc(fmt.Sprintf("h%d", i)))
	}

	// List all hosts with PerPage=0
	perPage0 := 0
	hosts0, err := ds.ListHosts(ctx, filter, fleet.HostListOptions{
		ListOptions: fleet.ListOptions{
			PerPage: uint(perPage0),
		},
	})
	require.NoError(t, err)
	require.Len(t, hosts0, hostCount)
	for i, host := range hosts0 {
		require.Equal(t, host.ID, hosts[i].ID)
	}

	// List hosts with PerPage=100
	perPage1 := 100
	hosts1, err := ds.ListHosts(ctx, filter, fleet.HostListOptions{
		ListOptions: fleet.ListOptions{
			PerPage: uint(perPage1),
		},
	})
	require.NoError(t, err)
	require.Len(t, hosts1, perPage1)
	for i, host := range hosts1 {
		require.Equal(t, host.ID, hosts[i].ID)
	}

	// List hosts with PerPage=120
	perPage2 := 120
	hosts2, err := ds.ListHosts(ctx, filter, fleet.HostListOptions{
		ListOptions: fleet.ListOptions{
			PerPage: uint(perPage2),
		},
	})
	require.NoError(t, err)
	require.Len(t, hosts2, perPage2)
	for i, host := range hosts2 {
		require.Equal(t, host.ID, hosts[i].ID)
	}

	// Count hosts.
	count, err := ds.CountHosts(ctx, filter, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Equal(t, hostCount, count)
}

func testLastRestarted(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Arbitrary value
	const uptimeVal = 16691000000000
	now := time.Now()
	newHostFunc := func(name string, uptimeZero bool) (*fleet.Host, time.Time) {
		newHost := &fleet.Host{
			DetailUpdatedAt: now,
			LabelUpdatedAt:  now,
			PolicyUpdatedAt: now,
			SeenTime:        now,
			NodeKey:         ptr.String(name),
			UUID:            name,
			Hostname:        "foo.local." + name,
		}

		var expectedLastRestartedAt time.Time

		if uptimeZero {
			newHost.Uptime = 0
		} else {
			newHost.Uptime = uptimeVal
			// Rounding to nearest second because the SQL query does integer division.
			expectedLastRestartedAt = newHost.DetailUpdatedAt.Add(-newHost.Uptime).Round(time.Second).UTC()
		}

		host, err := ds.NewHost(ctx, newHost)
		require.NoError(t, err)
		require.NotNil(t, host)
		return host, expectedLastRestartedAt
	}

	hostCount := 10
	hosts := make([]*fleet.Host, 0, hostCount)
	hostsToVals := make(map[uint]time.Time, 0)
	for i := 0; i < hostCount; i++ {
		nh, expectedVal := newHostFunc(fmt.Sprintf("h%d", i), i%2 == 0)
		hosts = append(hosts, nh)
		hostsToVals[nh.ID] = expectedVal
	}

	opts := fleet.HostListOptions{}

	userFilter := fleet.TeamFilter{User: test.UserAdmin}

	returnedHosts := listHostsCheckCount(t, ds, userFilter, opts, len(hosts))

	for i, h := range returnedHosts {
		require.Equal(t, hosts[i].Uptime, h.Uptime)
		require.Equal(t, hostsToVals[h.ID], h.LastRestartedAt)
	}

	h1 := hosts[0] // has Uptime == 0
	h2 := hosts[1] // has Uptime == uptimeVal

	host, err := ds.Host(ctx, h1.ID)
	require.NoError(t, err)
	require.Equal(t, h1.ID, host.ID)
	require.Equal(t, time.Duration(0), host.Uptime)
	require.Equal(t, hostsToVals[host.ID], host.LastRestartedAt)

	host, err = ds.Host(ctx, h2.ID)
	require.NoError(t, err)
	require.Equal(t, h2.ID, host.ID)
	require.Equal(t, time.Duration(uptimeVal), host.Uptime)
	require.Equal(t, hostsToVals[host.ID], host.LastRestartedAt)
}

func testHostHealth(t *testing.T, ds *Datastore) {
	_, err := ds.GetHostHealth(context.Background(), 1)
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.True(t, errors.As(err, &nfe))

	// We'll check TeamIDs because at this level they should still be populated
	team, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	now := time.Now()
	_, err = ds.NewHost(context.Background(), &fleet.Host{
		ID:                  1,
		OsqueryHostID:       ptr.String("foobar"),
		NodeKey:             ptr.String("nodekey"),
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
		TeamID:              ptr.Uint(team.ID),

		SeenTime: now,

		CPUType: "cpuType",
	})
	require.NoError(t, err)
	h, err := ds.Host(context.Background(), 1)
	require.NoError(t, err)

	// set up policies
	u := test.NewUser(t, ds, "Jack", "jack@example.com", true)

	q := test.NewQuery(t, ds, nil, "passing_query", "select 1", 0, true)
	passingPolicy, err := ds.NewGlobalPolicy(context.Background(), &u.ID, fleet.PolicyPayload{QueryID: &q.ID})
	require.NoError(t, err)

	q = test.NewQuery(t, ds, nil, "failing_query", "select 1", 0, true)
	failingPolicy, err := ds.NewGlobalPolicy(context.Background(), &u.ID, fleet.PolicyPayload{QueryID: &q.ID})
	require.NoError(t, err)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h, map[uint]*bool{passingPolicy.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h, map[uint]*bool{failingPolicy.ID: ptr.Bool(false)}, time.Now(), false))

	// set up vulnerable software
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "baz", Version: "0.0.4", Source: "apps"},
	}
	_, err = ds.UpdateHostSoftware(context.Background(), h.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(context.Background(), h, false))

	soft1 := h.Software[0]
	for _, item := range h.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	cpes := []fleet.SoftwareCPE{{SoftwareID: soft1.ID, CPE: "somecpe"}}
	_, err = ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	// Reload software so that 'GeneratedCPEID is set.
	require.NoError(t, ds.LoadHostSoftware(context.Background(), h, false))
	soft1 = h.Software[0]
	for _, item := range h.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	inserted, err := ds.InsertSoftwareVulnerability(
		context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: soft1.ID,
			CVE:        "cve-123-123-132",
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.True(t, inserted)

	hh, err := ds.GetHostHealth(context.Background(), h.ID)
	require.NoError(t, err)
	require.Equal(t, h.Platform, hh.Platform)
	require.Equal(t, h.DiskEncryptionEnabled, hh.DiskEncryptionEnabled)
	require.Equal(t, h.OSVersion, hh.OsVersion)
	require.Equal(t, ptr.Uint(team.ID), hh.TeamID)
	require.Equal(t, h.UpdatedAt, hh.UpdatedAt)
	require.Len(t, hh.FailingPolicies, 1)
	require.Equal(t, failingPolicy.ID, hh.FailingPolicies[0].ID)
	require.Len(t, hh.VulnerableSoftware, 1)
	require.Equal(t, soft1.ID, hh.VulnerableSoftware[0].ID)

	// Validate a host with no software or policies or team
	_, err = ds.NewHost(context.Background(), &fleet.Host{
		ID:                  2,
		OsqueryHostID:       ptr.String("empty"),
		NodeKey:             ptr.String("empty_nodekey"),
		Hostname:            "empty.local",
		UUID:                "uuid123",
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
	h, err = ds.Host(context.Background(), 2)
	require.NoError(t, err)

	hh, err = ds.GetHostHealth(context.Background(), h.ID)
	require.NoError(t, err)
	require.Equal(t, h.Platform, hh.Platform)
	require.Equal(t, h.DiskEncryptionEnabled, hh.DiskEncryptionEnabled)
	require.Equal(t, h.OSVersion, hh.OsVersion)
	require.Empty(t, hh.FailingPolicies)
	require.Empty(t, hh.VulnerableSoftware)
	require.Equal(t, h.TeamID, hh.TeamID)
}

func testGetHostOrbitInfo(t *testing.T, ds *Datastore) {
	host, err := ds.NewHost(
		context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String("1"),
			UUID:            "1",
			Hostname:        "foo.local",
			PrimaryIP:       "192.168.1.1",
			PrimaryMac:      "30-65-EC-6F-C4-58",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, host)

	_, err = ds.GetHostOrbitInfo(context.Background(), host.ID)
	require.True(t, fleet.IsNotFound(err))

	orbitVersion := "1.1.0"
	err = ds.SetOrUpdateHostOrbitInfo(
		context.Background(), host.ID, orbitVersion, sql.NullString{Valid: false}, sql.NullBool{Valid: false},
	)
	require.NoError(t, err)
	hostOrbitInfo, err := ds.GetHostOrbitInfo(context.Background(), host.ID)
	require.NoError(t, err)
	assert.Nil(t, hostOrbitInfo.ScriptsEnabled)

	err = ds.SetOrUpdateHostOrbitInfo(
		context.Background(), host.ID, orbitVersion, sql.NullString{Valid: false}, sql.NullBool{Bool: true, Valid: true},
	)
	require.NoError(t, err)
	hostOrbitInfo, err = ds.GetHostOrbitInfo(context.Background(), host.ID)
	require.NoError(t, err)
	assert.True(t, *hostOrbitInfo.ScriptsEnabled)
}

func testHostnamesByIdentifiers(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	// create a few hosts with different identifiers
	h1, err := ds.NewHost(
		ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(), LabelUpdatedAt: time.Now(),
			PolicyUpdatedAt: time.Now(), SeenTime: time.Now(),
			NodeKey:        ptr.String("abc"),
			UUID:           "def",
			Hostname:       "ghi.local",
			HardwareSerial: "jkl",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, h1)

	h2, err := ds.NewHost(
		ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(), LabelUpdatedAt: time.Now(),
			PolicyUpdatedAt: time.Now(), SeenTime: time.Now(),
			NodeKey:        ptr.String("def"),
			UUID:           "mno",
			Hostname:       "pqr.local",
			HardwareSerial: "sty",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, h2)

	h3, err := ds.NewHost(
		ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(), LabelUpdatedAt: time.Now(),
			PolicyUpdatedAt: time.Now(), SeenTime: time.Now(),
			NodeKey:        ptr.String("mno"),
			UUID:           "vwx",
			Hostname:       "yzA.local",
			HardwareSerial: "def",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, h3)

	cases := []struct {
		desc string
		in   []string
		out  []string
	}{
		{desc: "no identifier", in: nil, out: nil},
		{desc: "no match", in: []string{"ZZZ"}, out: nil},
		{desc: "single match", in: []string{"abc"}, out: []string{h1.Hostname}},
		{desc: "two matches", in: []string{"mno"}, out: []string{h2.Hostname, h3.Hostname}},
		{desc: "all matches", in: []string{"def"}, out: []string{h1.Hostname, h2.Hostname, h3.Hostname}},
		{desc: "multiple identifiers", in: []string{"abc", "mno", "vwx"}, out: []string{h1.Hostname, h2.Hostname, h3.Hostname}},
		{desc: "duplicate identifiers", in: []string{"abc", "abc", "ghi"}, out: []string{h1.Hostname}},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got, err := ds.HostnamesByIdentifiers(ctx, c.in)
			require.NoError(t, err)
			require.ElementsMatch(t, c.out, got)
		})
	}
}

func testHostsAddToTeamCleansUpTeamQueryResults(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	hostCount := 1
	newHost := func(teamID *uint) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			OsqueryHostID: ptr.String(fmt.Sprintf("foobar%d", hostCount)),
			NodeKey:       ptr.String(fmt.Sprintf("nodekey%d", hostCount)),
			TeamID:        teamID,
		})
		require.NoError(t, err)
		hostCount++
		return h
	}
	newQuery := func(name string, teamID *uint) *fleet.Query {
		q, err := ds.NewQuery(ctx, &fleet.Query{
			Name:    name,
			Query:   "SELECT 1:",
			TeamID:  teamID,
			Logging: fleet.LoggingSnapshot,
		})
		require.NoError(t, err)
		return q
	}

	h0 := newHost(nil)
	h1 := newHost(&team1.ID)
	h2 := newHost(&team2.ID)
	h3 := newHost(&team2.ID)

	hostStaticOnTeam1 := newHost(&team1.ID) // host that we won't move

	query0Global := newQuery("query0Global", nil)
	query1Team1 := newQuery("query1Team1", &team1.ID)
	query2Team2 := newQuery("query2Team2", &team2.ID)

	// Transfer h2 from team2 to team1 and back without any query results yet.
	err = ds.AddHostsToTeam(ctx, &team1.ID, []uint{h2.ID})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, &team2.ID, []uint{h2.ID})
	require.NoError(t, err)

	data := ptr.RawMessage(json.RawMessage(`{"foo": "bar"}`))
	h0Results := []*fleet.ScheduledQueryResultRow{
		{
			HostID:  h0.ID,
			QueryID: query0Global.ID,
			Data:    data,
		},
	}
	h1Global0Results := []*fleet.ScheduledQueryResultRow{
		{
			HostID:  h1.ID,
			QueryID: query0Global.ID,
			Data:    data,
		},
	}
	h1Query1Results := []*fleet.ScheduledQueryResultRow{
		{
			HostID:  h1.ID,
			QueryID: query1Team1.ID,
			Data:    data,
		},
	}
	h2Global0Results := []*fleet.ScheduledQueryResultRow{
		{
			HostID:  h2.ID,
			QueryID: query0Global.ID,
			Data:    data,
		},
	}
	h2Query2Results := []*fleet.ScheduledQueryResultRow{
		{
			HostID:  h2.ID,
			QueryID: query2Team2.ID,
			Data:    data,
		},
	}
	h3Global0Results := []*fleet.ScheduledQueryResultRow{
		{
			HostID:  h3.ID,
			QueryID: query0Global.ID,
			Data:    data,
		},
	}
	h3Query2Results := []*fleet.ScheduledQueryResultRow{
		{
			HostID:  h3.ID,
			QueryID: query2Team2.ID,
			Data:    data,
		},
	}
	h4Global0Results := []*fleet.ScheduledQueryResultRow{
		{
			HostID:  hostStaticOnTeam1.ID,
			QueryID: query0Global.ID,
			Data:    data,
		},
	}
	h4Query1Results := []*fleet.ScheduledQueryResultRow{
		{
			HostID:  hostStaticOnTeam1.ID,
			QueryID: query1Team1.ID,
			Data:    data,
		},
	}
	for _, results := range [][]*fleet.ScheduledQueryResultRow{
		h0Results,
		h1Global0Results,
		h1Query1Results,
		h2Global0Results,
		h2Query2Results,
		h3Global0Results,
		h3Query2Results,
		h4Global0Results,
		h4Query1Results,
	} {
		err = ds.OverwriteQueryResultRows(ctx, results, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
	}

	tf := fleet.TeamFilter{
		User: &fleet.User{
			GlobalRole: ptr.String(fleet.RoleAdmin),
		},
	}

	rows, err := ds.QueryResultRows(ctx, query0Global.ID, tf)
	require.NoError(t, err)
	require.Len(t, rows, 5)
	rows, err = ds.QueryResultRows(ctx, query1Team1.ID, tf)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	rows, err = ds.QueryResultRows(ctx, query2Team2.ID, tf)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	// Transfer h2 from team2 to team1.
	err = ds.AddHostsToTeam(ctx, &team1.ID, []uint{h2.ID})
	require.NoError(t, err)
	// Transfer h1 from team1 to team2.
	err = ds.AddHostsToTeam(ctx, &team2.ID, []uint{h1.ID})
	require.NoError(t, err)
	// Transfer h3 from team2 to global.
	err = ds.AddHostsToTeam(ctx, nil, []uint{h3.ID})
	require.NoError(t, err)

	// No global query results should be deleted
	rows, err = ds.QueryResultRows(ctx, query0Global.ID, tf)
	require.NoError(t, err)
	require.Len(t, rows, 5)
	// Results for h1 should be gone, and results for hostStaticOnTeam1 should be here.
	rows, err = ds.QueryResultRows(ctx, query1Team1.ID, tf)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, hostStaticOnTeam1.ID, rows[0].HostID)
	// Results for h2 and h3 should be gone.
	rows, err = ds.QueryResultRows(ctx, query2Team2.ID, tf)
	require.NoError(t, err)
	require.Empty(t, rows)

	// h1 should have only the global result.
	h1, err = ds.Host(ctx, h1.ID)
	require.NoError(t, err)
	require.Len(t, h1.PackStats, 1)
	require.Len(t, h1.PackStats[0].QueryStats, 1)
	require.Equal(t, query0Global.ID, h1.PackStats[0].QueryStats[0].ScheduledQueryID)

	// h2 should have only the global result.
	h2, err = ds.Host(ctx, h2.ID)
	require.NoError(t, err)
	require.Len(t, h2.PackStats, 1)
	require.Len(t, h2.PackStats[0].QueryStats, 1)
	require.Equal(t, query0Global.ID, h2.PackStats[0].QueryStats[0].ScheduledQueryID)

	// h3 should have only the global result.
	h3, err = ds.Host(ctx, h3.ID)
	require.NoError(t, err)
	require.Len(t, h3.PackStats, 1)
	require.Len(t, h3.PackStats[0].QueryStats, 1)
	require.Equal(t, query0Global.ID, h3.PackStats[0].QueryStats[0].ScheduledQueryID)

	// hostStaticOnTeam1 should have the global result and the team1 result.
	hostStaticOnTeam1, err = ds.Host(ctx, hostStaticOnTeam1.ID)
	require.NoError(t, err)
	require.Len(t, hostStaticOnTeam1.PackStats, 2)
	require.Len(t, hostStaticOnTeam1.PackStats[0].QueryStats, 1)
	require.Equal(t, query0Global.ID, hostStaticOnTeam1.PackStats[0].QueryStats[0].ScheduledQueryID)
	require.Len(t, hostStaticOnTeam1.PackStats[1].QueryStats, 1)
	require.Equal(t, query1Team1.ID, hostStaticOnTeam1.PackStats[1].QueryStats[0].ScheduledQueryID)
}

func testUpdateHostIssues(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	hosts := make([]*fleet.Host, 10)
	for i := range hosts {
		h, err := ds.NewHost(
			ctx, &fleet.Host{
				DetailUpdatedAt: time.Now(),
				LabelUpdatedAt:  time.Now(),
				PolicyUpdatedAt: time.Now(),
				SeenTime:        time.Now(),
				OsqueryHostID:   ptr.String(fmt.Sprintf("host%d", i)),
				NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
				UUID:            fmt.Sprintf("%d", i),
				Hostname:        fmt.Sprintf("foo.%d.local", i),
			},
		)
		require.NoError(t, err)
		hosts[i] = h
	}
	var hostIDs []uint
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID)
	}

	// Insert an issue for a non-existent host
	ExecAdhocSQL(
		t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `INSERT INTO host_issues (host_id) VALUES (?)`, hosts[len(hosts)-1].ID+1)
			return err
		},
	)

	// No issues with positive counts expected
	assert.NoError(t, ds.UpdateHostIssuesFailingPolicies(ctx, hostIDs))
	assert.NoError(t, ds.UpdateHostIssuesVulnerabilities(ctx))
	type issue struct {
		HostID uint `db:"host_id"`
		fleet.HostIssues
	}
	var issues []issue
	assert.NoError(
		t, sqlx.SelectContext(
			ctx, ds.reader(ctx), &issues,
			"SELECT host_id, failing_policies_count, critical_vulnerabilities_count, total_issues_count from host_issues",
		),
	)
	for _, is := range issues {
		assert.Zero(t, is.FailingPoliciesCount)
		assert.Zero(t, *is.CriticalVulnerabilitiesCount)
		assert.Zero(t, is.TotalIssuesCount)
	}

	// Clear the issues for non-existent hosts
	assert.NoError(t, ds.CleanupHostIssues(ctx))

	// Add some policy fails and critical vulnerabilities.
	// Hosts 0,1,8,9 don't have any issues
	// Hosts 2,3,4,5 have 2,3,4,5 policy fails
	// Hosts 4,5,6,7 have 1,2,3,4 critical vulnerabilities

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	policies := make([]*fleet.Policy, 0, 6)
	for i := 0; i < 6; i++ {
		q := test.NewQuery(t, ds, nil, fmt.Sprintf("query%d", i), "select 1", 0, true)
		p, err := ds.NewGlobalPolicy(
			context.Background(), &user1.ID, fleet.PolicyPayload{
				QueryID: &q.ID,
			},
		)
		require.NoError(t, err)
		policies = append(policies, p)
	}
	for i := 2; i < 6; i++ {
		results := make(map[uint]*bool, 6)
		for j := 0; j < 1; j++ {
			results[policies[j].ID] = ptr.Bool(true) // pass
		}
		for j := 1; j <= i; j++ {
			results[policies[j].ID] = ptr.Bool(false) // fail
		}
		for j := i + 1; j < 6; j++ {
			results[policies[j].ID] = ptr.Bool(true) // pass
		}
		require.NoError(
			// RecordPolicyQueryExecutions should call UpdateHostIssuesFailingPolicies, so we don't have to
			t, ds.RecordPolicyQueryExecutions(
				context.Background(), hosts[i], results, time.Now(), false,
			),
		)
	}

	// seed software
	software := []fleet.Software{
		{Name: "foo0", Version: "0", Source: "chrome_extensions"},
		{Name: "foo1", Version: "1", Source: "chrome_extensions"},
		{Name: "foo2", Version: "2", Source: "chrome_extensions"},
		{Name: "foo3", Version: "3", Source: "chrome_extensions"},
		{Name: "foo4", Version: "4", Source: "chrome_extensions"}, // vulnerable
		{Name: "foo5", Version: "5", Source: "chrome_extensions"}, // vulnerable
		{Name: "foo6", Version: "6", Source: "chrome_extensions"}, // vulnerable
		{Name: "foo7", Version: "7", Source: "chrome_extensions"}, // vulnerable
	}

	for i := 0; i < len(software); i++ {
		_, err := ds.UpdateHostSoftware(context.Background(), hosts[i].ID, software[:i+1])
		require.NoError(t, err)
	}

	softwareItems := make([]fleet.Software, 0, len(software))
	require.NoError(t, sqlx.SelectContext(ctx, ds.reader(ctx), &softwareItems, "SELECT id, version FROM software"))
	require.Len(t, softwareItems, len(software))

	for _, sw := range softwareItems {
		_, err := ds.InsertSoftwareVulnerability(
			context.Background(), fleet.SoftwareVulnerability{
				CVE:        fmt.Sprintf("CVE-%s", sw.Version),
				SoftwareID: sw.ID,
			}, fleet.NVDSource,
		)
		require.NoError(t, err)
	}
	require.NoError(
		t, ds.InsertCVEMeta(
			ctx, []fleet.CVEMeta{
				{
					CVE:       "CVE-3",
					CVSSScore: ptr.Float64(criticalCVSSScoreCutoff), // not critical
				},
				{
					CVE:       "CVE-4",
					CVSSScore: ptr.Float64(criticalCVSSScoreCutoff + 0.001),
				},
				{
					CVE:       "CVE-5",
					CVSSScore: ptr.Float64(criticalCVSSScoreCutoff + 0.01),
				},
				{
					CVE:       "CVE-6",
					CVSSScore: ptr.Float64(criticalCVSSScoreCutoff + 0.1),
				},
				{
					CVE:       "CVE-7",
					CVSSScore: ptr.Float64(criticalCVSSScoreCutoff + 1),
				},
			},
		),
	)

	// Test normal. UpdateHostIssuesFailingPolicies should not need to be called.
	assert.NoError(t, ds.UpdateHostIssuesVulnerabilities(ctx))
	issues = nil
	assert.NoError(
		t, sqlx.SelectContext(
			ctx, ds.reader(ctx), &issues,
			"SELECT host_id, failing_policies_count, critical_vulnerabilities_count, total_issues_count from host_issues ORDER BY host_id",
		),
	)
	nonZeroIssues := make([]issue, 0, 4)
	for _, hostIssue := range issues {
		if hostIssue.TotalIssuesCount == 0 {
			assert.Zero(t, hostIssue.FailingPoliciesCount)
			assert.Zero(t, *hostIssue.CriticalVulnerabilitiesCount)
			continue
		}
		nonZeroIssues = append(nonZeroIssues, hostIssue)
	}
	assert.Len(t, nonZeroIssues, 4)
	for i, hostIssue := range nonZeroIssues {
		count := i + 2
		assert.Equal(t, hosts[count].ID, hostIssue.HostID)
		assert.Equal(t, uint64(count), hostIssue.FailingPoliciesCount)
		assert.Zero(t, *hostIssue.CriticalVulnerabilitiesCount)
		assert.Equal(t, uint64(count), hostIssue.TotalIssuesCount)
	}

	// Test with small batch size and premium license
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	insertBatchSizeOrig := hostIssuesInsertBatchSize
	updateBatchSizeOrig := hostIssuesUpdateFailingPoliciesBatchSize
	t.Cleanup(
		func() {
			hostIssuesInsertBatchSize = insertBatchSizeOrig
			hostIssuesUpdateFailingPoliciesBatchSize = updateBatchSizeOrig
		},
	)
	hostIssuesInsertBatchSize = 2
	hostIssuesUpdateFailingPoliciesBatchSize = 2

	assert.NoError(t, ds.UpdateHostIssuesFailingPolicies(ctx, hostIDs))
	assert.NoError(t, ds.UpdateHostIssuesVulnerabilities(ctx))
	issues = nil
	assert.NoError(
		t, sqlx.SelectContext(
			ctx, ds.reader(ctx), &issues,
			"SELECT host_id, failing_policies_count, critical_vulnerabilities_count, total_issues_count from host_issues ORDER BY host_id",
		),
	)
	nonZeroIssues = make([]issue, 0, 6)
	for _, hostIssue := range issues {
		if hostIssue.TotalIssuesCount == 0 {
			assert.Zero(t, hostIssue.FailingPoliciesCount)
			assert.Zero(t, *hostIssue.CriticalVulnerabilitiesCount)
			continue
		}
		nonZeroIssues = append(nonZeroIssues, hostIssue)
	}
	assert.Len(t, nonZeroIssues, 6)
	for i, hostIssue := range nonZeroIssues {
		policiesCount := uint64(i + 2)
		criticalCount := uint64(0)
		if i > 1 {
			criticalCount = uint64(i - 1)
		}
		if i > 3 {
			policiesCount = 0
		}
		assert.Equal(t, hosts[i+2].ID, hostIssue.HostID)
		assert.Equal(t, policiesCount, hostIssue.FailingPoliciesCount)
		assert.Equal(t, criticalCount, *hostIssue.CriticalVulnerabilitiesCount)
		assert.Equal(t, policiesCount+criticalCount, hostIssue.TotalIssuesCount)
	}

	// Test with os vulnerability. First clear existing issues.
	ExecAdhocSQL(
		t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM policy_membership`)
			return err
		},
	)
	ExecAdhocSQL(
		t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM cve_meta`)
			return err
		},
	)

	// seed critical os vulnerability
	os := fleet.OperatingSystem{
		Name:          "Ubuntu",
		Version:       "20.4.0 LTS",
		Arch:          "x86_64",
		Platform:      "ubuntu",
		KernelVersion: "5.10.76-linuxkit",
	}
	require.NoError(t, ds.UpdateHostOperatingSystem(context.Background(), hosts[1].ID, os))
	var osID uint
	assert.NoError(
		t, sqlx.Get(
			ds.writer(ctx), &osID,
			"SELECT os_id FROM host_operating_system WHERE host_id = ?",
			hosts[1].ID,
		),
	)

	osVulns := []fleet.OSVulnerability{
		{
			OSID: osID,
			CVE:  "CVE-100",
		},
	}
	_, err := ds.InsertOSVulnerabilities(context.Background(), osVulns, fleet.NVDSource)
	require.NoError(t, err)
	require.NoError(
		t, ds.InsertCVEMeta(
			ctx, []fleet.CVEMeta{
				{
					CVE:       "CVE-100",
					CVSSScore: ptr.Float64(criticalCVSSScoreCutoff + 1), // critical
				},
			},
		),
	)
	assert.NoError(t, ds.UpdateHostIssuesFailingPolicies(ctx, hostIDs))
	assert.NoError(t, ds.UpdateHostIssuesVulnerabilities(ctx))
	issues = nil
	assert.NoError(
		t, sqlx.SelectContext(
			ctx, ds.reader(ctx), &issues,
			"SELECT host_id, failing_policies_count, critical_vulnerabilities_count, total_issues_count from host_issues ORDER BY host_id",
		),
	)
	hostIssueFound := false
	for _, hostIssue := range issues {
		if hostIssue.HostID == hosts[1].ID {
			hostIssueFound = true
			assert.Equal(t, hosts[1].ID, hostIssue.HostID)
			assert.Zero(t, hostIssue.FailingPoliciesCount)
			assert.Equal(t, uint64(1), *hostIssue.CriticalVulnerabilitiesCount)
			assert.Equal(t, uint64(1), hostIssue.TotalIssuesCount)
			continue
		}
		assert.Zero(t, hostIssue.FailingPoliciesCount)
		assert.Zero(t, *hostIssue.CriticalVulnerabilitiesCount)
		assert.Zero(t, hostIssue.TotalIssuesCount, "host issue: %+v", hostIssue)
	}
	assert.True(t, hostIssueFound)
}

func testListUpcomingHostMaintenanceWindows(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	err = ds.ReplaceHostDeviceMapping(ctx, host.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host.ID,
			Email:  "foo@example.com",
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	// call before any calendare events exist
	mWs, err := ds.ListUpcomingHostMaintenanceWindows(ctx, host.ID)
	require.NoError(t, err)
	require.Empty(t, mWs)

	// create an event
	timeZone := "America/Argentina/Buenos_Aires"

	startTime := time.Now().UTC().Add(30 * time.Minute)
	endTime := startTime.Add(30 * time.Minute)
	calendarEvent, err := ds.CreateOrUpdateCalendarEvent(ctx, uuid.New().String(), "foo@example.com", startTime, endTime, []byte(`{}`),
		&timeZone, host.ID, fleet.CalendarWebhookStatusNone)
	require.NoError(t, err)
	require.Equal(t, *calendarEvent.TimeZone, timeZone)

	mWs, err = ds.ListUpcomingHostMaintenanceWindows(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(mWs))
	mW := mWs[0]
	// round to match MySQL setting to round to nearest second (as of 6/27/2024)
	require.Equal(t, startTime.Round(time.Second), mW.StartsAt)
	require.Equal(t, timeZone, *mW.TimeZone)
}

func testGetHostEmails(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            uuid.NewString(),
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	emails, err := ds.GetHostEmails(ctx, host.UUID, fleet.DeviceMappingMDMIdpAccounts)
	require.NoError(t, err)
	assert.Empty(t, emails)

	err = ds.ReplaceHostDeviceMapping(ctx, host.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host.ID,
			Email:  "foo@example.com",
			Source: fleet.DeviceMappingMDMIdpAccounts,
		},
		{
			HostID: host.ID,
			Email:  "bar@example.com",
			Source: fleet.DeviceMappingMDMIdpAccounts,
		},
	}, fleet.DeviceMappingMDMIdpAccounts)
	require.NoError(t, err)

	emails, err = ds.GetHostEmails(ctx, host.UUID, fleet.DeviceMappingMDMIdpAccounts)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"foo@example.com", "bar@example.com"}, emails)
}

func testGetMatchingHostSerialsMarkedDeleted(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	serials := []string{"foo", "bar", "baz"}
	team, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)
	abmTok, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: t.Name(), EncryptedToken: []byte("token")})
	require.NoError(t, err)
	var hosts []fleet.Host
	for i, serial := range serials {
		var tmID *uint
		if serial == "bar" {
			tmID = &team.ID
		}
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String(fmt.Sprint(i)),
			UUID:            fmt.Sprint(i),
			OsqueryHostID:   ptr.String(fmt.Sprint(i)),
			Hostname:        "foo.local",
			PrimaryIP:       "192.168.1.1",
			PrimaryMac:      "30-65-EC-6F-C4-58",
			HardwareSerial:  serial,
			TeamID:          tmID,
			ID:              uint(i),
		})
		require.NoError(t, err)
		require.NotNil(t, h)

		// Only "foo" and "baz" are
		if i%2 == 0 {
			hosts = append(hosts, *h)
		}
	}

	require.NoError(t, ds.UpsertMDMAppleHostDEPAssignments(ctx, hosts, abmTok.ID))
	require.NoError(t, ds.DeleteHostDEPAssignments(ctx, abmTok.ID, serials))

	cases := []struct {
		name string
		in   []string
		want map[string]struct{}
		err  string
	}{
		{"no serials provided", []string{}, map[string]struct{}{}, ""},
		{"no matching serials", []string{"oof", "rab", "bar"}, map[string]struct{}{}, ""},
		{
			"partial matches",
			[]string{"foo", "rab", "bar"},
			map[string]struct{}{"foo": {}},
			"",
		},
		{
			"all matching",
			[]string{"foo", "baz"},
			map[string]struct{}{
				"foo": {},
				"baz": {},
			},
			"",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ds.GetMatchingHostSerialsMarkedDeleted(ctx, tt.in)
			if tt.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.err)
			}
			require.Equal(t, tt.want, got)
		})
	}
}
