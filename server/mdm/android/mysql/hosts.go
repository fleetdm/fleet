package mysql

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) CreateDeviceTx(ctx context.Context, tx sqlx.ExtContext, device *android.Device) (*android.Device, error) {
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
	// It should not matter which duplicate we delete since the other one will be overwritten.
	err := ds.deleteDevice(ctx, tx, existing[0].ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting duplicate device")
	}
	device.ID = existing[1].ID
	return nil
}

func (ds *Datastore) deleteDevice(ctx context.Context, tx sqlx.ExtContext, id uint) error {
	deleteStmt := `DELETE FROM android_devices WHERE id = ?`
	_, err := tx.ExecContext(ctx, deleteStmt, id)
	return err
}

func (ds *Datastore) insertDevice(ctx context.Context, device *android.Device, tx sqlx.ExtContext) (*android.Device, error) {
	stmt := `INSERT INTO android_devices (host_id, device_id, enterprise_specific_id, android_policy_id, last_policy_sync_time) VALUES (?, ?, ?, ?,
?)`
	result, err := tx.ExecContext(ctx, stmt, device.HostID, device.DeviceID, device.EnterpriseSpecificID, device.AndroidPolicyID,
		device.LastPolicySyncTime)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting device")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting android_devices last insert ID")
	}
	device.ID = uint(id) // nolint:gosec
	return device, nil
}

func (ds *Datastore) updateDevice(ctx context.Context, device *android.Device, tx sqlx.ExtContext) (*android.Device, error) {
	stmt := `
	UPDATE android_devices SET
		host_id = :host_id,
		device_id = :device_id,
		enterprise_specific_id = :enterprise_specific_id,
		android_policy_id = :android_policy_id,
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

func (ds *Datastore) UpdateDeviceTx(ctx context.Context, tx sqlx.ExtContext, device *android.Device) error {
	_, err := ds.updateDevice(ctx, device, tx)
	return err
}

func (ds *Datastore) InsertHostLabelMembershipTx(ctx context.Context, tx sqlx.ExtContext, hostID uint) error {
	// Insert the host in the builtin label memberships, adding them to the "All
	// Hosts" and "Android" labels.
	var labels []struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}
	err := sqlx.SelectContext(ctx, tx, &labels, `SELECT id, name FROM labels WHERE label_type = 1 AND (name = ? OR name = ?)`,
		fleet.BuiltinLabelNameAllHosts, fleet.BuiltinLabelNameAndroid)
	switch {
	case err != nil:
		return ctxerr.Wrap(ctx, err, "get builtin labels")
	case len(labels) != 2:
		// Builtin labels can get deleted so it is important that we check that
		// they still exist before we continue.
		level.Error(ds.logger).Log("err", fmt.Sprintf("expected 2 builtin labels but got %d", len(labels)))
		return nil
	}

	// We cannot assume IDs on labels, thus we look by name.
	var allHostsLabelID, androidLabelID uint
	for _, label := range labels {
		switch label.Name {
		case fleet.BuiltinLabelNameAllHosts:
			allHostsLabelID = label.ID
		case fleet.BuiltinLabelNameAndroid:
			androidLabelID = label.ID
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO label_membership (host_id, label_id) VALUES (?, ?), (?, ?)
		ON DUPLICATE KEY UPDATE host_id = host_id`,
		hostID, allHostsLabelID, hostID, androidLabelID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set label membership")
	}
	return nil
}
