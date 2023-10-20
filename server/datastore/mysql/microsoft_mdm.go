package mysql

import (
	"context"
	"database/sql"

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

// MDMWindowsGetEnrolledDeviceWithDeviceID receives a Windows MDM device id and returns the device information.
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

// MDMWindowsDeleteEnrolledDeviceWithDeviceID deletes a give MDMWindowsEnrolledDevice entry from the database using the device id.
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

// MDMWindowsDeletePendingCommands deletes pending commands for a given device id.
func (ds *Datastore) MDMWindowsDeletePendingCommands(ctx context.Context, deviceID string) error {
	stmt := "DELETE FROM windows_mdm_pending_commands WHERE device_id = ?"

	_, err := ds.writer(ctx).ExecContext(ctx, stmt, deviceID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete MDMWindowsDeletePendingCommands")
	}

	return nil
}

func (ds *Datastore) MDMWindowsInsertPendingCommandForDevices(ctx context.Context, deviceIDs []string, cmd *fleet.MDMWindowsPendingCommand) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		for _, deviceID := range deviceIDs {
			cmd.DeviceID = deviceID
			if err := ds.mdmWindowsInsertPendingCommandDB(ctx, tx, cmd); err != nil {
				return err
			}
		}
		return nil
	})
}

// MDMWindowsInsertPendingCommand inserts a new WindowMDMPendingCommand in the database
func (ds *Datastore) MDMWindowsInsertPendingCommand(ctx context.Context, cmd *fleet.MDMWindowsPendingCommand) error {
	return ds.mdmWindowsInsertPendingCommandDB(ctx, ds.writer(ctx), cmd)
}

func (ds *Datastore) mdmWindowsInsertPendingCommandDB(ctx context.Context, tx sqlx.ExecerContext, cmd *fleet.MDMWindowsPendingCommand) error {
	stmt := `
		INSERT INTO windows_mdm_pending_commands (
		command_uuid,
		device_id,
		cmd_verb,
		setting_uri,
		setting_value,
		data_type,
		system_origin ) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := tx.ExecContext(
		ctx,
		stmt,
		cmd.CommandUUID,
		cmd.DeviceID,
		cmd.CmdVerb,
		cmd.SettingURI,
		cmd.SettingValue,
		cmd.DataType,
		cmd.SystemOrigin)
	if err != nil {
		if isDuplicate(err) {
			return ctxerr.Wrap(ctx, alreadyExists("MDMWindowsPendingCommand", cmd.CommandUUID))
		}
		return ctxerr.Wrap(ctx, err, "inserting MDMWindowsPendingCommand")
	}

	return nil
}

// MDMWindowsGetPendingCommands retrieves all commands for a given device ID from the windows_mdm_pending_commands table
func (ds *Datastore) MDMWindowsGetPendingCommands(ctx context.Context, deviceID string) ([]*fleet.MDMWindowsPendingCommand, error) {
	var commands []*fleet.MDMWindowsPendingCommand

	query := `
        SELECT
            command_uuid,
            device_id,
            cmd_verb,
            setting_uri,
            setting_value,
            data_type,
            system_origin,
            created_at,
            updated_at
        FROM
            windows_mdm_pending_commands wmpc
        WHERE
            wmpc.device_id = ? AND
            NOT EXISTS (SELECT 1 FROM windows_mdm_commands wmc WHERE wmpc.device_id = wmc.device_id AND wmpc.command_uuid = wmc.command_uuid)
    `

	// Retrieve commands first
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &commands, query, deviceID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get pending Windows MDM commands by device id")
	}

	return commands, nil
}

// MDMWindowsInsertCommand inserts a new WindowMDMCommand in the database
func (ds *Datastore) MDMWindowsInsertCommand(ctx context.Context, cmd *fleet.MDMWindowsCommand) error {
	stmt := `
		INSERT INTO windows_mdm_commands (
		command_uuid,
		device_id,
		session_id,
		message_id,
		command_id,
		cmd_verb,
		setting_uri,
		setting_value,
		system_origin,
		rx_error_code ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := ds.writer(ctx).ExecContext(
		ctx,
		stmt,
		cmd.CommandUUID,
		cmd.DeviceID,
		cmd.SessionID,
		cmd.MessageID,
		cmd.CommandID,
		cmd.CmdVerb,
		cmd.SettingURI,
		cmd.SettingValue,
		cmd.SystemOrigin,
		"")
	if err != nil {
		if isDuplicate(err) {
			return ctxerr.Wrap(ctx, alreadyExists("MDMWindowsCommand", cmd.CommandUUID))
		}
		return ctxerr.Wrap(ctx, err, "inserting MDMWindowsCommand")
	}

	return nil
}

// MDMWindowsUpdateCommandErrorCode updates the rx_error_code for a given command that matches with device_id, session_id, message_id and command_id.
func (ds *Datastore) MDMWindowsUpdateCommandErrorCode(ctx context.Context, deviceID, sessionID, messageID, commandID, errorCode string) error {
	query := `
        UPDATE
            windows_mdm_commands
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

// MDMWindowsListCommands retrieves all commands for a given device ID from the windows_mdm_commands table
func (ds *Datastore) MDMWindowsListCommands(ctx context.Context, deviceID string) ([]*fleet.MDMWindowsCommand, error) {
	var commands []*fleet.MDMWindowsCommand

	query := `
        SELECT
            command_uuid,
            device_id,
            session_id,
            message_id,
            command_id,
            cmd_verb,
            setting_uri,
            setting_value,
            system_origin,
            rx_error_code,
            created_at,
            updated_at
        FROM
            windows_mdm_commands
        WHERE
            device_id = ?
    `

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &commands, query, deviceID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get Windows MDM commands by device id")
	}

	return commands, nil
}
