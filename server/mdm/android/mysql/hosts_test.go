package mysql

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHosts(t *testing.T) {
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
	_, err := ds.getDeviceByDeviceID(testCtx(), "deviceID")
	assert.True(t, fleet.IsNotFound(err))

	device1 := &android.Device{
		HostID:               1,
		DeviceID:             "deviceID",
		EnterpriseSpecificID: ptr.String("enterpriseSpecificID"),
		AndroidPolicyID:      nil,
		LastPolicySyncTime:   nil,
	}
	result1, err := ds.createDevice(testCtx(), device1)
	require.NoError(t, err)
	assert.NotZero(t, result1.ID)
	device1.ID = result1.ID
	assert.Equal(t, device1, result1)

	device2 := &android.Device{
		HostID:               2,
		DeviceID:             "deviceID2",
		EnterpriseSpecificID: ptr.String("enterpriseSpecificID2"),
		AndroidPolicyID:      ptr.Uint(1),
		LastPolicySyncTime:   ptr.Time(time.Now().UTC().Truncate(time.Millisecond)),
	}
	result2, err := ds.createDevice(testCtx(), device2)
	require.NoError(t, err)
	assert.NotZero(t, result2.ID)
	device2.ID = result2.ID
	assert.Equal(t, device2, result2)

	result1, err = ds.getDeviceByDeviceID(testCtx(), device1.DeviceID)
	require.NoError(t, err)
	assert.Equal(t, device1, result1)
	result2, err = ds.getDeviceByDeviceID(testCtx(), device2.DeviceID)
	require.NoError(t, err)
	assert.EqualValues(t, device2, result2)
}

func (ds *Datastore) createDevice(ctx context.Context, device *android.Device) (*android.Device, error) {
	return ds.CreateDeviceTx(ctx, ds.Writer(ctx), device)
}

func (ds *Datastore) getDeviceByDeviceID(ctx context.Context, deviceID string) (*android.Device, error) {
	stmt := `SELECT id, host_id, device_id, enterprise_specific_id, android_policy_id, last_policy_sync_time FROM android_devices WHERE device_id = ?`
	var device android.Device
	err := sqlx.GetContext(ctx, ds.reader(ctx), &device, stmt, deviceID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, common_mysql.NotFound("Android device").WithName(deviceID)
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting device by device ID")
	}
	return &device, nil
}
