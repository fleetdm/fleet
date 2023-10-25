package mysql

import (
	"context"
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// MDMWindowsGetEnrolledDeviceWithDeviceID receives a Windows MDM device id and
// returns the device information.
func (ds *Datastore) MDMWindowsGetEnrolledDeviceWithDeviceID(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
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
		FROM mdm_windows_enrollments WHERE mdm_device_id = ?`

	var winMDMDevice fleet.MDMWindowsEnrolledDevice
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &winMDMDevice, stmt, mdmDeviceID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice").WithMessage(mdmDeviceID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get MDMWindowsGetEnrolledDeviceWithDeviceID")
	}
	return &winMDMDevice, nil
}

// MDMWindowsInsertEnrolledDevice inserts a new MDMWindowsEnrolledDevice in the
// database.
func (ds *Datastore) MDMWindowsInsertEnrolledDevice(ctx context.Context, device *fleet.MDMWindowsEnrolledDevice) error {
	// TODO(mna): I think this needs to support an ON DUPLICATE UPDATE to
	// handle the potential case of a device_id changing for a given hardware_id.
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

// TODO(mna): should we have something like host_dep_assignments for Windows? I don't remember exactly what
// problem this was solving, but seeing those enrollments deletion made me think of it.

// MDMWindowsDeleteEnrolledDevice deletes an MDMWindowsEnrolledDevice entry
// from the database using the device's hardward ID.
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

// MDMWindowsDeleteEnrolledDeviceWithDeviceID deletes a given
// MDMWindowsEnrolledDevice entry from the database using the device id.
func (ds *Datastore) MDMWindowsDeleteEnrolledDeviceWithDeviceID(ctx context.Context, mdmDeviceID string) error {
	stmt := "DELETE FROM mdm_windows_enrollments WHERE mdm_device_id = ?"

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, mdmDeviceID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete MDMWindowsDeleteEnrolledDeviceWithDeviceID")
	}

	deleted, _ := res.RowsAffected()
	if deleted == 1 {
		return nil
	}

	return ctxerr.Wrap(ctx, notFound("MDMWindowsDeleteEnrolledDeviceWithDeviceID"))
}

// TODO(mna): this receives hostUUIDs, not deviceIDs, and must translate them via the (altered) enrollments table.
func (ds *Datastore) MDMWindowsInsertCommandForHosts(ctx context.Context, hostUUIDs []string, cmd *fleet.MDMWindowsCommand) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// first, create the command entry
		stmt := `
		INSERT INTO windows_mdm_commands (
			command_uuid,
			raw_command,
			target_loc_uri
		)
		VALUES
			(?, ?, ?)
`
		if _, err := tx.ExecContext(ctx, stmt, cmd.CommandUUID, cmd.RawCommand, cmd.TargetLocURI); err != nil {
			if isDuplicate(err) {
				return ctxerr.Wrap(ctx, alreadyExists("MDMWindowsCommand", cmd.CommandUUID))
			}
			return ctxerr.Wrap(ctx, err, "inserting MDMWindowsCommand")
		}

		// create the command execution queue entries, one per host
		for _, hostUUID := range hostUUIDs {
			if err := ds.mdmWindowsInsertHostCommandDB(ctx, tx, hostUUID, cmd.CommandUUID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (ds *Datastore) mdmWindowsInsertHostCommandDB(ctx context.Context, tx sqlx.ExecerContext, hostUUID, commandUUID string) error {
	stmt := `
	INSERT INTO windows_mdm_command_queue (
		enrollment_id,
		command_uuid,
	)
	VALUES
		(
			SELECT
				id, ?
			FROM
				mdm_windows_enrollments
			WHERE
				host_uuid = ?
		)
`
	if _, err := tx.ExecContext(ctx, stmt, commandUUID, hostUUID); err != nil {
		if isDuplicate(err) {
			return ctxerr.Wrap(ctx, alreadyExists("MDMWindowsCommandQueue", commandUUID))
		}
		return ctxerr.Wrap(ctx, err, "inserting MDMWindowsCommandQueue")
	}

	return nil
}

// MDMWindowsGetPendingCommands retrieves all commands awaiting execution for a
// given device ID.
func (ds *Datastore) MDMWindowsGetPendingCommands(ctx context.Context, deviceID string) ([]*fleet.MDMWindowsCommand, error) {
	var commands []*fleet.MDMWindowsCommand

	query := `
SELECT
	wmc.command_uuid,
	wmc.raw_command,
	wmc.target_loc_uri,
	wmc.created_at,
	wmc.updated_at
FROM
	windows_mdm_command_queue wmcq
INNER JOIN
	mdm_windows_enrollments mwe
ON
	mwe.id = wmcq.enrollment_id
INNER JOIN
	windows_mdm_commands wmc
ON
	wmc.command_uuid = wmcq.command_uuid
WHERE
	mwe.device_id = ? AND
	wmcq.active AND
	NOT EXISTS (
		SELECT 1
		FROM
			windows_mdm_command_results wmcr
		WHERE
			wmcr.enrollment_id = wmcq.enrollment_id AND
			wmcr.command_uuid = wmcq.command_uuid
	)
`

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &commands, query, deviceID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get pending Windows MDM commands by device id")
	}

	return commands, nil
}

// TODO(mna): those UpdateCommandErrorCode and UpdateCommandReceivedResult must
// be replaced by a single method something like MDMWindowsSaveResponse(ctx,
// deviceID, fullResponse) that will first store the full response and then -
// in the same transaction - store the result for each known command present in
// the response's XML (each CmdRef matching a command_uuid in
// windows_mdm_command_queue entry for that device that has no result in
// windows_mdm_command_results yet). Both the Status and Results parts are
// stored.

/*
// MDMWindowsUpdateCommandErrorCode updates the rx_error_code for a given command that matches with device_id, session_id, message_id and command_id.
func (ds *Datastore) MDMWindowsUpdateCommandErrorCode(ctx context.Context, deviceID, sessionID, messageID, commandID, errorCode string) error {
	query := `
        UPDATE
            old_windows_mdm_commands
        SET
            rx_error_code = ?
        WHERE
            device_id = ? AND
            session_id = ? AND
            message_id = ? AND
            command_id = ?
    `

	_, err := ds.writer(ctx).ExecContext(ctx, query, errorCode, deviceID, sessionID, messageID, commandID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating windows command rx_error_code")
	}

	return nil
}

// MDMWindowsUpdateCommandReceivedResult updates the rx_cmd_result field for a given command that matches with device_id, session_id, message_id and command_id.
func (ds *Datastore) MDMWindowsUpdateCommandReceivedResult(ctx context.Context, deviceID, sessionID, messageID, commandID, receivedValue string) error {
	query := `
        UPDATE
            old_windows_mdm_commands
        SET
            rx_cmd_result = ?
        WHERE
            device_id = ? AND
            session_id = ? AND
            message_id = ? AND
            command_id = ?
    `

	_, err := ds.writer(ctx).ExecContext(ctx, query, receivedValue, deviceID, sessionID, messageID, commandID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating windows command rx_cmd_result")
	}

	return nil
}
*/
