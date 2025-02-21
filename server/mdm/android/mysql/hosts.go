package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) CreateDeviceTx(ctx context.Context, device *android.Device, tx sqlx.ExtContext) (*android.Device, error) {
	// Check for existing devices and duplicates
	stmt := `SELECT id, device_id, enterprise_specific_id FROM android_devices WHERE device_id = ? OR enterprise_specific_id = ?`
	var existing []android.Device
	err := sqlx.SelectContext(ctx, tx, &existing, stmt, device.DeviceID, device.EnterpriseSpecificID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "checking for existing Android device")
	}

	switch {
	case len(existing) == 0:
		return ds.insertDevice(ctx, device, tx)
	case len(existing) == 1:
		device.ID = existing[0].ID
		return ds.updateDevice(ctx, device, tx)
	case len(existing) == 2:
		err = ds.deleteDuplicate(ctx, device, tx, existing)
		if err != nil {
			return nil, err
		}
		return ds.updateDevice(ctx, device, tx)
	default:
		// Should never happen
		return nil, ctxerr.New(ctx, "unexpected number of existing devices")
	}
}

func (ds *Datastore) deleteDuplicate(ctx context.Context, device *android.Device, tx sqlx.ExtContext, existing []android.Device) error {
	// Duplicates should never happen. We log error and try to handle it gracefully.
	level.Error(ds.logger).Log("msg", "Found two Android devices with the same device ID or enterprise specific ID", "device_id",
		device.DeviceID, "enterprise_specific_id", device.EnterpriseSpecificID)
	var idToDelete, idToUpdate uint
	if existing[0].DeviceID == device.DeviceID {
		idToDelete = existing[1].ID
		idToUpdate = existing[0].ID
	} else {
		idToDelete = existing[0].ID
		idToUpdate = existing[1].ID
	}
	err := ds.deleteDevice(ctx, tx, idToDelete)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting duplicate device")
	}

	device.ID = idToUpdate
	return nil
}

func (ds *Datastore) deleteDevice(ctx context.Context, tx sqlx.ExtContext, id uint) error {
	deleteStmt := `DELETE FROM android_devices WHERE id = ?`
	_, err := tx.ExecContext(ctx, deleteStmt, id)
	return err
}

func (ds *Datastore) insertDevice(ctx context.Context, device *android.Device, tx sqlx.ExtContext) (*android.Device, error) {
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

func (ds *Datastore) updateDevice(ctx context.Context, device *android.Device, tx sqlx.ExtContext) (*android.Device, error) {
	stmt := `
	UPDATE android_devices SET
		host_id = :host_id,
		device_id = :device_id,
		enterprise_specific_id = :enterprise_specific_id,
		policy_id = :policy_id,
		last_policy_sync_time = :last_policy_sync_time
	WHERE id = :id`
	stmt, args, err := sqlx.Named(stmt, device)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "binding parameters for updating Android device")
	}
	_, err = tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating Android device")
	}
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
