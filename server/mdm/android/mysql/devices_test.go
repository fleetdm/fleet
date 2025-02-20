package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevice(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"CreateGetDevice", testCreateGetDevice},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer testing_utils.TruncateTables(t, ds.primary, ds.logger, nil)

			c.fn(t, ds)
		})
	}
}

func testCreateGetDevice(t *testing.T, ds *Datastore) {
	_, err := ds.GetDeviceByDeviceID(testCtx(), "deviceID")
	assert.True(t, fleet.IsNotFound(err))

	device1 := &android.Device{
		HostID:               1,
		DeviceID:             "deviceID",
		EnterpriseSpecificID: "enterpriseSpecificID",
		PolicyID:             nil,
		LastPolicySyncTime:   nil,
	}
	result1, err := ds.CreateDevice(testCtx(), device1)
	require.NoError(t, err)
	assert.NotZero(t, result1.ID)
	device1.ID = result1.ID
	assert.Equal(t, device1, result1)

	device2 := &android.Device{
		HostID:               2,
		DeviceID:             "deviceID2",
		EnterpriseSpecificID: "enterpriseSpecificID2",
		PolicyID:             ptr.Uint(1),
		LastPolicySyncTime:   ptr.Time(time.Now().UTC().Truncate(time.Millisecond)),
	}
	result2, err := ds.CreateDevice(testCtx(), device2)
	require.NoError(t, err)
	assert.NotZero(t, result2.ID)
	device2.ID = result2.ID
	assert.Equal(t, device2, result2)

	result1, err = ds.GetDeviceByDeviceID(testCtx(), device1.DeviceID)
	require.NoError(t, err)
	assert.Equal(t, device1, result1)
	result2, err = ds.GetDeviceByDeviceID(testCtx(), device2.DeviceID)
	require.NoError(t, err)
	assert.EqualValues(t, device2, result2)
}
