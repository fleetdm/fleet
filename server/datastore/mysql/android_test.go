package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
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
		{"GetMDMAndroidProfilesSummary", testMDMAndroidProfilesSummary},
		{"ListMDMAndroidProfilesToSend", testListMDMAndroidProfilesToSend},
		{"GetMDMAndroidProfilesContents", testGetMDMAndroidProfilesContents},
		{"BulkUpsertMDMAndroidHostProfiles", testBulkUpsertMDMAndroidHostProfiles},
		{"BulkUpsertMDMAndroidHostProfiles", testBulkUpsertMDMAndroidHostProfiles2},
		{"BulkUpsertMDMAndroidHostProfiles", testBulkUpsertMDMAndroidHostProfiles3},
		{"GetHostMDMAndroidProfiles", testGetHostMDMAndroidProfiles},
		{"GetAndroidPolicyRequestByUUID", testGetAndroidPolicyRequestByUUID},
		{"ListHostMDMAndroidProfilesPendingInstallWithVersion", testListHostMDMAndroidProfilesPendingInstallWithVersion},
		{"BulkDeleteMDMAndroidHostProfiles", testBulkDeleteMDMAndroidHostProfiles},
		{"BatchSetMDMAndroidProfiles_Associations", testBatchSetMDMAndroidProfiles_Associations},
		{"NewAndroidHostWithIdP", testNewAndroidHostWithIdP},
		{"AndroidBYODDetection", testAndroidBYODDetection},
		{"SetAndroidHostUnenrolled", testSetAndroidHostUnenrolled},
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

	resultLite, err = ds.AndroidHostLiteByHostUUID(testCtx(), result.Host.UUID)
	require.NoError(t, err)
	assert.Equal(t, result.Host.ID, resultLite.Host.ID)
	assert.Equal(t, result.Device.ID, resultLite.Device.ID)

	_, err = ds.AndroidHostLite(testCtx(), "non-existent")
	require.Error(t, err)
	_, err = ds.AndroidHostLiteByHostUUID(testCtx(), "no-such-host")
	require.Error(t, err)

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
	// Device ID needs to be unique per device
	deviceID := md5ChecksumBytes([]byte(enterpriseSpecificID))[:16]
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
			UUID:           enterpriseSpecificID,
		},
		Device: &android.Device{
			DeviceID:             deviceID,
			EnterpriseSpecificID: ptr.String(enterpriseSpecificID),
			AppliedPolicyID:      ptr.String("1"),
			AppliedPolicyVersion: ptr.Int64(1),
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
	host.Device.AppliedPolicyID = ptr.String("2")

	// Make sure host UUID is preserved during update
	host.Host.UUID = enterpriseSpecificID

	err = ds.UpdateAndroidHost(testCtx(), host, false)
	require.NoError(t, err)

	resultLite, err := ds.AndroidHostLite(testCtx(), enterpriseSpecificID)
	require.NoError(t, err)
	assert.Equal(t, host.Host.ID, resultLite.Host.ID)
	assert.EqualValues(t, host.Device, resultLite.Device)

	// Make sure UUID was preserved after update
	assert.Equal(t, enterpriseSpecificID, resultLite.Host.UUID, "UUID should be preserved after UpdateAndroidHost")

	// Regression: empty UUID doesn't corrupt existing data
	// This simulates a scenario where updateHost might not set UUID, resulting in empty value
	t.Run("Empty UUID regression test", func(t *testing.T) {
		const regressionESID = "regression-uuid-test"
		regressionHost := createAndroidHost(regressionESID)
		createdHost, err := ds.NewAndroidHost(testCtx(), regressionHost)
		require.NoError(t, err)
		require.Equal(t, regressionESID, createdHost.Host.UUID)

		// Simulate update where UUID might be accidentally cleared
		hostWithEmptyUUID := createdHost
		hostWithEmptyUUID.Host.UUID = ""
		hostWithEmptyUUID.Host.Hostname = "regression-hostname"

		// This should still work but UUID should be empty
		err = ds.UpdateAndroidHost(testCtx(), hostWithEmptyUUID, false)
		require.NoError(t, err)

		// UUID is now empty
		resultAfterBug, err := ds.AndroidHostLite(testCtx(), regressionESID)
		require.NoError(t, err)
		assert.Equal(t, "", resultAfterBug.Host.UUID, "UUID should be empty after update without UUID set (documents the bug)")

		// Update with UUID properly set
		hostWithUUID := resultAfterBug
		hostWithUUID.Host.UUID = regressionESID
		hostWithUUID.Host.Hostname = "fixed-hostname"

		err = ds.UpdateAndroidHost(testCtx(), hostWithUUID, false)
		require.NoError(t, err)

		// UUID is restored
		resultAfterFix, err := ds.AndroidHostLite(testCtx(), regressionESID)
		require.NoError(t, err)
		assert.Equal(t, regressionESID, resultAfterFix.Host.UUID, "UUID should be restored after fix")
	})
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
	// 3 Android hosts with UUID are counted as personal enrollment, 1 macOS host as manual
	require.Equal(t, fleet.AggregatedMDMStatus{HostsCount: 4, EnrolledManualHostsCount: 1, EnrolledPersonalHostsCount: 3}, statusStats)
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
	// All 3 Android hosts with UUID are counted as personal enrollment
	require.Equal(t, fleet.AggregatedMDMStatus{HostsCount: 3, EnrolledPersonalHostsCount: 3}, statusStats)
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
	// After re-enrollment, 1 Android host with UUID is counted as personal enrollment
	require.Equal(t, fleet.AggregatedMDMStatus{HostsCount: 3, UnenrolledHostsCount: 2, EnrolledPersonalHostsCount: 1}, statusStats)
	require.Len(t, solutionsStats, 1)
	require.Equal(t, 1, solutionsStats[0].HostsCount)
	require.Equal(t, serverURL, solutionsStats[0].ServerURL)
}

// Test that BatchSetMDMProfiles properly inserts Android profiles when the
// incoming profiles have empty ProfileUUIDs and still applies label
// associations (i.e. matching by team_id + name works).
func testBatchSetMDMAndroidProfiles_Associations(t *testing.T, ds *Datastore) {
	// Ensure builtin labels exist
	test.AddBuiltinLabels(t, ds)

	// Prepare an incoming Android profile without ProfileUUID and with a label
	teamID := uint(0)
	profName := "test-android-profile"
	incoming := &fleet.MDMAndroidConfigProfile{
		ProfileUUID: "", // intentionally empty to exercise DB-generated uuid flow
		Name:        profName,
		RawJSON:     json.RawMessage(`{"k":"v"}`),
		TeamID:      nil,
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{{
			LabelName: fleet.BuiltinLabelNameAndroid,
		}},
	}

	// Look up the builtin Android label id and set it on the incoming profile
	var lblID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(testCtx(), q, &lblID, `SELECT id FROM labels WHERE name = ?`, fleet.BuiltinLabelNameAndroid)
	})
	// assign the id so the label association insertion uses a valid FK
	if len(incoming.LabelsIncludeAll) > 0 {
		incoming.LabelsIncludeAll[0].LabelID = lblID
	}

	// Call BatchSetMDMProfiles with only android profiles populated
	_, err := ds.BatchSetMDMProfiles(testCtx(), &teamID, nil, nil, nil, []*fleet.MDMAndroidConfigProfile{incoming}, nil)
	require.NoError(t, err)

	// Verify the profile exists in the DB
	var dbCount int
	err = sqlx.GetContext(testCtx(), ds.writer(testCtx()), &dbCount, `SELECT COUNT(1) FROM mdm_android_configuration_profiles WHERE name = ? AND team_id = ?`, profName, teamID)
	require.NoError(t, err)
	assert.Equal(t, 1, dbCount)

	// Verify that a label association was created for the profile by querying
	// mdm_configuration_profile_labels joined to mdm_android_configuration_profiles
	var assocCount int
	query := `SELECT COUNT(1) FROM mdm_configuration_profile_labels l JOIN mdm_android_configuration_profiles p ON l.android_profile_uuid = p.profile_uuid WHERE p.name = ? AND p.team_id = ? AND l.label_name = ?`
	err = sqlx.GetContext(testCtx(), ds.writer(testCtx()), &assocCount, query, profName, teamID, fleet.BuiltinLabelNameAndroid)
	require.NoError(t, err)
	assert.Equal(t, 1, assocCount, "expected a label association for the inserted android profile")
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
			AppliedPolicyID:      ptr.String("1"),
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

	// set a host profile to mimic reconcilation has yet to run

	err = ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
		{
			HostUUID:      "test-host-1",
			ProfileUUID:   profile1.ProfileUUID,
			Status:        nil,
			OperationType: fleet.MDMOperationTypeInstall,
		},
		{
			HostUUID:      "test-host-2",
			ProfileUUID:   profile2.ProfileUUID,
			Status:        &fleet.MDMDeliveryPending,
			OperationType: fleet.MDMOperationTypeInstall,
		},
	})
	require.NoError(t, err)

	// Delete the first profile
	err = ds.DeleteMDMAndroidConfigProfile(ctx, profile1.ProfileUUID)
	require.NoError(t, err)

	// Verify the first profile is deleted and respective host profile is cancelled
	profile1, err = ds.GetMDMAndroidConfigProfile(ctx, profile1.ProfileUUID)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, profile1)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		stmt := `SELECT host_uuid, profile_uuid FROM host_mdm_android_profiles`
		var hosts []struct {
			HostUUID    string `db:"host_uuid"`
			ProfileUUID string `db:"profile_uuid"`
		}
		err := sqlx.SelectContext(ctx, q, &hosts, stmt)
		if err != nil {
			return err
		}

		require.NoError(t, err)
		require.Len(t, hosts, 1)
		require.Equal(t, "test-host-2", hosts[0].HostUUID)
		return nil
	})

	// Verify the second profile is untouched
	profile2, err = ds.GetMDMAndroidConfigProfile(ctx, profile2.ProfileUUID)
	require.NoError(t, err)
	require.NotNil(t, profile2)
	require.Equal(t, "testAndroid2", profile2.Name)
}

func testMDMAndroidProfilesSummary(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	checkMDMProfilesSummary := func(t *testing.T, teamID *uint, expected fleet.MDMProfilesSummary) {
		ps, err := ds.GetMDMAndroidProfilesSummary(ctx, teamID)
		require.NoError(t, err)
		require.NotNil(t, ps)
		require.Equal(t, expected, *ps)
	}

	checkListHostsFilterOSSettings := func(t *testing.T, teamID *uint, status fleet.OSSettingsStatus, expectedIDs []uint) {
		gotHosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{TeamFilter: teamID, OSSettingsFilter: status})
		require.NoError(t, err)
		if len(expectedIDs) != len(gotHosts) {
			gotIDs := make([]uint, len(gotHosts))
			for i, h := range gotHosts {
				gotIDs[i] = h.ID
			}
			require.Len(t, gotHosts, len(expectedIDs), fmt.Sprintf("status: %s expected: %v got: %v", status, expectedIDs, gotIDs))

		}
		for _, h := range gotHosts {
			require.Contains(t, expectedIDs, h.ID)
		}

		count, err := ds.CountHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{TeamFilter: teamID, OSSettingsFilter: status})
		require.NoError(t, err)
		require.Equal(t, len(expectedIDs), count, "status: %s", status)
	}

	type hostIDsByProfileStatus map[fleet.MDMDeliveryStatus][]uint

	checkExpected := func(t *testing.T, teamID *uint, ep hostIDsByProfileStatus) {
		checkMDMProfilesSummary(t, teamID, fleet.MDMProfilesSummary{
			Pending:   uint(len(ep[fleet.MDMDeliveryPending])),
			Failed:    uint(len(ep[fleet.MDMDeliveryFailed])),
			Verifying: uint(len(ep[fleet.MDMDeliveryVerifying])),
			Verified:  uint(len(ep[fleet.MDMDeliveryVerified])),
		})

		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsVerified, ep[fleet.MDMDeliveryVerified])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsVerifying, ep[fleet.MDMDeliveryVerifying])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsFailed, ep[fleet.MDMDeliveryFailed])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsPending, ep[fleet.MDMDeliveryPending])
	}

	upsertHostProfileStatus := func(t *testing.T, hostUUID string, profUUID string, status *fleet.MDMDeliveryStatus) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt := `INSERT INTO host_mdm_android_profiles (host_uuid, profile_uuid, status, operation_type) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE status = ?`
			_, err := q.ExecContext(ctx, stmt, hostUUID, profUUID, status, fleet.MDMOperationTypeInstall, status)
			return err
		})
	}

	cleanupTables := func(t *testing.T) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM host_mdm_android_profiles`)
			return err
		})
	}

	// Create some hosts
	var hosts []*fleet.Host
	for i := 0; i < 5; i++ {
		androidHost := createAndroidHost(fmt.Sprintf("enterprise-id-%d", i))
		newHost, err := ds.NewAndroidHost(ctx, androidHost)
		require.NoError(t, err)
		require.NotNil(t, newHost)
		hosts = append(hosts, newHost.Host)
	}

	t.Run("profiles summary empty when there are no hosts with statuses", func(t *testing.T) {
		expected := hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{},
			fleet.MDMDeliveryVerifying: []uint{},
			fleet.MDMDeliveryVerified:  []uint{},
			fleet.MDMDeliveryFailed:    []uint{},
		}
		checkExpected(t, nil, expected)
	})

	t.Run("profiles summary accounts for host profiles with mixed statuses", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			// upsert five profiles for hosts[0] with nil statuses
			upsertHostProfileStatus(t, hosts[0].UUID, fmt.Sprintf("some-android-profile-%d", i), nil)
			// upsert five profiles for hosts[1] with pending statuses
			upsertHostProfileStatus(t, hosts[1].UUID, fmt.Sprintf("some-android-profile-%d", i), &fleet.MDMDeliveryPending)
			// upsert five profiles for hosts[2] with verifying statuses
			upsertHostProfileStatus(t, hosts[2].UUID, fmt.Sprintf("some-android-profile-%d", i), &fleet.MDMDeliveryVerifying)
			// upsert five profiles for hosts[3] with verified statuses
			upsertHostProfileStatus(t, hosts[3].UUID, fmt.Sprintf("some-android-profile-%d", i), &fleet.MDMDeliveryVerified)
			// upsert five profiles for hosts[4] with failed statuses
			upsertHostProfileStatus(t, hosts[4].UUID, fmt.Sprintf("some-android-profile-%d", i), &fleet.MDMDeliveryFailed)
		}

		expected := hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{hosts[0].ID, hosts[1].ID},
			fleet.MDMDeliveryVerifying: []uint{hosts[2].ID},
			fleet.MDMDeliveryVerified:  []uint{hosts[3].ID},
			fleet.MDMDeliveryFailed:    []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// add some other android hosts that won't be be assigned any profiles
		for i := 0; i < 5; i++ {
			androidHost := createAndroidHost(fmt.Sprintf("enterprise-id-other-%d", i))
			newHost, err := ds.NewAndroidHost(ctx, androidHost)
			require.NoError(t, err)
			require.NotNil(t, newHost)
		}

		checkExpected(t, nil, expected)

		// upsert some-profile-0 to failed status for hosts[0:4]
		for i := 0; i < 5; i++ {
			upsertHostProfileStatus(t, hosts[i].UUID, "some-android-profile-0", &fleet.MDMDeliveryFailed)
		}
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{},
			fleet.MDMDeliveryVerifying: []uint{},
			fleet.MDMDeliveryVerified:  []uint{},
			fleet.MDMDeliveryFailed:    []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// upsert some-profile-0 to pending status for hosts[0:4]
		for i := 0; i < 5; i++ {
			upsertHostProfileStatus(t, hosts[i].UUID, "some-android-profile-0", &fleet.MDMDeliveryPending)
		}
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID},
			fleet.MDMDeliveryVerifying: []uint{},
			fleet.MDMDeliveryVerified:  []uint{},
			fleet.MDMDeliveryFailed:    []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// upsert some-profile-0 to verifying status for hosts[0:4]
		for i := 0; i < 5; i++ {
			upsertHostProfileStatus(t, hosts[i].UUID, "some-android-profile-0", &fleet.MDMDeliveryVerifying)
		}
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{hosts[0].ID, hosts[1].ID},
			fleet.MDMDeliveryVerifying: []uint{hosts[2].ID, hosts[3].ID},
			fleet.MDMDeliveryVerified:  []uint{},
			fleet.MDMDeliveryFailed:    []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// upsert some-profile-0 to verified status for hosts[0:4]
		for i := 0; i < 5; i++ {
			upsertHostProfileStatus(t, hosts[i].UUID, "some-android-profile-0", &fleet.MDMDeliveryVerified)
		}
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{hosts[0].ID, hosts[1].ID},
			fleet.MDMDeliveryVerifying: []uint{hosts[2].ID},
			fleet.MDMDeliveryVerified:  []uint{hosts[3].ID},
			fleet.MDMDeliveryFailed:    []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// create a new team
		t1, err := ds.NewTeam(ctx, &fleet.Team{Name: uuid.NewString()})
		require.NoError(t, err)
		require.NotNil(t, t1)

		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{},
			fleet.MDMDeliveryVerifying: []uint{},
			fleet.MDMDeliveryVerified:  []uint{},
			fleet.MDMDeliveryFailed:    []uint{},
		}
		checkExpected(t, &t1.ID, expected)

		// transfer hosts[1:2] to the team
		require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&t1.ID, []uint{hosts[1].ID, hosts[2].ID})))

		// hosts[1:2] now counted for the team, hosts[2] is counted as verifying again because
		// disk encryption is not enabled for the team
		expectedTeam1 := hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{hosts[1].ID},
			fleet.MDMDeliveryVerifying: []uint{hosts[2].ID},
		}
		checkExpected(t, &t1.ID, expectedTeam1)

		// set MDM to off for hosts[0]
		require.NoError(t, ds.SetOrUpdateMDMData(ctx, hosts[0].ID, false, false, "", false, "", "", false))
		// hosts[0] is no longer counted
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryVerified: []uint{hosts[3].ID},
			fleet.MDMDeliveryFailed:   []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		cleanupTables(t)
	})
}

func testGetHostMDMAndroidProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a host
	host := createAndroidHost("host-mdm-profiles-test")
	newHost, err := ds.NewAndroidHost(ctx, host)
	require.NoError(t, err)
	require.NotNil(t, newHost)

	// No profiles initially
	profiles, err := ds.GetHostMDMAndroidProfiles(ctx, newHost.UUID)
	require.NoError(t, err)
	require.Empty(t, profiles)

	// Create some profiles
	profile1 := androidProfileForTest("profile1")
	profile1, err = ds.NewMDMAndroidConfigProfile(ctx, *profile1)
	require.NoError(t, err)
	require.NotNil(t, profile1)

	profile2 := androidProfileForTest("profile2")
	profile2, err = ds.NewMDMAndroidConfigProfile(ctx, *profile2)
	require.NoError(t, err)
	require.NotNil(t, profile2)

	profile3 := androidProfileForTest("profile3")
	profile3, err = ds.NewMDMAndroidConfigProfile(ctx, *profile3)
	require.NoError(t, err)
	require.NotNil(t, profile3)

	// Assign profiles to host with different statuses
	upsertAndroidHostProfileStatus(t, ds, newHost.UUID, profile1.ProfileUUID, &fleet.MDMDeliveryVerified)
	upsertAndroidHostProfileStatus(t, ds, newHost.UUID, profile2.ProfileUUID, &fleet.MDMDeliveryPending)
	upsertAndroidHostProfileStatus(t, ds, newHost.UUID, profile3.ProfileUUID, nil)

	// Retrieve host profiles
	profiles, err = ds.GetHostMDMAndroidProfiles(ctx, newHost.UUID)
	require.NoError(t, err)
	require.Len(t, profiles, 3)
	byProfileUUID := make(map[string]fleet.HostMDMAndroidProfile)
	for _, p := range profiles {
		require.NotNil(t, p.Status)
		byProfileUUID[p.ProfileUUID] = p
	}
	require.Len(t, byProfileUUID, 3)
	require.Equal(t, fleet.MDMDeliveryVerified, *byProfileUUID[profile1.ProfileUUID].Status)
	require.Equal(t, fleet.MDMDeliveryPending, *byProfileUUID[profile2.ProfileUUID].Status)
	require.Equal(t, fleet.MDMDeliveryPending, *byProfileUUID[profile3.ProfileUUID].Status)

	// Change status of two profiles
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		// delivery failed
		_, err := q.ExecContext(ctx, `UPDATE host_mdm_android_profiles SET status = ? WHERE host_uuid = ? AND profile_uuid = ?`,
			fleet.MDMDeliveryFailed, newHost.UUID, profile2.ProfileUUID)
		require.NoError(t, err)
		// removal verifying
		_, err = q.ExecContext(ctx, `UPDATE host_mdm_android_profiles SET operation_type = ?, status = ? WHERE host_uuid = ? AND profile_uuid = ?`,
			fleet.MDMOperationTypeRemove, fleet.MDMDeliveryVerifying, newHost.UUID, profile3.ProfileUUID)
		return err
	})

	// Retrieve host profiles
	profiles, err = ds.GetHostMDMAndroidProfiles(ctx, newHost.UUID)
	require.NoError(t, err)
	require.Len(t, profiles, 2) // verifying removal profile not returned
	byProfileUUID = make(map[string]fleet.HostMDMAndroidProfile)
	for _, p := range profiles {
		require.NotNil(t, p.Status)
		byProfileUUID[p.ProfileUUID] = p
	}
	require.Len(t, byProfileUUID, 2)
	require.Equal(t, fleet.MDMDeliveryVerified, *byProfileUUID[profile1.ProfileUUID].Status)
	require.Equal(t, fleet.MDMDeliveryFailed, *byProfileUUID[profile2.ProfileUUID].Status)

	// Non-existent host returns empty slice
	profiles, err = ds.GetHostMDMAndroidProfiles(ctx, "non-existent-uuid")
	require.NoError(t, err)
	require.Empty(t, profiles)
}

func androidProfileForTest(name string, labels ...*fleet.Label) *fleet.MDMAndroidConfigProfile {
	payload := `{
		"maximumTimeToLock": "1234"
	}`

	profile := &fleet.MDMAndroidConfigProfile{
		RawJSON: []byte(payload),
		Name:    name,
	}

	for _, l := range labels {
		switch {
		case strings.HasPrefix(l.Name, "exclude-"):
			profile.LabelsExcludeAny = append(profile.LabelsExcludeAny, fleet.ConfigurationProfileLabel{LabelName: l.Name, LabelID: l.ID})
		case strings.HasPrefix(l.Name, "inclany-"):
			profile.LabelsIncludeAny = append(profile.LabelsIncludeAny, fleet.ConfigurationProfileLabel{LabelName: l.Name, LabelID: l.ID})
		default:
			profile.LabelsIncludeAll = append(profile.LabelsIncludeAll, fleet.ConfigurationProfileLabel{LabelName: l.Name, LabelID: l.ID})
		}
	}

	return profile
}

func upsertAndroidHostProfileStatus(t *testing.T, ds *Datastore, hostUUID string, profUUID string, status *fleet.MDMDeliveryStatus) {
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		stmt := `INSERT INTO host_mdm_android_profiles (host_uuid, profile_uuid, status, operation_type) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE status = ?`
		_, err := q.ExecContext(context.Background(), stmt, hostUUID, profUUID, status, fleet.MDMOperationTypeInstall, status)
		return err
	})
}

func expectAndroidProfiles(
	t *testing.T,
	ds *Datastore,
	tmID *uint,
	want []*fleet.MDMAndroidConfigProfile,
) {
	if tmID == nil {
		tmID = ptr.Uint(0)
	}

	ctx := t.Context()
	var gotUUIDs []string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &gotUUIDs,
			`SELECT profile_uuid FROM mdm_android_configuration_profiles WHERE team_id = ?`,
			tmID)
	})

	// load each profile, this will also load its labels
	var got []*fleet.MDMAndroidConfigProfile
	for _, profileUUID := range gotUUIDs {
		profile, err := ds.GetMDMAndroidConfigProfile(ctx, profileUUID)
		require.NoError(t, err)
		got = append(got, profile)
	}
	// create map of expected uuids keyed by name
	wantMap := make(map[string]*fleet.MDMAndroidConfigProfile, len(want))
	for _, cp := range want {
		wantMap[cp.Name] = cp
	}

	JSONRemarshal := func(bytes []byte) ([]byte, error) {
		var ifce interface{}
		err := json.Unmarshal(bytes, &ifce)
		if err != nil {
			return nil, err
		}
		return json.Marshal(ifce)
	}

	// compare only the fields we care about, and build the resulting map of
	// profile identifier as key to profile UUID as value
	for _, gotA := range got {

		wantA := wantMap[gotA.Name]

		if gotA.TeamID != nil && *gotA.TeamID == 0 {
			gotA.TeamID = nil
		}

		// ProfileUUID is non-empty and starts with "g", but otherwise we don't
		// care about it for test assertions.
		require.NotEmpty(t, gotA.ProfileUUID)
		require.True(t, strings.HasPrefix(gotA.ProfileUUID, fleet.MDMAndroidProfileUUIDPrefix))
		gotA.ProfileUUID = ""

		gotA.CreatedAt = time.Time{}
		gotA.AutoIncrement = 0

		gotBytes, err := JSONRemarshal(gotA.RawJSON)
		require.NoError(t, err)
		gotA.RawJSON = gotBytes

		// if an expected uploaded_at timestamp is provided for this profile, keep
		// its value, otherwise clear it as we don't care about asserting its
		// value.
		if wantA.UploadedAt.IsZero() {
			gotA.UploadedAt = time.Time{}
		}
	}

	require.ElementsMatch(t, want, got)
}

func testListMDMAndroidProfilesToSend(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create some hosts
	hosts := make([]*fleet.Host, 2)
	for i := range hosts {
		androidHost := createAndroidHost(fmt.Sprintf("enterprise-id-%d", i))
		newHost, err := ds.NewAndroidHost(ctx, androidHost)
		require.NoError(t, err)
		hosts[i] = newHost.Host
	}

	// without any profile, should return empty
	profs, toRemoveProfs, err := ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, profs)
	require.Empty(t, toRemoveProfs)

	// create a couple profiles for no team, and one for a team
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "team"})
	require.NoError(t, err)

	p1, err := ds.NewMDMAndroidConfigProfile(ctx, *androidProfileForTest("no-team-1"))
	require.NoError(t, err)
	p2, err := ds.NewMDMAndroidConfigProfile(ctx, *androidProfileForTest("no-team-2"))
	require.NoError(t, err)
	tmP3 := androidProfileForTest("team-1")
	tmP3.TeamID = &tm.ID
	p3, err := ds.NewMDMAndroidConfigProfile(ctx, *tmP3)
	require.NoError(t, err)

	// both no-team profiles should be applicable to both hosts
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.Len(t, profs, 4)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p2.Name},
	}, profs)

	// transfer host 1 to the team
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm.ID, []uint{hosts[1].ID}))
	require.NoError(t, err)

	// profiles for host 1 change to p3
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.Len(t, profs, 3)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p3.Name},
	}, profs)

	// test the include all labels condition
	lblIncAll1, err := ds.NewLabel(ctx, &fleet.Label{Name: "inclall-1", Query: "select 1"})
	require.NoError(t, err)
	lblIncAll2, err := ds.NewLabel(ctx, &fleet.Label{Name: "inclall-2", Query: "select 1"})
	require.NoError(t, err)
	p4, err := ds.NewMDMAndroidConfigProfile(ctx, *androidProfileForTest("no-team-4", lblIncAll1, lblIncAll2))
	require.NoError(t, err)

	// no change, host is not a member of both labels
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.Len(t, profs, 3)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p3.Name},
	}, profs)

	// make host[0] a member of only one of the labels
	_, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, lblIncAll1.ID, []uint{hosts[0].ID}, fleet.TeamFilter{})
	require.NoError(t, err)

	// no change, host is not a member of both labels
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.Len(t, profs, 3)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p3.Name},
	}, profs)

	// make host[0] a member of the other label
	_, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, lblIncAll2.ID, []uint{hosts[0].ID}, fleet.TeamFilter{})
	require.NoError(t, err)

	// now p4 is applicable to host 0
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.Len(t, profs, 4)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p4.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p4.Name},
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p3.Name},
	}, profs)

	// test the include any labels condition
	lblIncAny1, err := ds.NewLabel(ctx, &fleet.Label{Name: "inclany-1", Query: "select 1"})
	require.NoError(t, err)
	lblIncAny2, err := ds.NewLabel(ctx, &fleet.Label{Name: "inclany-2", Query: "select 1"})
	require.NoError(t, err)
	p5, err := ds.NewMDMAndroidConfigProfile(ctx, *androidProfileForTest("no-team-5", lblIncAny1, lblIncAny2))
	require.NoError(t, err)

	// no change, host 0 not a member yet
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.Len(t, profs, 4)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p4.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p4.Name},
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p3.Name},
	}, profs)

	// make host[0] a member of one of the labels
	_, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, lblIncAny1.ID, []uint{hosts[0].ID}, fleet.TeamFilter{})
	require.NoError(t, err)

	// now p5 is applicable to host 0
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.Len(t, profs, 5)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p4.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p4.Name},
		{ProfileUUID: p5.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p5.Name},
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p3.Name},
	}, profs)

	// test the exclude any labels condition
	lblExclAny1, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-1", Query: "select 1"})
	require.NoError(t, err)
	lblExclAny2, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-2", Query: "select 1"})
	require.NoError(t, err)
	p6, err := ds.NewMDMAndroidConfigProfile(ctx, *androidProfileForTest("no-team-6", lblExclAny1, lblExclAny2))
	require.NoError(t, err)

	// no change, label membership was not updated after labels created
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.Len(t, profs, 5)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p4.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p4.Name},
		{ProfileUUID: p5.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p5.Name},
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p3.Name},
	}, profs)

	// update the timestamp of when host label membership was updated
	hosts[0].LabelUpdatedAt = time.Now().UTC().Add(time.Second) // just to be extra safe in tests
	hosts[0].PolicyUpdatedAt = time.Now().UTC()
	err = ds.UpdateHost(ctx, hosts[0])
	require.NoError(t, err)

	// host 0 is _not_ a member of the excluded labels, so p6 is applicable
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.Len(t, profs, 6)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p4.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p4.Name},
		{ProfileUUID: p5.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p5.Name},
		{ProfileUUID: p6.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p6.Name},
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p3.Name},
	}, profs)

	// make host[0] a member of one of the exclude labels
	_, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, lblExclAny2.ID, []uint{hosts[0].ID}, fleet.TeamFilter{})
	require.NoError(t, err)

	// p6 is not applicable anymore
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.Len(t, profs, 5)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p4.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p4.Name},
		{ProfileUUID: p5.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p5.Name},
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p3.Name},
	}, profs)

	// add another host in team
	androidHost := createAndroidHost(fmt.Sprintf("enterprise-id-%d", 2))
	newHost, err := ds.NewAndroidHost(ctx, androidHost)
	require.NoError(t, err)
	hosts = append(hosts, newHost.Host)
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm.ID, []uint{hosts[2].ID}))
	require.NoError(t, err)

	// it is not included in noProfHosts as it has p3
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.Len(t, profs, 6)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p4.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p4.Name},
		{ProfileUUID: p5.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p5.Name},
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p3.Name},
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[2].UUID, ProfileName: p3.Name},
	}, profs)

	// simulate that host 2 already has p3 installed
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO host_mdm_android_profiles
			(host_uuid, profile_uuid, profile_name, included_in_policy_version, operation_type, status)
			VALUES (?, ?, ?, ?, ?, ?)`, hosts[2].UUID, p3.ProfileUUID, p3.Name, 1, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)
		return err
	})

	// host 2 is not included in the results as it has p3 installed
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.Len(t, profs, 5)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p4.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p4.Name},
		{ProfileUUID: p5.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p5.Name},
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: p3.Name},
	}, profs)

	// delete profile p3
	err = ds.DeleteMDMAndroidConfigProfile(ctx, p3.ProfileUUID)
	require.NoError(t, err)

	// host 2 is now a host with no profile (profile 3 needs to be cleared), host 1 is unlisted as it didn't have p3 installed
	profs, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p3.ProfileUUID, HostUUID: hosts[2].UUID, ProfileName: p3.Name},
	}, toRemoveProfs)
	require.Len(t, profs, 4)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: p1.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p1.Name},
		{ProfileUUID: p2.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p2.Name},
		{ProfileUUID: p4.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p4.Name},
		{ProfileUUID: p5.ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: p5.Name},
	}, profs)
}

func testGetMDMAndroidProfilesContents(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	p1 := androidProfileForTest("p1")
	p1.RawJSON = []byte(`{"v": 1}`)
	p2 := androidProfileForTest("p2")
	p2.RawJSON = []byte(`{"v": 2}`)
	p3 := androidProfileForTest("p3")
	p3.RawJSON = []byte(`{"v": 3}`)

	p1, err := ds.NewMDMAndroidConfigProfile(ctx, *p1)
	require.NoError(t, err)
	p2, err = ds.NewMDMAndroidConfigProfile(ctx, *p2)
	require.NoError(t, err)
	p3, err = ds.NewMDMAndroidConfigProfile(ctx, *p3)
	require.NoError(t, err)

	cases := []struct {
		uuids []string
		want  map[string]json.RawMessage
	}{
		{[]string{}, nil},
		{nil, nil},
		{[]string{p1.ProfileUUID}, map[string]json.RawMessage{p1.ProfileUUID: p1.RawJSON}},
		{[]string{p1.ProfileUUID, p2.ProfileUUID}, map[string]json.RawMessage{
			p1.ProfileUUID: p1.RawJSON,
			p2.ProfileUUID: p2.RawJSON,
		}},
		{[]string{p1.ProfileUUID, p2.ProfileUUID, p3.ProfileUUID}, map[string]json.RawMessage{
			p1.ProfileUUID: p1.RawJSON,
			p2.ProfileUUID: p2.RawJSON,
			p3.ProfileUUID: p3.RawJSON,
		}},
		{[]string{p1.ProfileUUID, p2.ProfileUUID, "no-such-uuid"}, map[string]json.RawMessage{
			p1.ProfileUUID: p1.RawJSON,
			p2.ProfileUUID: p2.RawJSON,
		}},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v", c.uuids), func(t *testing.T) {
			out, err := ds.GetMDMAndroidProfilesContents(ctx, c.uuids)
			require.NoError(t, err)
			require.Equal(t, c.want, out)
		})
	}
}

func testBulkUpsertMDMAndroidHostProfiles(t *testing.T, ds *Datastore) {
	testBulkUpsertMDMAndroidHostProfilesN(t, ds, 0)
}

func testBulkUpsertMDMAndroidHostProfiles2(t *testing.T, ds *Datastore) {
	testBulkUpsertMDMAndroidHostProfilesN(t, ds, 2)
}

func testBulkUpsertMDMAndroidHostProfiles3(t *testing.T, ds *Datastore) {
	testBulkUpsertMDMAndroidHostProfilesN(t, ds, 3)
}

func testBulkUpsertMDMAndroidHostProfilesN(t *testing.T, ds *Datastore, batchSize int) {
	ctx := t.Context()

	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "team"})
	require.NoError(t, err)

	// Create some hosts and some profiles
	hosts := make([]*fleet.Host, 3)
	for i := range hosts {
		androidHost := createAndroidHost(fmt.Sprintf("enterprise-id-%d", i))
		newHost, err := ds.NewAndroidHost(ctx, androidHost)
		require.NoError(t, err)
		hosts[i] = newHost.Host
		if i == len(hosts)-1 {
			// last host is in a team
			err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm.ID, []uint{hosts[i].ID}))
			require.NoError(t, err)
		}
	}

	profiles := make([]*fleet.MDMAndroidConfigProfile, 3)
	for i := range profiles {
		p := androidProfileForTest(fmt.Sprintf("profile-%d", i))
		if i == len(profiles)-1 {
			// last profile is for a team
			p.TeamID = &tm.ID
		}
		p, err := ds.NewMDMAndroidConfigProfile(ctx, *p)
		require.NoError(t, err)
		profiles[i] = p
	}

	err = ds.BulkUpsertMDMAndroidHostProfiles(ctx, nil)
	require.NoError(t, err)

	ds.testUpsertMDMDesiredProfilesBatchSize = batchSize
	t.Cleanup(func() { ds.testUpsertMDMDesiredProfilesBatchSize = 0 })

	hostProfiles, toRemoveProfs, err := ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: profiles[0].ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: profiles[0].Name},
		{ProfileUUID: profiles[1].ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: profiles[1].Name},
		{ProfileUUID: profiles[0].ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: profiles[0].Name},
		{ProfileUUID: profiles[1].ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: profiles[1].Name},
		{ProfileUUID: profiles[2].ProfileUUID, HostUUID: hosts[2].UUID, ProfileName: profiles[2].Name},
	}, hostProfiles)

	// mark all installed for hosts 0, profile 1 failed for host 1
	err = ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
		{
			HostUUID:                hosts[0].UUID,
			ProfileUUID:             profiles[0].ProfileUUID,
			ProfileName:             profiles[0].Name,
			OperationType:           fleet.MDMOperationTypeInstall,
			Status:                  &fleet.MDMDeliveryPending,
			IncludedInPolicyVersion: ptr.Int(1),
		},
		{
			HostUUID:                hosts[0].UUID,
			ProfileUUID:             profiles[1].ProfileUUID,
			ProfileName:             profiles[1].Name,
			OperationType:           fleet.MDMOperationTypeInstall,
			Status:                  &fleet.MDMDeliveryPending,
			IncludedInPolicyVersion: ptr.Int(1),
		},
		{
			HostUUID:                hosts[1].UUID,
			ProfileUUID:             profiles[1].ProfileUUID,
			ProfileName:             profiles[1].Name,
			OperationType:           fleet.MDMOperationTypeInstall,
			Status:                  &fleet.MDMDeliveryFailed,
			IncludedInPolicyVersion: ptr.Int(1),
		},
	})
	require.NoError(t, err)

	hostProfiles, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		// because host 1 still has a missing profile, it must resend both (as it merged them)
		{ProfileUUID: profiles[0].ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: profiles[0].Name},
		{ProfileUUID: profiles[1].ProfileUUID, HostUUID: hosts[1].UUID, ProfileName: profiles[1].Name},
		{ProfileUUID: profiles[2].ProfileUUID, HostUUID: hosts[2].UUID, ProfileName: profiles[2].Name},
	}, hostProfiles)

	// mark host 0 profile 1 as NULL, host 1 profile 0 as installed (so both are now installed), and host 2 profile 2 as installed
	err = ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
		{
			HostUUID:                hosts[0].UUID,
			ProfileUUID:             profiles[1].ProfileUUID,
			ProfileName:             profiles[1].Name,
			OperationType:           fleet.MDMOperationTypeInstall,
			Status:                  nil,
			IncludedInPolicyVersion: ptr.Int(1),
		},
		{
			HostUUID:                hosts[1].UUID,
			ProfileUUID:             profiles[0].ProfileUUID,
			ProfileName:             profiles[0].Name,
			OperationType:           fleet.MDMOperationTypeInstall,
			Status:                  &fleet.MDMDeliveryPending,
			IncludedInPolicyVersion: ptr.Int(1),
		},
		{
			HostUUID:                hosts[2].UUID,
			ProfileUUID:             profiles[2].ProfileUUID,
			ProfileName:             profiles[2].Name,
			OperationType:           fleet.MDMOperationTypeInstall,
			Status:                  &fleet.MDMDeliveryPending,
			IncludedInPolicyVersion: ptr.Int(1),
		},
	})
	require.NoError(t, err)

	hostProfiles, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemoveProfs)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		// host 0 now has a profile not installed, so it needs to resend both
		{ProfileUUID: profiles[0].ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: profiles[0].Name},
		{ProfileUUID: profiles[1].ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: profiles[1].Name},
		// host 1 now has both delivered, nothing to resend
		// host 2 profile is delivered, nothing to resend
	}, hostProfiles)

	// delete profile 2, which will cause host 2 to be resent as "no profiles" to remove it as it was delivered
	err = ds.DeleteMDMAndroidConfigProfile(ctx, profiles[2].ProfileUUID)
	require.NoError(t, err)

	hostProfiles, toRemoveProfs, err = ds.ListMDMAndroidProfilesToSend(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: profiles[2].ProfileUUID, HostUUID: hosts[2].UUID, ProfileName: profiles[2].Name},
	}, toRemoveProfs)
	require.ElementsMatch(t, []*fleet.MDMAndroidProfilePayload{
		{ProfileUUID: profiles[0].ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: profiles[0].Name},
		{ProfileUUID: profiles[1].ProfileUUID, HostUUID: hosts[0].UUID, ProfileName: profiles[1].Name},
	}, hostProfiles)
}

func testGetAndroidPolicyRequestByUUID(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	policyRequestUUID := uuid.New().String()

	t.Run("Returns not found", func(t *testing.T) {
		policyRequest, err := ds.GetAndroidPolicyRequestByUUID(ctx, policyRequestUUID)
		require.Contains(t, err.Error(), common_mysql.NotFound("AndroidPolicyRequest").WithName(policyRequestUUID).Error())
		require.Nil(t, policyRequest)
	})

	t.Run("Correctly retrieves the policy request", func(t *testing.T) {
		// Create a test policy request
		err := ds.NewAndroidPolicyRequest(ctx, &fleet.MDMAndroidPolicyRequest{
			RequestUUID: policyRequestUUID,
			Payload:     json.RawMessage(`{"key": "value"}`),
		})
		require.NoError(t, err)

		// Retrieve the policy request by UUID
		policyRequest, err := ds.GetAndroidPolicyRequestByUUID(ctx, policyRequestUUID)
		require.NoError(t, err)
		require.NotNil(t, policyRequest)
		require.Equal(t, policyRequestUUID, policyRequest.RequestUUID)
	})
}

func testListHostMDMAndroidProfilesPendingInstallWithVersion(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	profiles := make([]*fleet.MDMAndroidConfigProfile, 3)
	for i := range profiles {
		p := androidProfileForTest(fmt.Sprintf("profile-%d", i))
		p, err := ds.NewMDMAndroidConfigProfile(ctx, *p)
		require.NoError(t, err)
		profiles[i] = p
	}
	hostUUID := uuid.NewString()

	clearOutHostMDMAndroidProfilesTable := func() {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "DELETE FROM host_mdm_android_profiles WHERE host_uuid = ?", hostUUID)
			return err
		})
	}

	t.Run("Does not list other install statuses", func(t *testing.T) {
		// Arrange
		policyVersion := ptr.Int(1)
		err := ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[0].ProfileUUID,
				ProfileName:             profiles[0].Name,
				OperationType:           fleet.MDMOperationTypeInstall,
				Status:                  &fleet.MDMDeliveryFailed,
				IncludedInPolicyVersion: policyVersion,
			},
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[1].ProfileUUID,
				ProfileName:             profiles[1].Name,
				OperationType:           fleet.MDMOperationTypeInstall,
				Status:                  &fleet.MDMDeliveryVerified,
				IncludedInPolicyVersion: policyVersion,
			},
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[2].ProfileUUID,
				ProfileName:             profiles[2].Name,
				OperationType:           fleet.MDMOperationTypeInstall,
				Status:                  &fleet.MDMDeliveryVerifying,
				IncludedInPolicyVersion: policyVersion,
			},
		})
		require.NoError(t, err)
		t.Cleanup(clearOutHostMDMAndroidProfilesTable)

		hostProfiles, err := ds.ListHostMDMAndroidProfilesPendingInstallWithVersion(ctx, hostUUID, int64(*policyVersion))
		require.NoError(t, err)
		require.Len(t, hostProfiles, 0)
	})

	t.Run("Does not list higher versions than passed", func(t *testing.T) {
		// Arrange
		policyVersion := ptr.Int(2)
		err := ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[0].ProfileUUID,
				ProfileName:             profiles[0].Name,
				OperationType:           fleet.MDMOperationTypeInstall,
				Status:                  &fleet.MDMDeliveryFailed,
				IncludedInPolicyVersion: policyVersion,
			},
		})
		require.NoError(t, err)
		t.Cleanup(clearOutHostMDMAndroidProfilesTable)

		hostProfiles, err := ds.ListHostMDMAndroidProfilesPendingInstallWithVersion(ctx, hostUUID, int64(*policyVersion-1))
		require.NoError(t, err)
		require.Len(t, hostProfiles, 0)
	})

	t.Run("Does not list remove operation", func(t *testing.T) {
		// Arrange
		policyVersion := ptr.Int(1)
		err := ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[0].ProfileUUID,
				ProfileName:             profiles[0].Name,
				OperationType:           fleet.MDMOperationTypeRemove,
				Status:                  &fleet.MDMDeliveryFailed,
				IncludedInPolicyVersion: policyVersion,
			},
		})
		require.NoError(t, err)
		t.Cleanup(clearOutHostMDMAndroidProfilesTable)

		hostProfiles, err := ds.ListHostMDMAndroidProfilesPendingInstallWithVersion(ctx, hostUUID, int64(*policyVersion))
		require.NoError(t, err)
		require.Len(t, hostProfiles, 0)
	})

	t.Run("Does list pending install profiles with version less than or equal to applied policy version", func(t *testing.T) {
		// Arrange
		policyVersion := ptr.Int(1)
		err := ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[0].ProfileUUID,
				ProfileName:             profiles[0].Name,
				OperationType:           fleet.MDMOperationTypeInstall,
				Status:                  &fleet.MDMDeliveryPending,
				IncludedInPolicyVersion: policyVersion,
			},
		})
		require.NoError(t, err)
		t.Cleanup(clearOutHostMDMAndroidProfilesTable)

		hostProfiles, err := ds.ListHostMDMAndroidProfilesPendingInstallWithVersion(ctx, hostUUID, int64(*policyVersion))
		require.NoError(t, err)
		require.Len(t, hostProfiles, 1)
		require.Equal(t, &fleet.MDMDeliveryPending, hostProfiles[0].Status)
		require.Equal(t, fleet.MDMOperationTypeInstall, hostProfiles[0].OperationType)
		require.EqualValues(t, policyVersion, hostProfiles[0].IncludedInPolicyVersion)
	})
}

func testBulkDeleteMDMAndroidHostProfiles(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	profiles := make([]*fleet.MDMAndroidConfigProfile, 3)
	for i := range profiles {
		p := androidProfileForTest(fmt.Sprintf("profile-%d", i))
		p, err := ds.NewMDMAndroidConfigProfile(ctx, *p)
		require.NoError(t, err)
		profiles[i] = p
	}
	hostUUID := uuid.NewString()

	clearOutHostMDMAndroidProfilesTable := func() {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "DELETE FROM host_mdm_android_profiles WHERE host_uuid = ?", hostUUID)
			return err
		})
	}

	listAllHostMDMAndroidProfiles := func() []*fleet.MDMAndroidProfilePayload {
		var hostProfiles []*fleet.MDMAndroidProfilePayload
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			err := sqlx.SelectContext(ctx, q, &hostProfiles, "SELECT profile_uuid, host_uuid, profile_name, operation_type, status, detail, included_in_policy_version, policy_request_uuid, device_request_uuid, request_fail_count FROM host_mdm_android_profiles")
			require.NoError(t, err)
			return err
		})

		return hostProfiles
	}

	t.Run("Does not delete profiles not associated with host", func(t *testing.T) {
		// Arrange
		policyVersion := ptr.Int(1)
		err := ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[0].ProfileUUID,
				ProfileName:             profiles[0].Name,
				OperationType:           fleet.MDMOperationTypeInstall,
				Status:                  &fleet.MDMDeliveryPending,
				IncludedInPolicyVersion: policyVersion,
			},
		})
		require.NoError(t, err)
		t.Cleanup(clearOutHostMDMAndroidProfilesTable)

		// Act
		err = ds.BulkDeleteMDMAndroidHostProfiles(ctx, uuid.NewString(), int64(*policyVersion))
		require.NoError(t, err)

		// Assert
		hostProfiles := listAllHostMDMAndroidProfiles()
		require.Len(t, hostProfiles, 1)
	})

	t.Run("Does not delete install operation types", func(t *testing.T) {
		// Arrange
		policyVersion := ptr.Int(1)
		err := ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[0].ProfileUUID,
				ProfileName:             profiles[0].Name,
				OperationType:           fleet.MDMOperationTypeInstall,
				Status:                  &fleet.MDMDeliveryPending,
				IncludedInPolicyVersion: policyVersion,
			},
		})
		require.NoError(t, err)
		t.Cleanup(clearOutHostMDMAndroidProfilesTable)

		// Act
		err = ds.BulkDeleteMDMAndroidHostProfiles(ctx, hostUUID, int64(*policyVersion))
		require.NoError(t, err)

		// Assert
		hostProfiles := listAllHostMDMAndroidProfiles()
		require.Len(t, hostProfiles, 1)
	})

	t.Run("Does not delete other statuses with remove operation", func(t *testing.T) {
		// Arrange
		policyVersion := ptr.Int(1)
		err := ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[1].ProfileUUID,
				ProfileName:             profiles[1].Name,
				OperationType:           fleet.MDMOperationTypeRemove,
				Status:                  &fleet.MDMDeliveryVerifying,
				IncludedInPolicyVersion: policyVersion,
			},
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[2].ProfileUUID,
				ProfileName:             profiles[2].Name,
				OperationType:           fleet.MDMOperationTypeRemove,
				Status:                  &fleet.MDMDeliveryVerified,
				IncludedInPolicyVersion: policyVersion,
			},
		})
		require.NoError(t, err)
		t.Cleanup(clearOutHostMDMAndroidProfilesTable)

		// Act
		err = ds.BulkDeleteMDMAndroidHostProfiles(ctx, hostUUID, int64(*policyVersion))
		require.NoError(t, err)

		// Assert
		hostProfiles := listAllHostMDMAndroidProfiles()
		require.Len(t, hostProfiles, 2)
	})

	t.Run("Does not delete profiles with higher policy version than passed", func(t *testing.T) {
		// Arrange
		policyVersion := ptr.Int(2)
		err := ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[0].ProfileUUID,
				ProfileName:             profiles[0].Name,
				OperationType:           fleet.MDMOperationTypeRemove,
				Status:                  &fleet.MDMDeliveryPending,
				IncludedInPolicyVersion: policyVersion,
			},
		})
		require.NoError(t, err)
		t.Cleanup(clearOutHostMDMAndroidProfilesTable)

		// Act
		err = ds.BulkDeleteMDMAndroidHostProfiles(ctx, hostUUID, int64(*policyVersion-1))
		require.NoError(t, err)

		// Assert
		hostProfiles := listAllHostMDMAndroidProfiles()
		require.Len(t, hostProfiles, 1)
	})

	t.Run("Deletes pending or failed remove profiles with policy version lower than or equal to passed", func(t *testing.T) {
		// Arrange
		policyVersion := ptr.Int(2)
		err := ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[0].ProfileUUID,
				ProfileName:             profiles[0].Name,
				OperationType:           fleet.MDMOperationTypeRemove,
				Status:                  &fleet.MDMDeliveryPending,
				IncludedInPolicyVersion: policyVersion,
			},
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[1].ProfileUUID,
				ProfileName:             profiles[1].Name,
				OperationType:           fleet.MDMOperationTypeRemove,
				Status:                  &fleet.MDMDeliveryPending,
				IncludedInPolicyVersion: ptr.Int(*policyVersion - 1),
			},
			{
				HostUUID:                hostUUID,
				ProfileUUID:             profiles[2].ProfileUUID,
				ProfileName:             profiles[2].Name,
				OperationType:           fleet.MDMOperationTypeRemove,
				Status:                  &fleet.MDMDeliveryFailed,
				IncludedInPolicyVersion: policyVersion,
			},
		})
		require.NoError(t, err)
		t.Cleanup(clearOutHostMDMAndroidProfilesTable)

		// Act
		err = ds.BulkDeleteMDMAndroidHostProfiles(ctx, hostUUID, int64(*policyVersion))
		require.NoError(t, err)

		// Assert
		hostProfiles := listAllHostMDMAndroidProfiles()
		require.Len(t, hostProfiles, 0)
	})
}

func testNewAndroidHostWithIdP(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.AddBuiltinLabels(t, ds)

	// create IdP account... InsertMDMIdPAccount generates its own UUID
	idpAccount := &fleet.MDMIdPAccount{
		Username: "john.doe",
		Fullname: "John Doe",
		Email:    "john.doe@example.com",
	}
	err := ds.InsertMDMIdPAccount(ctx, idpAccount)
	require.NoError(t, err)

	// get the actual UUID that was generated
	insertedAccount, err := ds.GetMDMIdPAccountByEmail(ctx, "john.doe@example.com")
	require.NoError(t, err)
	require.NotNil(t, insertedAccount)
	idpAccount.UUID = insertedAccount.UUID

	// create Android host
	const enterpriseSpecificID = "enterprise_with_idp"
	host := createAndroidHost(enterpriseSpecificID)
	host.Host.UUID = "test-host-uuid" // Use a specific UUID for testing

	result, err := ds.NewAndroidHost(ctx, host)
	require.NoError(t, err)
	require.NotZero(t, result.Host.ID)

	// associate host with IdP account, triggering reconciliation
	err = ds.AssociateHostMDMIdPAccount(ctx, "test-host-uuid", idpAccount.UUID)
	require.NoError(t, err)

	// host_emails table has IdP email
	emails, err := ds.GetHostEmails(ctx, "test-host-uuid", fleet.DeviceMappingMDMIdpAccounts)
	require.NoError(t, err)
	require.Len(t, emails, 1)
	assert.Equal(t, "john.doe@example.com", emails[0])

	// is reconciliation idempotent?
	err = ds.AssociateHostMDMIdPAccount(ctx, "test-host-uuid", idpAccount.UUID)
	require.NoError(t, err)

	// still only one email (no duplicates)
	emails, err = ds.GetHostEmails(ctx, "test-host-uuid", fleet.DeviceMappingMDMIdpAccounts)
	require.NoError(t, err)
	require.Len(t, emails, 1, "Should still have exactly one email after reassociation")
	assert.Equal(t, "john.doe@example.com", emails[0])

	// remove IdP account association and trigger reconciliation
	_, err = ds.writer(ctx).ExecContext(ctx,
		`DELETE FROM host_mdm_idp_accounts WHERE host_uuid = ?`,
		"test-host-uuid")
	require.NoError(t, err)

	// test cleanup (in production this would happen on re-enrollment)
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := reconcileHostEmailsFromMdmIdpAccountsDB(ctx, tx, ds.logger, result.Host.ID)
		return err
	})
	require.NoError(t, err)

	// host_emails table no longer has IdP email
	emails, err = ds.GetHostEmails(ctx, "test-host-uuid", fleet.DeviceMappingMDMIdpAccounts)
	require.NoError(t, err)
	require.Empty(t, emails, "IdP email should be removed when association is deleted")
}

func testAndroidBYODDetection(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	test.AddBuiltinLabels(t, ds)

	// Test 1: Android host with non-empty UUID (BYOD/personal device)
	t.Run("personal enrollment with UUID", func(t *testing.T) {
		const enterpriseID = "test-enterprise-id-byod"
		host := createAndroidHost(enterpriseID)
		// Ensure UUID is set (createAndroidHost already does this)
		require.NotEmpty(t, host.Host.UUID)
		require.Equal(t, enterpriseID, host.Host.UUID)

		result, err := ds.NewAndroidHost(ctx, host)
		require.NoError(t, err)
		require.NotZero(t, result.Host.ID)

		// Query host_mdm table directly to verify is_personal_enrollment = 1
		var isPersonalEnrollment bool
		err = sqlx.GetContext(ctx, ds.reader(ctx), &isPersonalEnrollment,
			`SELECT is_personal_enrollment FROM host_mdm WHERE host_id = ?`,
			result.Host.ID)
		require.NoError(t, err)
		assert.True(t, isPersonalEnrollment, "BYOD device with UUID should have is_personal_enrollment = 1")
	})

	// Test 2: Android host without UUID (company-owned device)
	t.Run("company enrollment without UUID", func(t *testing.T) {
		const enterpriseID = "test-enterprise-id-company"
		host := createAndroidHost(enterpriseID)
		// Override UUID to be empty to simulate company-owned device
		host.Host.UUID = ""

		result, err := ds.NewAndroidHost(ctx, host)
		require.NoError(t, err)
		require.NotZero(t, result.Host.ID)

		// Query host_mdm table directly to verify is_personal_enrollment = 0
		var isPersonalEnrollment bool
		err = sqlx.GetContext(ctx, ds.reader(ctx), &isPersonalEnrollment,
			`SELECT is_personal_enrollment FROM host_mdm WHERE host_id = ?`,
			result.Host.ID)
		require.NoError(t, err)
		assert.False(t, isPersonalEnrollment, "Company device without UUID should have is_personal_enrollment = 0")
	})

	// Test 3: Verify update path also sets personal enrollment correctly
	t.Run("update existing host enrollment status", func(t *testing.T) {
		// Create a host initially without UUID
		const enterpriseID = "test-enterprise-id-update"
		host := createAndroidHost(enterpriseID)
		host.Host.UUID = ""

		result, err := ds.NewAndroidHost(ctx, host)
		require.NoError(t, err)
		require.NotZero(t, result.Host.ID)

		// Initially should not be personal enrollment
		var isPersonalEnrollment bool
		err = sqlx.GetContext(ctx, ds.reader(ctx), &isPersonalEnrollment,
			`SELECT is_personal_enrollment FROM host_mdm WHERE host_id = ?`,
			result.Host.ID)
		require.NoError(t, err)
		assert.False(t, isPersonalEnrollment, "Initially should not be personal enrollment")

		// Update the host with a UUID (simulating re-enrollment as BYOD)
		result.Host.UUID = enterpriseID
		err = ds.UpdateAndroidHost(ctx, result, true) // fromEnroll = true to trigger MDM info update
		require.NoError(t, err)

		// Now should be marked as personal enrollment
		err = sqlx.GetContext(ctx, ds.reader(ctx), &isPersonalEnrollment,
			`SELECT is_personal_enrollment FROM host_mdm WHERE host_id = ?`,
			result.Host.ID)
		require.NoError(t, err)
		assert.True(t, isPersonalEnrollment, "After update with UUID should have is_personal_enrollment = 1")
	})
}

// NEW TEST: verify single-host unenroll updates host_mdm correctly
func testSetAndroidHostUnenrolled(t *testing.T, ds *Datastore) {
	// Set a non-empty server URL so initial enrolled row has data to clear
	appCfg, err := ds.AppConfig(testCtx())
	require.NoError(t, err)
	appCfg.ServerSettings.ServerURL = "https://mdm.example.com"
	require.NoError(t, ds.SaveAppConfig(testCtx(), appCfg))

	// Create an Android host (this also upserts an enrolled host_mdm row)
	esid := "enterprise-" + uuid.NewString()
	h := createAndroidHost(esid)
	res, err := ds.NewAndroidHost(testCtx(), h)
	require.NoError(t, err)

	// Sanity check initial host_mdm values
	var enrolled int
	var serverURL string
	var mdmIDIsNull int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(testCtx(), q, &enrolled, `SELECT enrolled FROM host_mdm WHERE host_id = ?`, res.Host.ID)
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(testCtx(), q, &serverURL, `SELECT server_url FROM host_mdm WHERE host_id = ?`, res.Host.ID)
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(testCtx(), q, &mdmIDIsNull, `SELECT CASE WHEN mdm_id IS NULL THEN 1 ELSE 0 END FROM host_mdm WHERE host_id = ?`, res.Host.ID)
	})
	require.Equal(t, 1, enrolled)
	require.NotEmpty(t, serverURL)
	require.Equal(t, 0, mdmIDIsNull)

	upsertAndroidHostProfileStatus(t, ds, res.Host.UUID, "profile-1", &fleet.MDMDeliveryPending)
	upsertAndroidHostProfileStatus(t, ds, res.Host.UUID, "profile-2", &fleet.MDMDeliveryPending)

	// Perform single-host unenroll
	didUnenroll, err := ds.SetAndroidHostUnenrolled(testCtx(), res.Host.ID)
	require.NoError(t, err)
	require.True(t, didUnenroll)

	// Calling unenrolled again returns false
	didUnenroll, err = ds.SetAndroidHostUnenrolled(testCtx(), res.Host.ID)
	require.NoError(t, err)
	require.False(t, didUnenroll)

	profileCountForHost := 0

	// Validate host_mdm row updated
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(testCtx(), q, &enrolled, `SELECT enrolled FROM host_mdm WHERE host_id = ?`, res.Host.ID)
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(testCtx(), q, &serverURL, `SELECT server_url FROM host_mdm WHERE host_id = ?`, res.Host.ID)
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(testCtx(), q, &mdmIDIsNull, `SELECT CASE WHEN mdm_id IS NULL THEN 1 ELSE 0 END FROM host_mdm WHERE host_id = ?`, res.Host.ID)
	})
	// validate profile records deleted
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(testCtx(), q, &profileCountForHost, `SELECT COUNT(*) FROM host_mdm_android_profiles WHERE host_uuid=?`, res.Host.UUID)
	})
	assert.Equal(t, 0, enrolled)
	assert.Equal(t, "", serverURL)
	assert.Equal(t, 1, mdmIDIsNull)
	assert.Equal(t, 0, profileCountForHost)
}
