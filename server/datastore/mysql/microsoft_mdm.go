package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// MDMWindowsGetEnrolledDeviceWithDeviceID receives a Windows MDM device id and
// returns the device information.
func (ds *Datastore) MDMWindowsGetEnrolledDeviceWithDeviceID(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
	stmt := `SELECT
		id,
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
			not_in_oobe
		      )
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			mdm_device_id         = VALUES(mdm_device_id),
			device_state          = VALUES(device_state),
			device_type           = VALUES(device_type),
			device_name           = VALUES(device_name),
			enroll_type           = VALUES(enroll_type),
			enroll_user_id        = VALUES(enroll_user_id),
			enroll_proto_version  = VALUES(enroll_proto_version),
			enroll_client_version = VALUES(enroll_client_version),
			not_in_oobe           = VALUES(not_in_oobe)
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

// MDMWindowsDeleteEnrolledDevice deletes an MDMWindowsEnrolledDevice entry
// from the database using the device's hardware ID.
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

func (ds *Datastore) MDMWindowsInsertCommandForHosts(ctx context.Context, hostUUIDs []string, cmd *fleet.MDMWindowsCommand) error {
	if len(hostUUIDs) == 0 {
		return nil
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// first, create the command entry
		stmt := `
  INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri)
  VALUES (?, ?, ?)
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
INSERT INTO windows_mdm_command_queue (enrollment_id, command_uuid)
VALUES ((SELECT id FROM mdm_windows_enrollments WHERE host_uuid = ?), ?)
`

	if _, err := tx.ExecContext(ctx, stmt, hostUUID, commandUUID); err != nil {
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
	mwe.mdm_device_id = ? AND
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

func (ds *Datastore) MDMWindowsSaveResponse(ctx context.Context, deviceID string, fullResponse *fleet.SyncML) error {
	if len(fullResponse.Raw) == 0 {
		return ctxerr.New(ctx, "empty raw response")
	}

	const findCommandsStmt = `SELECT command_uuid FROM windows_mdm_commands WHERE command_uuid IN (?)`

	const saveFullRespStmt = `INSERT INTO windows_mdm_responses (enrollment_id, raw_response) VALUES (?, ?)`

	const dequeueCommandsStmt = `UPDATE windows_mdm_command_queue SET active = 0 WHERE command_uuid IN (?)`

	// raw_results and status_code values might be inserted on different requests?
	// TODO: which response_id should we be tracking then? for now, using
	// whatever comes first.
	const insertResultsStmt = `
INSERT INTO windows_mdm_command_results
    (enrollment_id, command_uuid, raw_result, response_id, status_code)
VALUES %s
ON DUPLICATE KEY UPDATE
    raw_result = COALESCE(VALUES(raw_result), raw_result),
    status_code = COALESCE(VALUES(status_code), status_code)
	`

	enrollment, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting enrollment with device ID")
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// grab all the incoming UUIDs
		var cmdUUIDs []string
		uuidsToStatus := make(map[string]fleet.SyncMLCmd)
		uuidsToResults := make(map[string]fleet.SyncMLCmd)
		for _, protoOp := range fullResponse.GetOrderedCmds() {
			// results and status should contain a command they're
			// referencing
			cmdRef := protoOp.Cmd.CmdRef
			if !protoOp.Cmd.ShouldBeTracked(protoOp.Verb) || cmdRef == nil {
				continue
			}

			switch protoOp.Verb {
			case fleet.CmdStatus:
				uuidsToStatus[*cmdRef] = protoOp.Cmd
				cmdUUIDs = append(cmdUUIDs, *cmdRef)
			case fleet.CmdResults:
				uuidsToResults[*cmdRef] = protoOp.Cmd
				cmdUUIDs = append(cmdUUIDs, *cmdRef)
			}
		}

		// no relevant commands to tracks is a noop
		if len(cmdUUIDs) == 0 {
			return nil
		}

		// store the full response
		sqlResult, err := tx.ExecContext(ctx, saveFullRespStmt, enrollment.ID, fullResponse.Raw)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "saving full response")
		}
		responseID, _ := sqlResult.LastInsertId()

		// find commands we sent that match the UUID responses we've got
		stmt, params, err := sqlx.In(findCommandsStmt, cmdUUIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building IN to search matching commands")
		}
		var matchingUUIDs []string
		err = sqlx.SelectContext(ctx, tx, &matchingUUIDs, stmt, params...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "selecting matching commands")
		}

		if len(matchingUUIDs) == 0 {
			ds.logger.Log("warn", "unmatched commands", "uuids", cmdUUIDs)
			return nil
		}

		// for all the matching UUIDs, try to find any <Status> or
		// <Result> entries to track them as responses.
		var args []any
		var sb strings.Builder
		for _, uuid := range matchingUUIDs {
			statusCode := ""
			if status, ok := uuidsToStatus[uuid]; ok && status.Data != nil {
				statusCode = *status.Data
			}

			rawResult := []byte{}
			if result, ok := uuidsToResults[uuid]; ok && result.Data != nil {
				rawResult = []byte(*result.Data)
			}
			args = append(args, enrollment.ID, uuid, rawResult, responseID, statusCode)
			sb.WriteString("(?, ?, ?, ?, ?),")
		}

		// store the command results
		stmt = fmt.Sprintf(insertResultsStmt, strings.TrimSuffix(sb.String(), ","))
		if _, err = tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting command results")
		}

		// dequeue the commands
		// TODO(roberto): is this  the right place to dequeue the commands?
		//
		// If we didn't immediately get a response for a command we
		// already sent, and we don't dequeue it, it's going to be sent
		// again.  Is this what we want?
		stmt, params, err = sqlx.In(dequeueCommandsStmt, matchingUUIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building IN to dequeue commands")
		}
		if _, err = tx.ExecContext(ctx, stmt, params...); err != nil {
			return ctxerr.Wrap(ctx, err, "dequeuing commands")
		}

		return nil
	})
}

func (ds *Datastore) GetMDMWindowsCommandResults(ctx context.Context, commandUUID string) ([]*fleet.MDMCommandResult, error) {
	query := `
SELECT
    mwe.host_uuid,
    wmcr.command_uuid,
    wmcr.status_code as status,
	wmcr.updated_at,
    wmc.target_loc_uri as request_type,
    wmcr.raw_result as result
FROM
    windows_mdm_command_results wmcr
INNER JOIN
    windows_mdm_commands wmc
ON
    wmcr.command_uuid = wmc.command_uuid
INNER JOIN
    mdm_windows_enrollments mwe
ON
    wmcr.enrollment_id = mwe.id
WHERE
    wmcr.command_uuid = ?
`

	var results []*fleet.MDMCommandResult
	err := sqlx.SelectContext(
		ctx,
		ds.reader(ctx),
		&results,
		query,
		commandUUID,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get command results")
	}
	return results, nil
}
