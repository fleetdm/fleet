package mysql

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func slowStats(t *testing.T, ds *Datastore, id uint, percentile int, column string) float64 {
	queriesSQL := fmt.Sprintf(
		`
		SELECT SUM(d.%[1]s) / SUM(d.executions)
		FROM scheduled_query_stats d
			JOIN queries q ON (d.scheduled_query_id=q.id)
		WHERE q.id=? AND d.executions > 0
		GROUP BY d.host_id
		ORDER BY (SUM(d.%[1]s) / SUM(d.executions))`, column,
	)
	rows, err := ds.writer(context.Background()).Queryx(queriesSQL, id)
	require.NoError(t, err)
	defer rows.Close()

	var vals []float64

	for rows.Next() {
		var val float64
		err := rows.Scan(&val)
		require.NoError(t, err)
		vals = append(vals, val)
	}

	if len(vals) == 0 {
		return 0.0
	}

	index := int(math.Floor(float64(len(vals)) * float64(percentile) / 100.0))
	return vals[index]
}

func TestAggregatedStats(t *testing.T) {
	ds := CreateMySQLDS(t)

	var args []interface{}

	batchSize := 4000
	hostCount := 10           // 2000
	scheduledQueryCount := 20 // 400
	queryCount := 30          // 1000

	start := time.Now()
	for i := 0; i < queryCount; i++ {
		_, err := ds.writer(context.Background()).Exec(`INSERT INTO queries(name, query, description) VALUES (?,?,?)`, fmt.Sprint(i), fmt.Sprint(i), fmt.Sprint(i))
		require.NoError(t, err)
	}
	for i := 0; i < scheduledQueryCount; i++ {
		_, err := ds.writer(context.Background()).Exec(`INSERT INTO scheduled_queries(query_id, name, query_name) VALUES (?,?,?)`, rand.Intn(queryCount)+1, fmt.Sprint(i), fmt.Sprint(i))
		require.NoError(t, err)
	}
	insertScheduledQuerySQL := `INSERT IGNORE INTO scheduled_query_stats(host_id, scheduled_query_id, system_time, user_time, executions, query_type) VALUES %s`
	scheduledQueryStatsCount := 100 // 1000000
	for i := 0; i < scheduledQueryStatsCount; i++ {
		if len(args) > batchSize {
			values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?),", len(args)/6), ",")
			_, err := ds.writer(context.Background()).Exec(fmt.Sprintf(insertScheduledQuerySQL, values), args...)
			require.NoError(t, err)
			args = []interface{}{}
		}
		// Occasionally set 0 executions
		executions := rand.Intn(10000) + 100
		if rand.Intn(100) < 5 {
			executions = 0
		}
		args = append(
			args, rand.Intn(hostCount)+1, rand.Intn(queryCount)+1, rand.Intn(10000)+100, rand.Intn(10000)+100, executions,
			rand.Intn(2),
		)
	}
	if len(args) > 0 {
		values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?),", len(args)/6), ",")
		_, err := ds.writer(context.Background()).Exec(fmt.Sprintf(insertScheduledQuerySQL, values), args...)
		require.NoError(t, err)
	}

	// Make sure we have some queries and scheduled queries that don't have stats
	for i := queryCount; i < queryCount+4; i++ {
		_, err := ds.writer(context.Background()).Exec(`INSERT INTO queries(name, query, description) VALUES (?,?,?)`, fmt.Sprint(i), fmt.Sprint(i), fmt.Sprint(i))
		require.NoError(t, err)
	}
	for i := scheduledQueryCount; i < scheduledQueryCount+4; i++ {
		_, err := ds.writer(context.Background()).Exec(`INSERT INTO scheduled_queries(query_id, name, query_name) VALUES (?,?,?)`, rand.Intn(queryCount)+1, fmt.Sprint(i), fmt.Sprint(i))
		require.NoError(t, err)
	}

	t.Log("Done inserting dummy data. Took:", time.Since(start))

	testcases := []struct {
		table     string
		aggregate fleet.AggregatedStatsType
		aggFunc   func(ctx context.Context) error
	}{
		{"queries", fleet.AggregatedStatsTypeScheduledQuery, ds.UpdateQueryAggregatedStats},
	}
	for _, tt := range testcases {
		t.Run(tt.table, func(t *testing.T) {
			start = time.Now()
			require.NoError(t, tt.aggFunc(context.Background()))
			t.Log("Generated stats for ", tt.table, " in:,", time.Since(start))

			var stats []struct {
				ID          uint `db:"id"`
				GlobalStats bool `db:"global_stats"`
				fleet.AggregatedStats
			}
			require.NoError(t,
				ds.writer(context.Background()).Select(&stats,
					`
select
       id,
	   global_stats,
       JSON_EXTRACT(json_value, '$.user_time_p50') as user_time_p50,
       JSON_EXTRACT(json_value, '$.user_time_p95') as user_time_p95,
       JSON_EXTRACT(json_value, '$.system_time_p50') as system_time_p50,
       JSON_EXTRACT(json_value, '$.system_time_p95') as system_time_p95,
       JSON_EXTRACT(json_value, '$.total_executions') as total_executions
from aggregated_stats where type=?`, tt.aggregate))

			require.True(t, len(stats) > 0)
			for _, stat := range stats {
				require.False(t, stat.GlobalStats)
				checkAgainstSlowStats(t, ds, stat.ID, 50, "user_time", stat.UserTimeP50)
				checkAgainstSlowStats(t, ds, stat.ID, 95, "user_time", stat.UserTimeP95)
				checkAgainstSlowStats(t, ds, stat.ID, 50, "system_time", stat.SystemTimeP50)
				checkAgainstSlowStats(t, ds, stat.ID, 95, "system_time", stat.SystemTimeP95)
				require.NotNil(t, stat.TotalExecutions)
				assert.True(t, *stat.TotalExecutions >= 0)
			}
		})
	}
}

func checkAgainstSlowStats(t *testing.T, ds *Datastore, id uint, percentile int, column string, against *float64) {
	slowp := slowStats(t, ds, id, percentile, column)
	if against != nil {
		assert.Equal(t, slowp, *against)
	} else {
		assert.Zero(t, slowp)
	}
}

func TestAndroidMDMStats(t *testing.T) {
	ds := CreateMySQLDS(t)
	const appleMDMURL = "/mdm/apple/mdm"
	const serverURL = "http://androidmdm.example.com"

	testCtx := func() context.Context {
		return t.Context()
	}
	appCfg, err := ds.AppConfig(testCtx())
	require.NoError(t, err)
	appCfg.ServerSettings.ServerURL = serverURL
	err = ds.SaveAppConfig(testCtx(), appCfg)
	require.NoError(t, err)

	// create a few android hosts
	hosts := make([]*fleet.Host, 3)
	var androidHost0 *android.Host
	for i := range hosts {
		host := createAndroidHost(uuid.NewString())
		result, err := ds.NewAndroidHost(testCtx(), serverURL, host)
		require.NoError(t, err)
		hosts[i] = &fleet.Host{
			ID:              result.ID,
			TeamID:          result.TeamID,
			OSVersion:       result.OSVersion,
			Build:           result.Build,
			Memory:          result.Memory,
			HardwareSerial:  result.HardwareSerial,
			CPUType:         result.CPUType,
			HardwareModel:   result.HardwareModel,
			HardwareVendor:  result.HardwareVendor,
			DetailUpdatedAt: result.DetailUpdatedAt,
			LabelUpdatedAt:  result.LabelUpdatedAt,
		}

		if androidHost0 == nil {
			androidHost0 = host
		}
	}

	// create a non-android host
	macHost, err := ds.NewHost(testCtx(), &fleet.Host{
		Hostname:       "test-host1-name",
		OsqueryHostID:  ptr.String("1337"),
		NodeKey:        ptr.String("1337"),
		UUID:           "test-uuid-1",
		Platform:       "darwin",
		HardwareSerial: uuid.NewString(),
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, macHost, false)
	err = ds.MDMAppleUpsertHost(testCtx(), macHost)
	require.NoError(t, err)

	// create a non-mdm host
	linuxHost, err := ds.NewHost(testCtx(), &fleet.Host{
		Hostname:       "test-host2-name",
		OsqueryHostID:  ptr.String("1338"),
		NodeKey:        ptr.String("1338"),
		UUID:           "test-uuid-2",
		Platform:       "linux",
		HardwareSerial: uuid.NewString(),
	})
	require.NoError(t, err)
	require.NotNil(t, linuxHost)

	// stats not computed yet
	statusStats, _, err := ds.AggregatedMDMStatus(testCtx(), nil, "")
	require.NoError(t, err)
	solutionsStats, _, err := ds.AggregatedMDMSolutions(testCtx(), nil, "")
	require.NoError(t, err)
	require.Equal(t, fleet.AggregatedMDMStatus{}, statusStats)
	require.Equal(t, []fleet.AggregatedMDMSolutions(nil), solutionsStats)

	// compute stats
	err = ds.GenerateAggregatedMunkiAndMDM(testCtx())
	require.NoError(t, err)

	statusStats, _, err = ds.AggregatedMDMStatus(testCtx(), nil, "")
	require.NoError(t, err)
	solutionsStats, _, err = ds.AggregatedMDMSolutions(testCtx(), nil, "")
	require.NoError(t, err)
	require.Equal(t, fleet.AggregatedMDMStatus{HostsCount: 4, EnrolledManualHostsCount: 4}, statusStats)
	require.Len(t, solutionsStats, 2)

	// both solutions are Fleet
	require.Equal(t, fleet.WellKnownMDMFleet, solutionsStats[0].Name)
	require.Equal(t, fleet.WellKnownMDMFleet, solutionsStats[1].Name)

	// one is the Android server URL, one is the Apple URL
	for _, sol := range solutionsStats {
		switch sol.ServerURL {
		case serverURL:
			require.Equal(t, 3, sol.HostsCount)
		case serverURL + appleMDMURL:
			require.Equal(t, 1, sol.HostsCount)
		default:
			require.Failf(t, "unexpected server URL: %v", sol.ServerURL)
		}
	}

	// filter on android
	statusStats, _, err = ds.AggregatedMDMStatus(testCtx(), nil, "android")
	require.NoError(t, err)
	solutionsStats, _, err = ds.AggregatedMDMSolutions(testCtx(), nil, "android")
	require.NoError(t, err)
	require.Equal(t, fleet.AggregatedMDMStatus{HostsCount: 3, EnrolledManualHostsCount: 3}, statusStats)
	require.Len(t, solutionsStats, 1)
	require.Equal(t, 3, solutionsStats[0].HostsCount)
	require.Equal(t, serverURL, solutionsStats[0].ServerURL)

	// turn MDM off for android
	err = ds.DeleteAllEnterprises(testCtx())
	require.NoError(t, err)
	err = ds.BulkSetAndroidHostsUnenrolled(testCtx())
	require.NoError(t, err)

	// compute stats
	err = ds.GenerateAggregatedMunkiAndMDM(testCtx())
	require.NoError(t, err)

	statusStats, _, err = ds.AggregatedMDMStatus(testCtx(), nil, "")
	require.NoError(t, err)
	solutionsStats, _, err = ds.AggregatedMDMSolutions(testCtx(), nil, "")
	require.NoError(t, err)
	require.Equal(t, fleet.AggregatedMDMStatus{HostsCount: 4, EnrolledManualHostsCount: 1, UnenrolledHostsCount: 3}, statusStats)
	require.Len(t, solutionsStats, 1)
	require.Equal(t, 1, solutionsStats[0].HostsCount)
	require.Equal(t, serverURL+appleMDMURL, solutionsStats[0].ServerURL)

	// filter on android
	statusStats, _, err = ds.AggregatedMDMStatus(testCtx(), nil, "android")
	require.NoError(t, err)
	solutionsStats, _, err = ds.AggregatedMDMSolutions(testCtx(), nil, "android")
	require.NoError(t, err)
	require.Equal(t, fleet.AggregatedMDMStatus{HostsCount: 3, UnenrolledHostsCount: 3}, statusStats)
	require.Len(t, solutionsStats, 0)

	// simulate an android host that re-enrolls
	err = ds.UpdateAndroidHost(testCtx(), serverURL, androidHost0, true)
	require.NoError(t, err)

	// compute stats
	err = ds.GenerateAggregatedMunkiAndMDM(testCtx())
	require.NoError(t, err)

	// filter on android
	statusStats, _, err = ds.AggregatedMDMStatus(testCtx(), nil, "android")
	require.NoError(t, err)
	solutionsStats, _, err = ds.AggregatedMDMSolutions(testCtx(), nil, "android")
	require.NoError(t, err)
	require.Equal(t, fleet.AggregatedMDMStatus{HostsCount: 3, UnenrolledHostsCount: 2, EnrolledManualHostsCount: 1}, statusStats)
	require.Len(t, solutionsStats, 1)
	require.Equal(t, 1, solutionsStats[0].HostsCount)
	require.Equal(t, serverURL, solutionsStats[0].ServerURL)
}

func createAndroidHost(enterpriseSpecificID string) *android.Host {
	host := &android.Host{
		OSVersion:      "Android 14",
		Build:          "build",
		Memory:         1024,
		TeamID:         nil,
		HardwareSerial: "hardware_serial",
		CPUType:        "cpu_type",
		HardwareModel:  "hardware_model",
		HardwareVendor: "hardware_vendor",
		Device: &android.Device{
			DeviceID:             "device_id",
			EnterpriseSpecificID: ptr.String(enterpriseSpecificID),
			AndroidPolicyID:      ptr.Uint(1),
			LastPolicySyncTime:   ptr.Time(time.Time{}),
		},
	}
	host.SetNodeKey(enterpriseSpecificID)
	return host
}
