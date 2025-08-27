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
		{"AndroidHostStorageData", testAndroidHostStorageData},
		{"NewMDMAndroidConfigProfile", testNewMDMAndroidConfigProfile},
		{"GetMDMAndroidConfigProfile", testGetMDMAndroidConfigProfile},
		{"DeleteMDMAndroidConfigProfile", testDeleteMDMAndroidConfigProfile},
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
			LastPolicySyncTime:   ptr.Time(time.Now().UTC().Truncate(time.Millisecond)),
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

func testAndroidHostStorageData(t *testing.T, ds *Datastore) {
	test.AddBuiltinLabels(t, ds)

	// Android host with storage data
	const enterpriseSpecificID = "storage_test_enterprise"
	host := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname:                  "android-storage-test",
			ComputerName:              "Android Storage Test Device",
			Platform:                  "android",
			OSVersion:                 "Android 14",
			Build:                     "UPB4.230623.005",
			Memory:                    8192, // 8GB RAM
			TeamID:                    nil,
			HardwareSerial:            "STORAGE-TEST-SERIAL",
			CPUType:                   "arm64-v8a",
			HardwareModel:             "Google Pixel 8 Pro",
			HardwareVendor:            "Google",
			GigsTotalDiskSpace:        128.0, // 64GB system + 64GB external
			GigsDiskSpaceAvailable:    35.0,  // 10GB + 25GB available
			PercentDiskSpaceAvailable: 27.34, // 35/128 * 100
		},
		Device: &android.Device{
			DeviceID:             "storage-test-device-id",
			EnterpriseSpecificID: ptr.String(enterpriseSpecificID),
			AndroidPolicyID:      ptr.Uint(1),
			LastPolicySyncTime:   ptr.Time(time.Now().UTC().Truncate(time.Millisecond)),
		},
	}
	host.SetNodeKey(enterpriseSpecificID)

	// NewAndroidHost with storage data
	result, err := ds.NewAndroidHost(testCtx(), host)
	require.NoError(t, err)
	require.NotZero(t, result.Host.ID)

	// storage data was saved correctly
	assert.Equal(t, 128.0, result.Host.GigsTotalDiskSpace, "Total disk space should be saved")
	assert.Equal(t, 35.0, result.Host.GigsDiskSpaceAvailable, "Available disk space should be saved")
	assert.Equal(t, 27.34, result.Host.PercentDiskSpaceAvailable, "Disk space percentage should be saved")

	// AndroidHostLite provides lightweight Android data (no storage data)
	resultLite, err := ds.AndroidHostLite(testCtx(), enterpriseSpecificID)
	require.NoError(t, err)
	assert.Equal(t, result.Host.ID, resultLite.Host.ID)

	// UpdateAndroidHost preserves storage data
	updatedHost := result
	updatedHost.Host.Hostname = "updated-hostname"
	updatedHost.Host.GigsTotalDiskSpace = 256.0       // Updated: 128GB system + 128GB external
	updatedHost.Host.GigsDiskSpaceAvailable = 64.0    // Updated: 20GB + 44GB available
	updatedHost.Host.PercentDiskSpaceAvailable = 25.0 // Updated: 64/256 * 100

	err = ds.UpdateAndroidHost(testCtx(), updatedHost, false)
	require.NoError(t, err)

	// verify updated host data via host query (includes storage from host_disks)
	finalResult, err := ds.AndroidHostLite(testCtx(), enterpriseSpecificID)
	require.NoError(t, err)

	// get host data to check storage updates
	updatedFullHost, err := ds.Host(testCtx(), finalResult.Host.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-hostname", updatedFullHost.Hostname, "Hostname should be updated")
	assert.Equal(t, 256.0, updatedFullHost.GigsTotalDiskSpace, "Updated total disk space should be saved in host_disks")
	assert.Equal(t, 64.0, updatedFullHost.GigsDiskSpaceAvailable, "Updated available disk space should be saved in host_disks")
	assert.Equal(t, 25.0, updatedFullHost.PercentDiskSpaceAvailable, "Updated disk space percentage should be saved in host_disks")
}

func testNewMDMAndroidConfigProfile(t *testing.T, ds *Datastore) {
	test.AddBuiltinLabels(t, ds)
	ctx := testCtx()

	// create some labels to test
	lblExcl, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-label-1", Query: "select 1"})
	require.NoError(t, err)
	lblInclAny, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-label-2", Query: "select 2"})
	require.NoError(t, err)
	lblInclAll, err := ds.NewLabel(ctx, &fleet.Label{Name: "inclall-label-3", Query: "select 3"})
	require.NoError(t, err)

	// New Android MDM config profile
	profile := fleet.MDMAndroidConfigProfile{
		Name:    "testAndroid",
		TeamID:  nil,
		RawJSON: []byte(`{"hello": "world"}`),
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{{
			LabelID:    lblInclAll.ID,
			LabelName:  lblInclAll.Name,
			RequireAll: true,
		}},
		LabelsIncludeAny: []fleet.ConfigurationProfileLabel{{
			LabelID:    lblInclAny.ID,
			LabelName:  lblInclAny.Name,
			RequireAll: false,
		}},
		LabelsExcludeAny: []fleet.ConfigurationProfileLabel{{
			LabelID:    lblExcl.ID,
			LabelName:  lblExcl.Name,
			RequireAll: false,
			Exclude:    true,
		}},
	}

	// Create the profile
	result, err := ds.NewMDMAndroidConfigProfile(ctx, profile)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ProfileUUID)

	// Create another profile just to have multiple entries
	profile2 := fleet.MDMAndroidConfigProfile{
		Name:    "testAndroid2",
		TeamID:  nil,
		RawJSON: []byte(`{"hello2": "world2"}`),
	}
	result2, err := ds.NewMDMAndroidConfigProfile(ctx, profile2)
	require.NoError(t, err)
	assert.NotEmpty(t, result2.ProfileUUID)

	returnedProfile, err := ds.GetMDMAndroidConfigProfile(ctx, result.ProfileUUID)
	require.NoError(t, err)
	require.NotNil(t, returnedProfile)

	// Verify the profile was created correctly
	assert.Equal(t, profile.RawJSON, returnedProfile.RawJSON)
	assert.Equal(t, profile.Name, returnedProfile.Name)
	require.NotNil(t, returnedProfile.TeamID)
	assert.Equal(t, uint(0), *returnedProfile.TeamID)
	require.ElementsMatch(t, profile.LabelsIncludeAll, returnedProfile.LabelsIncludeAll)
	require.ElementsMatch(t, profile.LabelsIncludeAny, returnedProfile.LabelsIncludeAny)
	require.ElementsMatch(t, profile.LabelsExcludeAny, returnedProfile.LabelsExcludeAny)

	// Create a Windows profile with a name, then make sure an error is returned when creating an
	// Android profile with that name
	windowsProfile := fleet.MDMWindowsConfigProfile{
		Name:   "testWindowsAndroidConflict",
		TeamID: nil,
		SyncML: []byte(`hello`),
	}
	_, err = ds.NewMDMWindowsConfigProfile(ctx, windowsProfile, nil)
	require.NoError(t, err)

	androidProfile := fleet.MDMAndroidConfigProfile{
		Name:    "testWindowsAndroidConflict",
		TeamID:  nil,
		RawJSON: []byte(`{"hello3": "world3"}`),
	}
	_, err = ds.NewMDMAndroidConfigProfile(ctx, androidProfile)
	require.ErrorContains(t, err, "already exists")

	// Create that same conflicting android profile but on a different team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "test team"})
	require.NoError(t, err)
	require.NotNil(t, team)
	androidProfile.TeamID = ptr.Uint(team.ID)
	otherTeamProfile, err := ds.NewMDMAndroidConfigProfile(ctx, androidProfile)
	require.NoError(t, err)

	// Verify we can GET the newly created profile
	otherTeamProfile, err = ds.GetMDMAndroidConfigProfile(ctx, otherTeamProfile.ProfileUUID)
	require.NoError(t, err)
	require.NotNil(t, otherTeamProfile)
	assert.Equal(t, androidProfile.RawJSON, otherTeamProfile.RawJSON)
	assert.Equal(t, androidProfile.Name, otherTeamProfile.Name)
	require.NotNil(t, otherTeamProfile.TeamID)
	assert.Equal(t, *androidProfile.TeamID, *otherTeamProfile.TeamID)
}

func testGetMDMAndroidConfigProfile(t *testing.T, ds *Datastore) {
	ctx := testCtx()
	profile, err := ds.GetMDMAndroidConfigProfile(ctx, "some-fake-uuid")
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, profile)
}

func testDeleteMDMAndroidConfigProfile(t *testing.T, ds *Datastore) {
	ctx := testCtx()
	err := ds.DeleteMDMAndroidConfigProfile(ctx, "some-fake-uuid")
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	profile1 := &fleet.MDMAndroidConfigProfile{
		Name:    "testAndroid",
		TeamID:  nil,
		RawJSON: []byte(`{"hello": "world"}`),
	}

	profile1, err = ds.NewMDMAndroidConfigProfile(ctx, *profile1)
	require.NoError(t, err)
	require.NotNil(t, profile1)

	profile2 := &fleet.MDMAndroidConfigProfile{
		Name:    "testAndroid2",
		TeamID:  nil,
		RawJSON: []byte(`{"hello": "world"}`),
	}
	profile2, err = ds.NewMDMAndroidConfigProfile(ctx, *profile2)
	require.NoError(t, err)
	require.NotNil(t, profile2)

	// Delete the first profile
	err = ds.DeleteMDMAndroidConfigProfile(ctx, profile1.ProfileUUID)
	require.NoError(t, err)

	// Verify the first profile is deleted
	profile1, err = ds.GetMDMAndroidConfigProfile(ctx, profile1.ProfileUUID)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, profile1)

	// Verify the second profile is untouched
	profile2, err = ds.GetMDMAndroidConfigProfile(ctx, profile2.ProfileUUID)
	require.NoError(t, err)
	require.NotNil(t, profile2)
	require.Equal(t, "testAndroid2", profile2.Name)
}
