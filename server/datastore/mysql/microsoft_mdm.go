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

func subqueryHostsMDMWindowsOSSettingsStatusFailed() (string, []interface{}) {
	sql := `
            SELECT
                1 FROM host_mdm_windows_profiles hmwp
            WHERE
                h.uuid = hmwp.host_uuid
                AND hmwp.status = ?`
	args := []interface{}{
		fleet.MDMDeliveryFailed,
	}

	return sql, args
}

func subqueryHostsMDMWindowsOSSettingsStatusPending() (string, []interface{}) {
	sql := `
            SELECT
                1 FROM host_mdm_windows_profiles hmwp
            WHERE
                h.uuid = hmwp.host_uuid
                AND (hmwp.status IS NULL OR hmwp.status = ?) 
                AND NOT EXISTS (
                    SELECT
                        1 FROM host_mdm_windows_profiles hmwp2
                    WHERE (h.uuid = hmwp2.host_uuid
                        AND hmwp2.status = ?))`
	args := []interface{}{
		fleet.MDMDeliveryPending,
		fleet.MDMDeliveryFailed,
	}
	return sql, args
}

func subqueryHostsMDMWindowsOSSettingsStatusVerifying() (string, []interface{}) {
	sql := `
            SELECT
                1 FROM host_mdm_windows_profiles hmwp
            WHERE
                h.uuid = hmwp.host_uuid
                AND hmwp.operation_type = ?
                AND hmwp.status = ?
                AND NOT EXISTS (
                    SELECT
                        1 FROM host_mdm_windows_profiles hmwp2
                    WHERE (h.uuid = hmwp2.host_uuid
                        AND hmwp2.operation_type = ?
                        AND(hmwp2.status IS NULL
                            OR hmwp2.status NOT IN(?, ?))))`

	args := []interface{}{
		fleet.MDMOperationTypeInstall,
		fleet.MDMDeliveryVerifying,
		fleet.MDMOperationTypeInstall,
		fleet.MDMDeliveryVerifying,
		fleet.MDMDeliveryVerified,
	}
	return sql, args
}

func subqueryHostsMDMWindowsOSSettingsStatusVerified() (string, []interface{}) {
	sql := `
            SELECT
                1 FROM host_mdm_windows_profiles hmwp
            WHERE
                h.uuid = hmwp.host_uuid 
                AND hmwp.operation_type = ?
                AND hmwp.status = ?
                AND NOT EXISTS (
                    SELECT
                        1 FROM host_mdm_windows_profiles hmwp2
                    WHERE (h.uuid = hmwp2.host_uuid
                        AND hmwp2.operation_type = ?
                        AND(hmwp2.status IS NULL
                            OR hmwp2.status != ?)))`
	args := []interface{}{
		fleet.MDMOperationTypeInstall,
		fleet.MDMDeliveryVerified,
		fleet.MDMOperationTypeInstall,
		fleet.MDMDeliveryVerified,
	}
	return sql, args
}

func (ds *Datastore) GetMDMWindowsProfilesSummary(ctx context.Context, teamID *uint) (*fleet.MDMProfilesSummary, error) {
	includeBitLocker, err := ds.getConfigEnableDiskEncryption(ctx, teamID)
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
		default:
			level.Debug(ds.logger).Log("msg", "unexpected mdm windows status count", "status", c.Status, "count", c.Count)
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
	subqueryFailed, subqueryFailedArgs := subqueryHostsMDMWindowsOSSettingsStatusFailed()
	args = append(args, subqueryFailedArgs...)
	subqueryPending, subqueryPendingArgs := subqueryHostsMDMWindowsOSSettingsStatusPending()
	args = append(args, subqueryPendingArgs...)
	subqueryVerifying, subqueryVeryingingArgs := subqueryHostsMDMWindowsOSSettingsStatusVerifying()
	args = append(args, subqueryVeryingingArgs...)
	subqueryVerified, subqueryVerifiedArgs := subqueryHostsMDMWindowsOSSettingsStatusVerified()
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
WHERE
	hmdm.mdm_id = (SELECT id FROM mobile_device_management_solutions WHERE name = '%s') AND 
	hmdm.is_server = 0 AND
	hmdm.enrolled = 1 AND 
    h.platform = 'windows' AND
    %s
GROUP BY
    status`,
		subqueryFailed,
		subqueryPending,
		subqueryVerifying,
		subqueryVerified,
		fleet.WellKnownMDMFleet,
		teamFilter,
	)

	var counts []statusCounts
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &counts, stmt, args...)
	if err != nil {
		return nil, err
	}
	return counts, nil
}

func getMDMWindowsStatusCountsProfilesAndBitLockerDB(ctx context.Context, ds *Datastore, teamID *uint) ([]statusCounts, error) {
	var args []interface{}
	subqueryFailed, subqueryFailedArgs := subqueryHostsMDMWindowsOSSettingsStatusFailed()
	args = append(args, subqueryFailedArgs...)
	subqueryPending, subqueryPendingArgs := subqueryHostsMDMWindowsOSSettingsStatusPending()
	args = append(args, subqueryPendingArgs...)
	subqueryVerifying, subqueryVeryingingArgs := subqueryHostsMDMWindowsOSSettingsStatusVerifying()
	args = append(args, subqueryVeryingingArgs...)
	subqueryVerified, subqueryVerifiedArgs := subqueryHostsMDMWindowsOSSettingsStatusVerified()
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
    ELSE 
        REPLACE((%s), 'bitlocker_', '')
    END as status, 
    SUM(1) as count
FROM
    hosts h
    JOIN host_mdm hmdm ON h.id = hmdm.host_id
    %s
WHERE
	hmdm.mdm_id = (SELECT id FROM mobile_device_management_solutions WHERE name = '%s') AND 
    hmdm.is_server = 0 AND
    hmdm.enrolled = 1 AND
    h.platform = 'windows' AND
    %s
GROUP BY
    status`,
		profilesStatus,
		bitlockerStatus,
		bitlockerStatus,
		bitlockerStatus,
		bitlockerJoin,
		fleet.WellKnownMDMFleet,
		teamFilter,
	)

	var counts []statusCounts
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &counts, stmt, args...)
	if err != nil {
		return nil, err
	}
	return counts, nil
}
