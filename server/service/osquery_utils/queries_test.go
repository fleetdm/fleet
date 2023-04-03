package osquery_utils

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetailQueryNetworkInterfaces(t *testing.T) {
	var initialHost fleet.Host
	host := initialHost

	ingest := GetDetailQueries(context.Background(), config.FleetConfig{}, nil, nil)["network_interface_unix"].IngestFunc

	assert.NoError(t, ingest(context.Background(), log.NewNopLogger(), &host, nil))
	assert.Equal(t, initialHost, host)

	var rows []map[string]string
	require.NoError(t, json.Unmarshal([]byte(`
[
  {"address":"10.0.1.2","mac":"bc:d0:74:4b:10:6d"}
]`),
		&rows,
	))

	assert.NoError(t, ingest(context.Background(), log.NewNopLogger(), &host, rows))
	assert.Equal(t, "10.0.1.2", host.PrimaryIP)
	assert.Equal(t, "bc:d0:74:4b:10:6d", host.PrimaryMac)
}

func TestDetailQueryScheduledQueryStats(t *testing.T) {
	host := fleet.Host{ID: 1}
	ds := new(mock.Store)
	task := async.NewTask(ds, nil, clock.C, config.OsqueryConfig{EnableAsyncHostProcessing: "false"})

	var gotPackStats []fleet.PackStats
	ds.SaveHostPackStatsFunc = func(ctx context.Context, hostID uint, stats []fleet.PackStats) error {
		if hostID != host.ID {
			return errors.New("not found")
		}
		gotPackStats = stats
		return nil
	}

	ingest := GetDetailQueries(context.Background(), config.FleetConfig{App: config.AppConfig{EnableScheduledQueryStats: true}}, nil, nil)["scheduled_query_stats"].DirectTaskIngestFunc

	ctx := context.Background()
	assert.NoError(t, ingest(ctx, log.NewNopLogger(), &host, task, nil))
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

	assert.NoError(t, ingest(ctx, log.NewNopLogger(), &host, task, rows))
	assert.Len(t, gotPackStats, 2)
	sort.Slice(gotPackStats, func(i, j int) bool {
		return gotPackStats[i].PackName < gotPackStats[j].PackName
	})
	assert.Equal(t, gotPackStats[0].PackName, "pack-2")
	assert.ElementsMatch(t, gotPackStats[0].QueryStats,
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
	assert.Equal(t, gotPackStats[1].PackName, "test")
	assert.ElementsMatch(t, gotPackStats[1].QueryStats,
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

	assert.NoError(t, ingest(ctx, log.NewNopLogger(), &host, task, nil))
	assert.Len(t, gotPackStats, 0)
}

func sortedKeysCompare(t *testing.T, m map[string]DetailQuery, expectedKeys []string) {
	var keys []string
	for key := range m {
		keys = append(keys, key)
	}
	assert.ElementsMatch(t, keys, expectedKeys)
}

func TestGetDetailQueries(t *testing.T) {
	queriesNoConfig := GetDetailQueries(context.Background(), config.FleetConfig{}, nil, nil)

	baseQueries := []string{
		"network_interface_unix",
		"network_interface_windows",
		"network_interface_chrome",
		"os_version",
		"os_version_windows",
		"osquery_flags",
		"osquery_info",
		"system_info",
		"uptime",
		"disk_space_unix",
		"disk_space_windows",
		"mdm",
		"mdm_windows",
		"munki_info",
		"google_chrome_profiles",
		"battery",
		"os_windows",
		"os_unix_like",
		"windows_update_history",
		"kubequery_info",
		"orbit_info",
		"disk_encryption_darwin",
		"disk_encryption_linux",
		"disk_encryption_windows",
	}

	require.Len(t, queriesNoConfig, len(baseQueries))
	sortedKeysCompare(t, queriesNoConfig, baseQueries)

	queriesWithoutWinOSVuln := GetDetailQueries(context.Background(), config.FleetConfig{Vulnerabilities: config.VulnerabilitiesConfig{DisableWinOSVulnerabilities: true}}, nil, nil)
	require.Len(t, queriesWithoutWinOSVuln, 23)

	queriesWithUsers := GetDetailQueries(context.Background(), config.FleetConfig{App: config.AppConfig{EnableScheduledQueryStats: true}}, nil, &fleet.Features{EnableHostUsers: true})
	qs := append(baseQueries, "users", "users_chrome", "scheduled_query_stats")
	require.Len(t, queriesWithUsers, len(qs))
	sortedKeysCompare(t, queriesWithUsers, qs)

	queriesWithUsersAndSoftware := GetDetailQueries(context.Background(), config.FleetConfig{App: config.AppConfig{EnableScheduledQueryStats: true}}, nil, &fleet.Features{EnableHostUsers: true, EnableSoftwareInventory: true})
	qs = append(baseQueries, "users", "users_chrome", "software_macos", "software_linux", "software_windows", "software_chrome", "scheduled_query_stats")
	require.Len(t, queriesWithUsersAndSoftware, len(qs))
	sortedKeysCompare(t, queriesWithUsersAndSoftware, qs)
}

func TestDetailQueriesOSVersionUnixLike(t *testing.T) {
	var initialHost fleet.Host
	host := initialHost

	ingest := GetDetailQueries(context.Background(), config.FleetConfig{}, nil, nil)["os_version"].IngestFunc

	assert.NoError(t, ingest(context.Background(), log.NewNopLogger(), &host, nil))
	assert.Equal(t, initialHost, host)

	// Rolling release for archlinux
	var rows []map[string]string
	require.NoError(t, json.Unmarshal([]byte(`
[{
    "hostname": "kube2",
    "arch": "x86_64",
    "build": "rolling",
    "codename": "",
    "major": "0",
    "minor": "0",
    "name": "Arch Linux",
    "patch": "0",
    "platform": "arch",
    "platform_like": "",
    "version": ""
}]`),
		&rows,
	))

	assert.NoError(t, ingest(context.Background(), log.NewNopLogger(), &host, rows))
	assert.Equal(t, "Arch Linux rolling", host.OSVersion)

	// Simulate a linux with a proper version
	require.NoError(t, json.Unmarshal([]byte(`
[{
    "hostname": "kube2",
    "arch": "x86_64",
    "build": "rolling",
    "codename": "",
    "major": "1",
    "minor": "2",
    "name": "Arch Linux",
    "patch": "3",
    "platform": "arch",
    "platform_like": "",
    "version": ""
}]`),
		&rows,
	))

	assert.NoError(t, ingest(context.Background(), log.NewNopLogger(), &host, rows))
	assert.Equal(t, "Arch Linux 1.2.3", host.OSVersion)

	// Simulate Ubuntu host with incorrect `patch` number
	require.NoError(t, json.Unmarshal([]byte(`
[{
    "hostname": "kube2",
    "arch": "x86_64",
    "build": "",
    "codename": "bionic",
    "major": "18",
    "minor": "4",
    "name": "Ubuntu",
    "patch": "0",
    "platform": "ubuntu",
    "platform_like": "debian",
    "version": "18.04.5 LTS (Bionic Beaver)"
}]`),
		&rows,
	))

	assert.NoError(t, ingest(context.Background(), log.NewNopLogger(), &host, rows))
	assert.Equal(t, "Ubuntu 18.04.5 LTS", host.OSVersion)
}

func TestDetailQueriesOSVersionWindows(t *testing.T) {
	var initialHost fleet.Host
	host := initialHost

	ingest := GetDetailQueries(context.Background(), config.FleetConfig{}, nil, nil)["os_version_windows"].IngestFunc

	assert.NoError(t, ingest(context.Background(), log.NewNopLogger(), &host, nil))
	assert.Equal(t, initialHost, host)

	var rows []map[string]string
	require.NoError(t, json.Unmarshal([]byte(`
[{
    "hostname": "WinBox",
    "arch": "64-bit",
    "build": "22000",
    "codename": "Microsoft Windows 11 Enterprise",
    "major": "10",
    "minor": "0",
    "name": "Microsoft Windows 11 Enterprise",
    "patch": "",
    "platform": "windows",
    "platform_like": "windows",
    "version": "10.0.22000",
	"display_version": "21H2",
	"release_id": ""
}]`),
		&rows,
	))

	assert.NoError(t, ingest(context.Background(), log.NewNopLogger(), &host, rows))
	assert.Equal(t, "Windows 11 Enterprise 21H2", host.OSVersion)

	require.NoError(t, json.Unmarshal([]byte(`
[{
    "hostname": "WinBox",
    "arch": "64-bit",
    "build": "17763",
    "codename": "Microsoft Windows 10 Enterprise LTSC",
    "major": "10",
    "minor": "0",
    "name": "Microsoft Windows 10 Enterprise LTSC",
    "patch": "",
    "platform": "windows",
    "platform_like": "windows",
    "version": "10.0.17763",
	"display_version": "",
	"release_id": "1809"
}]`),
		&rows,
	))

	assert.NoError(t, ingest(context.Background(), log.NewNopLogger(), &host, rows))
	assert.Equal(t, "Windows 10 Enterprise LTSC ", host.OSVersion)
}

func TestDetailQueriesOSVersionChrome(t *testing.T) {
	var initialHost fleet.Host
	host := initialHost

	ingest := GetDetailQueries(context.Background(), config.FleetConfig{}, nil, nil)["os_version"].IngestFunc

	assert.NoError(t, ingest(context.Background(), log.NewNopLogger(), &host, nil))
	assert.Equal(t, initialHost, host)

	var rows []map[string]string
	require.NoError(t, json.Unmarshal([]byte(`
[{
    "hostname": "chromeo",
    "arch": "x86_64",
    "build": "chrome-build",
    "codename": "",
    "major": "1",
    "minor": "3",
    "name": "chromeos",
    "patch": "7",
    "platform": "chrome",
    "platform_like": "chrome",
    "version": "1.3.3.7"
}]`),
		&rows,
	))

	assert.NoError(t, ingest(context.Background(), log.NewNopLogger(), &host, rows))
	assert.Equal(t, "chromeos 1.3.3.7", host.OSVersion)
}

func TestDirectIngestMDMMac(t *testing.T) {
	ds := new(mock.Store)
	ds.SetOrUpdateMDMDataFunc = func(ctx context.Context, hostID uint, isServer, enrolled bool, serverURL string, installedFromDep bool, name string) error {
		require.False(t, enrolled)
		require.False(t, installedFromDep)
		require.Empty(t, serverURL)
		return nil
	}

	var host fleet.Host

	err := directIngestMDMMac(context.Background(), log.NewNopLogger(), &host, ds, []map[string]string{
		{
			"enrolled":           "false",
			"installed_from_dep": "",
			"server_url":         "",
		},
	})
	require.NoError(t, err)
	require.True(t, ds.SetOrUpdateMDMDataFuncInvoked)
}

func TestDirectIngestMDMWindows(t *testing.T) {
	ds := new(mock.Store)
	ds.SetOrUpdateMDMDataFunc = func(ctx context.Context, hostID uint, isServer, enrolled bool, serverURL string, installedFromDep bool, name string) error {
		require.True(t, enrolled)
		require.True(t, installedFromDep)
		require.True(t, isServer)
		require.NotEmpty(t, serverURL)
		return nil
	}

	var host fleet.Host
	err := directIngestMDMWindows(context.Background(), log.NewNopLogger(), &host, ds, []map[string]string{
		{"key": "discovery_service_url", "value": "some url"},
		{"key": "autopilot", "value": "true"},
		{"key": "provider_id", "value": "1337"},
		{"key": "installation_type", "value": "Windows SeRvEr 99.9"},
	})
	require.NoError(t, err)
	require.True(t, ds.SetOrUpdateMDMDataFuncInvoked)
}

func TestDirectIngestChromeProfiles(t *testing.T) {
	ds := new(mock.Store)
	ds.ReplaceHostDeviceMappingFunc = func(ctx context.Context, hostID uint, mapping []*fleet.HostDeviceMapping) error {
		require.Equal(t, hostID, uint(1))
		require.Equal(t, mapping, []*fleet.HostDeviceMapping{
			{HostID: hostID, Email: "test@example.com", Source: "google_chrome_profiles"},
			{HostID: hostID, Email: "test+2@example.com", Source: "google_chrome_profiles"},
		})
		return nil
	}

	host := fleet.Host{
		ID: 1,
	}

	err := directIngestChromeProfiles(context.Background(), log.NewNopLogger(), &host, ds, []map[string]string{
		{"email": "test@example.com"},
		{"email": "test+2@example.com"},
	})

	require.NoError(t, err)
	require.True(t, ds.ReplaceHostDeviceMappingFuncInvoked)
}

func TestDirectIngestBattery(t *testing.T) {
	ds := new(mock.Store)
	ds.ReplaceHostBatteriesFunc = func(ctx context.Context, id uint, mappings []*fleet.HostBattery) error {
		require.Equal(t, mappings, []*fleet.HostBattery{
			{HostID: uint(1), SerialNumber: "a", CycleCount: 2, Health: "Good"},
			{HostID: uint(1), SerialNumber: "c", CycleCount: 3, Health: strings.Repeat("z", 40)},
		})
		return nil
	}

	host := fleet.Host{
		ID: 1,
	}

	err := directIngestBattery(context.Background(), log.NewNopLogger(), &host, ds, []map[string]string{
		{"serial_number": "a", "cycle_count": "2", "health": "Good"},
		{"serial_number": "c", "cycle_count": "3", "health": strings.Repeat("z", 100)},
	})

	require.NoError(t, err)
	require.True(t, ds.ReplaceHostBatteriesFuncInvoked)
}

func TestDirectIngestOSWindows(t *testing.T) {
	ds := new(mock.Store)

	testCases := []struct {
		expected fleet.OperatingSystem
		data     []map[string]string
	}{
		{
			expected: fleet.OperatingSystem{
				Name:          "Microsoft Windows 11 Enterprise",
				Version:       "21H2",
				Arch:          "64-bit",
				KernelVersion: "10.0.22000.795",
			},
			data: []map[string]string{
				{"name": "Microsoft Windows 11 Enterprise", "display_version": "21H2", "release_id": "", "arch": "64-bit", "kernel_version": "10.0.22000.795"},
			},
		},
	}

	host := fleet.Host{ID: 1}

	for _, tt := range testCases {
		ds.UpdateHostOperatingSystemFunc = func(ctx context.Context, hostID uint, hostOS fleet.OperatingSystem) error {
			require.Equal(t, host.ID, hostID)
			require.Equal(t, tt.expected, hostOS)
			return nil
		}

		err := directIngestOSWindows(context.Background(), log.NewNopLogger(), &host, ds, tt.data)
		require.NoError(t, err)

		require.True(t, ds.UpdateHostOperatingSystemFuncInvoked)
		ds.UpdateHostOperatingSystemFuncInvoked = false
	}
}

func TestDirectIngestOSUnixLike(t *testing.T) {
	ds := new(mock.Store)

	for i, tc := range []struct {
		data     []map[string]string
		expected fleet.OperatingSystem
	}{
		{
			data: []map[string]string{
				{
					"name":           "macOS",
					"version":        "12.5",
					"major":          "12",
					"minor":          "5",
					"patch":          "0",
					"build":          "21G72",
					"arch":           "x86_64",
					"kernel_version": "21.6.0",
				},
			},
			expected: fleet.OperatingSystem{
				Name:          "macOS",
				Version:       "12.5.0",
				Arch:          "x86_64",
				KernelVersion: "21.6.0",
			},
		},
		{
			data: []map[string]string{
				{
					"name":           "Ubuntu",
					"version":        "20.04.2 LTS (Focal Fossa)",
					"major":          "20",
					"minor":          "4",
					"patch":          "0",
					"build":          "",
					"arch":           "x86_64",
					"kernel_version": "5.10.76-linuxkit",
				},
			},
			expected: fleet.OperatingSystem{
				Name:          "Ubuntu",
				Version:       "20.04.2 LTS",
				Arch:          "x86_64",
				KernelVersion: "5.10.76-linuxkit",
			},
		},
		{
			data: []map[string]string{
				{
					"name":           "CentOS Linux",
					"version":        "CentOS Linux release 7.9.2009 (Core)",
					"major":          "7",
					"minor":          "9",
					"patch":          "2009",
					"build":          "",
					"arch":           "x86_64",
					"kernel_version": "5.10.76-linuxkit",
				},
			},
			expected: fleet.OperatingSystem{
				Name:          "CentOS Linux",
				Version:       "7.9.2009",
				Arch:          "x86_64",
				KernelVersion: "5.10.76-linuxkit",
			},
		},
		{
			data: []map[string]string{
				{
					"name":           "Debian GNU/Linux",
					"version":        "10 (buster)",
					"major":          "10",
					"minor":          "0",
					"patch":          "0",
					"build":          "",
					"arch":           "x86_64",
					"kernel_version": "5.10.76-linuxkit",
				},
			},
			expected: fleet.OperatingSystem{
				Name:          "Debian GNU/Linux",
				Version:       "10.0.0",
				Arch:          "x86_64",
				KernelVersion: "5.10.76-linuxkit",
			},
		},
		{
			data: []map[string]string{
				{
					"name":           "CentOS Linux",
					"version":        "CentOS Linux release 7.9.2009 (Core)",
					"major":          "7",
					"minor":          "9",
					"patch":          "2009",
					"build":          "",
					"arch":           "x86_64",
					"kernel_version": "5.10.76-linuxkit",
				},
			},
			expected: fleet.OperatingSystem{
				Name:          "CentOS Linux",
				Version:       "7.9.2009",
				Arch:          "x86_64",
				KernelVersion: "5.10.76-linuxkit",
			},
		},
	} {
		t.Run(tc.expected.Name, func(t *testing.T) {
			ds.UpdateHostOperatingSystemFunc = func(ctx context.Context, hostID uint, hostOS fleet.OperatingSystem) error {
				require.Equal(t, uint(i), hostID)
				require.Equal(t, tc.expected, hostOS)
				return nil
			}

			err := directIngestOSUnixLike(context.Background(), log.NewNopLogger(), &fleet.Host{ID: uint(i)}, ds, tc.data)

			require.NoError(t, err)
			require.True(t, ds.UpdateHostOperatingSystemFuncInvoked)
			ds.UpdateHostOperatingSystemFuncInvoked = false
		})
	}
}

func TestAppConfigReplaceQuery(t *testing.T) {
	queries := GetDetailQueries(context.Background(), config.FleetConfig{}, nil, &fleet.Features{EnableHostUsers: true})
	originalQuery := queries["users"].Query

	replacementMap := make(map[string]*string)
	replacementMap["users"] = ptr.String("select 1 from blah")
	queries = GetDetailQueries(context.Background(), config.FleetConfig{}, nil, &fleet.Features{EnableHostUsers: true, DetailQueryOverrides: replacementMap})
	assert.NotEqual(t, originalQuery, queries["users"].Query)
	assert.Equal(t, "select 1 from blah", queries["users"].Query)

	replacementMap["users"] = nil
	queries = GetDetailQueries(context.Background(), config.FleetConfig{}, nil, &fleet.Features{EnableHostUsers: true, DetailQueryOverrides: replacementMap})
	_, exists := queries["users"]
	assert.False(t, exists)
}

func TestDirectIngestSoftware(t *testing.T) {
	ds := new(mock.Store)

	t.Run("vendor gets truncated", func(t *testing.T) {
		for i, tc := range []struct {
			data     []map[string]string
			expected string
		}{
			{
				data: []map[string]string{
					{
						"name":              "Software 1",
						"version":           "12.5",
						"source":            "My backyard",
						"bundle_identifier": "",
						"vendor":            "Fleet",
					},
				},
				expected: "Fleet",
			},
			{
				data: []map[string]string{
					{
						"name":              "Software 1",
						"version":           "12.5",
						"source":            "My backyard",
						"bundle_identifier": "",
						"vendor":            `oFZTwTV5WxJt02EVHEBcnhLzuJ8wnxKwfbabPWy7yTSiQbabEcAGDVmoXKZEZJLWObGD0cVfYptInHYgKjtDeDsBh2a8669EnyAqyBECXbFjSh1111`,
					},
				},
				expected: `oFZTwTV5WxJt02EVHEBcnhLzuJ8wnxKwfbabPWy7yTSiQbabEcAGDVmoXKZEZJLWObGD0cVfYptInHYgKjtDeDsBh2a8669EnyAqyBECXbFjSh1...`,
			},
		} {
			ds.UpdateHostSoftwareFunc = func(ctx context.Context, hostID uint, software []fleet.Software) error {
				require.Len(t, software, 1)
				require.Equal(t, tc.expected, software[0].Vendor)
				return nil
			}

			err := directIngestSoftware(
				context.Background(),
				log.NewNopLogger(),
				&fleet.Host{ID: uint(i)},
				ds,
				tc.data,
			)

			require.NoError(t, err)
			require.True(t, ds.UpdateHostSoftwareFuncInvoked)
			ds.UpdateHostSoftwareFuncInvoked = false
		}
	})
}

func TestDirectIngestWindowsUpdateHistory(t *testing.T) {
	ds := new(mock.Store)
	ds.InsertWindowsUpdatesFunc = func(ctx context.Context, hostID uint, updates []fleet.WindowsUpdate) error {
		require.Len(t, updates, 6)
		require.ElementsMatch(t, []fleet.WindowsUpdate{
			{KBID: 2267602, DateEpoch: 1657929207},
			{KBID: 890830, DateEpoch: 1658226954},
			{KBID: 5013887, DateEpoch: 1658225364},
			{KBID: 5005463, DateEpoch: 1658225225},
			{KBID: 5010472, DateEpoch: 1658224963},
			{KBID: 4052623, DateEpoch: 1657929544},
		}, updates)
		return nil
	}

	host := fleet.Host{
		ID: 1,
	}

	payload := []map[string]string{
		{"date": "1659392951", "title": "Security Intelligence Update for Microsoft Defender Antivirus - KB2267602 (Version 1.371.1239.0)"},
		{"date": "1658271402", "title": "Security Intelligence Update for Microsoft Defender Antivirus - KB2267602 (Version 1.371.442.0)"},
		{"date": "1658228495", "title": "Security Intelligence Update for Microsoft Defender Antivirus - KB2267602 (Version 1.371.415.0)"},
		{"date": "1658226954", "title": "Windows Malicious Software Removal Tool x64 - v5.103 (KB890830)"},
		{"date": "1658225364", "title": "2022-06 Cumulative Update for .NET Framework 3.5 and 4.8 for Windows 10 Version 21H2 for x64 (KB5013887)"},
		{"date": "1658225225", "title": "2022-04 Update for Windows 10 Version 21H2 for x64-based Systems (KB5005463)"},
		{"date": "1658224963", "title": "2022-02 Cumulative Update Preview for .NET Framework 3.5 and 4.8 for Windows 10 Version 21H2 for x64 (KB5010472)"},
		{"date": "1658222131", "title": "Security Intelligence Update for Microsoft Defender Antivirus - KB2267602 (Version 1.371.400.0)"},
		{"date": "1658189063", "title": "Security Intelligence Update for Microsoft Defender Antivirus - KB2267602 (Version 1.371.376.0)"},
		{"date": "1658185542", "title": "Security Intelligence Update for Microsoft Defender Antivirus - KB2267602 (Version 1.371.386.0)"},
		{"date": "1657929544", "title": "Update for Microsoft Defender Antivirus antimalware platform - KB4052623 (Version 4.18.2205.7)"},
		{"date": "1657929207", "title": "Security Intelligence Update for Microsoft Defender Antivirus - KB2267602 (Version 1.371.203.0)"},
	}

	err := directIngestWindowsUpdateHistory(context.Background(), log.NewNopLogger(), &host, ds, payload)
	require.NoError(t, err)
	require.True(t, ds.InsertWindowsUpdatesFuncInvoked)
}

func TestIngestKubequeryInfo(t *testing.T) {
	err := ingestKubequeryInfo(context.Background(), log.NewNopLogger(), &fleet.Host{}, nil)
	require.Error(t, err)
	err = ingestKubequeryInfo(context.Background(), log.NewNopLogger(), &fleet.Host{}, []map[string]string{})
	require.Error(t, err)
	err = ingestKubequeryInfo(context.Background(), log.NewNopLogger(), &fleet.Host{}, []map[string]string{
		{
			"cluster_name": "foo",
		},
	})
	require.NoError(t, err)
}

func TestDirectDiskEncryption(t *testing.T) {
	ds := new(mock.Store)
	var expectEncrypted bool
	ds.SetOrUpdateHostDisksEncryptionFunc = func(ctx context.Context, id uint, encrypted bool) error {
		assert.Equal(t, expectEncrypted, encrypted)
		return nil
	}

	host := fleet.Host{
		ID: 1,
	}

	// set to true (osquery returned a row)
	expectEncrypted = true
	err := directIngestDiskEncryption(context.Background(), log.NewNopLogger(), &host, ds, []map[string]string{
		{"col1": "1"},
	})
	require.NoError(t, err)
	require.True(t, ds.SetOrUpdateHostDisksEncryptionFuncInvoked)
	ds.SetOrUpdateHostDisksEncryptionFuncInvoked = false

	// set to false (osquery returned nothing)
	expectEncrypted = false
	err = directIngestDiskEncryption(context.Background(), log.NewNopLogger(), &host, ds, []map[string]string{})
	require.NoError(t, err)
	require.True(t, ds.SetOrUpdateHostDisksEncryptionFuncInvoked)
	ds.SetOrUpdateHostDisksEncryptionFuncInvoked = false
}

func TestDirectIngestDiskEncryptionLinux(t *testing.T) {
	ds := new(mock.Store)
	var expectEncrypted bool
	ds.SetOrUpdateHostDisksEncryptionFunc = func(ctx context.Context, id uint, encrypted bool) error {
		assert.Equal(t, expectEncrypted, encrypted)
		return nil
	}
	host := fleet.Host{
		ID: 1,
	}

	expectEncrypted = false
	err := directIngestDiskEncryptionLinux(context.Background(), log.NewNopLogger(), &host, ds, []map[string]string{})
	require.NoError(t, err)
	require.True(t, ds.SetOrUpdateHostDisksEncryptionFuncInvoked)
	ds.SetOrUpdateHostDisksEncryptionFuncInvoked = false

	expectEncrypted = true
	err = directIngestDiskEncryptionLinux(context.Background(), log.NewNopLogger(), &host, ds, []map[string]string{
		{"path": "/etc/hosts", "encrypted": "0"},
		{"path": "/tmp", "encrypted": "0"},
		{"path": "/", "encrypted": "1"},
	})
	require.NoError(t, err)
	require.True(t, ds.SetOrUpdateHostDisksEncryptionFuncInvoked)
}

func TestDirectIngestDiskEncryptionKeyDarwin(t *testing.T) {
	ds := new(mock.Store)
	ctx := context.Background()
	logger := log.NewNopLogger()
	wantKey := "OTM5ODRDQTYtOUY1Mi00NERELTkxOUEtMDlBN0ZBOUUzNUY5Cg=="
	host := &fleet.Host{ID: 1}

	ds.SetOrUpdateHostDiskEncryptionKeyFunc = func(ctx context.Context, hostID uint, encryptedBase64Key string) error {
		require.Empty(t, encryptedBase64Key)
		require.Equal(t, host.ID, hostID)
		return nil
	}

	err := directIngestDiskEncryptionKeyDarwin(ctx, logger, host, ds, []map[string]string{})
	require.NoError(t, err)
	require.False(t, ds.SetOrUpdateHostDiskEncryptionKeyFuncInvoked)

	err = directIngestDiskEncryptionKeyDarwin(ctx, logger, host, ds, []map[string]string{{"encrypted": "0"}})
	require.NoError(t, err)
	require.False(t, ds.SetOrUpdateHostDiskEncryptionKeyFuncInvoked)

	err = directIngestDiskEncryptionKeyDarwin(ctx, logger, host, ds, []map[string]string{{"filevault_key": ""}})
	require.NoError(t, err)
	require.True(t, ds.SetOrUpdateHostDiskEncryptionKeyFuncInvoked)
	ds.SetOrUpdateHostDiskEncryptionKeyFuncInvoked = false

	ds.SetOrUpdateHostDiskEncryptionKeyFunc = func(ctx context.Context, hostID uint, encryptedBase64Key string) error {
		require.Equal(t, wantKey, encryptedBase64Key)
		require.Equal(t, host.ID, hostID)
		return nil
	}

	err = directIngestDiskEncryptionKeyDarwin(ctx, logger, host, ds, []map[string]string{{"filevault_key": wantKey}})
	require.NoError(t, err)
	require.True(t, ds.SetOrUpdateHostDiskEncryptionKeyFuncInvoked)
}
