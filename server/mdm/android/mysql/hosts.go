package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) EnrollHost(ctx context.Context, host *android.Host) error {
	return ds.WithRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error
		host.Host, err = ds.fleetDS.NewAndroidHost(ctx, tx, host.Host)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating new Android host")
		}
		host.Device, err = ds.createDeviceTx(ctx, host.Device, tx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating new Android device")
		}
		return nil
	})
}

func (ds *Datastore) CreateDevice(ctx context.Context, device *android.Device) (*android.Device, error) {
	return ds.createDeviceTx(ctx, device, ds.Writer(ctx))
}

func (ds *Datastore) createDeviceTx(ctx context.Context, device *android.Device, tx sqlx.ExtContext) (*android.Device, error) {
	stmt := `INSERT INTO android_devices (host_id, device_id, enterprise_specific_id, policy_id, last_policy_sync_time) VALUES (?, ?, ?, ?, ?)`
	result, err := tx.ExecContext(ctx, stmt, device.HostID, device.DeviceID, device.EnterpriseSpecificID, device.PolicyID,
		device.LastPolicySyncTime)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting device")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting android_devices last insert ID")
	}
	device.ID = uint(id)
	return device, nil
}

func (ds *Datastore) GetDeviceByDeviceID(ctx context.Context, deviceID string) (*android.Device, error) {
	stmt := `SELECT id, host_id, device_id, enterprise_specific_id, policy_id, last_policy_sync_time FROM android_devices WHERE device_id = ?`
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
