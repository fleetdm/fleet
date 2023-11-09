package mysql

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
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
		updated_at,
		host_uuid
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
			not_in_oobe,
			host_uuid)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			mdm_device_id         = VALUES(mdm_device_id),
			device_state          = VALUES(device_state),
			device_type           = VALUES(device_type),
			device_name           = VALUES(device_name),
			enroll_type           = VALUES(enroll_type),
			enroll_user_id        = VALUES(enroll_user_id),
			enroll_proto_version  = VALUES(enroll_proto_version),
			enroll_client_version = VALUES(enroll_client_version),
			not_in_oobe           = VALUES(not_in_oobe),
			host_uuid             = VALUES(host_uuid)
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
		device.MDMNotInOOBE,
		device.HostUUID)
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

func (ds *Datastore) MDMWindowsInsertCommandForHosts(ctx context.Context, hostUUIDsOrDeviceIDs []string, cmd *fleet.MDMWindowsCommand) error {
	if len(hostUUIDsOrDeviceIDs) == 0 {
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
		for _, hostUUIDOrDeviceID := range hostUUIDsOrDeviceIDs {
			if err := ds.mdmWindowsInsertHostCommandDB(ctx, tx, hostUUIDOrDeviceID, cmd.CommandUUID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (ds *Datastore) mdmWindowsInsertHostCommandDB(ctx context.Context, tx sqlx.ExecerContext, hostUUIDOrDeviceID, commandUUID string) error {
	stmt := `
INSERT INTO windows_mdm_command_queue (enrollment_id, command_uuid)
VALUES ((SELECT id FROM mdm_windows_enrollments WHERE host_uuid = ? OR mdm_device_id = ?), ?)
`

	if _, err := tx.ExecContext(ctx, stmt, hostUUIDOrDeviceID, hostUUIDOrDeviceID, commandUUID); err != nil {
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

	const dequeueCommandsStmt = `DELETE FROM windows_mdm_command_queue WHERE command_uuid IN (?)`

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
				var err error
				rawResult, err = xml.Marshal(result)
				if err != nil {
					ds.logger.Log("err", err, "marshaling command result", "cmd_uuid", uuid)
				}
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
    wmr.raw_response as result
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
INNER JOIN
    windows_mdm_responses wmr
ON
    wmr.id = wmcr.response_id
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

func (ds *Datastore) UpdateMDMWindowsEnrollmentsHostUUID(ctx context.Context, hostUUID string, mdmDeviceID string) error {
	stmt := `UPDATE mdm_windows_enrollments SET host_uuid = ? WHERE mdm_device_id = ?`
	if _, err := ds.writer(ctx).Exec(stmt, hostUUID, mdmDeviceID); err != nil {
		return ctxerr.Wrap(ctx, err, "setting host_uuid for windows enrollment")
	}
	return nil
}

// whereBitLockerStatus returns a string suitable for inclusion within a SQL WHERE clause to filter by
// the given status. The caller is responsible for ensuring the status is valid. In the case of an invalid
// status, the function will return the string "FALSE". The caller should also ensure that the query in
// which this is used joins the following tables with the specified aliases:
// - host_disk_encryption_keys: hdek
// - host_mdm: hmdm
// - host_disks: hd
func (ds *Datastore) whereBitLockerStatus(status fleet.DiskEncryptionStatus) string {
	const (
		whereNotServer        = `(hmdm.is_server IS NOT NULL AND hmdm.is_server = 0)`
		whereKeyAvailable     = `(hdek.base64_encrypted IS NOT NULL AND hdek.base64_encrypted != '' AND hdek.decryptable IS NOT NULL AND hdek.decryptable = 1)`
		whereEncrypted        = `(hd.encrypted IS NOT NULL AND hd.encrypted = 1)`
		whereHostDisksUpdated = `(hd.updated_at IS NOT NULL AND hdek.updated_at IS NOT NULL AND hd.updated_at >= hdek.updated_at)`
		whereClientError      = `(hdek.client_error IS NOT NULL AND hdek.client_error != '')`
		withinGracePeriod     = `(hdek.updated_at IS NOT NULL AND hdek.updated_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR))`
	)

	// TODO: what if windows sends us a key for an already encrypted volumne? could it get stuck
	// in pending or verifying? should we modify SetOrUpdateHostDiskEncryption to ensure that we
	// increment the updated_at timestamp on the host_disks table for all encrypted volumes
	// host_disks if the hdek timestamp is newer? What about SetOrUpdateHostDiskEncryptionKey?

	switch status {
	case fleet.DiskEncryptionVerified:
		return whereNotServer + `
AND NOT ` + whereClientError + `
AND ` + whereKeyAvailable + `
AND ` + whereEncrypted + `
AND ` + whereHostDisksUpdated

	case fleet.DiskEncryptionVerifying:
		// Possible verifying scenarios:
		// - we have the key and host_disks already encrypted before the key but hasn't been updated yet
		// - we have the key and host_disks reported unencrypted during the 1-hour grace period after key was updated
		return whereNotServer + `
AND NOT ` + whereClientError + `
AND ` + whereKeyAvailable + `
AND (
	(` + whereEncrypted + ` AND NOT ` + whereHostDisksUpdated + `)
	OR (NOT ` + whereEncrypted + ` AND ` + whereHostDisksUpdated + ` AND ` + withinGracePeriod + `)
)`

	case fleet.DiskEncryptionEnforcing:
		// Possible enforcing scenarios:
		// - we don't have the key
		// - we have the key and host_disks reported unencrypted before the key was updated or outside the 1-hour grace period after key was updated
		return whereNotServer + `
AND NOT ` + whereClientError + `
AND (
	NOT ` + whereKeyAvailable + `
	OR (` + whereKeyAvailable + `
		AND (NOT ` + whereEncrypted + `
			AND (NOT ` + whereHostDisksUpdated + ` OR NOT ` + withinGracePeriod + `)
		)
	)
)`

	case fleet.DiskEncryptionFailed:
		return whereNotServer + ` AND ` + whereClientError

	default:
		level.Debug(ds.logger).Log("msg", "unknown bitlocker status", "status", status)
		return "FALSE"
	}
}

func (ds *Datastore) GetMDMWindowsBitLockerSummary(ctx context.Context, teamID *uint) (*fleet.MDMWindowsBitLockerSummary, error) {
	enabled, err := ds.getConfigEnableDiskEncryption(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return &fleet.MDMWindowsBitLockerSummary{}, nil
	}

	// Note action_required and removing_enforcement are not applicable to Windows hosts
	sqlFmt := `
SELECT
    COUNT(if((%s), 1, NULL)) AS verified,
    COUNT(if((%s), 1, NULL)) AS verifying,
    0 AS action_required,
    COUNT(if((%s), 1, NULL)) AS enforcing,
    COUNT(if((%s), 1, NULL)) AS failed,
    0 AS removing_enforcement
FROM
    hosts h
    LEFT JOIN host_disk_encryption_keys hdek ON h.id = hdek.host_id
	LEFT JOIN host_mdm hmdm ON h.id = hmdm.host_id
	LEFT JOIN host_disks hd ON h.id = hd.host_id
WHERE
    h.platform = 'windows' AND hmdm.is_server = 0 AND %s`

	var args []interface{}
	teamFilter := "h.team_id IS NULL"
	if teamID != nil && *teamID > 0 {
		teamFilter = "h.team_id = ?"
		args = append(args, *teamID)
	}

	var res fleet.MDMWindowsBitLockerSummary
	stmt := fmt.Sprintf(
		sqlFmt,
		ds.whereBitLockerStatus(fleet.DiskEncryptionVerified),
		ds.whereBitLockerStatus(fleet.DiskEncryptionVerifying),
		ds.whereBitLockerStatus(fleet.DiskEncryptionEnforcing),
		ds.whereBitLockerStatus(fleet.DiskEncryptionFailed),
		teamFilter,
	)
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &res, stmt, args...); err != nil {
		return nil, err
	}

	return &res, nil
}

func (ds *Datastore) GetMDMWindowsBitLockerStatus(ctx context.Context, host *fleet.Host) (*fleet.HostMDMDiskEncryption, error) {
	if host == nil {
		return nil, errors.New("host cannot be nil")
	}

	if host.Platform != "windows" {
		// Generally, the caller should have already checked this, but just in case we log and
		// return nil
		level.Debug(ds.logger).Log("msg", "cannot get bitlocker status for non-windows host", "host_id", host.ID)
		return nil, nil
	}

	if host.MDMInfo != nil && host.MDMInfo.IsServer {
		// It is currently expected that server hosts do not have a bitlocker status so we can skip
		// the query and return nil. We log for potential debugging in case this changes in the future.
		level.Debug(ds.logger).Log("msg", "no bitlocker status for server host", "host_id", host.ID)
		return nil, nil
	}

	enabled, err := ds.getConfigEnableDiskEncryption(ctx, host.TeamID)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, nil
	}

	// Note action_required and removing_enforcement are not applicable to Windows hosts
	stmt := fmt.Sprintf(`
SELECT
	CASE
		WHEN (%s) THEN '%s'
		WHEN (%s) THEN '%s'
		WHEN (%s) THEN '%s'
		WHEN (%s) THEN '%s'
	END AS status,
	COALESCE(client_error, '') as detail
FROM
	host_mdm hmdm
	LEFT JOIN host_disk_encryption_keys hdek ON hmdm.host_id = hdek.host_id
	LEFT JOIN host_disks hd ON hmdm.host_id = hd.host_id
WHERE
	hmdm.host_id = ?`,
		ds.whereBitLockerStatus(fleet.DiskEncryptionVerified),
		fleet.DiskEncryptionVerified,
		ds.whereBitLockerStatus(fleet.DiskEncryptionVerifying),
		fleet.DiskEncryptionVerifying,
		ds.whereBitLockerStatus(fleet.DiskEncryptionEnforcing),
		fleet.DiskEncryptionEnforcing,
		ds.whereBitLockerStatus(fleet.DiskEncryptionFailed),
		fleet.DiskEncryptionFailed,
	)

	var dest struct {
		Status fleet.DiskEncryptionStatus `db:"status"`
		Detail string                     `db:"detail"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, stmt, host.ID); err != nil {
		if err != sql.ErrNoRows {
			return &fleet.HostMDMDiskEncryption{}, err
		}
		// At this point we know disk encryption is enabled so if there are no rows for the
		// host then we treat it as enforcing and log for potential debugging
		level.Debug(ds.logger).Log("msg", "no bitlocker status found for host", "host_id", host.ID)
		dest.Status = fleet.DiskEncryptionEnforcing
	}

	return &fleet.HostMDMDiskEncryption{
		Status: &dest.Status,
		Detail: dest.Detail,
	}, nil
}

func (ds *Datastore) GetMDMWindowsConfigProfile(ctx context.Context, profileUUID string) (*fleet.MDMWindowsConfigProfile, error) {
	stmt := `
SELECT
	profile_uuid,
	team_id,
	name,
	syncml,
	created_at,
	updated_at
FROM
	mdm_windows_configuration_profiles
WHERE
	profile_uuid=?`

	var res fleet.MDMWindowsConfigProfile
	err := sqlx.GetContext(ctx, ds.reader(ctx), &res, stmt, profileUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMWindowsProfile").WithName(profileUUID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get mdm windows config profile")
	}

	return &res, nil
}

func (ds *Datastore) DeleteMDMWindowsConfigProfile(ctx context.Context, profileUUID string) error {
	res, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM mdm_windows_configuration_profiles WHERE profile_uuid=?`, profileUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	deleted, _ := res.RowsAffected() // cannot fail for mysql
	if deleted != 1 {
		return ctxerr.Wrap(ctx, notFound("MDMWindowsProfile").WithName(profileUUID))
	}
	return nil
}

func (ds *Datastore) ListMDMWindowsProfilesToInstall(ctx context.Context) ([]*fleet.MDMWindowsProfilePayload, error) {
	// The query below is a set difference between:
	//
	// - Set A (ds), the desired state, can be obtained from a JOIN between
	//   mdm_windows_configuration_profiles and hosts.
	// - Set B, the current state given by host_mdm_windows_profiles.
	//
	// A - B gives us the profiles that need to be installed:
	//
	//   - profiles that are in A but not in B
	//
	//   - profiles that are in A and in B, with an operation type of "install"
	//   and a NULL status. Other statuses mean that the operation is already in
	//   flight (pending), the operation has been completed but is still subject
	//   to independent verification by Fleet (verifying), or has reached a terminal
	//   state (failed or verified). If the profile's content is edited, all relevant hosts will
	//   be marked as status NULL so that it gets re-installed.
	query := `
        SELECT
            ds.profile_uuid,
            ds.host_uuid
        FROM (
            SELECT mwcp.profile_uuid, h.uuid as host_uuid
            FROM mdm_windows_configuration_profiles mwcp
            JOIN hosts h ON h.team_id = mwcp.team_id OR (h.team_id IS NULL AND mwcp.team_id = 0)
            JOIN mdm_windows_enrollments mwe ON mwe.host_uuid = h.uuid
            WHERE h.platform = 'windows'
        ) as ds
        LEFT JOIN host_mdm_windows_profiles hmwp
            ON hmwp.profile_uuid = ds.profile_uuid AND hmwp.host_uuid = ds.host_uuid
        WHERE
        -- profiles in A but not in B
        ( hmwp.profile_uuid IS NULL AND hmwp.host_uuid IS NULL ) OR
        -- profiles in A and B with operation type "install" and NULL status
        ( hmwp.host_uuid IS NOT NULL AND hmwp.operation_type = ? AND hmwp.status IS NULL )
`

	var profiles []*fleet.MDMWindowsProfilePayload
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, query, fleet.MDMOperationTypeInstall)
	return profiles, err
}

func (ds *Datastore) ListMDMWindowsProfilesToRemove(ctx context.Context) ([]*fleet.MDMWindowsProfilePayload, error) {
	// The query below is a set difference between:
	//
	// - Set A (ds), the desired state, can be obtained from a JOIN between
	// mdm_windows_configuration_profiles and hosts.
	// - Set B, the current state given by host_mdm_windows_profiles.
	//
	// B - A gives us the profiles that need to be removed
	//
	// Any other case are profiles that are in both B and A, and as such are
	// processed by the ListMDMWindowsProfilesToInstall method (since they are in
	// both, their desired state is necessarily to be installed).
	query := `
        SELECT
	    hmwp.profile_uuid,
	    hmwp.host_uuid,
	    hmwp.operation_type,
	    COALESCE(hmwp.detail, '') as detail,
	    hmwp.status,
	    hmwp.command_uuid
          FROM (
            SELECT h.uuid, mwcp.profile_uuid
            FROM mdm_windows_configuration_profiles mwcp
            JOIN hosts h ON h.team_id = mwcp.team_id OR (h.team_id IS NULL AND mwcp.team_id = 0)
	    JOIN mdm_windows_enrollments mwe ON mwe.host_uuid = h.uuid
            WHERE h.platform = 'windows'
          ) as ds
          RIGHT JOIN host_mdm_windows_profiles hmwp
            ON hmwp.profile_uuid = ds.profile_uuid AND hmwp.host_uuid = ds.uuid
          -- profiles that are in B but not in A
          WHERE ds.profile_uuid IS NULL AND ds.uuid IS NULL
`

	var profiles []*fleet.MDMWindowsProfilePayload
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, query)
	return profiles, err
}

func (ds *Datastore) BulkUpsertMDMWindowsHostProfiles(ctx context.Context, payload []*fleet.MDMWindowsBulkUpsertHostProfilePayload) error {
	if len(payload) == 0 {
		return nil
	}

	executeUpsertBatch := func(valuePart string, args []any) error {
		stmt := fmt.Sprintf(`
	    INSERT INTO host_mdm_windows_profiles (
              profile_uuid,
	      host_uuid,
	      status,
	      operation_type,
	      detail,
	      command_uuid,
	      profile_name
            )
            VALUES %s
	    ON DUPLICATE KEY UPDATE
              status = VALUES(status),
              operation_type = VALUES(operation_type),
              detail = VALUES(detail),
              profile_name = VALUES(profile_name),
              command_uuid = VALUES(command_uuid)`,
			strings.TrimSuffix(valuePart, ","),
		)

		_, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
		return err
	}

	var (
		args       []any
		sb         strings.Builder
		batchCount int
	)

	const defaultBatchSize = 1000 // results in this times 9 placeholders
	batchSize := defaultBatchSize
	if testUpsertMDMDesiredProfilesBatchSize > 0 {
		batchSize = testUpsertMDMDesiredProfilesBatchSize
	}

	resetBatch := func() {
		batchCount = 0
		args = args[:0]
		sb.Reset()
	}

	for _, p := range payload {
		args = append(args, p.ProfileUUID, p.HostUUID, p.Status, p.OperationType, p.Detail, p.CommandUUID, p.ProfileName)
		sb.WriteString("(?, ?, ?, ?, ?, ?, ?),")
		batchCount++

		if batchCount >= batchSize {
			if err := executeUpsertBatch(sb.String(), args); err != nil {
				return err
			}
			resetBatch()
		}
	}

	if batchCount > 0 {
		if err := executeUpsertBatch(sb.String(), args); err != nil {
			return err
		}
	}
	return nil
}

func (ds *Datastore) GetMDMWindowsProfilesContents(ctx context.Context, uuids []string) (map[string][]byte, error) {
	if len(uuids) == 0 {
		return nil, nil
	}

	stmt := `
          SELECT profile_uuid, syncml
          FROM mdm_windows_configuration_profiles WHERE profile_uuid IN (?)
	`
	query, args, err := sqlx.In(stmt, uuids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building in statement")
	}

	var profs []struct {
		ProfileUUID string `db:"profile_uuid"`
		SyncML      []byte `db:"syncml"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &profs, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "running query")
	}

	results := make(map[string][]byte)
	for _, p := range profs {
		results[p.ProfileUUID] = p.SyncML
	}

	return results, nil
}

func (ds *Datastore) BulkDeleteMDMWindowsHostsConfigProfiles(ctx context.Context, profs []*fleet.MDMWindowsProfilePayload) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		return bulkDeleteMDMWindowsHostsConfigProfilesDB(ctx, tx, profs)
	})
}

func bulkDeleteMDMWindowsHostsConfigProfilesDB(ctx context.Context, tx sqlx.ExtContext, profs []*fleet.MDMWindowsProfilePayload) error {
	if len(profs) == 0 {
		return nil
	}

	executeDeleteBatch := func(valuePart string, args []any) error {
		stmt := fmt.Sprintf(`DELETE FROM host_mdm_windows_profiles WHERE (profile_uuid, host_uuid) IN (%s)`, strings.TrimSuffix(valuePart, ","))
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "error deleting host_mdm_windows_profiles")
		}
		return nil
	}

	var (
		args       []any
		sb         strings.Builder
		batchCount int
	)

	const defaultBatchSize = 1000 // results in this times 2 placeholders
	batchSize := defaultBatchSize
	if testDeleteMDMProfilesBatchSize > 0 {
		batchSize = testDeleteMDMProfilesBatchSize
	}

	resetBatch := func() {
		batchCount = 0
		args = args[:0]
		sb.Reset()
	}

	for _, p := range profs {
		args = append(args, p.ProfileUUID, p.HostUUID)
		sb.WriteString("(?, ?),")
		batchCount++

		if batchCount >= batchSize {
			if err := executeDeleteBatch(sb.String(), args); err != nil {
				return err
			}
			resetBatch()
		}
	}

	if batchCount > 0 {
		if err := executeDeleteBatch(sb.String(), args); err != nil {
			return err
		}
	}
	return nil
}
