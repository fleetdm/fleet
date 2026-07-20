package mysql

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// windowsMDMCommandQueueBatchSize is the number of enrollments/hosts to process per batch when enqueuing Windows MDM commands, resolving
// their enrollment IDs, and recomputing the denormalized has_pending_commands flag for the affected enrollments.
const windowsMDMCommandQueueBatchSize = 10000

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
		poll_schedule_relaxed,
		fleetd_sync_capable,
		has_pending_commands,
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

// setMDMWindowsEnrollmentPollScheduleRelaxedDB records the intended DMClient poll schedule for the given Windows MDM enrollment (relaxed vs
// the aggressive default), written within the caller's transaction. The management session re-enqueues a poll Replace only when the desired
// schedule differs from this, so it is written once per intended change; delivery/acknowledgment of the Replace is handled by the command queue.
func (ds *Datastore) setMDMWindowsEnrollmentPollScheduleRelaxedDB(ctx context.Context, tx sqlx.ExtContext, enrollmentID uint, relaxed bool) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE mdm_windows_enrollments SET poll_schedule_relaxed = ? WHERE id = ?`, relaxed, enrollmentID); err != nil {
		return ctxerr.Wrap(ctx, err, "set mdm windows enrollment poll schedule relaxed")
	}
	return nil
}

// MDMWindowsEnqueuePollScheduleCommand enqueues the DMClient poll-schedule Replace command and records the intended relaxed state for the
// enrollment in a single transaction.
func (ds *Datastore) MDMWindowsEnqueuePollScheduleCommand(
	ctx context.Context, mdmDeviceID string, enrollmentID uint, cmd *fleet.MDMWindowsCommand, relaxed bool,
) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := ds.mdmWindowsInsertCommandForHostsDB(ctx, tx, []string{mdmDeviceID}, cmd); err != nil {
			return ctxerr.Wrap(ctx, err, "enqueue windows MDM poll schedule command")
		}
		if err := ds.setMDMWindowsEnrollmentPollScheduleRelaxedDB(ctx, tx, enrollmentID, relaxed); err != nil {
			return ctxerr.Wrap(ctx, err, "record windows MDM poll schedule intended")
		}
		return nil
	})
}

// SetMDMWindowsEnrollmentFleetdSyncCapable persists the last-observed CapabilityWindowsMDMSync value for the host's most recent Windows MDM
// enrollment. The orbit-config endpoint calls it on-change (the live capability header is only present on that request), so the OMA-DM
// management session, which has no such header, can gate poll relaxation on the stored value.
func (ds *Datastore) SetMDMWindowsEnrollmentFleetdSyncCapable(ctx context.Context, hostUUID string, capable bool) error {
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE mdm_windows_enrollments SET fleetd_sync_capable = ? WHERE host_uuid = ? ORDER BY created_at DESC, id DESC LIMIT 1`,
		capable, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set mdm windows enrollment fleetd sync capable")
	}
	return nil
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

// MDMWindowsGetUnlinkedEnrolledDeviceWithDeviceName fetches the most recent Windows MDM enrollment whose host_uuid is
// not yet populated and whose device_name matches the given computer name. This is the BYOD-before-osquery-ingest
// fallback used by callers (e.g. the setup-experience cancel flow) that need to consult enrollment state during the
// brief window between orbit/enroll and osquery's directIngestMDMDeviceIDWindows linking host_uuid. Constraining on
// the empty host_uuid avoids matching rows already linked to a different host that happens to share the same Windows
// computer name.
func (ds *Datastore) MDMWindowsGetUnlinkedEnrolledDeviceWithDeviceName(ctx context.Context, deviceName string) (*fleet.MDMWindowsEnrolledDevice, error) {
	if deviceName == "" {
		return nil, ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice").WithMessage("empty device name"))
	}
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
		FROM mdm_windows_enrollments
		WHERE device_name = ? AND (host_uuid IS NULL OR host_uuid = '')
		ORDER BY created_at DESC, id DESC LIMIT 1`

	var winMDMDevice fleet.MDMWindowsEnrolledDevice
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &winMDMDevice, stmt, deviceName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice").WithMessage(deviceName))
		}
		return nil, ctxerr.Wrap(ctx, err, "get MDMWindowsGetUnlinkedEnrolledDeviceWithDeviceName")
	}
	return &winMDMDevice, nil
}

// WindowsHostLiteByHardwareSerial looks up a Windows host by its hardware_serial. If multiple Windows hosts share the
// same serial (e.g. VM gold images that did not regenerate SMBIOS), the caller cannot pick safely so we return NotFound.
//
// The read honors ctxdb.RequirePrimary: a caller linking a just-enrolled host (whose hosts row may have been inserted
// seconds ago) must pass a primary-required context, otherwise replica lag can return a false NotFound.
func (ds *Datastore) WindowsHostLiteByHardwareSerial(ctx context.Context, hardwareSerial string) (*fleet.HostLite, error) {
	if hardwareSerial == "" {
		return nil, ctxerr.Wrap(ctx, notFound("Host").WithMessage("empty hardware serial"))
	}
	const stmt = `
		SELECT ` + hostLiteColumns + `
		FROM hosts h
		LEFT JOIN host_seen_times hst ON h.id = hst.host_id
		WHERE h.hardware_serial = ? AND h.platform = 'windows'
		LIMIT 2`
	var hosts []*fleet.HostLite
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, stmt, hardwareSerial); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select windows host by hardware serial")
	}
	if len(hosts) != 1 {
		return nil, ctxerr.Wrap(ctx, notFound("Host").WithMessage(hardwareSerial))
	}
	return hosts[0], nil
}

// HasWindowsSetupExperienceItemsForTeam returns true if any active Windows setup-experience software
// installers with install_during_setup=TRUE are configured for the given team. teamID=0 means "no team /
// global", matching the value EnqueueSetupExperienceItems passes in for hosts on no team.
func (ds *Datastore) HasWindowsSetupExperienceItemsForTeam(ctx context.Context, teamID uint) (bool, error) {
	const stmt = `
SELECT EXISTS (
	SELECT 1 FROM software_installers
	WHERE platform = 'windows'
		AND install_during_setup = TRUE
		AND global_or_team_id = ?
		AND is_active = TRUE
)`
	var hasItems bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &hasItems, stmt, teamID); err != nil {
		return false, ctxerr.Wrap(ctx, err, "check setup experience items configured")
	}
	return hasItems, nil
}

// GetMDMWindowsHostConfigState returns the Windows MDM per-host state read on each orbit config check-in for a connected Windows host: the
// Autopilot ESP awaiting-configuration value and whether the host's most recent enrollment has queued, unacknowledged commands. This is a
// single indexed row read; has_pending_commands is a denormalized flag maintained in the enqueue/ack transactions (set directly on a
// non-poll enqueue, recomputed via EXISTS on acknowledgment), so the hot polling path never recomputes an EXISTS across the command queue.
// Internal poll-schedule Replaces are excluded from the flag, so tuning the poll cadence does not itself request an on-demand wake. Reader-backed;
// wrap the context with ctxdb.RequirePrimary for primary-routed reads.
func (ds *Datastore) GetMDMWindowsHostConfigState(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
	const stmt = `
		SELECT
			awaiting_configuration,
			has_pending_commands,
			fleetd_sync_capable
		FROM mdm_windows_enrollments
		WHERE host_uuid = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 1`
	var row struct {
		AwaitingConfiguration fleet.WindowsMDMAwaitingConfiguration `db:"awaiting_configuration"`
		HasPendingCommands    bool                                  `db:"has_pending_commands"`
		FleetdSyncCapable     bool                                  `db:"fleetd_sync_capable"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice").WithMessage(hostUUID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get MDMWindowsHostConfigState")
	}
	return &fleet.MDMWindowsHostConfigState{
		AwaitingConfiguration: row.AwaitingConfiguration,
		HasPendingCommands:    row.HasPendingCommands,
		FleetdSyncCapable:     row.FleetdSyncCapable,
	}, nil
}

// windowsMDMHasPendingCommandsExpr computes whether an enrollment (aliased e) has queued, unacknowledged commands other than internal
// poll-schedule Replaces. It backs the denormalized mdm_windows_enrollments.has_pending_commands flag. The single ? placeholder is the
// poll-schedule LocURI to exclude.
//
// Pending is "queued with acked_at still NULL": the ack transaction stamps acked_at on the rows it records results for
// (soft dequeue), so this is an index probe on (enrollment_id, acked_at) over only the actually-pending rows.
const windowsMDMHasPendingCommandsExpr = `EXISTS (
	SELECT 1
	FROM windows_mdm_command_queue q
	JOIN windows_mdm_commands c ON c.command_uuid = q.command_uuid
	WHERE q.enrollment_id = e.id
		AND q.acked_at IS NULL
		AND c.target_loc_uri <> ?
)`

// recomputeMDMWindowsHasPendingCommandsByEnrollmentIDs refreshes the has_pending_commands flag for the given enrollments by evaluating the
// full EXISTS. Call it inside the transaction that may have cleared an enrollment's last pending command (acknowledgment) so the flag can
// flip back to false. The enqueue paths do NOT use this: inserting a non-poll command can only set the flag true, so they call the cheaper
// markMDMWindowsHasPendingCommandsBy* helpers instead of recomputing the EXISTS.
func (ds *Datastore) recomputeMDMWindowsHasPendingCommandsByEnrollmentIDs(ctx context.Context, tx sqlx.ExtContext, enrollmentIDs []uint) error {
	if len(enrollmentIDs) == 0 {
		return nil
	}
	// Batch to match the bounded queue-insert path; a single IN (...) over a large fan-out could exceed MySQL's placeholder limit.
	return common_mysql.BatchProcessSimple(enrollmentIDs, windowsMDMCommandQueueBatchSize, func(batch []uint) error {
		// The has_pending_commands = 1 guard is defense-in-depth, not the primary idle-path optimization: the management
		// session already skips the refresh entirely (no statement at all) when the flag loaded at session start is 0,
		// so this clause only matters for callers that do not pre-gate, or when the loaded flag was stale. It is safe
		// because the refresh only exists for the 1 -> 0 transition - the enqueue paths own 0 -> 1 by setting the flag
		// directly.
		stmt, args, err := sqlx.In(
			`UPDATE mdm_windows_enrollments e SET e.has_pending_commands = `+windowsMDMHasPendingCommandsExpr+
				` WHERE e.id IN (?) AND e.has_pending_commands = 1`,
			syncml.DMClientPollIntervalLocURI, batch,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build recompute has_pending_commands by enrollment ids")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "recompute has_pending_commands by enrollment ids")
		}
		return nil
	})
}

// MDMWindowsRefreshHasPendingCommands recomputes the denormalized has_pending_commands flag for one enrollment on the
// writer. The management session flow calls it at most once per OMA-DM session: only when the pending-commands fetch for
// the reply comes back empty (the session has drained the queue, so the flag may flip to 0). Mid-session messages skip it
// entirely. The flag provably stays 1 while commands remain queued, having been set by the enqueue paths. Running it
// outside the ack transaction is safe: the EXISTS recompute is authoritative against the writer, so a concurrent enqueue
// between the fetch and this refresh still lands on has_pending_commands = 1.
func (ds *Datastore) MDMWindowsRefreshHasPendingCommands(ctx context.Context, enrollmentID uint) error {
	return ds.recomputeMDMWindowsHasPendingCommandsByEnrollmentIDs(ctx, ds.writer(ctx), []uint{enrollmentID})
}

// markMDMWindowsHasPendingCommandsByEnrollmentIDs sets has_pending_commands = 1 for the given enrollments without recomputing the EXISTS.
// The enqueue paths use it because inserting a non-poll command means a pending command now exists by construction; only the acknowledgment
// path, which can clear the last pending command, needs the full recompute.
func (ds *Datastore) markMDMWindowsHasPendingCommandsByEnrollmentIDs(ctx context.Context, tx sqlx.ExtContext, enrollmentIDs []uint) error {
	if len(enrollmentIDs) == 0 {
		return nil
	}
	// Batch to match the bounded queue-insert path; a single IN (...) over a large fan-out could exceed MySQL's placeholder limit.
	return common_mysql.BatchProcessSimple(enrollmentIDs, windowsMDMCommandQueueBatchSize, func(batch []uint) error {
		stmt, args, err := sqlx.In(`UPDATE mdm_windows_enrollments SET has_pending_commands = 1 WHERE id IN (?)`, batch)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build mark has_pending_commands by enrollment ids")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "mark has_pending_commands by enrollment ids")
		}
		return nil
	})
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
		// setup_experience_status_results.host_uuid is keyed by fleet.HostUUIDForSetupExperience; for Windows that's the
		// host's OsqueryHostID, NOT the Fleet host UUID stored on the MDM enrollment. Resolve via JOIN so we delete by
		// whichever identifier matches (works for both shapes).
		delSetupExpStmt = `DELETE ser FROM setup_experience_status_results ser
			JOIN hosts h ON ser.host_uuid = h.osquery_host_id OR ser.host_uuid = h.uuid
			WHERE h.uuid = ?`
		delUpcomingStmt = `DELETE ua FROM upcoming_activities ua JOIN hosts h ON h.id = ua.host_id WHERE h.uuid = ?`
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
				// Clear setup experience results so they get re-enqueued on the new enrollment.
				if _, err := tx.ExecContext(ctx, delSetupExpStmt, hostUUID.String); err != nil {
					return ctxerr.Wrap(ctx, err, "delete setup_experience_status_results for host")
				}
				// Clear ALL stale upcoming activities (any activity_type) so they don't block new activities on re-enrollment.
				if _, err := tx.ExecContext(ctx, delUpcomingStmt, hostUUID.String); err != nil {
					return ctxerr.Wrap(ctx, err, "delete upcoming_activities for host")
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

// MDMWindowsBulkInsertCommands inserts multiple MDM commands in a single
// multi-row INSERT. Duplicate commands are silently ignored. Complementary to
// MDMWindowsEnqueueCommandAndUpsertHostProfiles
func (ds *Datastore) MDMWindowsBulkInsertCommands(ctx context.Context, cmds []*fleet.MDMWindowsCommand) error {
	if len(cmds) == 0 {
		return nil
	}

	var sb strings.Builder
	var args []any
	for i, cmd := range cmds {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("(?, ?, ?)")
		args = append(args, cmd.CommandUUID, cmd.RawCommand, cmd.TargetLocURI)
	}

	stmt := fmt.Sprintf(`
		INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri)
		VALUES %s
		ON DUPLICATE KEY UPDATE command_uuid = command_uuid`, sb.String())
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk inserting MDMWindowsCommands")
	}
	return nil
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

	// Insert the command once, outside of the batched transactions.
	cmdStmt := `
		INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE command_uuid = command_uuid
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, cmdStmt, cmd.CommandUUID, cmd.RawCommand, cmd.TargetLocURI); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting MDMWindowsCommand")
	}

	return ds.MDMWindowsEnqueueCommandAndUpsertHostProfiles(ctx, hostUUIDs, cmd, payload)
}

// MDMWindowsEnqueueCommandAndUpsertHostProfiles enqueues a command for hosts
// and upserts their profile tracking rows. The command must already exist in
// windows_mdm_commands (inserted via MDMWindowsBulkInsertCommands or
// MDMWindowsInsertCommandAndUpsertHostProfilesForHosts).
func (ds *Datastore) MDMWindowsEnqueueCommandAndUpsertHostProfiles(ctx context.Context, hostUUIDs []string, cmd *fleet.MDMWindowsCommand, payload []*fleet.MDMWindowsBulkUpsertHostProfilePayload) error {
	if len(hostUUIDs) == 0 {
		return nil
	}

	const defaultBatchSize = 1000
	batchSize := defaultBatchSize
	if ds.testUpsertMDMDesiredProfilesBatchSize > 0 {
		batchSize = ds.testUpsertMDMDesiredProfilesBatchSize
	}

	// Build a map from host UUID to its corresponding profile payload for quick lookup.
	payloadByHostUUID := make(map[string]*fleet.MDMWindowsBulkUpsertHostProfilePayload, len(payload))
	for _, p := range payload {
		payloadByHostUUID[p.HostUUID] = p
	}

	// Insert command queue entries and host profile entries in batches.
	// The queue INSERT ... SELECT silently skips hosts without an
	// enrollment; profile rows are upserted for all hosts in the batch.
	var (
		queueHostUUIDs []string
		profileArgs    []any
		profileSB      strings.Builder
		batchCount     int
	)

	executeBatch := func() error {
		return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			// Enqueuing the command and flagging the affected enrollments both need the latest enrollment id per host UUID
			// (SELECT MAX(id) ... GROUP BY host_uuid).
			if len(queueHostUUIDs) > 0 {
				// Hosts whose enrollment was deleted resolve to no id and are silently skipped.
				enrollmentIDs, err := ds.getEnrollmentIDsByHostUUIDDB(ctx, tx, queueHostUUIDs)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "resolving enrollment ids for MDMWindowsCommandQueue insert")
				}
				if len(enrollmentIDs) > 0 {
					// Plain VALUES insert keyed by the resolved enrollment ids; the unique (enrollment_id, command_uuid) key still
					// surfaces a duplicate the same way the INSERT ... SELECT did.
					if err := common_mysql.BatchProcessSimple(enrollmentIDs, windowsMDMCommandQueueBatchSize, func(batch []uint) error {
						valuesPart := strings.TrimSuffix(strings.Repeat("(?, ?),", len(batch)), ",")
						args := make([]any, 0, len(batch)*2)
						for _, eid := range batch {
							args = append(args, eid, cmd.CommandUUID)
						}
						if _, err := tx.ExecContext(ctx,
							`INSERT INTO windows_mdm_command_queue (enrollment_id, command_uuid) VALUES `+valuesPart, args...); err != nil {
							if IsDuplicate(err) {
								return ctxerr.Wrap(ctx, alreadyExists("MDMWindowsCommandQueue", cmd.CommandUUID))
							}
							return ctxerr.Wrap(ctx, err, "batch inserting MDMWindowsCommandQueue")
						}
						return nil
					}); err != nil {
						return err
					}

					// A pending command now exists, so set the flag directly in this same transaction. Flag by the enrollment ids we already
					// resolved.
					if cmd.TargetLocURI != syncml.DMClientPollIntervalLocURI {
						if err := ds.markMDMWindowsHasPendingCommandsByEnrollmentIDs(ctx, tx, enrollmentIDs); err != nil {
							return err
						}
					}
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

// MDMWindowsInsertCommandForHostUUIDs is the fire-and-forget enqueue used by the profile-manager cron to deliver supplemental
// <Delete> commands (e.g. LocURIs removed from an edited profile) to a bounded batch of hosts. The command stands alone and is
// not tracked per host.
func (ds *Datastore) MDMWindowsInsertCommandForHostUUIDs(ctx context.Context, hostUUIDs []string, cmd *fleet.MDMWindowsCommand) error {
	if len(hostUUIDs) == 0 {
		return nil
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return ds.mdmWindowsInsertCommandForHostUUIDsDB(ctx, tx, hostUUIDs, cmd)
	})
}

// MDMWindowsInsertCommandsForHost atomically inserts a batch of Windows MDM commands targeting a single host
// (identified by host UUID or MDM device ID). All commands are inserted in one transaction: either every row
// is committed or none. Used by the ESP finalize path so the dropped-response retry safety net can't end up
// partially written on a transient DB error -- a partial write followed by a fresh-UUID retry would leave
// orphan rows in the queue.
//
// Returns notFound("MDMWindowsEnrolledDevice") if the identifier resolves to zero enrollments. Without this
// guard, mdmWindowsInsertCommandForEnrollmentIDsDB would still INSERT each row into windows_mdm_commands and
// return success while leaving the rows targeted at no host -- the ESP finalize would silently drop the
// retry safety net.
func (ds *Datastore) MDMWindowsInsertCommandsForHost(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
	if len(cmds) == 0 {
		return nil
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		enrollmentIDs, err := ds.getEnrollmentIDsByHostUUIDOrDeviceIDDB(ctx, tx, []string{hostUUIDOrDeviceID})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "fetching enrollment IDs for command queue")
		}
		if len(enrollmentIDs) == 0 {
			return ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice").WithName(hostUUIDOrDeviceID))
		}
		for _, cmd := range cmds {
			if err := ds.mdmWindowsInsertCommandForEnrollmentIDsDB(ctx, tx, enrollmentIDs, cmd); err != nil {
				return err
			}
		}
		return nil
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
	if err := common_mysql.BatchProcessSimple(enrollmentIDs, windowsMDMCommandQueueBatchSize, func(batch []uint) error {
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
	}); err != nil {
		return err
	}

	if cmd.TargetLocURI == syncml.DMClientPollIntervalLocURI {
		return nil
	}
	return ds.markMDMWindowsHasPendingCommandsByEnrollmentIDs(ctx, tx, enrollmentIDs)
}

// getEnrollmentIDsByHostUUIDDB fetches enrollment IDs for a list of host UUIDs
// using an indexed batch query. Returns the most recent enrollment per host.
func (ds *Datastore) getEnrollmentIDsByHostUUIDDB(ctx context.Context, tx sqlx.ExtContext, hostUUIDs []string) ([]uint, error) {
	var allIDs []uint
	err := common_mysql.BatchProcessSimple(hostUUIDs, windowsMDMCommandQueueBatchSize, func(batch []string) error {
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
	// Fast path: probe the queue. An MDM management session runs this query on every check-in, and the overwhelming majority of
	// devices have nothing queued, so short-circuit before paying for the fetch JOIN. SELECT EXISTS always returns a row, so the idle
	// path does not go through a sql.ErrNoRows branch. This is an index probe on (enrollment_id, acked_at).
	const probe = `SELECT EXISTS(
		SELECT 1 FROM windows_mdm_command_queue wmcq
		WHERE wmcq.enrollment_id = ? AND wmcq.acked_at IS NULL
	)`
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
	wmcq.acked_at IS NULL
ORDER BY
	wmc.created_at ASC
`

	var commands []*fleet.MDMWindowsCommand
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &commands, query, enrollmentID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get pending Windows MDM commands by enrollment id")
	}

	return commands, nil
}

// MDMWindowsGetESPReleaseAckStatus summarizes the delivery state of ESP release commands targeting the given LocURI for
// the enrollment. Attempts are matched on BOTH the target LocURI and the command_uuid prefix Fleet stamps on its own
// release attempts, so an admin-enqueued raw command that happens to target the same LocURI can neither trigger the
// resend phase nor complete the ESP.
func (ds *Datastore) MDMWindowsGetESPReleaseAckStatus(ctx context.Context, enrollmentID uint, targetLocURI, cmdUUIDPrefix string) (*fleet.MDMWindowsESPReleaseAckStatus, error) {
	const query = `
SELECT
	COUNT(*) > 0 AS attempted,
	COALESCE(MAX(r.status_code = '200'), FALSE) AS acked200,
	COALESCE(MAX(wmcq.acked_at IS NULL), FALSE) AS has_unacked,
	COALESCE(SUBSTRING_INDEX(GROUP_CONCAT(r.status_code ORDER BY wmcq.acked_at DESC), ',', 1), '') AS latest_status
FROM
	windows_mdm_command_queue wmcq
INNER JOIN
	windows_mdm_commands wmc ON wmc.command_uuid = wmcq.command_uuid
LEFT JOIN
	windows_mdm_command_results r ON r.enrollment_id = wmcq.enrollment_id AND r.command_uuid = wmcq.command_uuid
WHERE
	wmcq.enrollment_id = ? AND wmc.target_loc_uri = ? AND wmc.command_uuid LIKE CONCAT(?, '%')
`
	var row struct {
		Attempted    bool   `db:"attempted"`
		Acked200     bool   `db:"acked200"`
		HasUnacked   bool   `db:"has_unacked"`
		LatestStatus string `db:"latest_status"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, query, enrollmentID, targetLocURI, cmdUUIDPrefix); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get Windows ESP release ack status")
	}
	return &fleet.MDMWindowsESPReleaseAckStatus{
		Attempted:    row.Attempted,
		Acked200:     row.Acked200,
		HasUnacked:   row.HasUnacked,
		LatestStatus: row.LatestStatus,
	}, nil
}

// compressWindowsMDMResponse gzip-compresses a full SyncML response envelope
func compressWindowsMDMResponse(raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(raw); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decompressWindowsMDMResponse reverses compressWindowsMDMResponse.
func decompressWindowsMDMResponse(stored []byte) ([]byte, error) {
	if len(stored) == 0 {
		return stored, nil
	}
	gr, err := gzip.NewReader(bytes.NewReader(stored))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	return io.ReadAll(gr)
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

		// store the full response, gzip-compressed to shrink the row and reduce redo-log/commit-quorum pressure on this hot path
		compressedResp, err := compressWindowsMDMResponse(enrichedSyncML.Raw)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "compressing full response")
		}
		const saveFullRespStmt = `INSERT INTO windows_mdm_responses (enrollment_id, raw_response_gz) VALUES (?, ?)`
		sqlResult, err := tx.ExecContext(ctx, saveFullRespStmt, enrolledDevice.ID, compressedResp)
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
			// Suppress the warning when the only CmdRefs in the message are Fleet-internal commands that are
			// intentionally absent from windows_mdm_commands (e.g. the DevDetail/SMBIOSSerialNumber Get used for
			// unlinked-enrollment linkage). Without this filter, an unlinked Windows enrollment with no other pending
			// commands warns on every session until linkage completes.
			externalCmdRefs := make([]string, 0, len(enrichedSyncML.CmdRefUUIDs))
			for _, u := range enrichedSyncML.CmdRefUUIDs {
				if !fleet.IsFleetInternalCmdID(u) {
					externalCmdRefs = append(externalCmdRefs, u)
				}
			}
			if len(commandIDsBeingResent) == 0 && len(externalCmdRefs) > 0 {
				// Only log if not resending commands as we then can expect no matching commands
				ds.logger.WarnContext(ctx, "unmatched Windows MDM commands", "uuids", strings.Join(externalCmdRefs, ","), "mdm_device_id",
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

			// if the command is a Wipe, keep track of it so we can update host_mdm_actions accordingly.
			if fleet.LocURITargetsReservedNode(cmd.TargetLocURI, syncml.FleetRemoteWipeTargetLocURI) {
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

			if wipeCmdStatus != "" && rowsAffected > 0 {
				if wipeSucceeded {
					result = &fleet.MDMWindowsSaveResponseResult{
						WipeSucceeded: &fleet.MDMWindowsWipeResult{
							HostUUID: enrolledDevice.HostUUID,
						},
					}
				} else {
					result = &fleet.MDMWindowsSaveResponseResult{
						WipeFailed: &fleet.MDMWindowsWipeResult{
							HostUUID: enrolledDevice.HostUUID,
						},
					}
				}
			}
		}

		// Soft-dequeue the commands we just recorded results for: stamp acked_at on exactly those queue rows, in the
		// same transaction as the results insert so "has a result row" and "acked_at set" can never disagree. This is
		// the ONLY path that inserts windows_mdm_command_results; any new results-insert path must stamp acked_at too,
		// or the rows will look pending forever under the acked_at IS NULL predicates. The periodic GC range-deletes
		// stamped rows after an hour. COALESCE preserves the first ack time on duplicate acks so the GC age floor is
		// measured from the original acknowledgment.
		//
		// The has_pending_commands recompute deliberately does NOT happen here: it runs at most once per OMA-DM session (in
		// getManagementResponse, when the reply-building pending-commands fetch finds no non-poll commands remaining and the flag was
		// loaded as 1 at session start), not once per message.
		markStmt, markArgs, err := sqlx.In(
			`UPDATE windows_mdm_command_queue SET acked_at = COALESCE(acked_at, NOW(6)) WHERE enrollment_id = ? AND command_uuid IN (?)`,
			enrolledDevice.ID, matchingCmdUUIDs,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build acked_at stamp for queue rows")
		}
		if _, err := tx.ExecContext(ctx, markStmt, markArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "stamp acked_at on queue rows")
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// renewalIDManagedCertProfileUUIDsDB returns, among the given profile UUIDs, those that have a managed-certificate row
// for the host whose CA type carries a renewal-ID marker (custom SCEP proxy, NDES, or Smallstep).
func renewalIDManagedCertProfileUUIDsDB(ctx context.Context, tx sqlx.ExtContext, hostUUID string, profileUUIDs []string) (map[string]struct{}, error) {
	if len(profileUUIDs) == 0 {
		return nil, nil
	}
	caTypes := fleet.ListCATypesWithRenewalIDSupport()
	caTypeStrs := make([]string, 0, len(caTypes))
	for _, t := range caTypes {
		caTypeStrs = append(caTypeStrs, string(t))
	}
	stmt, args, err := sqlx.In(`
		SELECT profile_uuid
		FROM host_mdm_managed_certificates
		WHERE host_uuid = ? AND profile_uuid IN (?) AND type IN (?)`,
		hostUUID, profileUUIDs, caTypeStrs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building managed cert profile query")
	}
	var uuids []string
	if err := sqlx.SelectContext(ctx, tx, &uuids, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting renewal-ID managed cert profile uuids")
	}
	result := make(map[string]struct{}, len(uuids))
	for _, u := range uuids {
		result[u] = struct{}{}
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

	// Proxied SCEP profiles must not report "verified" off the device's SyncML ACK alone: a 2xx ACK only means the
	// SCEP <Exec> was accepted by the CSP, not that a certificate was issued (the exchange runs asynchronously after).
	// Downgrade those installs to "verifying" and let UpdateHostCertificates flip them to "verified" once the matching
	// certificate is observed on the host. Detect them by an existing renewal-ID-backed managed-certificate row (custom
	// SCEP proxy, NDES, or Smallstep).
	var verifiedInstallProfileUUIDs []string
	for _, hp := range matchingHostProfiles {
		payload := uuidsToPayloads[hp.CommandUUID]
		if payload == nil {
			continue
		}
		if hp.OperationType == fleet.MDMOperationTypeInstall && payload.Status != nil && *payload.Status == fleet.MDMDeliveryVerified {
			verifiedInstallProfileUUIDs = append(verifiedInstallProfileUUIDs, hp.ProfileUUID)
		}
	}
	scepProxyProfileUUIDs, err := renewalIDManagedCertProfileUUIDsDB(ctx, tx, hostUUID, verifiedInstallProfileUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking for proxied SCEP managed certificate profiles")
	}

	// Partition matching entries into upsert and delete buckets.
	var sb strings.Builder
	args = args[:0]
	var deleteCommandUUIDs []string
	for _, hp := range matchingHostProfiles {
		payload := uuidsToPayloads[hp.CommandUUID]
		if payload == nil {
			continue
		}
		if hp.OperationType == fleet.MDMOperationTypeInstall && payload.Status != nil && *payload.Status == fleet.MDMDeliveryVerified {
			if _, ok := scepProxyProfileUUIDs[hp.ProfileUUID]; ok {
				verifying := fleet.MDMDeliveryVerifying
				payload.Status = &verifying
			}
		}
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

func (ds *Datastore) SetMDMWindowsHostProfileFailed(ctx context.Context, hostUUID string, profileUUID string, detail string) error {
	// Only touch an existing install row (a removed profile is not resurrected). Never overwrite a row that already
	// reached "verified" (the certificate was observed, so a late/stale upstream error must not regress it).
	const stmt = `
		UPDATE host_mdm_windows_profiles
		SET status = ?, detail = ?
		WHERE host_uuid = ? AND profile_uuid = ? AND operation_type = ?
			AND (status IS NULL OR status <> ?)`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt,
		fleet.MDMDeliveryFailed, detail, hostUUID, profileUUID, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "set windows host profile failed")
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
    COALESCE(wmr.raw_response_gz, '') AS result,
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

	// raw_response_gz is stored gzip-compressed; restore the original envelope for the API response.
	for _, r := range results {
		decompressed, err := decompressWindowsMDMResponse(r.Result)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "decompressing windows mdm response")
		}
		r.Result = decompressed
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
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Retain the profile's content so the profile-manager cron can build <Delete> commands after the definition is gone.
		// This must run before the definition row is deleted. deleteMDMWindowsConfigProfile returns notFound if it does not exist.
		if err := ds.retainWindowsProfilePriorContentDB(ctx, tx, []string{profileUUID}); err != nil {
			return err
		}
		if err := deleteMDMWindowsConfigProfile(ctx, tx, profileUUID); err != nil {
			return err
		}
		return ds.cancelWindowsHostInstallsForDeletedMDMProfiles(ctx, tx, []string{profileUUID})
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

// cancelWindowsHostInstallsForDeletedMDMProfiles does the cheap, request-path-safe host_mdm_windows_profiles cleanup when config
// profiles are deleted. In a single statement it removes the rows the cron does not need to act on: already-resolved removes
// (operation_type=remove, status verified/failed/verifying) and never-sent installs (operation_type=install, status IS NULL).
// Everything else (sent installs, pending removes) is left for the profile-manager cron, which classifies the surviving rows as
// removes (current-not-desired) and generates the <Delete> commands asynchronously in its bounded batches.
func (ds *Datastore) cancelWindowsHostInstallsForDeletedMDMProfiles(
	ctx context.Context, tx sqlx.ExtContext, profileUUIDs []string,
) error {
	if len(profileUUIDs) == 0 {
		return nil
	}

	// Delete the host-profile rows the cron does not need to act on, in one pass:
	//   - in-flight removes (status = verifying): the <Delete> is already on the wire, so it is safe to drop. Windows removals are
	//     best-effort and their results are not persisted (BulkUpsertMDMWindowsHostProfiles deletes a remove row once its <Delete>
	//     resolves), so a remove row only ever persists while verifying.
	//   - never-sent installs (status IS NULL): nothing was delivered, so no <Delete> is needed.
	// Sent installs and pending removes are left untouched for the reconciler.
	delStmt, delArgs, err := sqlx.In(`
	DELETE FROM host_mdm_windows_profiles
	WHERE profile_uuid IN (?)
		AND ((operation_type = ? AND status = ?) OR (operation_type = ? AND status IS NULL))`,
		profileUUIDs, fleet.MDMOperationTypeRemove, fleet.MDMDeliveryVerifying, fleet.MDMOperationTypeInstall)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building IN for deleted-profile host-row cleanup")
	}
	if _, err := tx.ExecContext(ctx, delStmt, delArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning up host rows for deleted profiles")
	}

	return nil
}

// retainWindowsProfilePriorContentDB retains the CURRENT live content of the given Windows config profiles, keyed by
// (profile_uuid, checksum), so the profile-manager cron can build <Delete> commands from the exact version a host has after the
// live definition is gone (delete) or overwritten (edit).
func (ds *Datastore) retainWindowsProfilePriorContentDB(ctx context.Context, tx sqlx.ExtContext, profileUUIDs []string) error {
	if len(profileUUIDs) == 0 {
		return nil
	}
	stmt, args, err := sqlx.In(`
		INSERT IGNORE INTO mdm_windows_configuration_profiles_prior_content (profile_uuid, checksum, syncml)
		SELECT profile_uuid, checksum, syncml
		FROM mdm_windows_configuration_profiles
		WHERE profile_uuid IN (?)`, profileUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building IN for prior-content retention")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "retaining windows config profile prior content")
	}
	return nil
}

// GetWindowsMDMProfilePriorContents returns the retained syncml for the given (profile_uuid, checksum) version keys.
// Reader-backed; callers that cannot tolerate a replica-lag miss (the reconcile pass consumes each modify-install once) wrap the
// context with ctxdb.RequirePrimary.
func (ds *Datastore) GetWindowsMDMProfilePriorContents(ctx context.Context, keys []fleet.MDMWindowsProfileVersionKey) ([]fleet.MDMWindowsProfilePriorContent, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	// Composite (profile_uuid, checksum) IN ((?,?),...). The placeholder structure is constant, so this is a parameterized query.
	placeholders := strings.TrimSuffix(strings.Repeat("(?,?),", len(keys)), ",")
	stmt := `SELECT profile_uuid, checksum, syncml
		FROM mdm_windows_configuration_profiles_prior_content
		WHERE (profile_uuid, checksum) IN (` + placeholders + `)`
	args := make([]any, 0, len(keys)*2)
	for _, k := range keys {
		args = append(args, k.ProfileUUID, k.Checksum)
	}

	var rows []fleet.MDMWindowsProfilePriorContent
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting windows profile prior content")
	}
	return rows, nil
}

func (ds *Datastore) DeleteMDMWindowsConfigProfileByTeamAndName(ctx context.Context, teamID *uint, profileName string) error {
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	// Read the profile UUID before the transaction to keep it short.
	var profile struct {
		ProfileUUID string `db:"profile_uuid"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &profile,
		`SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE team_id=? AND name=?`,
		globalOrTeamID, profileName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil // nothing to delete
		}
		return ctxerr.Wrap(ctx, err, "reading profile before deletion")
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Retain the profile's content for the cron to build <Delete> commands from, before the definition row is deleted (#46993).
		if err := ds.retainWindowsProfilePriorContentDB(ctx, tx, []string{profile.ProfileUUID}); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM mdm_windows_configuration_profiles WHERE profile_uuid=?`, profile.ProfileUUID); err != nil {
			return ctxerr.Wrap(ctx, err)
		}
		return ds.cancelWindowsHostInstallsForDeletedMDMProfiles(ctx, tx, []string{profile.ProfileUUID})
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

// GetMDMWindowsProfilesContents returns the SyncML (and checksum) for the given profile UUIDs. Live profiles are looked up first; for
// any UUID not found live it falls back to the most recently retained version in mdm_windows_configuration_profiles_prior_content.
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

	results := make(map[string]fleet.MDMWindowsProfileContents, len(uuids))
	for _, p := range profs {
		results[p.ProfileUUID] = fleet.MDMWindowsProfileContents{
			SyncML:   p.SyncML,
			Checksum: p.Checksum,
		}
	}

	// Fall back to retained prior content for any UUID not found live (deleted profiles still being drained by the cron).
	var missing []string
	for _, u := range uuids {
		if _, ok := results[u]; !ok {
			missing = append(missing, u)
		}
	}
	if len(missing) > 0 {
		pcStmt, pcArgs, pcErr := sqlx.In(`
			SELECT profile_uuid, syncml, checksum
			FROM mdm_windows_configuration_profiles_prior_content
			WHERE profile_uuid IN (?)
			ORDER BY created_at ASC, checksum ASC`, missing)
		if pcErr != nil {
			return nil, ctxerr.Wrap(ctx, pcErr, "building in statement for prior-content contents")
		}
		var prior []struct {
			ProfileUUID string `db:"profile_uuid"`
			SyncML      []byte `db:"syncml"`
			Checksum    []byte `db:"checksum"`
		}
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &prior, pcStmt, pcArgs...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "running prior-content contents query")
		}
		for _, p := range prior {
			results[p.ProfileUUID] = fleet.MDMWindowsProfileContents{
				SyncML:   p.SyncML,
				Checksum: p.Checksum,
			}
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

		// An OS-update profile is tracked as the team's OS-update profile within
		// this transaction so it rolls back together on failure.
		if fleet.ProfileTargetsReservedLocURI(cp.SyncML, syncml.FleetOSUpdateTargetLocURI) {
			if err := trackWindowsUpdateConfigProfileDB(ctx, tx, teamID, profileUUID); err != nil {
				return err
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

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// For an existing profile whose content is changing, retain the outgoing version before the upsert overwrites it, so the
		// profile-manager cron can build <Delete> commands for LocURIs the new content drops (the same guarantee as
		// batchSetMDMWindowsProfilesDB's edit path). Reserved profiles updated through this method (e.g. Windows OS updates) are
		// delivered and reconciled like any other profile, so they need the same retention.
		var existing struct {
			ProfileUUID string `db:"profile_uuid"`
			SyncML      []byte `db:"syncml"`
		}
		err := sqlx.GetContext(ctx, tx, &existing,
			`SELECT profile_uuid, syncml FROM mdm_windows_configuration_profiles WHERE team_id = ? AND name = ?`, teamID, cp.Name)
		switch {
		case err == nil:
			if !bytes.Equal(existing.SyncML, cp.SyncML) {
				if err := ds.retainWindowsProfilePriorContentDB(ctx, tx, []string{existing.ProfileUUID}); err != nil {
					return ctxerr.Wrap(ctx, err, "retaining prior content for updated profile")
				}
			}
		case errors.Is(err, sql.ErrNoRows):
			// new profile, nothing to retain
		default:
			return ctxerr.Wrap(ctx, err, "loading existing windows profile before upsert")
		}

		res, err := tx.ExecContext(ctx, stmt, profileUUID, teamID, cp.Name, cp.SyncML, cp.Name, teamID, cp.Name, teamID, cp.Name, teamID)
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
	})
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
		stmt                string
		args                []any
		result              sql.Result
		deletedProfileUUIDs []string
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

	// Step 2: Retain the content of profiles being deleted so the profile-manager cron can build <Delete> commands after the
	// definition rows are gone. Must run before Step 3 deletes the definitions.
	if len(deletedProfileUUIDs) > 0 {
		if err := ds.retainWindowsProfilePriorContentDB(ctx, tx, deletedProfileUUIDs); err != nil {
			return false, ctxerr.Wrap(ctx, err, "retain deleted profiles for async removal")
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

	// Step 4: Clean up host-profile rows for deleted profiles (terminal removes + never-sent installs). The actual <Delete>
	// commands for already-delivered profiles are issued asynchronously by the profile-manager cron from the retained content.
	if len(deletedProfileUUIDs) > 0 {
		if err := ds.cancelWindowsHostInstallsForDeletedMDMProfiles(ctx, tx, deletedProfileUUIDs); err != nil {
			return false, ctxerr.Wrap(ctx, err, "cancel installs of deleted profiles")
		}
	}

	// For profiles being updated (same name, different content), retain the OUTGOING version's content before the upsert below
	// overwrites it. A re-install only Replaces/Adds the new content; it never reverts a LocURI the edit dropped, so the device must
	// receive a <Delete> for each removed LocURI. The profile-manager cron, when it re-installs the modified profile, diffs this
	// retained prior version against the new content and enqueues the <Delete> commands asynchronously in its bounded batches, with
	// per-host LocURI protection. This reuses the same retention table and async path as profile deletion.
	//
	// This is an edge case (most edits change values, not remove LocURIs), and the reverts are best-effort, not surfaced in the UI/API.
	var editedProfileUUIDs []string
	for _, existing := range existingProfiles {
		if incoming := incomingProfs[existing.Name]; incoming != nil && !bytes.Equal(existing.SyncML, incoming.SyncML) {
			editedProfileUUIDs = append(editedProfileUUIDs, existing.ProfileUUID)
		}
	}
	if err := ds.retainWindowsProfilePriorContentDB(ctx, tx, editedProfileUUIDs); err != nil {
		return false, ctxerr.Wrap(ctx, err, "retaining prior content for edited profiles")
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

func (ds *Datastore) CleanupWindowsMDMCommandQueue(ctx context.Context) error {
	const batchSize = 1000
	// Acknowledged rows carry acked_at (stamped in the ack transaction), so GC is a single-table index range delete on
	// (enrollment_id, acked_at)'s acked_at part. The 1-hour age floor preserves the resend/debugging window the join-based predicate
	// had via r.created_at. ORDER BY makes the LIMIT deterministic (oldest acknowledged rows first) and keeps the optimizer on the
	// acked_at index for a bounded range delete.
	const stmt = `
DELETE FROM windows_mdm_command_queue
WHERE acked_at IS NOT NULL AND acked_at < NOW() - INTERVAL 1 HOUR
ORDER BY acked_at
LIMIT ?`
	const maxBatches = 500 // cap total work per cron tick (500k rows)
	var totalDeleted int64
	exhausted := true
	for range maxBatches {
		res, err := ds.writer(ctx).ExecContext(ctx, stmt, batchSize)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "cleanup windows mdm command queue")
		}
		n, _ := res.RowsAffected()
		totalDeleted += n
		if n < int64(batchSize) {
			exhausted = false
			break
		}
	}
	if exhausted {
		ds.logger.WarnContext(ctx, "cleanup windows mdm command queue did not finish, remaining rows will be cleaned on next run",
			"deleted", totalDeleted, "max_batches", maxBatches)
	}
	return nil
}

// CleanupWindowsMDMProfilePriorContent garbage-collects retained prior profile content once no host_mdm_windows_profiles row
// still has that version installed. Reference-counted on (profile_uuid, checksum) rather than age-based, so a version's content
// survives exactly as long as some host still has it installed and could still need its <Delete> (e.g. a host that was offline
// when the profile was edited or deleted). Once every host has moved past that version (re-installed to a newer one, drained its
// removal, or unenrolled), the row is dropped.
func (ds *Datastore) CleanupWindowsMDMProfilePriorContent(ctx context.Context) error {
	const stmt = `
DELETE pc FROM mdm_windows_configuration_profiles_prior_content pc
WHERE NOT EXISTS (
	SELECT 1 FROM host_mdm_windows_profiles hmwp
	WHERE hmwp.profile_uuid = pc.profile_uuid AND hmwp.checksum = pc.checksum
)`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt); err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup windows mdm profile prior content")
	}
	return nil
}

func (ds *Datastore) HasWindowsUpdateConfigProfileConfigured(ctx context.Context, teamID uint) (bool, error) {
	const stmt = `
	SELECT COUNT(*) > 0 FROM mdm_configuration_profile_update_settings mcpus
    	INNER JOIN mdm_windows_configuration_profiles mwcp ON mwcp.profile_uuid = mcpus.windows_profile_uuid
	WHERE mwcp.team_id = ?`

	var configured bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &configured, stmt, teamID); err != nil {
		return false, ctxerr.Wrap(ctx, err, "check if windows update config profile is configured")
	}

	return configured, nil
}

// trackWindowsUpdateConfigProfileDB records profileUUID as the team's OS-update
// profile within the caller's transaction
func trackWindowsUpdateConfigProfileDB(ctx context.Context, tx sqlx.ExtContext, teamID uint, profileUUID string) error {
	const insertStmt = `INSERT INTO mdm_configuration_profile_update_settings (windows_profile_uuid) VALUES (?)
		ON DUPLICATE KEY UPDATE windows_profile_uuid = windows_profile_uuid`
	if _, err := tx.ExecContext(ctx, insertStmt, profileUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting software update profile")
	}
	return nil
}
