package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// MDMWindowsGetEnrolledDevice receives a Windows MDM device id and returns the device information.
func (ds *Datastore) MDMWindowsGetEnrolledDevice(ctx context.Context, mdmDeviceHWID string) (*fleet.MDMWindowsEnrolledDevice, error) {
	stmt := `SELECT 
		mdm_device_id, 
		mdm_hardware_id, 
		device_state,
		device_type, 
		device_name, 
		enroll_type, 
		enroll_user_id, 
		enroll_proto_version, 
		enroll_client_version, 
		not_in_oobe,
		created_at, 
		updated_at
		FROM mdm_windows_enrollments WHERE mdm_hardware_id = ?`

	var winMDMDevice fleet.MDMWindowsEnrolledDevice
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &winMDMDevice, stmt, mdmDeviceHWID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice").WithMessage(mdmDeviceHWID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get MDMWindowsEnrolledDevice")
	}
	return &winMDMDevice, nil
}

// MDMWindowsInsertEnrolledDevice inserts a new MDMWindowsEnrolledDevice in the database
func (ds *Datastore) MDMWindowsInsertEnrolledDevice(ctx context.Context, device *fleet.MDMWindowsEnrolledDevice) error {
	stmt := `
		INSERT INTO mdm_windows_enrollments (
		mdm_device_id,
		mdm_hardware_id,
		device_state,
		device_type,
		device_name,
		enroll_type,
		enroll_user_id,
		enroll_proto_version,
		enroll_client_version,
		not_in_oobe ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := ds.writer(ctx).ExecContext(
		ctx,
		stmt,
		device.MDMDeviceID,
		device.MDMHardwareID,
		device.MDMDeviceState,
		device.MDMDeviceType,
		device.MDMDeviceName,
		device.MDMEnrollType,
		device.MDMEnrollUserID,
		device.MDMEnrollProtoVersion,
		device.MDMEnrollClientVersion,
		device.MDMNotInOOBE)
	if err != nil {
		if isDuplicate(err) {
			return ctxerr.Wrap(ctx, alreadyExists("MDMWindowsEnrolledDevice", device.MDMHardwareID))
		}
		return ctxerr.Wrap(ctx, err, "inserting MDMWindowsEnrolledDevice")
	}

	return nil
}

// MDMWindowsDeleteEnrolledDevice deletes a give MDMWindowsEnrolledDevice entry from the database using the device id.
func (ds *Datastore) MDMWindowsDeleteEnrolledDevice(ctx context.Context, mdmDeviceHWID string) error {
	stmt := "DELETE FROM mdm_windows_enrollments WHERE mdm_hardware_id = ?"

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, mdmDeviceHWID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete MDMWindowsEnrolledDevice")
	}

	deleted, _ := res.RowsAffected()
	if deleted == 1 {
		return nil
	}

	return ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice"))
}

func (ds *Datastore) GetMDMWindowsBitLockerSummary(ctx context.Context, teamID *uint) (*fleet.MDMWindowsBitLockerSummary, error) {
	// Note verifying, action_required, and removing_enforcement are not applicable to Windows hosts
	sqlFmt := `
SELECT
    COUNT( if(hdek.decryptable = 1, 1, NULL)) AS verified,
    0 AS verifying,
    0 AS action_required,
    COUNT( if(hdek.host_id IS NOT NULL AND hdek.decryptable IS NULL AND hdek.client_error = '', 1, NULL) ) AS enforcing,
    COUNT( if(hdek.host_id IS NOT NULL AND hdek.client_error != '', 1, NULL) ) AS failed,
    0 AS removing_enforcement
FROM
    hosts h
    LEFT JOIN host_disk_encryption_keys hdek ON h.id = hdek.host_id
	LEFT JOIN host_mdm hmdm ON h.id = hmdm.host_id
WHERE
    h.platform = 'windows' AND hmdm.is_server = 0 AND %s`

	// TODO: Consider if we should use right joins instead?

	var args []interface{}

	teamFilter := "h.team_id IS NULL"
	if teamID != nil && *teamID > 0 {
		teamFilter = "h.team_id = ?"
		args = append(args, *teamID)
	}

	stmt := fmt.Sprintf(sqlFmt, teamFilter)

	var res fleet.MDMWindowsBitLockerSummary
	err := sqlx.GetContext(ctx, ds.reader(ctx), &res, stmt, args...)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
