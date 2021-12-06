package osquery_utils

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetailQueryNetworkInterfaces(t *testing.T) {
	var initialHost fleet.Host
	host := initialHost

	ingest := GetDetailQueries(nil)["network_interface"].IngestFunc

	assert.NoError(t, ingest(log.NewNopLogger(), &host, nil))
	assert.Equal(t, initialHost, host)

	var rows []map[string]string
	require.NoError(t, json.Unmarshal([]byte(`
[
  {"address":"127.0.0.1","mac":"00:00:00:00:00:00"},
  {"address":"::1","mac":"00:00:00:00:00:00"},
  {"address":"fe80::1%lo0","mac":"00:00:00:00:00:00"},
  {"address":"fe80::df:429b:971c:d051%en0","mac":"f4:5c:89:92:57:5b"},
  {"address":"192.168.1.3","mac":"f4:5d:79:93:58:5b"},
  {"address":"fe80::241a:9aff:fe60:d80a%awdl0","mac":"27:1b:aa:60:e8:0a"},
  {"address":"fe80::3a6f:582f:86c5:8296%utun0","mac":"00:00:00:00:00:00"}
]`),
		&rows,
	))

	assert.NoError(t, ingest(log.NewNopLogger(), &host, rows))
	assert.Equal(t, "192.168.1.3", host.PrimaryIP)
	assert.Equal(t, "f4:5d:79:93:58:5b", host.PrimaryMac)

	// Only IPv6
	require.NoError(t, json.Unmarshal([]byte(`
[
  {"address":"127.0.0.1","mac":"00:00:00:00:00:00"},
  {"address":"::1","mac":"00:00:00:00:00:00"},
  {"address":"fe80::1%lo0","mac":"00:00:00:00:00:00"},
  {"address":"fe80::df:429b:971c:d051%en0","mac":"f4:5c:89:92:57:5b"},
  {"address":"2604:3f08:1337:9411:cbe:814f:51a6:e4e3","mac":"27:1b:aa:60:e8:0a"},
  {"address":"3333:3f08:1337:9411:cbe:814f:51a6:e4e3","mac":"bb:1b:aa:60:e8:bb"},
  {"address":"fe80::3a6f:582f:86c5:8296%utun0","mac":"00:00:00:00:00:00"}
]`),
		&rows,
	))

	assert.NoError(t, ingest(log.NewNopLogger(), &host, rows))
	assert.Equal(t, "2604:3f08:1337:9411:cbe:814f:51a6:e4e3", host.PrimaryIP)
	assert.Equal(t, "27:1b:aa:60:e8:0a", host.PrimaryMac)

	// IPv6 appears before IPv4 (v4 should be prioritized)
	require.NoError(t, json.Unmarshal([]byte(`
[
  {"address":"127.0.0.1","mac":"00:00:00:00:00:00"},
  {"address":"::1","mac":"00:00:00:00:00:00"},
  {"address":"fe80::1%lo0","mac":"00:00:00:00:00:00"},
  {"address":"fe80::df:429b:971c:d051%en0","mac":"f4:5c:89:92:57:5b"},
  {"address":"2604:3f08:1337:9411:cbe:814f:51a6:e4e3","mac":"27:1b:aa:60:e8:0a"},
  {"address":"205.111.43.79","mac":"ab:1b:aa:60:e8:0a"},
  {"address":"205.111.44.80","mac":"bb:bb:aa:60:e8:0a"},
  {"address":"fe80::3a6f:582f:86c5:8296%utun0","mac":"00:00:00:00:00:00"}
]`),
		&rows,
	))

	assert.NoError(t, ingest(log.NewNopLogger(), &host, rows))
	assert.Equal(t, "205.111.43.79", host.PrimaryIP)
	assert.Equal(t, "ab:1b:aa:60:e8:0a", host.PrimaryMac)

	// Only link-local/loopback
	require.NoError(t, json.Unmarshal([]byte(`
[
  {"address":"127.0.0.1","mac":"00:00:00:00:00:00"},
  {"address":"::1","mac":"00:00:00:00:00:00"},
  {"address":"fe80::1%lo0","mac":"00:00:00:00:00:00"},
  {"address":"fe80::df:429b:971c:d051%en0","mac":"f4:5c:89:92:57:5b"},
  {"address":"fe80::241a:9aff:fe60:d80a%awdl0","mac":"27:1b:aa:60:e8:0a"},
  {"address":"fe80::3a6f:582f:86c5:8296%utun0","mac":"00:00:00:00:00:00"}
]`),
		&rows,
	))

	assert.NoError(t, ingest(log.NewNopLogger(), &host, rows))
	assert.Equal(t, "127.0.0.1", host.PrimaryIP)
	assert.Equal(t, "00:00:00:00:00:00", host.PrimaryMac)
}

func TestDetailQueryScheduledQueryStats(t *testing.T) {
	host := fleet.Host{}

	ingest := GetDetailQueries(nil)["scheduled_query_stats"].IngestFunc

	assert.NoError(t, ingest(log.NewNopLogger(), &host, nil))
	assert.Len(t, host.PackStats, 0)

	resJSON := `
[
  {
    "average_memory":"33",
    "delimiter":"/",
    "denylisted":"0",
    "executions":"1",
    "interval":"33",
    "last_executed":"1620325191",
    "name":"pack/pack-2/time",
    "output_size":"",
    "query":"SELECT * FROM time",
    "system_time":"100",
    "user_time":"60",
    "wall_time":"180"
  },
  {
    "average_memory":"8000",
    "delimiter":"/",
    "denylisted":"0",
    "executions":"164",
    "interval":"30",
    "last_executed":"1620325191",
    "name":"pack/test/osquery info",
    "output_size":"1337",
    "query":"SELECT * FROM osquery_info",
    "system_time":"150",
    "user_time":"180",
    "wall_time":"0"
  },
  {
    "average_memory":"50400",
    "delimiter":"/",
    "denylisted":"1",
    "executions":"188",
    "interval":"30",
    "last_executed":"1620325203",
    "name":"pack/test/processes?",
    "output_size":"",
    "query":"SELECT * FROM processes",
    "system_time":"140",
    "user_time":"190",
    "wall_time":"1"
  },
  {
    "average_memory":"0",
    "delimiter":"/",
    "denylisted":"0",
    "executions":"1",
    "interval":"3600",
    "last_executed":"1620323381",
    "name":"pack/test/processes?-1",
    "output_size":"",
    "query":"SELECT * FROM processes",
    "system_time":"0",
    "user_time":"0",
    "wall_time":"0"
  },
  {
    "average_memory":"0",
    "delimiter":"/",
    "denylisted":"0",
    "executions":"105",
    "interval":"47",
    "last_executed":"1620325190",
    "name":"pack/test/time",
    "output_size":"",
    "query":"SELECT * FROM time",
    "system_time":"70",
    "user_time":"50",
    "wall_time":"1"
  }
]
`

	var rows []map[string]string
	require.NoError(t, json.Unmarshal([]byte(resJSON), &rows))

	assert.NoError(t, ingest(log.NewNopLogger(), &host, rows))
	assert.Len(t, host.PackStats, 2)
	sort.Slice(host.PackStats, func(i, j int) bool {
		return host.PackStats[i].PackName < host.PackStats[j].PackName
	})
	assert.Equal(t, host.PackStats[0].PackName, "pack-2")
	assert.ElementsMatch(t, host.PackStats[0].QueryStats,
		[]fleet.ScheduledQueryStats{
			{
				ScheduledQueryName: "time",
				PackName:           "pack-2",
				AverageMemory:      33,
				Denylisted:         false,
				Executions:         1,
				Interval:           33,
				LastExecuted:       time.Unix(1620325191, 0).UTC(),
				OutputSize:         0,
				SystemTime:         100,
				UserTime:           60,
				WallTime:           180,
			},
		},
	)
	assert.Equal(t, host.PackStats[1].PackName, "test")
	assert.ElementsMatch(t, host.PackStats[1].QueryStats,
		[]fleet.ScheduledQueryStats{
			{
				ScheduledQueryName: "osquery info",
				PackName:           "test",
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
			{
				ScheduledQueryName: "processes?",
				PackName:           "test",
				AverageMemory:      50400,
				Denylisted:         true,
				Executions:         188,
				Interval:           30,
				LastExecuted:       time.Unix(1620325203, 0).UTC(),
				OutputSize:         0,
				SystemTime:         140,
				UserTime:           190,
				WallTime:           1,
			},
			{
				ScheduledQueryName: "processes?-1",
				PackName:           "test",
				AverageMemory:      0,
				Denylisted:         false,
				Executions:         1,
				Interval:           3600,
				LastExecuted:       time.Unix(1620323381, 0).UTC(),
				OutputSize:         0,
				SystemTime:         0,
				UserTime:           0,
				WallTime:           0,
			},
			{
				ScheduledQueryName: "time",
				PackName:           "test",
				AverageMemory:      0,
				Denylisted:         false,
				Executions:         105,
				Interval:           47,
				LastExecuted:       time.Unix(1620325190, 0).UTC(),
				OutputSize:         0,
				SystemTime:         70,
				UserTime:           50,
				WallTime:           1,
			},
		},
	)

	assert.NoError(t, ingest(log.NewNopLogger(), &host, nil))
	assert.Len(t, host.PackStats, 0)
}

func sortedKeysCompare(t *testing.T, m map[string]DetailQuery, expectedKeys []string) {
	var keys []string
	for key := range m {
		keys = append(keys, key)
	}
	assert.ElementsMatch(t, keys, expectedKeys)
}

func TestGetDetailQueries(t *testing.T) {
	queriesNoConfig := GetDetailQueries(nil)
	require.Len(t, queriesNoConfig, 9)
	baseQueries := []string{
		"network_interface",
		"os_version",
		"osquery_flags",
		"osquery_info",
		"scheduled_query_stats",
		"system_info",
		"uptime",
		"disk_space_unix",
		"disk_space_windows",
	}
	sortedKeysCompare(t, queriesNoConfig, baseQueries)

	queriesWithUsers := GetDetailQueries(&fleet.AppConfig{HostSettings: fleet.HostSettings{EnableHostUsers: true}})
	require.Len(t, queriesWithUsers, 10)
	sortedKeysCompare(t, queriesWithUsers, append(baseQueries, "users"))

	queriesWithUsersAndSoftware := GetDetailQueries(&fleet.AppConfig{HostSettings: fleet.HostSettings{EnableHostUsers: true, EnableSoftwareInventory: true}})
	require.Len(t, queriesWithUsersAndSoftware, 13)
	sortedKeysCompare(t, queriesWithUsersAndSoftware,
		append(baseQueries, "users", "software_macos", "software_linux", "software_windows"))
}
