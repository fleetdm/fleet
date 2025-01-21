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
	"github.com/fleetdm/fleet/v4/server/mdm"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
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
		if IsDuplicate(err) {
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
		return ds.mdmWindowsInsertCommandForHostsDB(ctx, tx, hostUUIDsOrDeviceIDs, cmd)
	})
}

func (ds *Datastore) mdmWindowsInsertCommandForHostsDB(ctx context.Context, tx sqlx.ExecerContext, hostUUIDsOrDeviceIDs []string, cmd *fleet.MDMWindowsCommand) error {
	// first, create the command entry
	stmt := `
		INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri)
		VALUES (?, ?, ?)
  `
	if _, err := tx.ExecContext(ctx, stmt, cmd.CommandUUID, cmd.RawCommand, cmd.TargetLocURI); err != nil {
		if IsDuplicate(err) {
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
}

func (ds *Datastore) mdmWindowsInsertHostCommandDB(ctx context.Context, tx sqlx.ExecerContext, hostUUIDOrDeviceID, commandUUID string) error {
	stmt := `
INSERT INTO windows_mdm_command_queue (enrollment_id, command_uuid)
VALUES ((SELECT id FROM mdm_windows_enrollments WHERE host_uuid = ? OR mdm_device_id = ? ORDER BY created_at DESC LIMIT 1), ?)
`

	if _, err := tx.ExecContext(ctx, stmt, hostUUIDOrDeviceID, hostUUIDOrDeviceID, commandUUID); err != nil {
		if IsDuplicate(err) {
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

func (ds *Datastore) MDMWindowsSaveResponse(ctx context.Context, deviceID string, enrichedSyncML fleet.EnrichedSyncML) error {
	if len(enrichedSyncML.Raw) == 0 {
		return ctxerr.New(ctx, "empty raw response")
	}

	enrolledDevice, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting enrolled device with device ID")
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// store the full response
		const saveFullRespStmt = `INSERT INTO windows_mdm_responses (enrollment_id, raw_response) VALUES (?, ?)`
		sqlResult, err := tx.ExecContext(ctx, saveFullRespStmt, enrolledDevice.ID, enrichedSyncML.Raw)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "saving full response")
		}
		responseID, _ := sqlResult.LastInsertId()

		// find commands we sent that match the UUID responses we've got
		const findCommandsStmt = `SELECT command_uuid, raw_command, target_loc_uri FROM windows_mdm_commands WHERE command_uuid IN (?)`
		stmt, params, err := sqlx.In(findCommandsStmt, enrichedSyncML.CmdRefUUIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building IN to search matching commands")
		}
		var matchingCmds []fleet.MDMWindowsCommand
		err = sqlx.SelectContext(ctx, tx, &matchingCmds, stmt, params...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "selecting matching commands")
		}

		if len(matchingCmds) == 0 {
			level.Warn(ds.logger).Log("msg", "unmatched Windows MDM commands", "uuids", enrichedSyncML.CmdRefUUIDs, "mdm_device_id",
				deviceID)
			return nil
		}

		// for all the matching UUIDs, try to find any <Status> or
		// <Result> entries to track them as responses.
		var (
			args                     []any
			sb                       strings.Builder
			potentialProfilePayloads []*fleet.MDMWindowsProfilePayload

			wipeCmdUUID   string
			wipeCmdStatus string
		)

		for _, cmd := range matchingCmds {
			statusCode := ""
			if status, ok := enrichedSyncML.CmdRefUUIDToStatus[cmd.CommandUUID]; ok && status.Data != nil {
				statusCode = *status.Data
				if status.Cmd != nil && *status.Cmd == fleet.CmdAtomic {
					// The raw MDM command may contain a $FLEET_SECRET_XXX, which should never be exposed or stored unencrypted.
					// Note: As of 2024/12/17, on <Add>, <Replace>, and <Exec> commands are exposed to Windows MDM users, so we should not see any secrets in <Atomic> commands. This code is here for future-proofing.
					rawCommandStr := string(cmd.RawCommand)
					rawCommandWithSecret, err := ds.ExpandEmbeddedSecrets(ctx, rawCommandStr)
					if err != nil {
						// This error should never happen since we validate the presence of needed secrets on profile upload.
						return ctxerr.Wrap(ctx, err, "expanding embedded secrets")
					}
					// Secret may be found in the command, so we make a new struct with the expanded secret.
					cmdWithSecret := cmd
					cmdWithSecret.RawCommand = []byte(rawCommandWithSecret)
					pp, err := fleet.BuildMDMWindowsProfilePayloadFromMDMResponse(cmdWithSecret, enrichedSyncML.CmdRefUUIDToStatus,
						enrolledDevice.HostUUID)
					if err != nil {
						return err
					}
					potentialProfilePayloads = append(potentialProfilePayloads, pp)
				}
			}

			rawResult := []byte{}
			if result, ok := enrichedSyncML.CmdRefUUIDToResults[cmd.CommandUUID]; ok && result.Data != nil {
				var err error
				rawResult, err = xml.Marshal(result)
				if err != nil {
					ds.logger.Log("err", err, "marshaling command result", "cmd_uuid", cmd.CommandUUID)
				}
			}
			args = append(args, enrolledDevice.ID, cmd.CommandUUID, rawResult, responseID, statusCode)
			sb.WriteString("(?, ?, ?, ?, ?),")

			// if the command is a Wipe, keep track of it so we can update
			// host_mdm_actions accordingly.
			if strings.Contains(cmd.TargetLocURI, "/Device/Vendor/MSFT/RemoteWipe/") {
				wipeCmdUUID = cmd.CommandUUID
				wipeCmdStatus = statusCode
			}
		}

		if err := updateMDMWindowsHostProfileStatusFromResponseDB(ctx, tx, potentialProfilePayloads); err != nil {
			return ctxerr.Wrap(ctx, err, "updating host profile status")
		}

		// store the command results
		const insertResultsStmt = `
INSERT INTO windows_mdm_command_results
    (enrollment_id, command_uuid, raw_result, response_id, status_code)
VALUES %s
ON DUPLICATE KEY UPDATE
    raw_result = COALESCE(VALUES(raw_result), raw_result),
    status_code = COALESCE(VALUES(status_code), status_code)
`
		stmt = fmt.Sprintf(insertResultsStmt, strings.TrimSuffix(sb.String(), ","))
		if _, err = tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting command results")
		}

		// if we received a Wipe command result, update the host's status
		if wipeCmdUUID != "" {
			if err := updateHostLockWipeStatusFromResultAndHostUUID(ctx, tx, enrolledDevice.HostUUID,
				"wipe_ref", wipeCmdUUID, strings.HasPrefix(wipeCmdStatus, "2"), false,
			); err != nil {
				return ctxerr.Wrap(ctx, err, "updating wipe command result in host_mdm_actions")
			}
		}

		// dequeue the commands
		var matchingUUIDs []string
		for _, cmd := range matchingCmds {
			matchingUUIDs = append(matchingUUIDs, cmd.CommandUUID)
		}
		const dequeueCommandsStmt = `DELETE FROM windows_mdm_command_queue WHERE enrollment_id = ? AND command_uuid IN (?)`
		stmt, params, err = sqlx.In(dequeueCommandsStmt, enrolledDevice.ID, matchingUUIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building IN to dequeue commands")
		}
		if _, err = tx.ExecContext(ctx, stmt, params...); err != nil {
			return ctxerr.Wrap(ctx, err, "dequeuing commands")
		}

		return nil
	})
}

// updateMDMWindowsHostProfileStatusFromResponseDB takes a slice of potential
// profile payloads and updates the corresponding `status` and `detail` columns
// in `host_mdm_windows_profiles`
// TODO(roberto): much of this logic should be living in the service layer,
// would be nice to get the time to properly plan and implement.
func updateMDMWindowsHostProfileStatusFromResponseDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	payloads []*fleet.MDMWindowsProfilePayload,
) error {
	if len(payloads) == 0 {
		return nil
	}

	// this statement will act as a batch-update, no new host profiles
	// should be inserted from a device MDM response, so we first check for
	// matching entries and then perform the INSERT ... ON DUPLICATE KEY to
	// update their detail and status.
	const updateHostProfilesStmt = `
		INSERT INTO host_mdm_windows_profiles
			(host_uuid, profile_uuid, detail, status, retries, command_uuid)
		VALUES %s
		ON DUPLICATE KEY UPDATE
			detail = VALUES(detail),
			status = VALUES(status),
			retries = VALUES(retries)`

	// MySQL will use the `host_uuid` part of the primary key as a first
	// pass, and then filter that subset by `command_uuid`.
	const getMatchingHostProfilesStmt = `
		SELECT host_uuid, profile_uuid, command_uuid, retries
		FROM host_mdm_windows_profiles
		WHERE host_uuid = ? AND command_uuid IN (?)`

	// grab command UUIDs to find matching entries using `getMatchingHostProfilesStmt`
	commandUUIDs := make([]string, 0, len(payloads))
	// also grab the payloads keyed by the command uuid, so we can easily
	// grab the corresponding `Detail` and `Status` from the matching
	// command later on.
	uuidsToPayloads := make(map[string]*fleet.MDMWindowsProfilePayload, len(payloads))
	hostUUID := payloads[0].HostUUID
	for _, payload := range payloads {
		if payload.HostUUID != hostUUID {
			return errors.New("all payloads must be for the same host uuid")
		}
		commandUUIDs = append(commandUUIDs, payload.CommandUUID)
		uuidsToPayloads[payload.CommandUUID] = payload
	}

	// find the matching entries for the given host_uuid, command_uuid combinations.
	stmt, args, err := sqlx.In(getMatchingHostProfilesStmt, hostUUID, commandUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building sqlx.In query")
	}
	var matchingHostProfiles []fleet.MDMWindowsProfilePayload
	if err := sqlx.SelectContext(ctx, tx, &matchingHostProfiles, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "running query to get matching profiles")
	}

	// batch-update the matching entries with the desired detail and status
	var sb strings.Builder
	args = args[:0]
	for _, hp := range matchingHostProfiles {
		payload := uuidsToPayloads[hp.CommandUUID]
		if payload.Status != nil && *payload.Status == fleet.MDMDeliveryFailed {
			if hp.Retries < mdm.MaxProfileRetries {
				// if we haven't hit the max retries, we set
				// the host profile status to nil (which causes
				// an install profile command to be enqueued
				// the next time the profile manager cron runs)
				// and increment the retry count
				payload.Status = nil
				hp.Retries++
			}
		}
		args = append(args, hp.HostUUID, hp.ProfileUUID, payload.Detail, payload.Status, hp.Retries)
		sb.WriteString("(?, ?, ?, ?, ?, command_uuid),")
	}

	stmt = fmt.Sprintf(updateHostProfilesStmt, strings.TrimSuffix(sb.String(), ","))
	_, err = tx.ExecContext(ctx, stmt, args...)
	return ctxerr.Wrap(ctx, err, "updating host profiles")
}

func (ds *Datastore) GetMDMWindowsCommandResults(ctx context.Context, commandUUID string) ([]*fleet.MDMCommandResult, error) {
	query := `
SELECT
    mwe.host_uuid,
    wmcr.command_uuid,
    wmcr.status_code as status,
    wmcr.updated_at,
    wmc.target_loc_uri as request_type,
    wmr.raw_response as result,
    wmc.raw_command as payload
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
	enabled, err := ds.GetConfigEnableDiskEncryption(ctx, teamID)
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
		return nil, ctxerr.New(ctx, "cannot get bitlocker status for nil host")
	}

	if host.Platform != "windows" {
		// the caller should have already checked this
		return nil, ctxerr.Errorf(ctx, "cannot get bitlocker status for non-windows host %d", host.ID)
	}

	mdmInfo, err := ds.GetHostMDM(ctx, host.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, ctxerr.Wrap(ctx, err, "cannot get bitlocker status because mdm info lookup failed")
	}

	if mdmInfo.IsServer {
		// It is currently expected that server hosts do not have a bitlocker status so we can skip
		// the query and return nil. We log for potential debugging in case this changes in the future.
		level.Debug(ds.logger).Log("msg", "no bitlocker status for server host", "host_id", host.ID)
		return nil, nil
	}

	enabled, err := ds.GetConfigEnableDiskEncryption(ctx, host.TeamID)
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
		ELSE ''
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

	if dest.Status == "" {
		// This is unexpected. We know that disk encryption is enabled so we treat it failed to draw
		// attention to the issue and log potential debugging
		level.Debug(ds.logger).Log("msg", "no bitlocker status found for host", "host_id", host.ID, "mdm_info")
		dest.Status = fleet.DiskEncryptionFailed
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
	uploaded_at
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

	labels, err := ds.listProfileLabelsForProfiles(ctx, []string{res.ProfileUUID}, nil, nil)
	if err != nil {
		return nil, err
	}
	for _, lbl := range labels {
		switch {
		case lbl.Exclude && lbl.RequireAll:
			// this should never happen so log it for debugging
			level.Debug(ds.logger).Log("msg", "unsupported profile label: cannot be both exclude and require all",
				"profile_uuid", lbl.ProfileUUID,
				"label_name", lbl.LabelName,
			)
		case lbl.Exclude && !lbl.RequireAll:
			res.LabelsExcludeAny = append(res.LabelsExcludeAny, lbl)
		case !lbl.Exclude && !lbl.RequireAll:
			res.LabelsIncludeAny = append(res.LabelsIncludeAny, lbl)
		default:
			// default include all
			res.LabelsIncludeAll = append(res.LabelsIncludeAll, lbl)
		}
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

func (ds *Datastore) DeleteMDMWindowsConfigProfileByTeamAndName(ctx context.Context, teamID *uint, profileName string) error {
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}
	_, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM mdm_windows_configuration_profiles WHERE team_id=? AND name=?`, globalOrTeamID, profileName)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	return nil
}

func subqueryHostsMDMWindowsOSSettingsStatusFailed() (string, []interface{}, error) {
	sql := `
            SELECT
                1 FROM host_mdm_windows_profiles hmwp
            WHERE
                h.uuid = hmwp.host_uuid
                AND hmwp.status = ?
                AND hmwp.profile_name NOT IN(?)`
	args := []interface{}{
		fleet.MDMDeliveryFailed,
		mdm.ListFleetReservedWindowsProfileNames(),
	}

	return sqlx.In(sql, args...)
}

func subqueryHostsMDMWindowsOSSettingsStatusPending() (string, []interface{}, error) {
	sql := `
            SELECT
                1 FROM host_mdm_windows_profiles hmwp
            WHERE
                h.uuid = hmwp.host_uuid
                AND (hmwp.status IS NULL OR hmwp.status = ?)
				AND hmwp.profile_name NOT IN(?)
                AND NOT EXISTS (
                    SELECT
                        1 FROM host_mdm_windows_profiles hmwp2
                    WHERE (h.uuid = hmwp2.host_uuid
                        AND hmwp2.status = ?
                        AND hmwp2.profile_name NOT IN(?)))`
	args := []interface{}{
		fleet.MDMDeliveryPending,
		mdm.ListFleetReservedWindowsProfileNames(),
		fleet.MDMDeliveryFailed,
		mdm.ListFleetReservedWindowsProfileNames(),
	}
	return sqlx.In(sql, args...)
}

func subqueryHostsMDMWindowsOSSettingsStatusVerifying() (string, []interface{}, error) {
	sql := `
            SELECT
                1 FROM host_mdm_windows_profiles hmwp
            WHERE
                h.uuid = hmwp.host_uuid
                AND hmwp.operation_type = ?
                AND hmwp.status = ?
                AND hmwp.profile_name NOT IN(?)
                AND NOT EXISTS (
                    SELECT
                        1 FROM host_mdm_windows_profiles hmwp2
                    WHERE (h.uuid = hmwp2.host_uuid
                        AND hmwp2.operation_type = ?
                        AND hmwp2.profile_name NOT IN(?)
                        AND(hmwp2.status IS NULL
                            OR hmwp2.status NOT IN(?))))`

	args := []interface{}{
		fleet.MDMOperationTypeInstall,
		fleet.MDMDeliveryVerifying,
		mdm.ListFleetReservedWindowsProfileNames(),
		fleet.MDMOperationTypeInstall,
		mdm.ListFleetReservedWindowsProfileNames(),
		[]interface{}{fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerified},
	}
	return sqlx.In(sql, args...)
}

func subqueryHostsMDMWindowsOSSettingsStatusVerified() (string, []interface{}, error) {
	sql := `
            SELECT
                1 FROM host_mdm_windows_profiles hmwp
            WHERE
                h.uuid = hmwp.host_uuid
                AND hmwp.operation_type = ?
                AND hmwp.status = ?
                AND hmwp.profile_name NOT IN(?)
                AND NOT EXISTS (
                    SELECT
                        1 FROM host_mdm_windows_profiles hmwp2
                    WHERE (h.uuid = hmwp2.host_uuid
                        AND hmwp2.operation_type = ?
                        AND hmwp2.profile_name NOT IN(?)
                        AND(hmwp2.status IS NULL
                            OR hmwp2.status != ?)))`
	args := []interface{}{
		fleet.MDMOperationTypeInstall,
		fleet.MDMDeliveryVerified,
		mdm.ListFleetReservedWindowsProfileNames(),
		fleet.MDMOperationTypeInstall,
		mdm.ListFleetReservedWindowsProfileNames(),
		fleet.MDMDeliveryVerified,
	}
	return sqlx.In(sql, args...)
}

func (ds *Datastore) GetMDMWindowsProfilesSummary(ctx context.Context, teamID *uint) (*fleet.MDMProfilesSummary, error) {
	includeBitLocker, err := ds.GetConfigEnableDiskEncryption(ctx, teamID)
	if err != nil {
		return nil, err
	}

	var counts []statusCounts
	if !includeBitLocker {
		counts, err = getMDMWindowsStatusCountsProfilesOnlyDB(ctx, ds, teamID)
	} else {
		counts, err = getMDMWindowsStatusCountsProfilesAndBitLockerDB(ctx, ds, teamID)
	}
	if err != nil {
		return nil, err
	}

	var res fleet.MDMProfilesSummary
	for _, c := range counts {
		switch c.Status {
		case "failed":
			res.Failed = c.Count
		case "pending":
			res.Pending = c.Count
		case "verifying":
			res.Verifying = c.Count
		case "verified":
			res.Verified = c.Count
		case "":
			level.Debug(ds.logger).Log("msg", fmt.Sprintf("counted %d windows hosts on team %v with mdm turned on but no profiles or bitlocker status", c.Count, teamID))
		default:
			return nil, ctxerr.New(ctx, fmt.Sprintf("unexpected mdm windows status count: status=%s, count=%d", c.Status, c.Count))
		}
	}

	return &res, nil
}

type statusCounts struct {
	Status string `db:"status"`
	Count  uint   `db:"count"`
}

func getMDMWindowsStatusCountsProfilesOnlyDB(ctx context.Context, ds *Datastore, teamID *uint) ([]statusCounts, error) {
	var args []interface{}
	subqueryFailed, subqueryFailedArgs, err := subqueryHostsMDMWindowsOSSettingsStatusFailed()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "subqueryHostsMDMWindowsOSSettingsStatusFailed")
	}
	args = append(args, subqueryFailedArgs...)
	subqueryPending, subqueryPendingArgs, err := subqueryHostsMDMWindowsOSSettingsStatusPending()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "subqueryHostsMDMWindowsOSSettingsStatusPending")
	}
	args = append(args, subqueryPendingArgs...)
	subqueryVerifying, subqueryVeryingingArgs, err := subqueryHostsMDMWindowsOSSettingsStatusVerifying()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "subqueryHostsMDMWindowsOSSettingsStatusVerifying")
	}
	args = append(args, subqueryVeryingingArgs...)
	subqueryVerified, subqueryVerifiedArgs, err := subqueryHostsMDMWindowsOSSettingsStatusVerified()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "subqueryHostsMDMWindowsOSSettingsStatusVerified")
	}
	args = append(args, subqueryVerifiedArgs...)

	teamFilter := "h.team_id IS NULL"
	if teamID != nil && *teamID > 0 {
		teamFilter = "h.team_id = ?"
		args = append(args, *teamID)
	}

	stmt := fmt.Sprintf(`
SELECT
    CASE
        WHEN EXISTS (%s) THEN
            'failed'
        WHEN EXISTS (%s) THEN
            'pending'
        WHEN EXISTS (%s) THEN
            'verifying'
        WHEN EXISTS (%s) THEN
            'verified'
        ELSE
            ''
    END AS status,
    SUM(1) AS count
FROM
    hosts h
    JOIN host_mdm hmdm ON h.id = hmdm.host_id
    JOIN mdm_windows_enrollments mwe ON h.uuid = mwe.host_uuid
WHERE
    mwe.device_state = '%s' AND
    h.platform = 'windows' AND
    hmdm.is_server = 0 AND
    %s
GROUP BY
    status`,
		subqueryFailed,
		subqueryPending,
		subqueryVerifying,
		subqueryVerified,
		microsoft_mdm.MDMDeviceStateEnrolled,
		teamFilter,
	)

	var counts []statusCounts
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &counts, stmt, args...)
	if err != nil {
		return nil, err
	}
	return counts, nil
}

func getMDMWindowsStatusCountsProfilesAndBitLockerDB(ctx context.Context, ds *Datastore, teamID *uint) ([]statusCounts, error) {
	var args []interface{}
	subqueryFailed, subqueryFailedArgs, err := subqueryHostsMDMWindowsOSSettingsStatusFailed()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "subqueryHostsMDMWindowsOSSettingsStatusFailed")
	}
	args = append(args, subqueryFailedArgs...)
	subqueryPending, subqueryPendingArgs, err := subqueryHostsMDMWindowsOSSettingsStatusPending()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "subqueryHostsMDMWindowsOSSettingsStatusPending")
	}
	args = append(args, subqueryPendingArgs...)
	subqueryVerifying, subqueryVeryingingArgs, err := subqueryHostsMDMWindowsOSSettingsStatusVerifying()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "subqueryHostsMDMWindowsOSSettingsStatusVerifying")
	}
	args = append(args, subqueryVeryingingArgs...)
	subqueryVerified, subqueryVerifiedArgs, err := subqueryHostsMDMWindowsOSSettingsStatusVerified()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "subqueryHostsMDMWindowsOSSettingsStatusVerified")
	}
	args = append(args, subqueryVerifiedArgs...)

	profilesStatus := fmt.Sprintf(`
        CASE WHEN EXISTS (%s) THEN
            'profiles_failed'
        WHEN EXISTS (%s) THEN
            'profiles_pending'
        WHEN EXISTS (%s) THEN
            'profiles_verifying'
        WHEN EXISTS (%s) THEN
            'profiles_verified'
        ELSE
            ''
        END`,
		subqueryFailed,
		subqueryPending,
		subqueryVerifying,
		subqueryVerified,
	)

	teamFilter := "h.team_id IS NULL"
	if teamID != nil && *teamID > 0 {
		teamFilter = "h.team_id = ?"
		args = append(args, *teamID)
	}
	bitlockerJoin := `
    LEFT JOIN host_disk_encryption_keys hdek ON hdek.host_id = h.id
    LEFT JOIN host_disks hd ON hd.host_id = h.id`

	bitlockerStatus := fmt.Sprintf(`
            CASE WHEN (%s) THEN
                'bitlocker_verified'
            WHEN (%s) THEN
                'bitlocker_verifying'
            WHEN (%s) THEN
                'bitlocker_pending'
            WHEN (%s) THEN
                'bitlocker_failed'
            ELSE
                ''
            END`,
		ds.whereBitLockerStatus(fleet.DiskEncryptionVerified),
		ds.whereBitLockerStatus(fleet.DiskEncryptionVerifying),
		ds.whereBitLockerStatus(fleet.DiskEncryptionEnforcing),
		ds.whereBitLockerStatus(fleet.DiskEncryptionFailed),
	)

	stmt := fmt.Sprintf(`
SELECT
    CASE (SELECT (%s) FROM hosts h2 WHERE h2.id = h.id)
    WHEN 'profiles_failed' THEN
        'failed'
    WHEN 'profiles_pending' THEN (
        CASE (%s)
        WHEN 'bitlocker_failed' THEN
            'failed'
        ELSE
            'pending'
        END)
    WHEN 'profiles_verifying' THEN (
        CASE (%s)
        WHEN 'bitlocker_failed' THEN
            'failed'
        WHEN 'bitlocker_pending' THEN
            'pending'
        ELSE
            'verifying'
        END)
    WHEN 'profiles_verified' THEN (
        CASE (%s)
        WHEN 'bitlocker_failed' THEN
            'failed'
        WHEN 'bitlocker_pending' THEN
            'pending'
        WHEN 'bitlocker_verifying' THEN
            'verifying'
        ELSE
            'verified'
        END)
    ELSE
        REPLACE((%s), 'bitlocker_', '')
    END as status,
    SUM(1) as count
FROM
    hosts h
    JOIN host_mdm hmdm ON h.id = hmdm.host_id
    JOIN mdm_windows_enrollments mwe ON h.uuid = mwe.host_uuid
    %s
WHERE
    mwe.device_state = '%s' AND
    h.platform = 'windows' AND
    hmdm.is_server = 0 AND
    %s
GROUP BY
    status`,
		profilesStatus,
		bitlockerStatus,
		bitlockerStatus,
		bitlockerStatus,
		bitlockerStatus,
		bitlockerJoin,
		microsoft_mdm.MDMDeviceStateEnrolled,
		teamFilter,
	)

	var counts []statusCounts
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &counts, stmt, args...)
	if err != nil {
		return nil, err
	}
	return counts, nil
}

const windowsMDMProfilesDesiredStateQuery = `
	-- non label-based profiles
	SELECT
		mwcp.profile_uuid,
		mwcp.name,
		h.uuid as host_uuid,
		0 as count_profile_labels,
		0 as count_non_broken_labels,
		0 as count_host_labels,
		0 as count_host_updated_after_labels
	FROM
		mdm_windows_configuration_profiles mwcp
			JOIN hosts h
				ON h.team_id = mwcp.team_id OR (h.team_id IS NULL AND mwcp.team_id = 0)
			JOIN mdm_windows_enrollments mwe
				ON mwe.host_uuid = h.uuid
	WHERE
		h.platform = 'windows' AND
		NOT EXISTS (
			SELECT 1
			FROM mdm_configuration_profile_labels mcpl
			WHERE mcpl.windows_profile_uuid = mwcp.profile_uuid
		) AND
		( %s )

	UNION

	-- label-based profiles where the host is a member of all the labels (include-all).
	-- by design, "include" labels cannot match if they are broken (the host cannot be
	-- a member of a deleted label).
	SELECT
		mwcp.profile_uuid,
		mwcp.name,
		h.uuid as host_uuid,
		COUNT(*) as count_profile_labels,
		COUNT(mcpl.label_id) as count_non_broken_labels,
		COUNT(lm.label_id) as count_host_labels,
		0 as count_host_updated_after_labels
	FROM
		mdm_windows_configuration_profiles mwcp
			JOIN hosts h
				ON h.team_id = mwcp.team_id OR (h.team_id IS NULL AND mwcp.team_id = 0)
			JOIN mdm_windows_enrollments mwe
				ON mwe.host_uuid = h.uuid
			JOIN mdm_configuration_profile_labels mcpl
				ON mcpl.windows_profile_uuid = mwcp.profile_uuid AND mcpl.exclude = 0 AND mcpl.require_all = 1
			LEFT OUTER JOIN label_membership lm
				ON lm.label_id = mcpl.label_id AND lm.host_id = h.id
	WHERE
		h.platform = 'windows' AND
		( %s )
	GROUP BY
		mwcp.profile_uuid, mwcp.name, h.uuid
	HAVING
		count_profile_labels > 0 AND count_host_labels = count_profile_labels

	UNION

	-- label-based entities where the host is NOT a member of any of the labels (exclude-any).
	-- explicitly ignore profiles with broken excluded labels so that they are never applied,
	-- and ignore profiles that depend on labels created _after_ the label_updated_at timestamp
	-- of the host (because we don't have results for that label yet, the host may or may not be
	-- a member).
	SELECT
		mwcp.profile_uuid,
		mwcp.name,
		h.uuid as host_uuid,
		COUNT(*) as count_profile_labels,
		COUNT(mcpl.label_id) as count_non_broken_labels,
		COUNT(lm.label_id) as count_host_labels,
		-- this helps avoid the case where the host is not a member of a label
		-- just because it hasn't reported results for that label yet.
		SUM(CASE WHEN lbl.created_at IS NOT NULL AND h.label_updated_at >= lbl.created_at THEN 1 ELSE 0 END) as count_host_updated_after_labels
	FROM
		mdm_windows_configuration_profiles mwcp
			JOIN hosts h
				ON h.team_id = mwcp.team_id OR (h.team_id IS NULL AND mwcp.team_id = 0)
			JOIN mdm_windows_enrollments mwe
				ON mwe.host_uuid = h.uuid
			JOIN mdm_configuration_profile_labels mcpl
				ON mcpl.windows_profile_uuid = mwcp.profile_uuid AND mcpl.exclude = 1 AND mcpl.require_all = 0
			LEFT OUTER JOIN labels lbl
				ON lbl.id = mcpl.label_id
			LEFT OUTER JOIN label_membership lm
				ON lm.label_id = mcpl.label_id AND lm.host_id = h.id
	WHERE
		h.platform = 'windows' AND
		( %s )
	GROUP BY
		mwcp.profile_uuid, mwcp.name, h.uuid
	HAVING
		-- considers only the profiles with labels, without any broken label, with results reported after all labels were created and with the host not in any label
		count_profile_labels > 0 AND count_profile_labels = count_non_broken_labels AND count_profile_labels = count_host_updated_after_labels AND count_host_labels = 0

	UNION

	-- label-based profiles where the host is a member of any of the labels (include-any).
	-- by design, "include" labels cannot match if they are broken (the host cannot be
	-- a member of a deleted label).
	SELECT
		mwcp.profile_uuid,
		mwcp.name,
		h.uuid as host_uuid,
		COUNT(*) as count_profile_labels,
		COUNT(mcpl.label_id) as count_non_broken_labels,
		COUNT(lm.label_id) as count_host_labels,
		0 as count_host_updated_after_labels
	FROM
		mdm_windows_configuration_profiles mwcp
			JOIN hosts h
				ON h.team_id = mwcp.team_id OR (h.team_id IS NULL AND mwcp.team_id = 0)
			JOIN mdm_windows_enrollments mwe
				ON mwe.host_uuid = h.uuid
			JOIN mdm_configuration_profile_labels mcpl
				ON mcpl.windows_profile_uuid = mwcp.profile_uuid AND mcpl.exclude = 0 AND mcpl.require_all = 0
			LEFT OUTER JOIN label_membership lm
				ON lm.label_id = mcpl.label_id AND lm.host_id = h.id
	WHERE
		h.platform = 'windows' AND
		( %s )
	GROUP BY
		mwcp.profile_uuid, mwcp.name, h.uuid
	HAVING
		count_profile_labels > 0 AND count_host_labels >= 1
`

func (ds *Datastore) ListMDMWindowsProfilesToInstall(ctx context.Context) ([]*fleet.MDMWindowsProfilePayload, error) {
	var result []*fleet.MDMWindowsProfilePayload
	// TODO(mna): why is this in a transaction/reading from the primary, but not
	// Apple's implementation? I see that the called private method is sometimes
	// called inside a transaction, but when called from here it could (should?)
	// be without and use the reader replica?
	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		var err error
		result, err = listMDMWindowsProfilesToInstallDB(ctx, tx, nil, nil)
		return err
	})
	return result, err
}

func listMDMWindowsProfilesToInstallDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostUUIDs []string,
	onlyProfileUUIDs []string,
) ([]*fleet.MDMWindowsProfilePayload, error) {
	// The query below is a set difference between:
	//
	// - Set A (ds), the "desired state", can be obtained from a JOIN between
	//   mdm_windows_configuration_profiles and hosts.
	//
	// - Set B, the "current state" given by host_mdm_windows_profiles.
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
	//
	// Note that for label-based profiles, only fully-satisfied profiles are
	// considered for installation. This means that a broken label-based profile,
	// where one of the labels does not exist anymore, will not be considered for
	// installation.

	query := fmt.Sprintf(`
	SELECT
		ds.profile_uuid,
		ds.host_uuid,
		ds.name as profile_name
	FROM ( %s ) as ds
		LEFT JOIN host_mdm_windows_profiles hmwp
			ON hmwp.profile_uuid = ds.profile_uuid AND hmwp.host_uuid = ds.host_uuid
	WHERE
		-- profiles in A but not in B
		( hmwp.profile_uuid IS NULL AND hmwp.host_uuid IS NULL ) OR
		-- profiles in A and B with operation type "install" and NULL status
		( hmwp.host_uuid IS NOT NULL AND hmwp.operation_type = ? AND hmwp.status IS NULL )
`, windowsMDMProfilesDesiredStateQuery)

	hostFilter := "TRUE"
	if len(hostUUIDs) > 0 {
		if len(onlyProfileUUIDs) > 0 {
			hostFilter = "mwcp.profile_uuid IN (?) AND h.uuid IN (?)"
		} else {
			hostFilter = "h.uuid IN (?)"
		}
	}

	var err error
	args := []any{fleet.MDMOperationTypeInstall}
	query = fmt.Sprintf(query, hostFilter, hostFilter, hostFilter, hostFilter)
	if len(hostUUIDs) > 0 {
		if len(onlyProfileUUIDs) > 0 {
			query, args, err = sqlx.In(
				query,
				onlyProfileUUIDs, hostUUIDs,
				onlyProfileUUIDs, hostUUIDs,
				onlyProfileUUIDs, hostUUIDs,
				onlyProfileUUIDs, hostUUIDs,
				fleet.MDMOperationTypeInstall,
			)
		} else {
			query, args, err = sqlx.In(query, hostUUIDs, hostUUIDs, hostUUIDs, hostUUIDs, fleet.MDMOperationTypeInstall)
		}
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "building sqlx.In")
		}
	}

	var profiles []*fleet.MDMWindowsProfilePayload
	err = sqlx.SelectContext(ctx, tx, &profiles, query, args...)
	return profiles, err
}

func (ds *Datastore) ListMDMWindowsProfilesToRemove(ctx context.Context) ([]*fleet.MDMWindowsProfilePayload, error) {
	var result []*fleet.MDMWindowsProfilePayload
	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		var err error
		result, err = listMDMWindowsProfilesToRemoveDB(ctx, tx, nil, nil)
		return err
	})

	return result, err
}

func listMDMWindowsProfilesToRemoveDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostUUIDs []string,
	onlyProfileUUIDs []string,
) ([]*fleet.MDMWindowsProfilePayload, error) {
	// The query below is a set difference between:
	//
	// - Set A (ds), the desired state, can be obtained from a JOIN between
	// mdm_windows_configuration_profiles and hosts.
	// - Set B, the current state given by host_mdm_windows_profiles.
	//
	// B - A gives us the profiles that need to be removed
	//
	// Any other case are profiles that are in both B and A, and as such are
	// processed by the ListMDMWindowsProfilesToInstall method (since they are
	// in both, their desired state is necessarily to be installed).
	//
	// Note that for label-based profiles, only those that are fully-sastisfied
	// by the host are considered for install (are part of the desired state used
	// to compute the ones to remove). However, as a special case, a broken
	// label-based profile will NOT be removed from a host where it was
	// previously installed. However, if a host used to satisfy a label-based
	// profile but no longer does (and that label-based profile is not "broken"),
	// the profile will be removed from the host.

	hostFilter := "TRUE"
	if len(hostUUIDs) > 0 {
		if len(onlyProfileUUIDs) > 0 {
			hostFilter = "hmwp.profile_uuid IN (?) AND hmwp.host_uuid IN (?)"
		} else {
			hostFilter = "hmwp.host_uuid IN (?)"
		}
	}

	query := fmt.Sprintf(`
	SELECT
		hmwp.profile_uuid,
		hmwp.host_uuid,
		hmwp.operation_type,
		COALESCE(hmwp.detail, '') as detail,
		hmwp.status,
		hmwp.command_uuid
	FROM ( %s ) as ds
		RIGHT JOIN host_mdm_windows_profiles hmwp
			ON hmwp.profile_uuid = ds.profile_uuid AND hmwp.host_uuid = ds.host_uuid
	WHERE
		-- profiles that are in B but not in A
		ds.profile_uuid IS NULL AND ds.host_uuid IS NULL AND
		-- TODO(mna): why don't we have the same exception for "remove" operations as for Apple

		-- except "would be removed" profiles if they are a broken label-based profile
		-- (regardless of if it is an include-all or exclude-any label)
		NOT EXISTS (
			SELECT 1
			FROM mdm_configuration_profile_labels mcpl
			WHERE
				mcpl.windows_profile_uuid = hmwp.profile_uuid AND
				mcpl.label_id IS NULL
		) AND
		(%s)
`, fmt.Sprintf(windowsMDMProfilesDesiredStateQuery, "TRUE", "TRUE", "TRUE", "TRUE"), hostFilter)

	var err error
	var args []any
	if len(hostUUIDs) > 0 {
		if len(onlyProfileUUIDs) > 0 {
			query, args, err = sqlx.In(query, onlyProfileUUIDs, hostUUIDs)
		} else {
			query, args, err = sqlx.In(query, hostUUIDs)
		}
		if err != nil {
			return nil, err
		}
	}

	var profiles []*fleet.MDMWindowsProfilePayload
	err = sqlx.SelectContext(ctx, tx, &profiles, query, args...)
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
	if ds.testUpsertMDMDesiredProfilesBatchSize > 0 {
		batchSize = ds.testUpsertMDMDesiredProfilesBatchSize
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
		return ds.bulkDeleteMDMWindowsHostsConfigProfilesDB(ctx, tx, profs)
	})
}

func (ds *Datastore) bulkDeleteMDMWindowsHostsConfigProfilesDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	profs []*fleet.MDMWindowsProfilePayload,
) error {
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
	if ds.testDeleteMDMProfilesBatchSize > 0 {
		batchSize = ds.testDeleteMDMProfilesBatchSize
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

func (ds *Datastore) NewMDMWindowsConfigProfile(ctx context.Context, cp fleet.MDMWindowsConfigProfile) (*fleet.MDMWindowsConfigProfile, error) {
	profileUUID := "w" + uuid.New().String()
	insertProfileStmt := `
INSERT INTO
    mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml, uploaded_at)
(SELECT ?, ?, ?, ?, CURRENT_TIMESTAMP() FROM DUAL WHERE
	NOT EXISTS (
		SELECT 1 FROM mdm_apple_configuration_profiles WHERE name = ? AND team_id = ?
	) AND NOT EXISTS (
		SELECT 1 FROM mdm_apple_declarations WHERE name = ? AND team_id = ?
	)
)`

	var teamID uint
	if cp.TeamID != nil {
		teamID = *cp.TeamID
	}

	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, insertProfileStmt, profileUUID, teamID, cp.Name, cp.SyncML, cp.Name, teamID, cp.Name, teamID)
		if err != nil {
			switch {
			case IsDuplicate(err):
				return &existsError{
					ResourceType: "MDMWindowsConfigProfile.Name",
					Identifier:   cp.Name,
					TeamID:       cp.TeamID,
				}
			default:
				return ctxerr.Wrap(ctx, err, "creating new windows mdm config profile")
			}
		}

		aff, _ := res.RowsAffected()
		if aff == 0 {
			return &existsError{
				ResourceType: "MDMWindowsConfigProfile.Name",
				Identifier:   cp.Name,
				TeamID:       cp.TeamID,
			}
		}

		labels := make([]fleet.ConfigurationProfileLabel, 0, len(cp.LabelsIncludeAll)+len(cp.LabelsIncludeAny)+len(cp.LabelsExcludeAny))
		for i := range cp.LabelsIncludeAll {
			cp.LabelsIncludeAll[i].ProfileUUID = profileUUID
			cp.LabelsIncludeAll[i].RequireAll = true
			cp.LabelsIncludeAll[i].Exclude = false
			labels = append(labels, cp.LabelsIncludeAll[i])
		}
		for i := range cp.LabelsIncludeAny {
			cp.LabelsIncludeAny[i].ProfileUUID = profileUUID
			cp.LabelsIncludeAny[i].RequireAll = false
			cp.LabelsIncludeAny[i].Exclude = false
			labels = append(labels, cp.LabelsIncludeAny[i])
		}
		for i := range cp.LabelsExcludeAny {
			cp.LabelsExcludeAny[i].ProfileUUID = profileUUID
			cp.LabelsExcludeAny[i].RequireAll = false
			cp.LabelsExcludeAny[i].Exclude = true
			labels = append(labels, cp.LabelsExcludeAny[i])
		}
		if _, err := batchSetProfileLabelAssociationsDB(ctx, tx, labels, "windows"); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting windows profile label associations")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &fleet.MDMWindowsConfigProfile{
		ProfileUUID: profileUUID,
		Name:        cp.Name,
		SyncML:      cp.SyncML,
		TeamID:      cp.TeamID,
	}, nil
}

func (ds *Datastore) SetOrUpdateMDMWindowsConfigProfile(ctx context.Context, cp fleet.MDMWindowsConfigProfile) error {
	profileUUID := "w" + uuid.New().String()
	stmt := `
INSERT INTO
	mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml, uploaded_at)
(SELECT ?, ?, ?, ?, CURRENT_TIMESTAMP() FROM DUAL WHERE
	NOT EXISTS (
		SELECT 1 FROM mdm_apple_configuration_profiles WHERE name = ? AND team_id = ?
	) AND NOT EXISTS (
		SELECT 1 FROM mdm_apple_declarations WHERE name = ? AND team_id = ?
	)
)
ON DUPLICATE KEY UPDATE
	uploaded_at = IF(syncml = VALUES(syncml), uploaded_at, CURRENT_TIMESTAMP()),
	syncml = VALUES(syncml)
`

	var teamID uint
	if cp.TeamID != nil {
		teamID = *cp.TeamID
	}

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, profileUUID, teamID, cp.Name, cp.SyncML, cp.Name, teamID, cp.Name, teamID)
	if err != nil {
		switch {
		case IsDuplicate(err):
			return &existsError{
				ResourceType: "MDMWindowsConfigProfile.Name",
				Identifier:   cp.Name,
				TeamID:       cp.TeamID,
			}
		default:
			return ctxerr.Wrap(ctx, err, "creating new windows mdm config profile")
		}
	}

	aff, _ := res.RowsAffected()
	if aff == 0 {
		return &existsError{
			ResourceType: "MDMWindowsConfigProfile.Name",
			Identifier:   cp.Name,
			TeamID:       cp.TeamID,
		}
	}

	return nil
}

// batchSetMDMWindowsProfilesDB must be called from inside a transaction.
func (ds *Datastore) batchSetMDMWindowsProfilesDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	tmID *uint,
	profiles []*fleet.MDMWindowsConfigProfile,
) (updatedDB bool, err error) {
	const loadExistingProfiles = `
SELECT
  name,
  profile_uuid,
  syncml
FROM
  mdm_windows_configuration_profiles
WHERE
  team_id = ? AND
  name IN (?)
`

	const deleteProfilesNotInList = `
DELETE FROM
  mdm_windows_configuration_profiles
WHERE
  team_id = ? AND
  name NOT IN (?)
`

	const deleteAllProfilesForTeam = `
DELETE FROM
  mdm_windows_configuration_profiles
WHERE
  team_id = ?
`

	// For Windows profiles, if team_id and name are the same, we do an update. Otherwise, we do an insert.
	const insertNewOrEditedProfile = `
INSERT INTO
  mdm_windows_configuration_profiles (
    profile_uuid, team_id, name, syncml, uploaded_at
  )
VALUES
  -- see https://stackoverflow.com/a/51393124/1094941
  ( CONCAT('w', CONVERT(UUID() USING utf8mb4)), ?, ?, ?, CURRENT_TIMESTAMP() )
ON DUPLICATE KEY UPDATE
  uploaded_at = IF(syncml = VALUES(syncml) AND name = VALUES(name), uploaded_at, CURRENT_TIMESTAMP()),
  name = VALUES(name),
  syncml = VALUES(syncml)
`

	// use a profile team id of 0 if no-team
	var profTeamID uint
	if tmID != nil {
		profTeamID = *tmID
	}

	// build a list of names for the incoming profiles, will keep the
	// existing ones if there's a match and no change
	incomingNames := make([]string, len(profiles))
	// at the same time, index the incoming profiles keyed by name for ease
	// or processing
	incomingProfs := make(map[string]*fleet.MDMWindowsConfigProfile, len(profiles))
	for i, p := range profiles {
		incomingNames[i] = p.Name
		incomingProfs[p.Name] = p
	}

	var existingProfiles []*fleet.MDMWindowsConfigProfile

	if len(incomingNames) > 0 {
		// load existing profiles that match the incoming profiles by name
		stmt, args, err := sqlx.In(loadExistingProfiles, profTeamID, incomingNames)
		if err != nil || strings.HasPrefix(ds.testBatchSetMDMWindowsProfilesErr, "inselect") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMWindowsProfilesErr)
			}
			return false, ctxerr.Wrap(ctx, err, "build query to load existing profiles")
		}
		if err := sqlx.SelectContext(ctx, tx, &existingProfiles, stmt, args...); err != nil || strings.HasPrefix(ds.testBatchSetMDMWindowsProfilesErr, "select") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMWindowsProfilesErr)
			}
			return false, ctxerr.Wrap(ctx, err, "load existing profiles")
		}
	}

	// figure out if we need to delete any profiles
	keepNames := make([]string, 0, len(incomingNames))
	for _, p := range existingProfiles {
		if newP := incomingProfs[p.Name]; newP != nil {
			keepNames = append(keepNames, p.Name)
		}
	}
	for n := range mdm.FleetReservedProfileNames() {
		if _, ok := incomingProfs[n]; !ok {
			// always keep reserved profiles even if they're not incoming
			keepNames = append(keepNames, n)
		}
	}

	var (
		stmt string
		args []interface{}
	)
	// delete the obsolete profiles (all those that are not in keepNames)
	var result sql.Result
	if len(keepNames) > 0 {
		stmt, args, err = sqlx.In(deleteProfilesNotInList, profTeamID, keepNames)
		if err != nil || strings.HasPrefix(ds.testBatchSetMDMWindowsProfilesErr, "indelete") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMWindowsProfilesErr)
			}
			return false, ctxerr.Wrap(ctx, err, "build statement to delete obsolete profiles")
		}
		if result, err = tx.ExecContext(ctx, stmt, args...); err != nil || strings.HasPrefix(ds.testBatchSetMDMWindowsProfilesErr,
			"delete") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMWindowsProfilesErr)
			}
			return false, ctxerr.Wrap(ctx, err, "delete obsolete profiles")
		}
	} else {
		if result, err = tx.ExecContext(ctx, deleteAllProfilesForTeam,
			profTeamID); err != nil || strings.HasPrefix(ds.testBatchSetMDMWindowsProfilesErr, "delete") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMWindowsProfilesErr)
			}
			return false, ctxerr.Wrap(ctx, err, "delete all profiles for team")
		}
	}
	if result != nil {
		rows, _ := result.RowsAffected()
		updatedDB = rows > 0
	}

	// insert the new profiles and the ones that have changed
	for _, p := range incomingProfs {
		if result, err = tx.ExecContext(ctx, insertNewOrEditedProfile, profTeamID, p.Name,
			p.SyncML); err != nil || strings.HasPrefix(ds.testBatchSetMDMWindowsProfilesErr, "insert") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMWindowsProfilesErr)
			}
			return false, ctxerr.Wrapf(ctx, err, "insert new/edited profile with name %q", p.Name)
		}
		updatedDB = updatedDB || insertOnDuplicateDidInsertOrUpdate(result)
	}

	// build a list of labels so the associations can be batch-set all at once
	// TODO: with minor changes this chunk of code could be shared
	// between macOS and Windows, but at the time of this
	// implementation we're under tight time constraints.
	incomingLabels := []fleet.ConfigurationProfileLabel{}
	if len(incomingNames) > 0 {
		var newlyInsertedProfs []*fleet.MDMWindowsConfigProfile
		// load current profiles (again) that match the incoming profiles by name to grab their uuids
		stmt, args, err := sqlx.In(loadExistingProfiles, profTeamID, incomingNames)
		if err != nil || strings.HasPrefix(ds.testBatchSetMDMWindowsProfilesErr, "inreselect") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMWindowsProfilesErr)
			}
			return false, ctxerr.Wrap(ctx, err, "build query to load newly inserted profiles")
		}
		if err := sqlx.SelectContext(ctx, tx, &newlyInsertedProfs, stmt, args...); err != nil || strings.HasPrefix(ds.testBatchSetMDMWindowsProfilesErr, "reselect") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMWindowsProfilesErr)
			}
			return false, ctxerr.Wrap(ctx, err, "load newly inserted profiles")
		}

		for _, newlyInsertedProf := range newlyInsertedProfs {
			incomingProf, ok := incomingProfs[newlyInsertedProf.Name]
			if !ok {
				return false, ctxerr.Wrapf(ctx, err, "profile %q is in the database but was not incoming", newlyInsertedProf.Name)
			}

			for _, label := range incomingProf.LabelsIncludeAll {
				label.ProfileUUID = newlyInsertedProf.ProfileUUID
				label.Exclude = false
				label.RequireAll = true
				incomingLabels = append(incomingLabels, label)
			}
			for _, label := range incomingProf.LabelsIncludeAny {
				label.ProfileUUID = newlyInsertedProf.ProfileUUID
				label.Exclude = false
				label.RequireAll = false
				incomingLabels = append(incomingLabels, label)
			}
			for _, label := range incomingProf.LabelsExcludeAny {
				label.ProfileUUID = newlyInsertedProf.ProfileUUID
				label.Exclude = true
				label.RequireAll = false
				incomingLabels = append(incomingLabels, label)
			}
		}
	}

	// insert/delete the label associations
	var updatedLabels bool
	if updatedLabels, err = batchSetProfileLabelAssociationsDB(ctx, tx, incomingLabels,
		"windows"); err != nil || strings.HasPrefix(ds.testBatchSetMDMWindowsProfilesErr, "labels") {
		if err == nil {
			err = errors.New(ds.testBatchSetMDMWindowsProfilesErr)
		}
		return false, ctxerr.Wrap(ctx, err, "inserting windows profile label associations")
	}

	return updatedDB || updatedLabels, nil
}

func (ds *Datastore) bulkSetPendingMDMWindowsHostProfilesDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostUUIDs []string,
	onlyProfileUUIDs []string,
) (updatedDB bool, err error) {
	if len(hostUUIDs) == 0 {
		return false, nil
	}

	profilesToInstall, err := listMDMWindowsProfilesToInstallDB(ctx, tx, hostUUIDs, onlyProfileUUIDs)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "list profiles to install")
	}

	profilesToRemove, err := listMDMWindowsProfilesToRemoveDB(ctx, tx, hostUUIDs, onlyProfileUUIDs)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "list profiles to remove")
	}

	if len(profilesToInstall) == 0 && len(profilesToRemove) == 0 {
		return false, nil
	}

	if len(profilesToRemove) > 0 {
		if err := ds.bulkDeleteMDMWindowsHostsConfigProfilesDB(ctx, tx, profilesToRemove); err != nil {
			return false, ctxerr.Wrap(ctx, err, "bulk delete profiles to remove")
		}
		updatedDB = true
	}
	if len(profilesToInstall) == 0 {
		return updatedDB, nil
	}

	var (
		pargs            []any
		profilesToInsert = make(map[string]*fleet.MDMWindowsProfilePayload)
		psb              strings.Builder
		batchCount       int
	)

	const defaultBatchSize = 1000
	batchSize := defaultBatchSize
	if ds.testUpsertMDMDesiredProfilesBatchSize > 0 {
		batchSize = ds.testUpsertMDMDesiredProfilesBatchSize
	}

	resetBatch := func() {
		batchCount = 0
		pargs = pargs[:0]
		clear(profilesToInsert)
		psb.Reset()
	}

	executeUpsertBatch := func(valuePart string, args []any) error {
		// Check if the update needs to be done at all.
		selectStmt := fmt.Sprintf(`
			SELECT
				profile_uuid,
				host_uuid,
				status,
				COALESCE(operation_type, '') AS operation_type,
				COALESCE(detail, '') AS detail,
				COALESCE(command_uuid, '') AS command_uuid,
				COALESCE(profile_name, '') AS profile_name
			FROM host_mdm_windows_profiles WHERE (profile_uuid, host_uuid) IN (%s)`,
			strings.TrimSuffix(strings.Repeat("(?,?),", len(profilesToInsert)), ","))
		var selectArgs []any
		for _, p := range profilesToInsert {
			selectArgs = append(selectArgs, p.ProfileUUID, p.HostUUID)
		}
		var existingProfiles []fleet.MDMWindowsProfilePayload
		if err := sqlx.SelectContext(ctx, tx, &existingProfiles, selectStmt, selectArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "bulk set pending profile status select existing")
		}
		var updateNeeded bool
		if len(existingProfiles) == len(profilesToInsert) {
			for _, exist := range existingProfiles {
				insert, ok := profilesToInsert[fmt.Sprintf("%s\n%s", exist.ProfileUUID, exist.HostUUID)]
				if !ok || !exist.Equal(*insert) {
					updateNeeded = true
					break
				}
			}
		} else {
			updateNeeded = true
		}
		if !updateNeeded {
			// All profiles are already in the database, no need to update.
			return nil
		}

		baseStmt := fmt.Sprintf(`
				INSERT INTO host_mdm_windows_profiles (
					profile_uuid,
					host_uuid,
					profile_name,
					operation_type,
					status,
					command_uuid
				)
				VALUES %s
				ON DUPLICATE KEY UPDATE
					operation_type = VALUES(operation_type),
					status = NULL,
					command_uuid = VALUES(command_uuid),
					detail = ''
			`, strings.TrimSuffix(valuePart, ","))

		_, err := tx.ExecContext(ctx, baseStmt, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "bulk set pending profile status execute batch")
		}
		updatedDB = true
		return nil
	}

	for _, p := range profilesToInstall {
		profilesToInsert[fmt.Sprintf("%s\n%s", p.ProfileUUID, p.HostUUID)] = &fleet.MDMWindowsProfilePayload{
			ProfileUUID:   p.ProfileUUID,
			ProfileName:   p.ProfileName,
			HostUUID:      p.HostUUID,
			Status:        nil,
			OperationType: fleet.MDMOperationTypeInstall,
			Detail:        p.Detail,
			CommandUUID:   p.CommandUUID,
			Retries:       p.Retries,
		}
		pargs = append(
			pargs, p.ProfileUUID, p.HostUUID, p.ProfileName,
			fleet.MDMOperationTypeInstall)
		psb.WriteString("(?, ?, ?, ?, NULL, ''),")
		batchCount++
		if batchCount >= batchSize {
			if err := executeUpsertBatch(psb.String(), pargs); err != nil {
				return false, err
			}
			resetBatch()
		}
	}

	if batchCount > 0 {
		if err := executeUpsertBatch(psb.String(), pargs); err != nil {
			return false, err
		}
	}

	return updatedDB, nil
}

func (ds *Datastore) GetHostMDMWindowsProfiles(ctx context.Context, hostUUID string) ([]fleet.HostMDMWindowsProfile, error) {
	stmt := fmt.Sprintf(`
SELECT
	profile_uuid,
	profile_name AS name,
	-- internally, a NULL status implies that the cron needs to pick up
	-- this profile, for the user that difference doesn't exist, the
	-- profile is effectively pending. This is consistent with all our
	-- aggregation functions.
	COALESCE(status, '%s') AS status,
	COALESCE(operation_type, '') AS operation_type,
	COALESCE(detail, '') AS detail
FROM
	host_mdm_windows_profiles
WHERE
host_uuid = ? AND profile_name NOT IN(?) AND NOT (operation_type = '%s' AND COALESCE(status, '%s') IN('%s', '%s'))`,
		fleet.MDMDeliveryPending,
		fleet.MDMOperationTypeRemove,
		fleet.MDMDeliveryPending,
		fleet.MDMDeliveryVerifying,
		fleet.MDMDeliveryVerified,
	)

	stmt, args, err := sqlx.In(stmt, hostUUID, mdm.ListFleetReservedWindowsProfileNames())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building in statement")
	}

	var profiles []fleet.HostMDMWindowsProfile
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, stmt, args...); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (ds *Datastore) WipeHostViaWindowsMDM(ctx context.Context, host *fleet.Host, cmd *fleet.MDMWindowsCommand) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := ds.mdmWindowsInsertCommandForHostsDB(ctx, tx, []string{host.UUID}, cmd); err != nil {
			return err
		}

		stmt := `
			INSERT INTO host_mdm_actions (
				host_id,
				wipe_ref,
				fleet_platform
			)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE
				wipe_ref   = VALUES(wipe_ref)`

		if _, err := tx.ExecContext(ctx, stmt, host.ID, cmd.CommandUUID, host.FleetPlatform()); err != nil {
			return ctxerr.Wrap(ctx, err, "modifying host_mdm_actions for wipe_ref")
		}

		return nil
	})
}
