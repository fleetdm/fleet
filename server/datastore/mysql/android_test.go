package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAndroid(t *testing.T) {
	ds := CreateMySQLDS(t)
	TruncateTables(t, ds)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"NewAndroidHost", testNewAndroidHost},
		{"UpdateAndroidHost", testUpdateAndroidHost},
		{"AndroidMDMStats", testAndroidMDMStats},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testNewAndroidHost(t *testing.T, ds *Datastore) {
	test.AddBuiltinLabels(t, ds)

	const enterpriseSpecificID = "enterprise_specific_id"
	host := createAndroidHost(enterpriseSpecificID)

	result, err := ds.NewAndroidHost(testCtx(), host)
	require.NoError(t, err)
	assert.NotZero(t, result.Host.ID)
	assert.NotZero(t, result.Device.ID)

	lbls, err := ds.ListLabelsForHost(testCtx(), result.Host.ID)
	require.NoError(t, err)
	require.Len(t, lbls, 2)
	names := []string{lbls[0].Name, lbls[1].Name}
	require.ElementsMatch(t, []string{fleet.BuiltinLabelNameAllHosts, fleet.BuiltinLabelNameAndroid}, names)

	resultLite, err := ds.AndroidHostLite(testCtx(), enterpriseSpecificID)
	require.NoError(t, err)
	assert.Equal(t, result.Host.ID, resultLite.Host.ID)
	assert.Equal(t, result.Device.ID, resultLite.Device.ID)

	// Inserting the same host again should be fine.
	// This may occur when 2 Fleet servers received the same host information via pubsub.
	resultCopy, err := ds.NewAndroidHost(testCtx(), host)
	require.NoError(t, err)
	assert.Equal(t, result.Host.ID, resultCopy.Host.ID)
	assert.Equal(t, result.Device.ID, resultCopy.Device.ID)

	// create another host, this time delete the Android label
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(testCtx(), `DELETE FROM labels WHERE name = ?`, fleet.BuiltinLabelNameAndroid)
		return err
	})
	const enterpriseSpecificID2 = "enterprise_specific_id2"
	host2 := createAndroidHost(enterpriseSpecificID2)

	// still passes, but no label membership was recorded
	result, err = ds.NewAndroidHost(testCtx(), host2)
	require.NoError(t, err)

	lbls, err = ds.ListLabelsForHost(testCtx(), result.Host.ID)
	require.NoError(t, err)
	require.Empty(t, lbls)
}

func createAndroidHost(enterpriseSpecificID string) *fleet.AndroidHost {
	host := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname:       "hostname",
			ComputerName:   "computer_name",
			Platform:       "android",
			OSVersion:      "Android 14",
			Build:          "build",
			Memory:         1024,
			TeamID:         nil,
			HardwareSerial: "hardware_serial",
			CPUType:        "cpu_type",
			HardwareModel:  "hardware_model",
			HardwareVendor: "hardware_vendor",
		},
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

func testCtx() context.Context {
	return context.Background()
}

func testUpdateAndroidHost(t *testing.T, ds *Datastore) {
	const enterpriseSpecificID = "es_id_update"
	host := createAndroidHost(enterpriseSpecificID)

	result, err := ds.NewAndroidHost(testCtx(), host)
	require.NoError(t, err)
	assert.NotZero(t, result.Host.ID)
	assert.NotZero(t, result.Device.ID)

	// Dummy update
	err = ds.UpdateAndroidHost(testCtx(), result, false)
	require.NoError(t, err)

	host = result
	host.Host.DetailUpdatedAt = time.Now()
	host.Host.LabelUpdatedAt = time.Now()
	host.Host.Hostname = "hostname_updated"
	host.Host.ComputerName = "computer_name_updated"
	host.Host.Platform = "android_updated"
	host.Host.OSVersion = "Android 15"
	host.Host.Build = "build_updated"
	host.Host.Memory = 2048
	host.Host.HardwareSerial = "hardware_serial_updated"
	host.Host.CPUType = "cpu_type_updated"
	host.Host.HardwareModel = "hardware_model_updated"
	host.Host.HardwareVendor = "hardware_vendor_updated"
	host.Device.AndroidPolicyID = ptr.Uint(2)
	err = ds.UpdateAndroidHost(testCtx(), host, false)
	require.NoError(t, err)

	resultLite, err := ds.AndroidHostLite(testCtx(), enterpriseSpecificID)
	require.NoError(t, err)
	assert.Equal(t, host.Host.ID, resultLite.Host.ID)
	assert.EqualValues(t, host.Device, resultLite.Device)
}

func testAndroidMDMStats(t *testing.T, ds *Datastore) {
	const appleMDMURL = "/mdm/apple/mdm"
	const serverURL = "http://androidmdm.example.com"

	appCfg, err := ds.AppConfig(testCtx())
	require.NoError(t, err)
	appCfg.ServerSettings.ServerURL = serverURL
	err = ds.SaveAppConfig(testCtx(), appCfg)
	require.NoError(t, err)

	// create a few android hosts
	hosts := make([]*fleet.Host, 3)
	var androidHost0 *fleet.AndroidHost
	for i := range hosts {
		host := createAndroidHost(uuid.NewString())
		result, err := ds.NewAndroidHost(testCtx(), host)
		require.NoError(t, err)
		hosts[i] = result.Host

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
	err = ds.MDMAppleUpsertHost(testCtx(), macHost, false)
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
	err = ds.UpdateAndroidHost(testCtx(), androidHost0, true)
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
