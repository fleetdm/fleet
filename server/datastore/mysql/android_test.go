package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
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
		{"AndroidHostStorageData", testAndroidHostStorageData},
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
