package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// windowsMDMProfileDeleteBatchSize is the number of rows to process per batch
// when enqueuing <Delete> commands, resolving enrollment IDs, and updating
// host profile rows during profile deletion.
//
// 10,000 stays under MySQL's 65,535 placeholder limit on every caller. The
// densest caller is the batched UPDATE with tuple IN + CASE per profile in
// cancelWindowsHostInstallsForDeletedMDMProfiles, whose placeholder count is
//
//	2 (constants) + 2*distinctProfilesInBatch (CASE arms) + 2*rowsInBatch (IN)
//
// At batchSize=10,000 the realistic case (tens of profiles × many hosts each)
// is ~20,000 placeholders, and the worst case (up to 10,000 distinct profiles
// sharing a 10,000-row batch) is 40,002, still well under 65,535. The value
// also matches the batch sizes used elsewhere in Fleet for host-table bulk ops.
const windowsMDMProfileDeleteBatchSize = 10000

func isWindowsHostConnectedToFleetMDM(ctx context.Context, q sqlx.QueryerContext, h *fleet.Host) (bool, error) {
	var unused string

	// safe to use with interpolation rather than prepared statements because we're using a numeric ID here
	err := sqlx.GetContext(ctx, q, &unused, fmt.Sprintf(`
	  SELECT mwe.host_uuid
	  FROM mdm_windows_enrollments mwe
	    JOIN hosts h ON h.uuid = mwe.host_uuid
	    JOIN host_mdm hm ON hm.host_id = h.id
	  WHERE h.id = %d
	    AND mwe.device_state = '`+microsoft_mdm.MDMDeviceStateEnrolled+`'
	    AND hm.enrolled = 1 LIMIT 1
	`, h.ID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// MDMWindowsGetEnrolledDeviceWithDeviceID receives a Windows MDM device id and
// returns the device information.
func (ds *Datastore) MDMWindowsGetEnrolledDeviceWithDeviceID(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
	// Only fetch the most recently enrolled entry which matches the one we enqueue commands for
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
		awaiting_configuration,
		awaiting_configuration_at,
		credentials_hash,
		credentials_acknowledged,
		created_at,
		updated_at,
		host_uuid
		FROM mdm_windows_enrollments WHERE mdm_device_id = ? ORDER BY created_at DESC, id DESC LIMIT 1`

	var winMDMDevice fleet.MDMWindowsEnrolledDevice
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &winMDMDevice, stmt, mdmDeviceID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice").WithMessage(mdmDeviceID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get MDMWindowsGetEnrolledDeviceWithDeviceID")
	}
	return &winMDMDevice, nil
}

// MDMWindowsGetEnrolledDeviceWithDeviceID receives a Windows MDM device id and
// returns the device information.
func (ds *Datastore) MDMWindowsGetEnrolledDeviceWithHostUUID(ctx context.Context, hostUUID string) (*fleet.MDMWindowsEnrolledDevice, error) {
	// Only fetch the most recently enrolled entry which matches the one we enqueue commands for
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
		awaiting_configuration,
		awaiting_configuration_at,
		credentials_hash,
		credentials_acknowledged,
		created_at,
		updated_at,
		host_uuid
		FROM mdm_windows_enrollments WHERE host_uuid = ? ORDER BY created_at DESC, id DESC LIMIT 1`

	var winMDMDevice fleet.MDMWindowsEnrolledDevice
	// use the writer because this is sometimes fetched soon after updating the host UUID
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &winMDMDevice, stmt, hostUUID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice").WithMessage(hostUUID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get MDMWindowsGetEnrolledDeviceWithHostUUID")
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
			awaiting_configuration,
			awaiting_configuration_at,
			host_uuid,
			credentials_hash,
			credentials_acknowledged)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			awaiting_configuration = VALUES(awaiting_configuration),
			awaiting_configuration_at = VALUES(awaiting_configuration_at),
			host_uuid             = VALUES(host_uuid),
			credentials_hash      = VALUES(credentials_hash),
			credentials_acknowledged = VALUES(credentials_acknowledged)
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
		device.AwaitingConfiguration,
		device.AwaitingConfigurationAt,
		device.HostUUID,
		device.CredentialsHash,
		device.CredentialsAcknowledged)
	if err != nil {
		if IsDuplicate(err) {
			return ctxerr.Wrap(ctx, alreadyExists("MDMWindowsEnrolledDevice", device.MDMHardwareID))
		}
		return ctxerr.Wrap(ctx, err, "inserting MDMWindowsEnrolledDevice")
	}

	return nil
}

// MDMWindowsDeleteEnrolledDeviceOnReenrollment deletes a Windows device
// enrollment entry from the database using the device's hardware ID as it is
// re-enrolling. It also cleans up host_mdm_windows_profiles so profile
// delivery statuses are reset for the new enrollment.
func (ds *Datastore) MDMWindowsDeleteEnrolledDeviceOnReenrollment(ctx context.Context, mdmDeviceHWID string) error {
	const (
		delStmt         = "DELETE FROM mdm_windows_enrollments WHERE mdm_hardware_id = ?"
		loadStmt        = "SELECT host_uuid FROM mdm_windows_enrollments WHERE mdm_hardware_id = ? LIMIT 1"
		delActionsStmt  = "DELETE FROM host_mdm_actions WHERE host_id = (SELECT id FROM hosts WHERE uuid = ? LIMIT 1)"
		delProfilesStmt = "DELETE FROM host_mdm_windows_profiles WHERE host_uuid = ?"
	)

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var hostUUID sql.NullString
		switch err := sqlx.GetContext(ctx, tx, &hostUUID, loadStmt, mdmDeviceHWID); err {
		case nil:
			if hostUUID.Valid {
				// Clear lock/wipe status
				if _, err := tx.ExecContext(ctx, delActionsStmt, hostUUID.String); err != nil {
					return ctxerr.Wrap(ctx, err, "delete host_mdm_actions for host")
				}
				// Clear profile delivery statuses so they get re-delivered
				// on the new enrollment.
				if _, err := tx.ExecContext(ctx, delProfilesStmt, hostUUID.String); err != nil {
					return ctxerr.Wrap(ctx, err, "delete host_mdm_windows_profiles for host")
				}
			}

		case sql.ErrNoRows:
			// nothing to delete, return early
			return ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice"))

		default:
			return ctxerr.Wrap(ctx, err, "load host_uuid for MDMWindowsEnrolledDevice")
		}

		res, err := tx.ExecContext(ctx, delStmt, mdmDeviceHWID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete MDMWindowsEnrolledDevice")
		}

		deleted, _ := res.RowsAffected()
		if deleted == 1 {
			return nil
		}

		return ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice"))
	})
}

// MDMWindowsDeleteEnrolledDeviceWithDeviceID deletes a given
// MDMWindowsEnrolledDevice entry from the database using the device id.
// It also cleans up all host_mdm_windows_profiles rows for the host since
// the device can no longer receive MDM commands after unenrollment.
func (ds *Datastore) MDMWindowsDeleteEnrolledDeviceWithDeviceID(ctx context.Context, mdmDeviceID string) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Look up host_uuid before deleting the enrollment so we can clean up profile rows.
		// Use sql.NullString since host_uuid may be NULL if the enrollment hasn't been linked to a host yet.
		var hostUUID sql.NullString
		err := sqlx.GetContext(ctx, tx,
			&hostUUID, `SELECT host_uuid FROM mdm_windows_enrollments WHERE mdm_device_id = ? ORDER BY created_at DESC LIMIT 1`, mdmDeviceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice"))
			}
			return ctxerr.Wrap(ctx, err, "looking up host_uuid for enrolled device")
		}

		res, err := tx.ExecContext(ctx, `DELETE FROM mdm_windows_enrollments WHERE mdm_device_id = ?`, mdmDeviceID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting Windows enrolled device")
		}

		deleted, _ := res.RowsAffected()
		if deleted != 1 {
			return ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice"))
		}

		// Clean up all host_mdm_windows_profiles rows for this host since it can no longer receive MDM commands.
		if hostUUID.Valid && hostUUID.String != "" {
			if _, err := tx.ExecContext(ctx,
				`DELETE FROM host_mdm_windows_profiles WHERE host_uuid = ?`, hostUUID.String); err != nil {
				return ctxerr.Wrap(ctx, err, "cleaning up Windows host MDM profiles after unenrollment")
			}
		}

		return nil
	})
}

// this function inserts both the host_mdm_windows_profile entries and the actual mdm_windows_command_queue entries for a given command and list of hosts.
// We do the host-targeting pieces in a transaction to ensure that we don't end up with queued commands that don't have corresponding host profile entries,
// which would previously cause issues when processing responses from the device if there was a long delay between enqueing the command and the host profile
// entry insertion. It is done in batches for performance reasons and the command itself is inserted before the batches begin. It need not be one big tranasaction
// as long as a given host's command queue entry and host profile entry are inserted in the same transaction. Note that unlike the insert command function below
// this does not work with device IDs, only host UUIDs
func (ds *Datastore) MDMWindowsInsertCommandAndUpsertHostProfilesForHosts(ctx context.Context, hostUUIDs []string, cmd *fleet.MDMWindowsCommand, payload []*fleet.MDMWindowsBulkUpsertHostProfilePayload) error {
	if len(hostUUIDs) == 0 {
		return nil
	}

	const defaultBatchSize = 1000
	batchSize := defaultBatchSize
	if ds.testUpsertMDMDesiredProfilesBatchSize > 0 {
		batchSize = ds.testUpsertMDMDesiredProfilesBatchSize
	}

	// Insert the command once, outside of the batched transactions.
	cmdStmt := `
		INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri)
		VALUES (?, ?, ?)
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, cmdStmt, cmd.CommandUUID, cmd.RawCommand, cmd.TargetLocURI); err != nil {
		if IsDuplicate(err) {
			return ctxerr.Wrap(ctx, alreadyExists("MDMWindowsCommand", cmd.CommandUUID))
		}
		return ctxerr.Wrap(ctx, err, "inserting MDMWindowsCommand")
	}

	// Build a map from host UUID to its corresponding profile payload for quick lookup.
	payloadByHostUUID := make(map[string]*fleet.MDMWindowsBulkUpsertHostProfilePayload, len(payload))
	for _, p := range payload {
		payloadByHostUUID[p.HostUUID] = p
	}

	// Insert command queue entries and host profile entries in batches, each
	// batch in its own transaction to limit lock contention. Each host gets
	// one command queue row and one host_mdm_windows_profiles row, inserted
	// together so they stay consistent within the batch and so we don't end
	// up with queued commands that don't have corresponding host profile entries.
	var (
		queueHostUUIDs []string
		profileArgs    []any
		profileSB      strings.Builder
		batchCount     int
	)

	executeBatch := func() error {
		return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			// Insert command queue entries via INSERT ... SELECT so that
			// hosts whose enrollment was deleted produce 0 rows instead
			// of a NULL enrollment_id / FK error.
			if len(queueHostUUIDs) > 0 {
				queueQuery, queueArgs, err := sqlx.In(`
					INSERT INTO windows_mdm_command_queue (enrollment_id, command_uuid)
					SELECT MAX(mwe.id), ?
					FROM mdm_windows_enrollments mwe
					WHERE mwe.host_uuid IN (?)
					GROUP BY mwe.host_uuid`,
					cmd.CommandUUID, queueHostUUIDs,
				)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "building IN for MDMWindowsCommandQueue insert")
				}
				if _, err := tx.ExecContext(ctx, queueQuery, queueArgs...); err != nil {
					if IsDuplicate(err) {
						return ctxerr.Wrap(ctx, alreadyExists("MDMWindowsCommandQueue", cmd.CommandUUID))
					}
					return ctxerr.Wrap(ctx, err, "batch inserting MDMWindowsCommandQueue")
				}
			}

			// Should never happen
			if len(profileArgs) == 0 {
				return nil
			}

			// Upsert host profile entries.
			profileStmt := fmt.Sprintf(`
				INSERT INTO host_mdm_windows_profiles (
					profile_uuid, host_uuid, status, operation_type,
					detail, command_uuid, profile_name, checksum
				)
				VALUES %s
				ON DUPLICATE KEY UPDATE
					status = VALUES(status),
					operation_type = VALUES(operation_type),
					detail = VALUES(detail),
					profile_name = VALUES(profile_name),
					checksum = VALUES(checksum),
					command_uuid = VALUES(command_uuid)`,
				strings.TrimSuffix(profileSB.String(), ","),
			)
			if _, err := tx.ExecContext(ctx, profileStmt, profileArgs...); err != nil {
				return ctxerr.Wrap(ctx, err, "batch upserting host_mdm_windows_profiles")
			}

			return nil
		})
	}

	resetBatch := func() {
		batchCount = 0
		queueHostUUIDs = queueHostUUIDs[:0]
		profileArgs = profileArgs[:0]
		profileSB.Reset()
	}

	for _, hostUUID := range hostUUIDs {
		// This may seem odd running the batch up front but it helps ensure we don't run oversized batches if for instance a caller
		// makes an error leading to the warning below about mismatch host profile/command entries
		if batchCount >= batchSize {
			if err := executeBatch(); err != nil {
				return err
			}
			resetBatch()
		}

		// Host profile entry.
		p := payloadByHostUUID[hostUUID]

		if p == nil {
			ds.logger.WarnContext(ctx, "skipping host with no corresponding profile payload", "host_uuid", hostUUID, "command_uuid", cmd.CommandUUID)
			continue
		}

		batchCount++
		queueHostUUIDs = append(queueHostUUIDs, hostUUID)
		profileSB.WriteString("(?, ?, ?, ?, ?, ?, ?, ?),")
		profileArgs = append(profileArgs, p.ProfileUUID, p.HostUUID, p.Status, p.OperationType, p.Detail, p.CommandUUID, p.ProfileName, p.Checksum)

	}

	if batchCount > 0 {
		if err := executeBatch(); err != nil {
			return err
		}
	}

	return nil
}

func (ds *Datastore) MDMWindowsInsertCommandForHosts(ctx context.Context, hostUUIDsOrDeviceIDs []string, cmd *fleet.MDMWindowsCommand) error {
	if len(hostUUIDsOrDeviceIDs) == 0 {
		return nil
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return ds.mdmWindowsInsertCommandForHostsDB(ctx, tx, hostUUIDsOrDeviceIDs, cmd)
	})
}

func (ds *Datastore) mdmWindowsInsertCommandForHostsDB(ctx context.Context, tx sqlx.ExtContext, hostUUIDsOrDeviceIDs []string, cmd *fleet.MDMWindowsCommand) error {
	// Resolve host UUIDs / device IDs to enrollment IDs using the general-purpose
	// lookup (supports both host_uuid and mdm_device_id via subquery).
	enrollmentIDs, err := ds.getEnrollmentIDsByHostUUIDOrDeviceIDDB(ctx, tx, hostUUIDsOrDeviceIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching enrollment IDs for command queue")
	}
	return ds.mdmWindowsInsertCommandForEnrollmentIDsDB(ctx, tx, enrollmentIDs, cmd)
}

// mdmWindowsInsertCommandForHostUUIDsDB is the fast path for bulk operations
// that always have host UUIDs (not device IDs). Uses an indexed batch SELECT
// instead of per-row subqueries.
func (ds *Datastore) mdmWindowsInsertCommandForHostUUIDsDB(ctx context.Context, tx sqlx.ExtContext, hostUUIDs []string, cmd *fleet.MDMWindowsCommand) error {
	enrollmentIDs, err := ds.getEnrollmentIDsByHostUUIDDB(ctx, tx, hostUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching enrollment IDs by host UUID")
	}
	return ds.mdmWindowsInsertCommandForEnrollmentIDsDB(ctx, tx, enrollmentIDs, cmd)
}

// mdmWindowsInsertCommandForEnrollmentIDsDB inserts the command and queues it
// for the given enrollment IDs.
func (ds *Datastore) mdmWindowsInsertCommandForEnrollmentIDsDB(ctx context.Context, tx sqlx.ExtContext, enrollmentIDs []uint, cmd *fleet.MDMWindowsCommand) error {
	// Create the command entry.
	stmt := `INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri) VALUES (?, ?, ?)`
	if _, err := tx.ExecContext(ctx, stmt, cmd.CommandUUID, cmd.RawCommand, cmd.TargetLocURI); err != nil {
		if IsDuplicate(err) {
			return ctxerr.Wrap(ctx, alreadyExists("MDMWindowsCommand", cmd.CommandUUID))
		}
		return ctxerr.Wrap(ctx, err, "inserting MDMWindowsCommand")
	}

	if len(enrollmentIDs) == 0 {
		return nil
	}

	// Batch insert into command queue.
	return common_mysql.BatchProcessSimple(enrollmentIDs, windowsMDMProfileDeleteBatchSize, func(batch []uint) error {
		valuesPart := strings.Repeat("(?, ?),", len(batch))
		valuesPart = strings.TrimSuffix(valuesPart, ",")

		args := make([]any, 0, len(batch)*2)
		for _, eid := range batch {
			args = append(args, eid, cmd.CommandUUID)
		}

		batchStmt := `INSERT INTO windows_mdm_command_queue (enrollment_id, command_uuid) VALUES ` + valuesPart
		if _, err := tx.ExecContext(ctx, batchStmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "batch inserting MDMWindowsCommandQueue")
		}
		return nil
	})
}

// getEnrollmentIDsByHostUUIDDB fetches enrollment IDs for a list of host UUIDs
// using an indexed batch query. Returns the most recent enrollment per host.
func (ds *Datastore) getEnrollmentIDsByHostUUIDDB(ctx context.Context, tx sqlx.ExtContext, hostUUIDs []string) ([]uint, error) {
	var allIDs []uint
	err := common_mysql.BatchProcessSimple(hostUUIDs, windowsMDMProfileDeleteBatchSize, func(batch []string) error {
		stmt, args, err := sqlx.In(
			`SELECT MAX(id) FROM mdm_windows_enrollments WHERE host_uuid IN (?) GROUP BY host_uuid`,
			batch)
		if err != nil {
			return err
		}
		var ids []uint
		if err := sqlx.SelectContext(ctx, tx, &ids, stmt, args...); err != nil {
			return err
		}
		allIDs = append(allIDs, ids...)
		return nil
	})
	return allIDs, err
}

// getEnrollmentIDsByHostUUIDOrDeviceIDDB fetches enrollment IDs using a
// per-row SELECT that supports both host_uuid and mdm_device_id lookups.
// Used by the general-purpose command insertion path (typically 1-2 IDs).
func (ds *Datastore) getEnrollmentIDsByHostUUIDOrDeviceIDDB(ctx context.Context, tx sqlx.ExtContext, hostUUIDsOrDeviceIDs []string) ([]uint, error) {
	var allIDs []uint
	for _, id := range hostUUIDsOrDeviceIDs {
		var eid uint
		err := sqlx.GetContext(ctx, tx, &eid,
			`SELECT id FROM mdm_windows_enrollments WHERE host_uuid = ? OR mdm_device_id = ? ORDER BY created_at DESC, id DESC LIMIT 1`,
			id, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue // host not enrolled, skip
			}
			return nil, ctxerr.Wrap(ctx, err, "looking up enrollment ID")
		}
		allIDs = append(allIDs, eid)
	}
	return allIDs, nil
}

// MDMWindowsGetPendingCommands retrieves all commands awaiting execution for the given enrollment.
func (ds *Datastore) MDMWindowsGetPendingCommands(ctx context.Context, enrollmentID uint) ([]*fleet.MDMWindowsCommand, error) {
	// Fast path: probe the queue. An MDM management session runs this query on every
	// check-in, and the overwhelming majority of devices have nothing queued, so short-circuit
	// before paying for the full scan + anti-join. SELECT EXISTS always returns a row, so the
	// idle path does not go through a sql.ErrNoRows branch.
	const probe = `SELECT EXISTS(SELECT 1 FROM windows_mdm_command_queue WHERE enrollment_id = ?)`
	var hasPending bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &hasPending, probe, enrollmentID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "probe pending Windows MDM commands")
	}
	if !hasPending {
		return nil, nil
	}

	const query = `
SELECT
	wmc.command_uuid,
	wmc.raw_command,
	wmc.target_loc_uri,
	wmc.created_at,
	wmc.updated_at
FROM
	windows_mdm_command_queue wmcq
INNER JOIN
	windows_mdm_commands wmc
ON
	wmc.command_uuid = wmcq.command_uuid
WHERE
	wmcq.enrollment_id = ? AND
	NOT EXISTS (
		SELECT 1
		FROM
			windows_mdm_command_results wmcr
		WHERE
			wmcr.enrollment_id = wmcq.enrollment_id AND
			wmcr.command_uuid = wmcq.command_uuid
	)
ORDER BY
	wmc.created_at ASC
`

	var commands []*fleet.MDMWindowsCommand
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &commands, query, enrollmentID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get pending Windows MDM commands by enrollment id")
	}

	return commands, nil
}

func (ds *Datastore) MDMWindowsSaveResponse(ctx context.Context, enrolledDevice *fleet.MDMWindowsEnrolledDevice, enrichedSyncML fleet.EnrichedSyncML, commandIDsBeingResent []string) (*fleet.MDMWindowsSaveResponseResult, error) {
	if len(enrichedSyncML.Raw) == 0 {
		return nil, ctxerr.New(ctx, "empty raw response")
	}
	if enrolledDevice == nil {
		return nil, ctxerr.New(ctx, "enrolled device is nil")
	}

	var result *fleet.MDMWindowsSaveResponseResult
	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		result = nil

		// store the full response
		const saveFullRespStmt = `INSERT INTO windows_mdm_responses (enrollment_id, raw_response) VALUES (?, ?)`
		sqlResult, err := tx.ExecContext(ctx, saveFullRespStmt, enrolledDevice.ID, enrichedSyncML.Raw)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "saving full response")
		}
		responseID, _ := sqlResult.LastInsertId()

		// find commands we sent that match the UUID responses we've got
		findCommandsStmt := `SELECT command_uuid, raw_command, target_loc_uri FROM windows_mdm_commands WHERE command_uuid IN (?)`
		stmt, params, err := sqlx.In(findCommandsStmt, enrichedSyncML.CmdRefUUIDs)
		if len(commandIDsBeingResent) > 0 {
			// If we're resending any commands, avoid selecting them here
			placeholders := make([]string, len(commandIDsBeingResent))
			for i, id := range commandIDsBeingResent {
				placeholders[i] = "?"
				params = append(params, id)
			}
			stmt += fmt.Sprintf(" AND command_uuid NOT IN (%s)", strings.Join(placeholders, ","))

		}
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building IN to search matching commands")
		}
		var matchingCmds []fleet.MDMWindowsCommand
		err = sqlx.SelectContext(ctx, tx, &matchingCmds, stmt, params...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "selecting matching commands")
		}

		if len(matchingCmds) == 0 {
			if len(commandIDsBeingResent) == 0 {
				// Only log if not resending commands as we then can expect no matching commands
				ds.logger.WarnContext(ctx, "unmatched Windows MDM commands", "uuids", strings.Join(enrichedSyncML.CmdRefUUIDs, ","), "mdm_device_id",
					enrolledDevice.MDMDeviceID)
			}
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

		// Look up operation types for matching commands so we can pass isRemoveOperation to BuildMDMWindowsProfilePayloadFromMDMResponse.
		cmdOperationTypes := make(map[string]fleet.MDMOperationType)
		matchingCmdUUIDs := make([]string, 0, len(matchingCmds))
		for _, cmd := range matchingCmds {
			matchingCmdUUIDs = append(matchingCmdUUIDs, cmd.CommandUUID)
		}
		const getOpTypesStmt = `SELECT command_uuid, operation_type FROM host_mdm_windows_profiles WHERE host_uuid = ? AND command_uuid IN (?)`
		opStmt, opArgs, opErr := sqlx.In(getOpTypesStmt, enrolledDevice.HostUUID, matchingCmdUUIDs)
		if opErr != nil {
			return ctxerr.Wrap(ctx, opErr, "building IN for operation types")
		}
		var opResults []struct {
			CommandUUID   string                 `db:"command_uuid"`
			OperationType fleet.MDMOperationType `db:"operation_type"`
		}
		if err := sqlx.SelectContext(ctx, tx, &opResults, opStmt, opArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting operation types for matching commands")
		}
		for _, r := range opResults {
			cmdOperationTypes[r.CommandUUID] = r.OperationType
		}

		for _, cmd := range matchingCmds {
			statusCode := ""
			if status, ok := enrichedSyncML.CmdRefUUIDToStatus[cmd.CommandUUID]; ok && status.Data != nil {
				statusCode = *status.Data
				if status.Cmd != nil {
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
						enrolledDevice.HostUUID, cmdOperationTypes[cmd.CommandUUID] == fleet.MDMOperationTypeRemove)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "building profile payload from MDM response")
					}
					potentialProfilePayloads = append(potentialProfilePayloads, pp)
				}
			}

			rawResult := []byte{}
			if result, ok := enrichedSyncML.CmdRefUUIDToResults[cmd.CommandUUID]; ok && result.Data != nil {
				var err error
				rawResult, err = xml.Marshal(result)
				if err != nil {
					ds.logger.ErrorContext(ctx, "marshaling command result", "err", err, "cmd_uuid", cmd.CommandUUID)
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
			wipeSucceeded := strings.HasPrefix(wipeCmdStatus, "2")
			rowsAffected, err := updateHostLockWipeStatusFromResultAndHostUUID(ctx, tx, enrolledDevice.HostUUID,
				"wipe_ref", wipeCmdUUID, wipeSucceeded, false,
			)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "updating wipe command result in host_mdm_actions")
			}

			if wipeCmdStatus != "" && !wipeSucceeded && rowsAffected > 0 {
				result = &fleet.MDMWindowsSaveResponseResult{
					WipeFailed: &fleet.MDMWindowsWipeResult{
						HostUUID: enrolledDevice.HostUUID,
					},
				}
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
	}); err != nil {
		return nil, err
	}
	return result, nil
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
			(host_uuid, profile_uuid, detail, status, retries, command_uuid, checksum)
		VALUES %s
		ON DUPLICATE KEY UPDATE
			checksum = VALUES(checksum),
			detail = VALUES(detail),
			status = VALUES(status),
			retries = VALUES(retries)`

	// MySQL will use the `host_uuid` part of the primary key as a first
	// pass, and then filter that subset by `command_uuid`.
	const getMatchingHostProfilesStmt = `
		SELECT host_uuid, profile_uuid, command_uuid, retries, checksum, operation_type
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

	// Partition matching entries into upsert and delete buckets.
	var sb strings.Builder
	args = args[:0]
	var deleteCommandUUIDs []string
	for _, hp := range matchingHostProfiles {
		payload := uuidsToPayloads[hp.CommandUUID]
		if payload.Status != nil && *payload.Status == fleet.MDMDeliveryFailed {
			// Don't retry remove operations; removal is best-effort. Only retry install operations up to the max retry count.
			if hp.OperationType != fleet.MDMOperationTypeRemove && hp.Retries < mdm.MaxWindowsProfileRetries {
				// if we haven't hit the max retries, we set
				// the host profile status to nil (which causes
				// an install profile command to be enqueued
				// the next time the profile manager cron runs)
				// and increment the retry count
				payload.Status = nil
				hp.Retries++
			}
		}

		// Delete bucket: remove operations that resolved to a terminal state.
		// Removes are best-effort; both verified and failed are terminal since
		// failed removes are non-retryable and should not surface as host-level
		// failures in profile summaries.
		if hp.OperationType == fleet.MDMOperationTypeRemove && payload.Status != nil &&
			(*payload.Status == fleet.MDMDeliveryVerified || *payload.Status == fleet.MDMDeliveryFailed) {
			deleteCommandUUIDs = append(deleteCommandUUIDs, hp.CommandUUID)
			continue
		}

		args = append(args, hp.HostUUID, hp.ProfileUUID, payload.Detail, payload.Status, hp.Retries, hp.Checksum)
		sb.WriteString("(?, ?, ?, ?, ?, command_uuid, ?),")
	}

	// Execute batched UPSERT for the upsert bucket.
	values := strings.TrimSuffix(sb.String(), ",")
	if len(values) > 0 {
		stmt = fmt.Sprintf(updateHostProfilesStmt, values)
		if _, err = tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "updating host profiles")
		}
	}

	// Execute batched DELETE for terminal remove operations.
	if len(deleteCommandUUIDs) > 0 {
		deleteStmt, deleteArgs, err := sqlx.In(`
			DELETE FROM host_mdm_windows_profiles
			WHERE host_uuid = ? AND command_uuid IN (?)`,
			hostUUID, deleteCommandUUIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building IN for remove cleanup")
		}
		if _, err = tx.ExecContext(ctx, deleteStmt, deleteArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "cleaning up completed remove profiles")
		}
	}

	return nil
}

func (ds *Datastore) GetMDMWindowsCommandResults(ctx context.Context, commandUUID string, hostUUID string) ([]*fleet.MDMCommandResult, error) {
	query := `SELECT
    mwe.host_uuid,
    wmc.command_uuid,
    COALESCE(wmcr.status_code, '101') AS status,
    COALESCE(
        wmcr.updated_at,
        wmc.updated_at
    ) as updated_at,
    wmc.target_loc_uri AS request_type,
    COALESCE(wmr.raw_response, '') AS result,
    wmc.raw_command AS payload
FROM
    windows_mdm_commands wmc
    LEFT JOIN windows_mdm_command_results wmcr ON wmcr.command_uuid = wmc.command_uuid
    LEFT JOIN windows_mdm_responses wmr ON wmr.id = wmcr.response_id
    LEFT JOIN windows_mdm_command_queue wmcq ON wmcq.command_uuid = wmc.command_uuid AND wmcr.command_uuid IS NULL
    LEFT JOIN mdm_windows_enrollments mwe ON mwe.id = COALESCE(
        wmcr.enrollment_id,
        wmcq.enrollment_id
    )
WHERE
    wmc.command_uuid = ?`

	args := []any{commandUUID}
	if hostUUID != "" {
		query += " AND mwe.host_uuid = ?"
		args = append(args, hostUUID)
	}

	var results []*fleet.MDMCommandResult
	err := sqlx.SelectContext(
		ctx,
		ds.reader(ctx),
		&results,
		query,
		args...,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get command results")
	}

	return results, nil
}

func (ds *Datastore) UpdateMDMWindowsEnrollmentsHostUUID(ctx context.Context, hostUUID string, mdmDeviceID string) (bool, error) {
	// The final clause ensures we only update if the host UUID changes so we can tell the caller as this basically
	// signals a new MDM enrollment in certain cases, as it is the first time we associate a host with an enrollment
	stmt := `UPDATE mdm_windows_enrollments SET host_uuid = ? WHERE mdm_device_id = ? AND host_uuid <> ?`
	res, err := ds.writer(ctx).Exec(stmt, hostUUID, mdmDeviceID, hostUUID)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "setting host_uuid for windows enrollment")
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "checking rows affected when setting host_uuid for windows enrollment")
	}
	return aff > 0, nil
}

func (ds *Datastore) SetMDMWindowsAwaitingConfiguration(ctx context.Context, mdmDeviceID string, expectFrom, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
	stmt := `UPDATE mdm_windows_enrollments SET awaiting_configuration = ? WHERE mdm_device_id = ? AND awaiting_configuration = ?`
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, to, mdmDeviceID, expectFrom)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "set windows awaiting configuration")
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "rows affected for set windows awaiting configuration")
	}
	return aff > 0, nil
}

// whereBitLockerStatus returns a string suitable for inclusion within a SQL WHERE clause to filter by
// the given status. The caller is responsible for ensuring the status is valid. In the case of an invalid
// status, the function will return the string "FALSE". The caller should also ensure that the query in
// which this is used joins the following tables with the specified aliases:
// - host_disk_encryption_keys: hdek
// - host_mdm: hmdm
// - host_disks: hd
func (ds *Datastore) whereBitLockerStatus(ctx context.Context, status fleet.DiskEncryptionStatus, bitLockerPINRequired bool) string {
	const (
		whereNotServer        = `(hmdm.is_server IS NOT NULL AND hmdm.is_server = 0)`
		whereKeyAvailable     = `(hdek.base64_encrypted IS NOT NULL AND hdek.base64_encrypted != '' AND hdek.decryptable IS NOT NULL AND hdek.decryptable = 1)`
		whereEncrypted        = `(hd.encrypted IS NOT NULL AND hd.encrypted = 1)`
		whereHostDisksUpdated = `(hd.updated_at IS NOT NULL AND hdek.updated_at IS NOT NULL AND hd.updated_at >= hdek.updated_at)`
		whereClientError      = `(hdek.client_error IS NOT NULL AND hdek.client_error != '')`
		withinGracePeriod     = `(hdek.updated_at IS NOT NULL AND hdek.updated_at >= DATE_SUB(NOW(6), INTERVAL 1 HOUR))`
		whereProtectionOn     = `(hd.bitlocker_protection_status IS NULL OR hd.bitlocker_protection_status != 0)`
		whereProtectionOff    = `(hd.bitlocker_protection_status = 0)`
	)

	whereBitLockerPINSet := `TRUE`
	if bitLockerPINRequired {
		whereBitLockerPINSet = `(hd.tpm_pin_set = true)`
	}

	// TODO: what if windows sends us a key for an already encrypted volumne? could it get stuck
	// in pending or verifying? should we modify SetOrUpdateHostDiskEncryption to ensure that we
	// increment the updated_at timestamp on the host_disks table for all encrypted volumes
	// host_disks if the hdek timestamp is newer? What about SetOrUpdateHostDiskEncryptionKey?

	switch status {
	case fleet.DiskEncryptionVerified:
		// Verified requires protection to be on (or unknown/NULL for backward compatibility).
		return whereNotServer + `
AND NOT ` + whereClientError + `
AND ` + whereKeyAvailable + `
AND ` + whereEncrypted + `
AND ` + whereHostDisksUpdated + `
AND ` + whereProtectionOn + `
AND ` + whereBitLockerPINSet

	case fleet.DiskEncryptionVerifying:
		// Possible verifying scenarios:
		// - we have the key and host_disks already encrypted before the key but hasn't been updated yet
		// - we have the key and host_disks reported unencrypted during the 1-hour grace period after key was updated
		// Protection must be on for encrypted disks. For the grace period path (encryption
		// still in progress), protection is expected to be off so we don't check it.
		return whereNotServer + `
AND NOT ` + whereClientError + `
AND ` + whereKeyAvailable + `
AND (
    (` + whereEncrypted + ` AND NOT ` + whereHostDisksUpdated + ` AND ` + whereProtectionOn + `)
    OR (NOT ` + whereEncrypted + ` AND ` + whereHostDisksUpdated + ` AND ` + withinGracePeriod + `)
)
AND ` + whereBitLockerPINSet

	case fleet.DiskEncryptionActionRequired:
		// Action required when:
		// 1. We _would_ be in verified/verifying but PIN is required and not set, OR
		// 2. Disk is encrypted and key is escrowed but BitLocker protection is off
		//    (e.g., suspended for a BIOS update, or a TPM configuration issue)
		return whereNotServer + `
AND NOT ` + whereClientError + `
AND ` + whereKeyAvailable + `
AND (` + whereEncrypted + ` OR (NOT ` + whereEncrypted + ` AND ` + whereHostDisksUpdated + ` AND ` + withinGracePeriod + `))
AND (NOT ` + whereBitLockerPINSet + ` OR (` + whereEncrypted + ` AND ` + whereProtectionOff + `))`

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
		ds.logger.DebugContext(ctx, "unknown bitlocker status", "status", status)
		return "FALSE"
	}
}

func (ds *Datastore) GetMDMWindowsBitLockerSummary(ctx context.Context, teamID *uint) (*fleet.MDMWindowsBitLockerSummary, error) {
	diskEncryptionConfig, err := ds.GetConfigEnableDiskEncryption(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if !diskEncryptionConfig.Enabled {
		return &fleet.MDMWindowsBitLockerSummary{}, nil
	}

	// Note removing_enforcement is not applicable to Windows hosts
	sqlFmt := `
SELECT
    COUNT(if((%s), 1, NULL)) AS verified,
    COUNT(if((%s), 1, NULL)) AS verifying,
    COUNT(if((%s), 1, NULL)) AS action_required,
    COUNT(if((%s), 1, NULL)) AS enforcing,
    COUNT(if((%s), 1, NULL)) AS failed,
    0 AS removing_enforcement
FROM
    hosts h
    JOIN host_mdm hmdm ON h.id = hmdm.host_id
    JOIN mdm_windows_enrollments mwe ON h.uuid = mwe.host_uuid
    LEFT JOIN host_disk_encryption_keys hdek ON h.id = hdek.host_id
    LEFT JOIN host_disks hd ON h.id = hd.host_id
WHERE
    mwe.device_state = '%s' AND
    h.platform = 'windows' AND
    hmdm.is_server = 0 AND
    hmdm.enrolled = 1 AND
    %s`

	var args []interface{}
	teamFilter := "h.team_id IS NULL"
	if teamID != nil && *teamID > 0 {
		teamFilter = "h.team_id = ?"
		args = append(args, *teamID)
	}

	var res fleet.MDMWindowsBitLockerSummary
	stmt := fmt.Sprintf(
		sqlFmt,
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionVerified, diskEncryptionConfig.BitLockerPINRequired),
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionVerifying, diskEncryptionConfig.BitLockerPINRequired),
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionActionRequired, diskEncryptionConfig.BitLockerPINRequired),
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionEnforcing, diskEncryptionConfig.BitLockerPINRequired),
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionFailed, diskEncryptionConfig.BitLockerPINRequired),
		microsoft_mdm.MDMDeviceStateEnrolled,
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
		ds.logger.DebugContext(ctx, "no bitlocker status for server host", "host_id", host.ID)
		return nil, nil
	}

	diskEncryptionConfig, err := ds.GetConfigEnableDiskEncryption(ctx, host.TeamID)
	if err != nil {
		return nil, err
	}
	if !diskEncryptionConfig.Enabled {
		return nil, nil
	}

	stmt := fmt.Sprintf(`
SELECT
	CASE
		WHEN (%s) THEN '%s'
		WHEN (%s) THEN '%s'
		WHEN (%s) THEN '%s'
		WHEN (%s) THEN '%s'
		WHEN (%s) THEN '%s'
		ELSE ''
	END AS status,
	COALESCE(client_error, '') as detail,
	hd.bitlocker_protection_status,
	COALESCE(hd.tpm_pin_set, false) as tpm_pin_set
FROM
	host_mdm hmdm
	LEFT JOIN host_disk_encryption_keys hdek ON hmdm.host_id = hdek.host_id
	LEFT JOIN host_disks hd ON hmdm.host_id = hd.host_id
WHERE
	hmdm.host_id = ?`,
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionActionRequired, diskEncryptionConfig.BitLockerPINRequired),
		fleet.DiskEncryptionActionRequired,
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionVerified, diskEncryptionConfig.BitLockerPINRequired),
		fleet.DiskEncryptionVerified,
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionVerifying, diskEncryptionConfig.BitLockerPINRequired),
		fleet.DiskEncryptionVerifying,
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionEnforcing, diskEncryptionConfig.BitLockerPINRequired),
		fleet.DiskEncryptionEnforcing,
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionFailed, diskEncryptionConfig.BitLockerPINRequired),
		fleet.DiskEncryptionFailed,
	)

	var dest struct {
		Status           fleet.DiskEncryptionStatus `db:"status"`
		Detail           string                     `db:"detail"`
		ProtectionStatus *int                       `db:"bitlocker_protection_status"`
		TpmPinSet        bool                       `db:"tpm_pin_set"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, stmt, host.ID); err != nil {
		if err != sql.ErrNoRows {
			return &fleet.HostMDMDiskEncryption{}, err
		}
		// At this point we know disk encryption is enabled so if there are no rows for the
		// host then we treat it as enforcing and log for potential debugging
		ds.logger.DebugContext(ctx, "no bitlocker status found for host", "host_id", host.ID)
		dest.Status = fleet.DiskEncryptionEnforcing
	}

	if dest.Status == "" {
		// This is unexpected. We know that disk encryption is enabled so we treat it failed to draw
		// attention to the issue and log potential debugging
		ds.logger.DebugContext(ctx, "no bitlocker status found for host", "host_id", host.ID)
		dest.Status = fleet.DiskEncryptionFailed
	}

	// Build a meaningful detail message for action_required when there's no client error.
	if dest.Status == fleet.DiskEncryptionActionRequired && dest.Detail == "" {
		protectionOff := dest.ProtectionStatus != nil && *dest.ProtectionStatus == fleet.BitLockerProtectionStatusOff
		pinMissing := diskEncryptionConfig.BitLockerPINRequired && !dest.TpmPinSet

		switch {
		case protectionOff && pinMissing:
			dest.Detail = "BitLocker protection is off and a required startup PIN is not set. The disk is encrypted but the TPM protector is not active, and a BitLocker PIN must be configured."
		case protectionOff:
			dest.Detail = "BitLocker protection is off. The disk is encrypted but the TPM protector is not active. This may be due to a suspended BitLocker state or a TPM configuration issue."
		case pinMissing:
			dest.Detail = "A required BitLocker startup PIN is not set. The disk is encrypted but a PIN must be configured for compliance."
		}
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

	labels, err := ds.listProfileLabelsForProfiles(ctx, []string{res.ProfileUUID}, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	for _, lbl := range labels {
		switch {
		case lbl.Exclude && lbl.RequireAll:
			// this should never happen so log it for debugging
			ds.logger.DebugContext(ctx, "unsupported profile label: cannot be both exclude and require all",
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
	// SyncML bytes and team ID are needed to generate <Delete> commands and
	// scope the LocURI protection set to the profile's team.
	var profile struct {
		TeamID uint   `db:"team_id"`
		SyncML []byte `db:"syncml"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &profile,
		`SELECT team_id, syncml FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`, profileUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ctxerr.Wrap(ctx, notFound("MDMWindowsProfile").WithName(profileUUID))
		}
		return ctxerr.Wrap(ctx, err, "reading profile syncml before deletion")
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := deleteMDMWindowsConfigProfile(ctx, tx, profileUUID); err != nil {
			return err
		}

		profileContents := map[string][]byte{profileUUID: profile.SyncML}
		if err := ds.cancelWindowsHostInstallsForDeletedMDMProfiles(ctx, tx, profile.TeamID, []string{profileUUID}, profileContents); err != nil {
			return err
		}

		return nil
	})
}

func deleteMDMWindowsConfigProfile(ctx context.Context, tx sqlx.ExtContext, profileUUID string) error {
	res, err := tx.ExecContext(ctx, `DELETE FROM mdm_windows_configuration_profiles WHERE profile_uuid=?`, profileUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	deleted, _ := res.RowsAffected() // cannot fail for mysql
	if deleted != 1 {
		return ctxerr.Wrap(ctx, notFound("MDMWindowsProfile").WithName(profileUUID))
	}
	return nil
}

// cancelWindowsHostInstallsForDeletedMDMProfiles handles host-profile cleanup
// when config profiles are deleted. It uses a two-phase approach:
//   - Phase 1: Delete rows that were never sent to the device (NULL status + install)
//   - Phase 2: For rows that were sent (non-NULL status + install), generate SyncML
//     <Delete> commands and enqueue them, then mark the rows for removal.
//
// profTeamID is the team the deleted profiles belong to (0 for "No team"/Unassigned).
// It is passed by the caller rather than derived from the hosts table because
// host_mdm_windows_profiles rows can reference profile UUIDs from a host's
// previous team after the host has been moved (the rows stay marked for removal
// until the reconciler dispatches <Delete> commands). Deriving the team from
// those rows would return the host's current team, not the profile's team.
// The team is used to scope the LocURI protection set built from OTHER active
// profiles; without a fixed team scope, a profile in one team could spuriously
// suppress deletes for hosts whose rows happen to join to a different team.
func (ds *Datastore) cancelWindowsHostInstallsForDeletedMDMProfiles(
	ctx context.Context, tx sqlx.ExtContext,
	profTeamID uint, profileUUIDs []string, profileContents map[string][]byte,
) error {
	if len(profileUUIDs) == 0 {
		return nil
	}

	// Phase 0: Clean up remove+failed rows from previous failed removal attempts.
	// These are terminal: the device already processed them, nothing more to do.
	terminalStatuses := []fleet.MDMDeliveryStatus{fleet.MDMDeliveryFailed, fleet.MDMDeliveryVerified, fleet.MDMDeliveryVerifying}
	delRemStmt, delRemArgs, err := sqlx.In(`
	DELETE FROM host_mdm_windows_profiles
	WHERE profile_uuid IN (?) AND operation_type = ? AND status IN (?)`,
		profileUUIDs, fleet.MDMOperationTypeRemove, terminalStatuses)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building IN for phase 0 remove cleanup")
	}
	if _, err := tx.ExecContext(ctx, delRemStmt, delRemArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning up terminal remove rows")
	}

	// Phase 1: Delete host-profile rows that were never sent to the device.
	const delNeverSentStmt = `
	DELETE FROM host_mdm_windows_profiles
	WHERE profile_uuid IN (?) AND status IS NULL AND operation_type = ?`

	delStmt, delArgs, err := sqlx.In(delNeverSentStmt, profileUUIDs, fleet.MDMOperationTypeInstall)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building IN for phase 1 delete")
	}
	if _, err := tx.ExecContext(ctx, delStmt, delArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting never-sent host profiles")
	}

	// Phase 2: Find rows that need <Delete> commands. This includes:
	// - install rows with non-NULL status (profile was sent to device)
	// - rows already marked for removal but whose <Delete> command hasn't
	//   been sent yet (e.g. the host moved teams and the profile was flagged
	//   for removal, but the command wasn't generated before the team was deleted)
	const selectSentStmt = `
	SELECT host_uuid, profile_uuid
	FROM host_mdm_windows_profiles
	WHERE profile_uuid IN (?)
	  AND ((status IS NOT NULL AND operation_type = ?) OR (operation_type = ? AND status IS NULL))`

	selStmt, selArgs, err := sqlx.In(selectSentStmt, profileUUIDs, fleet.MDMOperationTypeInstall, fleet.MDMOperationTypeRemove)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building IN for phase 2 select")
	}
	var rowsToRemove []struct {
		HostUUID    string `db:"host_uuid"`
		ProfileUUID string `db:"profile_uuid"`
	}
	if err := sqlx.SelectContext(ctx, tx, &rowsToRemove, selStmt, selArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting sent host profiles for removal")
	}

	if len(rowsToRemove) == 0 {
		return nil
	}

	// Group hosts by profile UUID for efficient command generation.
	type removeTarget struct {
		cmdUUID   string
		hostUUIDs []string
	}
	targets := make(map[string]*removeTarget)
	for _, row := range rowsToRemove {
		t := targets[row.ProfileUUID]
		if t == nil {
			t = &removeTarget{cmdUUID: uuid.NewString()}
			targets[row.ProfileUUID] = t
		}
		t.hostUUIDs = append(t.hostUUIDs, row.HostUUID)
	}

	// Generate and enqueue <Delete> commands for each profile.
	// Track which profiles were successfully enqueued so we only
	// update rows that have a corresponding queued command.
	// Collect LocURIs from OTHER active profiles so we
	// don't send <Delete> for settings still enforced by a remaining profile.
	// This prevents deleting one profile from undoing settings in another.
	//
	// This is a two-pass approach for performance:
	// Pass 1 (team-wide): Build a global protection set from ALL other profiles
	//   in the team. This is fast and handles the common case.
	// Pass 2 (per-host, only if needed): For any LocURIs that were protected in
	//   pass 1, check if the protecting profile actually applies to each host
	//   (considering label scope). If it doesn't, send the <Delete> anyway.
	activeLocURIs := make(map[string]struct{})
	// Map each protected LocURI to the profile UUIDs that protect it,
	// so pass 2 can check per-host applicability.
	locURIToProtectingProfiles := make(map[string][]string)
	if len(profileUUIDs) > 0 {
		// Query profile UUIDs and SyncML from profiles in the same team that
		// are NOT being deleted. Failures here must not be swallowed: an empty
		// activeLocURIs would make every LocURI look safe to delete and could
		// undo settings still enforced by remaining profiles.
		const activeProfilesStmt = `
			SELECT profile_uuid, syncml FROM mdm_windows_configuration_profiles
			WHERE team_id = ? AND profile_uuid NOT IN (?)`
		apStmt, apArgs, err := sqlx.In(activeProfilesStmt, profTeamID, profileUUIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building IN for active profiles LocURI protection")
		}
		var activeProfiles []struct {
			ProfileUUID string `db:"profile_uuid"`
			SyncML      []byte `db:"syncml"`
		}
		if err := sqlx.SelectContext(ctx, tx, &activeProfiles, apStmt, apArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting active profiles for LocURI protection")
		}
		for _, ap := range activeProfiles {
			// Substitute SCEP variable so LocURIs are compared on
			// resolved paths, consistent with the deleted profile side.
			resolved := fleet.FleetVarSCEPWindowsCertificateIDRegexp.ReplaceAll(ap.SyncML, []byte(ap.ProfileUUID))
			for _, uri := range fleet.ExtractLocURIsFromProfileBytes(resolved) {
				activeLocURIs[uri] = struct{}{}
				locURIToProtectingProfiles[uri] = append(locURIToProtectingProfiles[uri], ap.ProfileUUID)
			}
		}
	}

	enqueuedTargets := make(map[string]*removeTarget)
	var pass2Params []locURIProtectionParams
	for profUUID, target := range targets {
		syncML, ok := profileContents[profUUID]
		if !ok || len(syncML) == 0 {
			ds.logger.WarnContext(ctx, "skipping delete command generation: no SyncML content", "profile.uuid", profUUID)
			continue
		}

		// Extract all LocURIs from this profile (done once, reused for pass 2).
		allURIs := fleet.ExtractLocURIsFromProfileBytes(
			fleet.FleetVarSCEPWindowsCertificateIDRegexp.ReplaceAll(syncML, []byte(profUUID)),
		)

		// Partition into safe (not protected) and protected (in activeLocURIs).
		var safeURIs, protectedURIs []string
		for _, uri := range allURIs {
			if _, isProtected := activeLocURIs[uri]; isProtected {
				protectedURIs = append(protectedURIs, uri)
			} else {
				safeURIs = append(safeURIs, uri)
			}
		}

		// Generate <Delete> commands for the safe (unprotected) LocURIs.
		deleteCmd, err := fleet.BuildDeleteCommandFromLocURIs(safeURIs, target.cmdUUID)
		if err != nil {
			ds.logger.ErrorContext(ctx, "skipping delete command generation: build error",
				"profile.uuid", profUUID, "err", err)
			ctxerr.Handle(ctx, err)
			continue
		}
		if deleteCmd != nil {
			// Enqueue the primary delete command for unprotected LocURIs.
			if err := ds.mdmWindowsInsertCommandForHostUUIDsDB(ctx, tx, target.hostUUIDs, deleteCmd); err != nil {
				return ctxerr.Wrap(ctx, err, "inserting delete commands for hosts")
			}
			enqueuedTargets[profUUID] = target
		} else {
			// No primary delete command (all LocURIs protected or profile only
			// has Exec commands). Delete the host-profile rows since the config
			// profile is being removed and there's no command to track.
			delSkipStmt, delSkipArgs, delSkipErr := sqlx.In(
				`DELETE FROM host_mdm_windows_profiles WHERE profile_uuid = ? AND host_uuid IN (?)`,
				profUUID, target.hostUUIDs)
			if delSkipErr != nil {
				return ctxerr.Wrap(ctx, delSkipErr, "building IN for protected profile cleanup")
			}
			if _, err := tx.ExecContext(ctx, delSkipStmt, delSkipArgs...); err != nil {
				return ctxerr.Wrap(ctx, err, "cleaning up protected profile rows")
			}
		}

		// Collect protected URIs for pass 2 (label-scoped check).
		if len(protectedURIs) > 0 {
			pass2Params = append(pass2Params, locURIProtectionParams{
				protectedURIs: protectedURIs,
				hostUUIDs:     target.hostUUIDs,
			})
		}
	}

	// Pass 2: For LocURIs that were protected in pass 1, check if the protecting
	// profile is label-scoped and doesn't actually apply to some hosts. If so,
	// send supplemental <Delete> commands for those specific hosts.
	// This only runs when there are protected LocURIs, which is rare.
	if len(pass2Params) > 0 {
		if err := ds.checkAndEnqueueLabelScopedDeletes(ctx, tx, pass2Params, locURIToProtectingProfiles); err != nil {
			return ctxerr.Wrap(ctx, err, "label-scoped LocURI protection check")
		}
	}

	// Update host-profile rows only for profiles that had delete commands enqueued.
	// This covers both install rows (being flipped to remove) and remove+NULL rows
	// (being given a command_uuid and set to pending).
	//
	// Flatten (host_uuid, profile_uuid, cmd_uuid) triples across all profiles and
	// batch them into a single UPDATE per batch. Each batch can span multiple
	// profiles, with a CASE mapping each row's profile_uuid to its command_uuid.
	// The WHERE clause uses a tuple IN on (host_uuid, profile_uuid), which matches
	// the PK and lets the optimizer perform direct PK point lookups. This avoids
	// the previous per-profile loop, which under-utilized batches when profiles
	// affected fewer than batchSize hosts.
	//
	// Profile UUIDs are iterated in sorted order so concurrent callers
	// acquire InnoDB row locks on host_mdm_windows_profiles in the same
	// order, reducing the deadlock surface on this path. The SQL text
	// itself is placeholder-only and already deterministic for a given
	// batch size, so iteration order does not affect plan-cache / query
	// digest stability.
	type pendingRemoveRow struct {
		hostUUID    string
		profileUUID string
		cmdUUID     string
	}
	sortedProfUUIDs := slices.Sorted(maps.Keys(enqueuedTargets))
	totalRows := 0
	for _, profUUID := range sortedProfUUIDs {
		totalRows += len(enqueuedTargets[profUUID].hostUUIDs)
	}
	rows := make([]pendingRemoveRow, 0, totalRows)
	for _, profUUID := range sortedProfUUIDs {
		target := enqueuedTargets[profUUID]
		for _, hostUUID := range target.hostUUIDs {
			rows = append(rows, pendingRemoveRow{
				hostUUID:    hostUUID,
				profileUUID: profUUID,
				cmdUUID:     target.cmdUUID,
			})
		}
	}

	if err := common_mysql.BatchProcessSimple(rows, windowsMDMProfileDeleteBatchSize, func(batch []pendingRemoveRow) error {
		// Collect the profile_uuid -> cmd_uuid mapping needed by this batch. Most
		// batches span 1 to N profiles; we only need one CASE arm per distinct
		// profile in the batch.
		profileCmds := make(map[string]string)
		for _, r := range batch {
			profileCmds[r.profileUUID] = r.cmdUUID
		}
		sortedBatchProfUUIDs := slices.Sorted(maps.Keys(profileCmds))

		var sb strings.Builder
		sb.WriteString(`UPDATE host_mdm_windows_profiles
			SET operation_type = ?,
			    status = ?,
			    detail = '',
			    command_uuid = CASE profile_uuid`)
		args := make([]any, 0, 2+2*len(profileCmds)+2*len(batch))
		args = append(args, fleet.MDMOperationTypeRemove, fleet.MDMDeliveryPending)
		for _, profUUID := range sortedBatchProfUUIDs {
			sb.WriteString(" WHEN ? THEN ?")
			args = append(args, profUUID, profileCmds[profUUID])
		}
		// ELSE command_uuid is defensive: WHERE restricts the update to rows
		// whose profile_uuid is present in profileCmds, so in practice every
		// updated row matches a WHEN arm.
		sb.WriteString(` ELSE command_uuid END
			WHERE (host_uuid, profile_uuid) IN (`)
		for i, r := range batch {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString("(?,?)")
			args = append(args, r.hostUUID, r.profileUUID)
		}
		sb.WriteByte(')')

		if _, err := tx.ExecContext(ctx, sb.String(), args...); err != nil {
			return ctxerr.Wrap(ctx, err, "updating host profiles to remove")
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// checkAndEnqueueLabelScopedDeletes identifies which protecting profiles are
// label-scoped and, if any, sends supplemental <Delete> commands for hosts
// where the protector doesn't apply.
func (ds *Datastore) checkAndEnqueueLabelScopedDeletes(
	ctx context.Context,
	tx sqlx.ExtContext,
	toCheck []locURIProtectionParams,
	locURIToProtectingProfiles map[string][]string,
) error {
	// Collect all protecting profile UUIDs.
	allProtectingUUIDs := make(map[string]struct{})
	for _, uuids := range locURIToProtectingProfiles {
		for _, u := range uuids {
			allProtectingUUIDs[u] = struct{}{}
		}
	}
	if len(allProtectingUUIDs) == 0 {
		return nil
	}

	// Check which are label-scoped.
	lsStmt, lsArgs, lsErr := sqlx.In(
		`SELECT DISTINCT windows_profile_uuid FROM mdm_configuration_profile_labels
		WHERE windows_profile_uuid IN (?)`, slices.Collect(maps.Keys(allProtectingUUIDs)))
	if lsErr != nil {
		return ctxerr.Wrap(ctx, lsErr, "building IN for label-scoped profile check")
	}
	var labelScoped []string
	if err := sqlx.SelectContext(ctx, tx, &labelScoped, lsStmt, lsArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "querying label-scoped profiles")
	}

	labelScopedProfiles := make(map[string]struct{})
	for _, u := range labelScoped {
		labelScopedProfiles[u] = struct{}{}
	}
	if len(labelScopedProfiles) == 0 {
		return nil
	}

	return ds.enqueueSupplementalDeletesForLabelScopedProtection(
		ctx, tx, toCheck, locURIToProtectingProfiles, labelScopedProfiles)
}

// locURIProtectionParams holds the data needed by enqueueSupplementalDeletesForLabelScopedProtection.
type locURIProtectionParams struct {
	// protectedURIs are the LocURIs from this profile that were filtered
	// out by the team-wide protection in pass 1 (i.e., another profile in
	// the team also targets them). Pass 2 checks per-host if the protector
	// actually applies.
	protectedURIs []string
	hostUUIDs     []string
}

// enqueueSupplementalDeletesForLabelScopedProtection handles pass 2 of
// LocURI protection. For each profile being deleted, it checks if any
// protected LocURIs are only protected by label-scoped profiles. If a
// label-scoped protector doesn't actually apply to a host, a supplemental
// <Delete> is enqueued for that host.
//
// Label type handling (include-any, include-all, exclude-any): this function
// does NOT re-implement label matching logic. Instead, it checks
// host_mdm_windows_profiles for an existing install assignment. The reconciler
// already evaluated all label types when it created those rows, so a row with
// operation_type='install' means the profile applies to that host regardless
// of how the label matching was computed.
//
// This is only called when there are protected LocURIs AND at least one
// protecting profile is label-scoped, which is rare.
func (ds *Datastore) enqueueSupplementalDeletesForLabelScopedProtection(
	ctx context.Context,
	tx sqlx.ExtContext,
	profilesToCheck []locURIProtectionParams,
	locURIToProtectingProfiles map[string][]string,
	labelScopedProfiles map[string]struct{},
) error {
	for _, p := range profilesToCheck {
		if len(p.protectedURIs) == 0 || len(p.hostUUIDs) == 0 {
			continue
		}

		// Filter to LocURIs where at least one protector is label-scoped.
		var labelProtectedURIs []string
		for _, uri := range p.protectedURIs {
			for _, protector := range locURIToProtectingProfiles[uri] {
				if _, isScoped := labelScopedProfiles[protector]; isScoped {
					labelProtectedURIs = append(labelProtectedURIs, uri)
					break
				}
			}
		}
		if len(labelProtectedURIs) == 0 {
			continue
		}

		// Batch: get which label-scoped protecting profiles are installed on which hosts.
		type hostProfile struct {
			HostUUID    string `db:"host_uuid"`
			ProfileUUID string `db:"profile_uuid"`
		}
		var hostProfs []hostProfile
		hpStmt, hpArgs, hpErr := sqlx.In(
			`SELECT host_uuid, profile_uuid FROM host_mdm_windows_profiles
			WHERE host_uuid IN (?) AND profile_uuid IN (?) AND operation_type = 'install'`,
			p.hostUUIDs, slices.Collect(maps.Keys(labelScopedProfiles)))
		if hpErr != nil {
			return ctxerr.Wrap(ctx, hpErr, "building IN for host-profile label check")
		}
		if err := sqlx.SelectContext(ctx, tx, &hostProfs, hpStmt, hpArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "querying host-profile assignments for label check")
		}

		hostHasProfile := make(map[string]map[string]struct{})
		for _, hp := range hostProfs {
			if hostHasProfile[hp.HostUUID] == nil {
				hostHasProfile[hp.HostUUID] = make(map[string]struct{})
			}
			hostHasProfile[hp.HostUUID][hp.ProfileUUID] = struct{}{}
		}

		// For each host, determine which protected LocURIs are safe to delete.
		// Group hosts by their safe-URI set so we can batch the command insertion.
		// Key: sorted comma-joined URIs; Value: list of host UUIDs.
		hostsByURISet := make(map[string][]string)
		for _, hostUUID := range p.hostUUIDs {
			var hostSafeURIs []string
			for _, uri := range labelProtectedURIs {
				protectorApplies := false
				for _, protectorUUID := range locURIToProtectingProfiles[uri] {
					if _, isScoped := labelScopedProfiles[protectorUUID]; !isScoped {
						protectorApplies = true // non-label profile, always applies
						break
					}
					if _, ok := hostHasProfile[hostUUID][protectorUUID]; ok {
						protectorApplies = true
						break
					}
				}
				if !protectorApplies {
					hostSafeURIs = append(hostSafeURIs, uri)
				}
			}
			if len(hostSafeURIs) > 0 {
				slices.Sort(hostSafeURIs)
				key := strings.Join(hostSafeURIs, ",")
				hostsByURISet[key] = append(hostsByURISet[key], hostUUID)
			}
		}

		// One command per unique URI set, shared across all hosts in the group.
		for uriKey, hostUUIDs := range hostsByURISet {
			uris := strings.Split(uriKey, ",")
			cmdUUID := uuid.NewString()
			deleteCmd, err := fleet.BuildDeleteCommandFromLocURIs(uris, cmdUUID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "building supplemental delete command")
			}
			if deleteCmd == nil {
				continue
			}
			if err := ds.mdmWindowsInsertCommandForHostUUIDsDB(ctx, tx, hostUUIDs, deleteCmd); err != nil {
				return ctxerr.Wrap(ctx, err, "enqueuing supplemental delete for label-scoped LocURI")
			}
		}
	}
	return nil
}

func (ds *Datastore) DeleteMDMWindowsConfigProfileByTeamAndName(ctx context.Context, teamID *uint, profileName string) error {
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	// Read the profile UUID and SyncML before the transaction to keep it short.
	var profile struct {
		ProfileUUID string `db:"profile_uuid"`
		SyncML      []byte `db:"syncml"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &profile,
		`SELECT profile_uuid, syncml FROM mdm_windows_configuration_profiles WHERE team_id=? AND name=?`,
		globalOrTeamID, profileName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil // nothing to delete
		}
		return ctxerr.Wrap(ctx, err, "reading profile before deletion")
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM mdm_windows_configuration_profiles WHERE profile_uuid=?`, profile.ProfileUUID); err != nil {
			return ctxerr.Wrap(ctx, err)
		}

		profileContents := map[string][]byte{profile.ProfileUUID: profile.SyncML}
		return ds.cancelWindowsHostInstallsForDeletedMDMProfiles(ctx, tx, globalOrTeamID, []string{profile.ProfileUUID}, profileContents)
	})
}

// windowsHostProfileStatusSubquery returns a correlated SQL scalar subquery
// that resolves to one of `<statusPrefix>failed`, `<statusPrefix>pending`,
// `<statusPrefix>verifying`, `<statusPrefix>verified`, or '<empty>' for the host
// identified by h.uuid in the outer query.
//
// The subquery does a single aggregation pass over host_mdm_windows_profiles
// via the PK(host_uuid, profile_uuid) prefix.
//
// The returned SQL does NOT include outer parentheses; callers wrap in
// `(...)` as needed for the context (scalar subquery or CASE switch).
//
// Priority logic:
//   - failed: any non-reserved profile has status='failed'.
//   - pending: any non-reserved profile has status NULL or 'pending'.
//   - verifying: at least one non-reserved install-type profile has
//     status='verifying'.
//     At this CASE branch we already know failed=0 and pending=0, so no
//     profile has status NULL/pending/failed; since profile status is always
//     one of {NULL,pending,failed,verifying,verified}, that leaves only
//     verifying and verified for install-type rows.
//   - verified: at least one non-reserved install-type profile has
//     status='verified' and no install verifying exists (enforced by the
//     earlier verifying branch).
func windowsHostProfileStatusSubquery(statusPrefix string) (string, []any, error) {
	reserved := mdm.ListFleetReservedWindowsProfileNames()

	stmt := fmt.Sprintf(`
        SELECT CASE
            WHEN SUM(CASE WHEN hmwp.status = ? AND hmwp.profile_name NOT IN (?) THEN 1 ELSE 0 END) > 0
                THEN '%sfailed'
            WHEN SUM(CASE WHEN (hmwp.status IS NULL OR hmwp.status = ?) AND hmwp.profile_name NOT IN (?) THEN 1 ELSE 0 END) > 0
                THEN '%spending'
            WHEN SUM(CASE WHEN hmwp.operation_type = ? AND hmwp.status = ? AND hmwp.profile_name NOT IN (?) THEN 1 ELSE 0 END) > 0
                THEN '%sverifying'
            WHEN SUM(CASE WHEN hmwp.operation_type = ? AND hmwp.status = ? AND hmwp.profile_name NOT IN (?) THEN 1 ELSE 0 END) > 0
                THEN '%sverified'
            ELSE ''
        END
        FROM host_mdm_windows_profiles hmwp
        WHERE hmwp.host_uuid = h.uuid`,
		statusPrefix, statusPrefix, statusPrefix, statusPrefix,
	)

	args := []any{
		fleet.MDMDeliveryFailed, reserved,
		fleet.MDMDeliveryPending, reserved,
		fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying, reserved,
		fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified, reserved,
	}

	return sqlx.In(stmt, args...)
}

func (ds *Datastore) GetMDMWindowsProfilesSummary(ctx context.Context, teamID *uint) (*fleet.MDMProfilesSummary, error) {
	diskEncryptionConfig, err := ds.GetConfigEnableDiskEncryption(ctx, teamID)
	if err != nil {
		return nil, err
	}

	var counts []statusCounts
	if !diskEncryptionConfig.Enabled {
		counts, err = getMDMWindowsStatusCountsProfilesOnlyDB(ctx, ds, teamID)
	} else {
		counts, err = getMDMWindowsStatusCountsProfilesAndBitLockerDB(ctx, ds, teamID, diskEncryptionConfig.BitLockerPINRequired)
	}
	if err != nil {
		return nil, err
	}

	var res fleet.MDMProfilesSummary
	// Note that hosts with "BitLocker action required" are counted as pending.
	for _, c := range counts {
		switch c.Status {
		case "failed":
			res.Failed = c.Count
		case "pending":
			res.Pending += c.Count
		case "verifying":
			res.Verifying = c.Count
		case "verified":
			res.Verified = c.Count
		case "action_required":
			res.Pending += c.Count
		case "":
			ds.logger.DebugContext(ctx, fmt.Sprintf("counted %d windows hosts on team %v with mdm turned on but no profiles or bitlocker status", c.Count, teamID))
		default:
			return nil, ctxerr.New(ctx, fmt.Sprintf("unexpected mdm windows status count: status=%s, count=%d", c.Status, c.Count))
		}
	}

	return &res, nil
}

type statusCounts struct {
	Status string `db:"final_status"`
	Count  uint   `db:"count"`
}

func getMDMWindowsStatusCountsProfilesOnlyDB(ctx context.Context, ds *Datastore, teamID *uint) ([]statusCounts, error) {
	profilesStatus, profilesStatusArgs, err := windowsHostProfileStatusSubquery("")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "windows host profile status subquery")
	}

	args := make([]any, 0, len(profilesStatusArgs)+1)
	args = append(args, profilesStatusArgs...)

	teamFilter := "h.team_id IS NULL"
	if teamID != nil && *teamID > 0 {
		teamFilter = "h.team_id = ?"
		args = append(args, *teamID)
	}

	// profilesStatus is a correlated scalar subquery that does one aggregation
	// pass over host_mdm_windows_profiles per host (via the PK(host_uuid,
	// profile_uuid) prefix) and resolves directly to one of
	// 'failed'|'pending'|'verifying'|'verified'|''. It replaces the previous
	// four correlated EXISTS (three with a nested NOT EXISTS) with a single
	// PK range scan per outer row. The outer SELECT/FROM/WHERE/GROUP BY shape
	// is preserved verbatim so row-level counts (including duplicate enrolled
	// rows in mdm_windows_enrollments, if any) match the prior implementation.
	stmt := fmt.Sprintf(`
SELECT
    (%s) AS final_status,
    SUM(1) AS count
FROM
    hosts h
    JOIN host_mdm hmdm ON h.id = hmdm.host_id
    JOIN mdm_windows_enrollments mwe ON h.uuid = mwe.host_uuid
WHERE
    mwe.device_state = '%s' AND
    h.platform = 'windows' AND
    hmdm.is_server = 0 AND
    hmdm.enrolled = 1 AND
    %s
GROUP BY
    final_status`,
		profilesStatus,
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

func getMDMWindowsStatusCountsProfilesAndBitLockerDB(ctx context.Context, ds *Datastore, teamID *uint, bitLockerPINRequired bool) ([]statusCounts, error) {
	profilesStatus, profilesStatusArgs, err := windowsHostProfileStatusSubquery("profiles_")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "windows host profile status subquery")
	}

	args := make([]any, 0, len(profilesStatusArgs)+1)
	args = append(args, profilesStatusArgs...)

	teamFilter := "h.team_id IS NULL"
	if teamID != nil && *teamID > 0 {
		teamFilter = "h.team_id = ?"
		args = append(args, *teamID)
	}

	bitlockerStatus := fmt.Sprintf(`
            CASE WHEN (%s) THEN
                'bitlocker_verified'
            WHEN (%s) THEN
                'bitlocker_verifying'
            WHEN (%s) THEN
                'bitlocker_action_required'
            WHEN (%s) THEN
                'bitlocker_pending'
            WHEN (%s) THEN
                'bitlocker_failed'
            ELSE
                ''
            END`,
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionVerified, bitLockerPINRequired),
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionVerifying, bitLockerPINRequired),
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionActionRequired, bitLockerPINRequired),
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionEnforcing, bitLockerPINRequired),
		ds.whereBitLockerStatus(ctx, fleet.DiskEncryptionFailed, bitLockerPINRequired),
	)

	// profilesStatus is a scalar subquery that does one aggregation pass over
	// host_mdm_windows_profiles per host (correlated on h.uuid).
	stmt := fmt.Sprintf(`
SELECT
    CASE (%s)
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
        WHEN 'bitlocker_action_required' THEN
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
        WHEN 'bitlocker_action_required' THEN
            'pending'
        WHEN 'bitlocker_verifying' THEN
            'verifying'
        ELSE
            'verified'
        END)
    ELSE
        REPLACE((%s), 'bitlocker_', '')
    END as final_status,
    SUM(1) as count
FROM
    hosts h
    JOIN host_mdm hmdm ON h.id = hmdm.host_id
    JOIN mdm_windows_enrollments mwe ON h.uuid = mwe.host_uuid
    LEFT JOIN host_disk_encryption_keys hdek ON hdek.host_id = h.id
    LEFT JOIN host_disks hd ON hd.host_id = h.id
WHERE
    mwe.device_state = '%s' AND
    h.platform = 'windows' AND
    hmdm.is_server = 0 AND
    hmdm.enrolled = 1 AND
    %s
GROUP BY
    final_status`,
		profilesStatus,
		bitlockerStatus,
		bitlockerStatus,
		bitlockerStatus,
		bitlockerStatus,
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
		mwcp.checksum,
		mwcp.secrets_updated_at,
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
		mwcp.checksum,
		mwcp.secrets_updated_at,
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
		mwcp.checksum,
		mwcp.secrets_updated_at,
		h.uuid as host_uuid,
		COUNT(*) as count_profile_labels,
		COUNT(mcpl.label_id) as count_non_broken_labels,
		COUNT(lm.label_id) as count_host_labels,
		-- this helps avoid the case where the host is not a member of a label
		-- just because it hasn't reported results for that label yet. But we
		-- only need consider this for dynamic labels - manual(type=1) can be
		-- considered at any time
		SUM(
			CASE WHEN lbl.label_membership_type <> 1 AND lbl.created_at IS NOT NULL AND h.label_updated_at >= lbl.created_at THEN 1
			WHEN lbl.label_membership_type = 1 AND lbl.created_at IS NOT NULL THEN 1
			ELSE 0 END) as count_host_updated_after_labels
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
		mwcp.checksum,
		mwcp.secrets_updated_at,
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
		result, err = ds.listAllMDMWindowsProfilesToInstallDB(ctx, tx)
		return err
	})
	return result, err
}

// The query below is a set difference between:
//
//   - Set A (ds), the "desired state", can be obtained from a JOIN between
//     mdm_windows_configuration_profiles and hosts.
//
// - Set B, the "current state" given by host_mdm_windows_profiles.
//
// A - B gives us the profiles that need to be installed:
//
//   - profiles that are in A but not in B
//
//   - profiles that are in A and in B, with an operation type of "install"
//     and a NULL status. Other statuses mean that the operation is already in
//     flight (pending), the operation has been completed but is still subject
//     to independent verification by Fleet (verifying), or has reached a terminal
//     state (failed or verified). If the profile's content is edited, all relevant hosts will
//     be marked as status NULL so that it gets re-installed.
//
// Note that for label-based profiles, only fully-satisfied profiles are
// considered for installation. This means that a broken label-based profile,
// where one of the labels does not exist anymore, will not be considered for
// installation.
const windowsProfilesToInstallQuery = `
	SELECT
		ds.profile_uuid,
		ds.host_uuid,
		ds.name as profile_name,
		ds.checksum,
		ds.secrets_updated_at
	FROM ( ` + windowsMDMProfilesDesiredStateQuery + ` ) as ds
		LEFT JOIN host_mdm_windows_profiles hmwp
			ON hmwp.profile_uuid = ds.profile_uuid AND hmwp.host_uuid = ds.host_uuid
	WHERE
		-- profile or secret variables have been updated
		( hmwp.checksum != ds.checksum ) OR IFNULL(hmwp.secrets_updated_at < ds.secrets_updated_at, FALSE) OR
		-- profiles in A but not in B
		( hmwp.profile_uuid IS NULL AND hmwp.host_uuid IS NULL ) OR
		-- profiles in A and B with operation type "install" and NULL status
		( hmwp.host_uuid IS NOT NULL AND hmwp.operation_type = ? AND hmwp.status IS NULL ) OR
		-- profiles in desired state that are currently marked for removal need
		-- to be re-installed, excluding in-flight or completed removals
		( hmwp.host_uuid IS NOT NULL AND hmwp.operation_type = ? AND COALESCE(hmwp.status, '') NOT IN ('verifying', 'verified') )
`

func (ds *Datastore) listAllMDMWindowsProfilesToInstallDB(ctx context.Context, tx sqlx.ExtContext) ([]*fleet.MDMWindowsProfilePayload, error) {
	var profiles []*fleet.MDMWindowsProfilePayload
	err := sqlx.SelectContext(ctx, tx, &profiles, fmt.Sprintf(windowsProfilesToInstallQuery, "TRUE", "TRUE", "TRUE", "TRUE"), fleet.MDMOperationTypeInstall, fleet.MDMOperationTypeRemove)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "selecting windows MDM profiles to install")
	}

	return profiles, nil
}

func (ds *Datastore) listMDMWindowsProfilesToInstallDB(
	ctx context.Context,
	tx sqlx.QueryerContext,
	hostUUIDs []string,
	onlyProfileUUIDs []string,
) (profiles []*fleet.MDMWindowsProfilePayload, err error) {
	if len(hostUUIDs) == 0 {
		return profiles, nil
	}

	hostFilter := "h.uuid IN (?)"
	if len(onlyProfileUUIDs) > 0 {
		hostFilter = "mwcp.profile_uuid IN (?) AND h.uuid IN (?)"
	}

	toInstallQuery := fmt.Sprintf(windowsProfilesToInstallQuery, hostFilter, hostFilter, hostFilter, hostFilter)

	// use a 10k host batch size to match what we do on the macOS side.
	selectProfilesBatchSize := 10_000
	if ds.testSelectMDMProfilesBatchSize > 0 {
		selectProfilesBatchSize = ds.testSelectMDMProfilesBatchSize
	}
	selectProfilesTotalBatches := int(math.Ceil(float64(len(hostUUIDs)) / float64(selectProfilesBatchSize)))

	for i := range selectProfilesTotalBatches {
		start := i * selectProfilesBatchSize
		end := min(start+selectProfilesBatchSize, len(hostUUIDs))

		batchUUIDs := hostUUIDs[start:end]

		var args []any
		var stmt string
		if len(onlyProfileUUIDs) > 0 {
			stmt, args, err = sqlx.In(
				toInstallQuery,
				onlyProfileUUIDs, batchUUIDs,
				onlyProfileUUIDs, batchUUIDs,
				onlyProfileUUIDs, batchUUIDs,
				onlyProfileUUIDs, batchUUIDs,
				fleet.MDMOperationTypeInstall, fleet.MDMOperationTypeRemove,
			)
		} else {
			stmt, args, err = sqlx.In(toInstallQuery, batchUUIDs, batchUUIDs, batchUUIDs, batchUUIDs, fleet.MDMOperationTypeInstall, fleet.MDMOperationTypeRemove)
		}
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "building sqlx.In for list MDM windows profiles to install, batch %d of %d", i, selectProfilesTotalBatches)
		}

		var partialResult []*fleet.MDMWindowsProfilePayload
		err = sqlx.SelectContext(ctx, tx, &partialResult, stmt, args...)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "selecting windows MDM profiles to install, batch %d of %d", i, selectProfilesTotalBatches)
		}

		profiles = append(profiles, partialResult...)
	}

	return profiles, nil
}

func (ds *Datastore) ListMDMWindowsProfilesToInstallForHost(ctx context.Context, hostUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
	return ds.listMDMWindowsProfilesToInstallDB(ctx, ds.reader(ctx), []string{hostUUID}, nil)
}

func (ds *Datastore) ListMDMWindowsProfilesToRemove(ctx context.Context) ([]*fleet.MDMWindowsProfilePayload, error) {
	var result []*fleet.MDMWindowsProfilePayload
	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		var err error
		result, err = ds.listAllMDMWindowsProfilesToRemoveDB(ctx, tx)
		return err
	})

	return result, err
}

// ListMDMWindowsProfilesToInstallForHosts is the scoped variant of
// ListMDMWindowsProfilesToInstall: it returns only rows for the given host
// UUIDs. Used by the cron's batched reconciliation path to bound per-tick
// work; see ReconcileWindowsProfiles.
func (ds *Datastore) ListMDMWindowsProfilesToInstallForHosts(ctx context.Context, hostUUIDs []string) ([]*fleet.MDMWindowsProfilePayload, error) {
	if len(hostUUIDs) == 0 {
		return nil, nil
	}
	return ds.listMDMWindowsProfilesToInstallDB(ctx, ds.reader(ctx), hostUUIDs, nil)
}

// ListMDMWindowsProfilesToRemoveForHosts is the scoped variant of
// ListMDMWindowsProfilesToRemove: it returns only rows for the given host
// UUIDs. Used by the cron's batched reconciliation path to bound per-tick
// work; see ReconcileWindowsProfiles.
func (ds *Datastore) ListMDMWindowsProfilesToRemoveForHosts(ctx context.Context, hostUUIDs []string) ([]*fleet.MDMWindowsProfilePayload, error) {
	if len(hostUUIDs) == 0 {
		return nil, nil
	}
	return ds.listMDMWindowsProfilesToRemoveDB(ctx, ds.reader(ctx), hostUUIDs, nil)
}

// ListNextPendingMDMWindowsHostUUIDs returns up to batchSize host UUIDs
// (sorted ascending, lexicographic) where host_uuid > afterHostUUID and
// the host has any pending Windows MDM profile reconciliation work
// (install or remove). If afterHostUUID is empty, scanning starts from
// the beginning. The cron uses this to slice its per-tick work into a
// bounded host window; see ReconcileWindowsProfiles.
func (ds *Datastore) ListNextPendingMDMWindowsHostUUIDs(ctx context.Context, afterHostUUID string, batchSize int) ([]string, error) {
	// Push the cursor predicate (host_uuid > ?) into each branch of the
	// UNION so the optimizer applies it before deduplication. The install
	// query has 4 host-filter slots, one per UNION branch in the
	// desired-state subquery; each gets h.uuid > ?. The remove query
	// inverts desired-state membership (it keeps rows where ds.host_uuid
	// IS NULL), so its 4 desired-state slots stay TRUE; the cursor goes
	// in the 5th slot, which filters hmwp.host_uuid after the RIGHT JOIN
	// to host_mdm_windows_profiles. hmwp.host_uuid is the leading column
	// of that table's PK, so this is a clean PK range scan.
	toInstall := fmt.Sprintf(windowsProfilesToInstallQuery, "h.uuid > ?", "h.uuid > ?", "h.uuid > ?", "h.uuid > ?")
	toRemove := fmt.Sprintf(windowsProfilesToRemoveQuery, "TRUE", "TRUE", "TRUE", "TRUE", "hmwp.host_uuid > ?")

	stmt := fmt.Sprintf(`
		SELECT host_uuid FROM (
			SELECT host_uuid FROM (%s) AS install_set
			UNION
			SELECT host_uuid FROM (%s) AS remove_set
		) AS combined
		ORDER BY host_uuid
		LIMIT %d
	`, toInstall, toRemove, batchSize)

	// Placeholder order in stmt:
	//   install branches: 4 cursor (h.uuid > ?), 2 op-type (install, remove)
	//   remove branches:  1 cursor (hmwp.host_uuid > ?)
	var hostUUIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostUUIDs, stmt,
		afterHostUUID, afterHostUUID, afterHostUUID, afterHostUUID,
		fleet.MDMOperationTypeInstall, fleet.MDMOperationTypeRemove,
		afterHostUUID,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing next pending MDM windows host UUIDs")
	}
	return hostUUIDs, nil
}

// GetMDMWindowsReconcileCursor returns the persisted host_uuid cursor
// used by the Windows MDM reconciliation cron to bound per-tick work.
// Returns "" if no cursor is set or if the underlying datastore does not
// support cursor persistence (the bare mysql.Datastore in unit tests
// returns "" here; the mysqlredis wrapper backs it with Redis).
//
// See ReconcileWindowsProfiles.
func (ds *Datastore) GetMDMWindowsReconcileCursor(_ context.Context) (string, error) {
	return "", nil
}

// SetMDMWindowsReconcileCursor persists the host_uuid cursor used by the
// Windows MDM reconciliation cron. The bare mysql.Datastore is a no-op
// here; the mysqlredis wrapper writes to Redis. See
// GetMDMWindowsReconcileCursor.
func (ds *Datastore) SetMDMWindowsReconcileCursor(_ context.Context, _ string) error {
	return nil
}

// The query below is a set difference between:
//
// - Set A (ds), the desired state, can be obtained from a JOIN between
// mdm_windows_configuration_profiles and hosts.
// - Set B, the current state given by host_mdm_windows_profiles.
//
// # B - A gives us the profiles that need to be removed
//
// Any other case are profiles that are in both B and A, and as such are
// processed by the ListMDMWindowsProfilesToInstall method (since they are
// in both, their desired state is necessarily to be installed).
//
// Note that for label-based profiles, only those that are fully-satisfied
// by the host are considered for install (are part of the desired state used
// to compute the ones to remove). However, as a special case, a broken
// label-based profile will NOT be removed from a host where it was
// previously installed. However, if a host used to satisfy a label-based
// profile but no longer does (and that label-based profile is not "broken"),
// the profile will be removed from the host.
const windowsProfilesToRemoveQuery = `
	SELECT
		hmwp.profile_uuid,
		hmwp.host_uuid,
		hmwp.profile_name,
		hmwp.operation_type,
		COALESCE(hmwp.detail, '') as detail,
		hmwp.status,
		hmwp.command_uuid
	FROM ( ` + windowsMDMProfilesDesiredStateQuery + ` ) as ds
		RIGHT JOIN host_mdm_windows_profiles hmwp
			ON hmwp.profile_uuid = ds.profile_uuid AND hmwp.host_uuid = ds.host_uuid
	WHERE
		-- profiles that are in B but not in A
		ds.profile_uuid IS NULL AND ds.host_uuid IS NULL AND
		-- only target hosts that still have a valid Windows MDM enrollment;
		-- orphaned host_mdm_windows_profiles rows (where the enrollment was
		-- deleted) cannot receive MDM commands and must be skipped.
		EXISTS (
			SELECT 1 FROM mdm_windows_enrollments mwe
			WHERE mwe.host_uuid = hmwp.host_uuid
		) AND
		-- exclude remove operations with non-NULL status (already processed;
		-- matches the pattern used by Fleet's Apple MDM profile removal)
		(hmwp.operation_type != 'remove' OR hmwp.status IS NULL) AND

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
`

func (ds *Datastore) listAllMDMWindowsProfilesToRemoveDB(ctx context.Context, tx sqlx.ExtContext) (profiles []*fleet.MDMWindowsProfilePayload, err error) {
	err = sqlx.SelectContext(ctx, tx, &profiles, fmt.Sprintf(windowsProfilesToRemoveQuery, "TRUE", "TRUE", "TRUE", "TRUE", "TRUE"))
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "selecting windows MDM profiles to remove")
	}

	return profiles, nil
}

func (ds *Datastore) listMDMWindowsProfilesToRemoveDB(
	ctx context.Context,
	tx sqlx.QueryerContext,
	hostUUIDs []string,
	onlyProfileUUIDs []string,
) (profiles []*fleet.MDMWindowsProfilePayload, err error) {
	if len(hostUUIDs) == 0 {
		return profiles, nil
	}

	hostFilter := "hmwp.host_uuid IN (?)"
	if len(onlyProfileUUIDs) > 0 {
		hostFilter = "hmwp.profile_uuid IN (?) AND hmwp.host_uuid IN (?)"
	}

	toRemoveQuery := fmt.Sprintf(windowsProfilesToRemoveQuery, "TRUE", "TRUE", "TRUE", "TRUE", hostFilter)

	// use a 10k host batch size to match what we do on the macOS side.
	selectProfilesBatchSize := 10_000
	if ds.testSelectMDMProfilesBatchSize > 0 {
		selectProfilesBatchSize = ds.testSelectMDMProfilesBatchSize
	}
	selectProfilesTotalBatches := int(math.Ceil(float64(len(hostUUIDs)) / float64(selectProfilesBatchSize)))

	for i := range selectProfilesTotalBatches {
		start := i * selectProfilesBatchSize
		end := min(start+selectProfilesBatchSize, len(hostUUIDs))

		batchUUIDs := hostUUIDs[start:end]

		var err error
		var args []any
		var stmt string
		if len(onlyProfileUUIDs) > 0 {
			stmt, args, err = sqlx.In(toRemoveQuery, onlyProfileUUIDs, batchUUIDs)
		} else {
			stmt, args, err = sqlx.In(toRemoveQuery, batchUUIDs)
		}
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "building sqlx.In for list MDM windows profiles to remove, batch %d of %d", i, selectProfilesTotalBatches)
		}

		var partialResult []*fleet.MDMWindowsProfilePayload
		err = sqlx.SelectContext(ctx, tx, &partialResult, stmt, args...)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "selecting windows MDM profiles to remove, batch %d of %d", i, selectProfilesTotalBatches)
		}

		profiles = append(profiles, partialResult...)
	}

	return profiles, nil
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
	      profile_name,
	      checksum
            )
            VALUES %s
	    ON DUPLICATE KEY UPDATE
              status = VALUES(status),
              operation_type = VALUES(operation_type),
              detail = VALUES(detail),
              profile_name = VALUES(profile_name),
              checksum = VALUES(checksum),
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
		args = append(args, p.ProfileUUID, p.HostUUID, p.Status, p.OperationType, p.Detail, p.CommandUUID, p.ProfileName, p.Checksum)
		sb.WriteString("(?, ?, ?, ?, ?, ?, ?, ?),")
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

// GetExistingMDMWindowsProfileUUIDs returns a set of the given profile UUIDs
// that still exist in mdm_windows_configuration_profiles. The cron
// reconciler uses this just before upserting host_mdm_windows_profiles rows
// to skip profiles that an admin deleted between the initial list and the
// upsert; without this guard a <Delete> command could never be built later
// (SyncML is gone), leaving a zombie install row.
func (ds *Datastore) GetExistingMDMWindowsProfileUUIDs(ctx context.Context, profileUUIDs []string) (map[string]struct{}, error) {
	if len(profileUUIDs) == 0 {
		return map[string]struct{}{}, nil
	}
	stmt, args, err := sqlx.In(
		`SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE profile_uuid IN (?)`,
		profileUUIDs,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building IN for existing Windows profile UUIDs")
	}
	var rows []string
	// Force a primary read: the guard exists to catch admin deletes that
	// happened seconds ago (between the cron's initial list and the upsert).
	// Replica lag could show a just-deleted profile as still present and
	// defeat the guard.
	ctx = ctxdb.RequirePrimary(ctx, true)
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting existing Windows profile UUIDs")
	}
	result := make(map[string]struct{}, len(rows))
	for _, u := range rows {
		result[u] = struct{}{}
	}
	return result, nil
}

func (ds *Datastore) GetMDMWindowsProfilesContents(ctx context.Context, uuids []string) (map[string]fleet.MDMWindowsProfileContents, error) {
	if len(uuids) == 0 {
		return nil, nil
	}

	stmt := `
          SELECT profile_uuid, syncml, checksum
          FROM mdm_windows_configuration_profiles WHERE profile_uuid IN (?)
	`
	query, args, err := sqlx.In(stmt, uuids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building in statement")
	}

	var profs []struct {
		ProfileUUID string `db:"profile_uuid"`
		SyncML      []byte `db:"syncml"`
		Checksum    []byte `db:"checksum"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &profs, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "running query")
	}

	results := make(map[string]fleet.MDMWindowsProfileContents, len(profs))
	for _, p := range profs {
		results[p.ProfileUUID] = fleet.MDMWindowsProfileContents{
			SyncML:   p.SyncML,
			Checksum: p.Checksum,
		}
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

func (ds *Datastore) NewMDMWindowsConfigProfile(ctx context.Context, cp fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
	profileUUID := "w" + uuid.New().String()
	insertProfileStmt := `
INSERT INTO
    mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml, uploaded_at)
(SELECT ?, ?, ?, ?, CURRENT_TIMESTAMP() FROM DUAL WHERE
	NOT EXISTS (
		SELECT 1 FROM mdm_apple_configuration_profiles WHERE name = ? AND team_id = ?
	) AND NOT EXISTS (
		SELECT 1 FROM mdm_apple_declarations WHERE name = ? AND team_id = ?
	) AND NOT EXISTS (
		SELECT 1 FROM mdm_android_configuration_profiles WHERE name = ? AND team_id = ?
	)
)`

	var teamID uint
	if cp.TeamID != nil {
		teamID = *cp.TeamID
	}

	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, insertProfileStmt, profileUUID, teamID, cp.Name, cp.SyncML, cp.Name, teamID, cp.Name, teamID, cp.Name, teamID)
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
		var profsWithoutLabel []string
		if len(labels) == 0 {
			profsWithoutLabel = append(profsWithoutLabel, profileUUID)
		}
		if _, err := batchSetProfileLabelAssociationsDB(ctx, tx, labels, profsWithoutLabel, "windows"); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting windows profile label associations")
		}

		// Save Fleet variables associated with this Windows profile
		if len(usesFleetVars) > 0 {
			profilesVarsToUpsert := []fleet.MDMProfileUUIDFleetVariables{
				{
					ProfileUUID:    profileUUID,
					FleetVariables: usesFleetVars,
				},
			}
			if _, err := batchSetProfileVariableAssociationsDB(ctx, tx, profilesVarsToUpsert, "windows", false); err != nil {
				return ctxerr.Wrap(ctx, err, "inserting windows profile variable associations")
			}
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
	profileUUID := fleet.MDMWindowsProfileUUIDPrefix + uuid.New().String()
	stmt := `
INSERT INTO
	mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml, uploaded_at)
(SELECT ?, ?, ?, ?, CURRENT_TIMESTAMP() FROM DUAL WHERE
	NOT EXISTS (
		SELECT 1 FROM mdm_apple_configuration_profiles WHERE name = ? AND team_id = ?
	) AND NOT EXISTS (
		SELECT 1 FROM mdm_apple_declarations WHERE name = ? AND team_id = ?
	) AND NOT EXISTS (
		SELECT 1 FROM mdm_android_configuration_profiles WHERE name = ? AND team_id = ?
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

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, profileUUID, teamID, cp.Name, cp.SyncML, cp.Name, teamID, cp.Name, teamID, cp.Name, teamID)
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
	profilesVariablesByIdentifier []fleet.MDMProfileIdentifierFleetVariables,
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

	const loadToBeDeletedProfilesNotInList = `
SELECT
	profile_uuid
FROM
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

	const loadToBeDeletedProfiles = `
SELECT
	profile_uuid
FROM
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
  ( CONCAT('` + fleet.MDMWindowsProfileUUIDPrefix + `', CONVERT(UUID() USING utf8mb4)), ?, ?, ?, CURRENT_TIMESTAMP() )
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
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "build query to load existing profiles")
		}
		if err := sqlx.SelectContext(ctx, tx, &existingProfiles, stmt, args...); err != nil {
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

	// Identify, read SyncML for, delete, and handle host cleanup for obsolete
	// profiles in a single sequential flow.
	var (
		stmt                   string
		args                   []any
		result                 sql.Result
		deletedProfileUUIDs    []string
		deletedProfileContents = make(map[string][]byte)
	)

	// Step 1: Load UUIDs of profiles to be deleted.
	if len(keepNames) > 0 {
		stmt, args, err = sqlx.In(loadToBeDeletedProfilesNotInList, profTeamID, keepNames)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "build statement to load obsolete profiles")
		}
	} else {
		stmt, args = loadToBeDeletedProfiles, []any{profTeamID}
	}
	if err = sqlx.SelectContext(ctx, tx, &deletedProfileUUIDs, stmt, args...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "load obsolete profiles")
	}

	// Step 2: Read SyncML bytes before deletion (needed to generate <Delete> commands).
	if len(deletedProfileUUIDs) > 0 {
		const readSyncMLStmt = `SELECT profile_uuid, syncml FROM mdm_windows_configuration_profiles WHERE profile_uuid IN (?)`
		rdStmt, rdArgs, rdErr := sqlx.In(readSyncMLStmt, deletedProfileUUIDs)
		if rdErr != nil {
			return false, ctxerr.Wrap(ctx, rdErr, "building IN to read deleted profile syncml")
		}
		var profileRows []struct {
			ProfileUUID string `db:"profile_uuid"`
			SyncML      []byte `db:"syncml"`
		}
		if err := sqlx.SelectContext(ctx, tx, &profileRows, rdStmt, rdArgs...); err != nil {
			return false, ctxerr.Wrap(ctx, err, "reading deleted profile syncml")
		}
		for _, r := range profileRows {
			deletedProfileContents[r.ProfileUUID] = r.SyncML
		}
	}

	// Step 3: Delete the config profile rows.
	if len(keepNames) > 0 {
		stmt, args, err = sqlx.In(deleteProfilesNotInList, profTeamID, keepNames)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "build statement to delete obsolete profiles")
		}
	} else {
		stmt, args = deleteAllProfilesForTeam, []any{profTeamID}
	}
	if result, err = tx.ExecContext(ctx, stmt, args...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "delete obsolete profiles")
	}
	rows, _ := result.RowsAffected()
	updatedDB = rows > 0

	// Step 4: Cancel pending installs and enqueue <Delete> commands for delivered profiles.
	if len(deletedProfileUUIDs) > 0 {
		if err := ds.cancelWindowsHostInstallsForDeletedMDMProfiles(ctx, tx, profTeamID, deletedProfileUUIDs, deletedProfileContents); err != nil {
			return false, ctxerr.Wrap(ctx, err, "cancel installs of deleted profiles")
		}
	}

	// For profiles being updated (same name, different content), diff the old
	// and new LocURIs. Generate <Delete> commands for LocURIs that were removed
	// so the device reverts those settings.
	//
	// This is an edge case (most edits change values, not remove LocURIs).
	// The delete commands are best-effort and currently not visible to the
	// IT admin in the UI or API. They are fire-and-forget MDM commands
	// with no corresponding host_mdm_windows_profiles status entry.

	// Two-pass LocURI protection for edited profiles:
	// Pass 1 (team-wide): Build protection set from all retained profiles.
	// Pass 2 (per-host): For protected LocURIs where the protector is
	//   label-scoped, check per-host if it actually applies.
	//
	// Known limitation: pass 2 runs before the INSERT (line ~2967) and
	// batchSetLabelAndVariableAssociations (line ~2987), so:
	//  (a) Brand-new profiles don't have UUIDs yet (generated by MySQL on
	//      INSERT), so they appear in allRetainedURIs (pass 1 protects their
	//      LocURIs) but NOT in editLocURIProtectors. Pass 2 can't check their
	//      label scope.
	//  (b) Existing profiles whose label associations change in the same batch
	//      are checked against stale mdm_configuration_profile_labels rows and
	//      stale host_mdm_windows_profiles install rows.
	// In both cases the result is over-protection: the delete is suppressed on
	// all hosts even if the protector doesn't apply. The setting stays enforced
	// on hosts outside the protector's label scope. Fixing this requires
	// restructuring so pass 2 runs after the INSERT and label association.
	allRetainedURIs := make(map[string]struct{})
	// Track which profile UUID protects which LocURI for pass 2.
	editLocURIProtectors := make(map[string][]string) // uri -> []profileUUID
	// Build name-to-UUID lookup for incoming profiles.
	incomingNameToUUID := make(map[string]string)
	for _, ep := range existingProfiles {
		incomingNameToUUID[ep.Name] = ep.ProfileUUID
	}
	for _, p := range incomingProfs {
		// Normalize SCEP placeholders so LocURIs are compared on resolved
		// paths, consistent with the delete path in cancelWindowsHostInstallsForDeletedMDMProfiles.
		resolvedSyncML := p.SyncML
		if puuid, ok := incomingNameToUUID[p.Name]; ok {
			resolvedSyncML = fleet.FleetVarSCEPWindowsCertificateIDRegexp.ReplaceAll(p.SyncML, []byte(puuid))
		}
		for _, uri := range fleet.ExtractLocURIsFromProfileBytes(resolvedSyncML) {
			allRetainedURIs[uri] = struct{}{}
			if puuid, ok := incomingNameToUUID[p.Name]; ok {
				editLocURIProtectors[uri] = append(editLocURIProtectors[uri], puuid)
			}
		}
	}
	// Include LocURIs from reserved profiles that are always kept. Reserved
	// profiles may not be in existingProfiles (which only loads profiles
	// matching incomingNames), so query them separately.
	reservedNames := mdm.ListFleetReservedWindowsProfileNames()
	if len(reservedNames) > 0 {
		rpStmt, rpArgs, rpErr := sqlx.In(
			`SELECT profile_uuid, syncml FROM mdm_windows_configuration_profiles WHERE team_id = ? AND name IN (?)`,
			profTeamID, reservedNames)
		if rpErr != nil {
			return false, ctxerr.Wrap(ctx, rpErr, "building IN for reserved profiles query")
		}
		var reservedProfiles []struct {
			ProfileUUID string `db:"profile_uuid"`
			SyncML      []byte `db:"syncml"`
		}
		if err := sqlx.SelectContext(ctx, tx, &reservedProfiles, rpStmt, rpArgs...); err != nil {
			return false, ctxerr.Wrap(ctx, err, "querying reserved profiles for LocURI protection")
		}
		for _, rp := range reservedProfiles {
			resolved := fleet.FleetVarSCEPWindowsCertificateIDRegexp.ReplaceAll(rp.SyncML, []byte(rp.ProfileUUID))
			for _, uri := range fleet.ExtractLocURIsFromProfileBytes(resolved) {
				allRetainedURIs[uri] = struct{}{}
				editLocURIProtectors[uri] = append(editLocURIProtectors[uri], rp.ProfileUUID)
			}
		}
	}

	for _, existing := range existingProfiles {
		incoming := incomingProfs[existing.Name]
		if incoming == nil || bytes.Equal(existing.SyncML, incoming.SyncML) {
			continue
		}

		// Normalize SCEP placeholders for consistent LocURI comparison.
		resolvedOld := fleet.FleetVarSCEPWindowsCertificateIDRegexp.ReplaceAll(existing.SyncML, []byte(existing.ProfileUUID))
		resolvedNew := fleet.FleetVarSCEPWindowsCertificateIDRegexp.ReplaceAll(incoming.SyncML, []byte(existing.ProfileUUID))
		oldURIs := fleet.ExtractLocURIsFromProfileBytes(resolvedOld)
		newURIs := fleet.ExtractLocURIsFromProfileBytes(resolvedNew)

		newSet := make(map[string]bool, len(newURIs))
		for _, u := range newURIs {
			newSet[u] = true
		}

		// Pass 1: team-wide protection.
		var removedURIs []string
		var protectedURIs []string
		for _, u := range oldURIs {
			if newSet[u] {
				continue // still in updated profile
			}
			if _, ok := allRetainedURIs[u]; ok {
				protectedURIs = append(protectedURIs, u)
			} else {
				removedURIs = append(removedURIs, u)
			}
		}

		// Find hosts that have this profile installed (not pending removal).
		var hostUUIDs []string
		if len(removedURIs) > 0 || len(protectedURIs) > 0 {
			if err := sqlx.SelectContext(ctx, tx, &hostUUIDs,
				`SELECT host_uuid FROM host_mdm_windows_profiles WHERE profile_uuid = ? AND operation_type = ? AND status IS NOT NULL`,
				existing.ProfileUUID, fleet.MDMOperationTypeInstall); err != nil {
				return false, ctxerr.Wrap(ctx, err, "selecting hosts for edited profile LocURI cleanup")
			}
		}

		// Send deletes for unprotected LocURIs (applies to all hosts).
		if len(removedURIs) > 0 && len(hostUUIDs) > 0 {
			cmdUUID := uuid.NewString()
			deleteCmd, err := fleet.BuildDeleteCommandFromLocURIs(removedURIs, cmdUUID)
			if err == nil && deleteCmd != nil {
				ds.logger.InfoContext(ctx, "sending delete commands for LocURIs removed from edited profile",
					"profile.name", existing.Name, "profile.uuid", existing.ProfileUUID, "removed_loc_uris", len(removedURIs))
				if err := ds.mdmWindowsInsertCommandForHostUUIDsDB(ctx, tx, hostUUIDs, deleteCmd); err != nil {
					return false, ctxerr.Wrap(ctx, err, "inserting delete commands for removed LocURIs")
				}
			}
		}

		// Pass 2: for protected LocURIs where the protector is label-scoped,
		// check per-host if the protector actually applies.
		if len(protectedURIs) > 0 && len(hostUUIDs) > 0 {
			if err := ds.checkAndEnqueueLabelScopedDeletes(
				ctx, tx,
				[]locURIProtectionParams{{
					protectedURIs: protectedURIs,
					hostUUIDs:     hostUUIDs,
				}},
				editLocURIProtectors,
			); err != nil {
				return false, ctxerr.Wrap(ctx, err, "label-scoped LocURI protection check for edited profile")
			}
		}
	}

	// insert the new profiles and the ones that have changed
	for _, p := range incomingProfs {
		if result, err = tx.ExecContext(ctx, insertNewOrEditedProfile, profTeamID, p.Name,
			p.SyncML); err != nil {
			return false, ctxerr.Wrapf(ctx, err, "insert new/edited profile with name %q", p.Name)
		}
		updatedDB = updatedDB || insertOnDuplicateDidInsertOrUpdate(result)
	}

	var mappedIncomingProfiles []*BatchSetAssociationIncomingProfile
	for _, p := range profiles {
		mappedIncomingProfiles = append(mappedIncomingProfiles, &BatchSetAssociationIncomingProfile{
			Name:             p.Name,
			ProfileUUID:      p.ProfileUUID,
			LabelsIncludeAll: p.LabelsIncludeAll,
			LabelsIncludeAny: p.LabelsIncludeAny,
			LabelsExcludeAny: p.LabelsExcludeAny,
		})
	}

	updatedLabels, err := ds.batchSetLabelAndVariableAssociations(ctx, tx, "windows", tmID, mappedIncomingProfiles, profilesVariablesByIdentifier)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "setting labels and variable associations")
	}

	return updatedDB || updatedLabels, nil
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
	COALESCE(detail, '') AS detail,
	command_uuid
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

func (ds *Datastore) GetWindowsHostMDMCertificateProfile(ctx context.Context, hostUUID string,
	profileUUID string, caName string,
) (*fleet.HostMDMCertificateProfile, error) {
	stmt := `
	SELECT
		hmwp.host_uuid,
		hmwp.profile_uuid,
		hmwp.status,
		hmmc.challenge_retrieved_at,
		hmmc.not_valid_before,
		hmmc.not_valid_after,
		hmmc.type,
		hmmc.ca_name,
		hmmc.serial
	FROM
		host_mdm_windows_profiles hmwp
	JOIN host_mdm_managed_certificates hmmc
		ON hmwp.host_uuid = hmmc.host_uuid AND hmwp.profile_uuid = hmmc.profile_uuid
	WHERE
		hmmc.host_uuid = ? AND hmmc.profile_uuid = ? AND hmmc.ca_name = ?`
	var profile fleet.HostMDMCertificateProfile
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &profile, stmt, hostUUID, profileUUID, caName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

func (ds *Datastore) GetWindowsMDMCommandsForResending(ctx context.Context, deviceID string, failedCommandIds []string) ([]*fleet.MDMWindowsCommand, error) {
	if len(failedCommandIds) == 0 {
		return []*fleet.MDMWindowsCommand{}, nil
	}

	stmt := `SELECT wmc.command_uuid, wmc.raw_command, wmc.target_loc_uri, wmc.created_at, wmc.updated_at
		FROM windows_mdm_commands wmc INNER JOIN windows_mdm_command_queue wmcq ON wmcq.enrollment_id = (SELECT id from mdm_windows_enrollments WHERE mdm_device_id = ? ORDER BY created_at DESC, id DESC LIMIT 1) AND wmcq.command_uuid = wmc.command_uuid WHERE`

	args := []any{deviceID}
	for idx, commandId := range failedCommandIds {
		if commandId == "" {
			continue
		}

		stmt += " wmc.raw_command LIKE ? OR "
		args = append(args, "%<CmdID>"+commandId+"</CmdID>%")
		if idx == len(failedCommandIds)-1 {
			stmt = strings.TrimSuffix(stmt, " OR ")
		}
	}

	if len(args) == 1 {
		// No valid command IDs were provided, return empty result to avoid returning all commands for the device.
		return []*fleet.MDMWindowsCommand{}, nil
	}

	stmt += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d", len(failedCommandIds))

	var commands []*fleet.MDMWindowsCommand
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &commands, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting windows mdm commands for resending")
	}

	return commands, nil
}

func (ds *Datastore) ResendWindowsMDMCommand(ctx context.Context, mdmDeviceId string, newCmd *fleet.MDMWindowsCommand, oldCmd *fleet.MDMWindowsCommand) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		// First clear out any existing command queue references for the host
		_, err := tx.ExecContext(ctx, `
			DELETE FROM windows_mdm_command_queue WHERE enrollment_id = (
				SELECT id FROM mdm_windows_enrollments WHERE mdm_device_id = ? ORDER BY created_at DESC, id DESC LIMIT 1
			) AND command_uuid = ?`, mdmDeviceId, oldCmd.CommandUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting existing command queue entries for old command")
		}

		if err := ds.mdmWindowsInsertCommandForHostsDB(ctx, tx, []string{mdmDeviceId}, newCmd); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting new windows mdm command for hosts")
		}

		updateStmt := fmt.Sprintf(`
			UPDATE host_mdm_windows_profiles
			SET command_uuid = ?,
			status = '%s',
			retries = retries, -- Keep retries the same to avoid endlessly resending.
			detail = ''
			WHERE host_uuid = (SELECT host_uuid FROM mdm_windows_enrollments WHERE mdm_device_id = ? ORDER BY created_at DESC, id DESC LIMIT 1) AND command_uuid = ?`, fleet.MDMDeliveryPending)
		// Keep the profile in pending while we resend with Replace.

		_, err = tx.ExecContext(ctx, updateStmt, newCmd.CommandUUID, mdmDeviceId, oldCmd.CommandUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "updating host_mdm_windows_profiles with new command uuid")
		}

		return nil
	})
}

func (ds *Datastore) MDMWindowsUpdateEnrolledDeviceCredentials(ctx context.Context, deviceId string, credentialsHash []byte) error {
	if deviceId == "" {
		return nil
	}

	_, err := ds.writer(ctx).ExecContext(ctx, `
		UPDATE mdm_windows_enrollments
		SET credentials_hash = ?
		WHERE mdm_device_id = ?`,
		credentialsHash, deviceId,
	)
	return err
}

func (ds *Datastore) MDMWindowsAcknowledgeEnrolledDeviceCredentials(ctx context.Context, deviceId string) error {
	if deviceId == "" {
		return nil
	}

	_, err := ds.writer(ctx).ExecContext(ctx, `
		UPDATE mdm_windows_enrollments
		SET credentials_acknowledged = TRUE
		WHERE mdm_device_id = ?`,
		deviceId,
	)
	return err
}
