package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// deviceNameEligibleHostsJoins and deviceNameEligibleHostsWhere express the
// shared eligibility predicate for host-name enforcement: the host must be on an
// Apple platform, enrolled in Fleet's own MDM (the nano_enrollments join is the
// Fleet-server signal), and not a personal BYOD enrollment. Personal enrollments
// are skipped because Apple rejects the DeviceName setting on them, mirroring the
// skip in GetHostsForRecoveryLockAction.
const deviceNameEligibleHostsJoins = `
	FROM hosts h
	JOIN nano_enrollments ne ON ne.device_id = h.uuid
	JOIN host_mdm hm ON hm.host_id = h.id`

const deviceNameEligibleHostsWhere = `
	h.platform IN ('darwin', 'ios', 'ipados')
	AND ne.enabled = 1
	AND ne.type IN ('Device', 'User Enrollment (Device)')
	AND hm.enrolled = 1
	AND hm.is_personal_enrollment = 0`

func (ds *Datastore) BulkUpsertHostDeviceNameEnforcement(ctx context.Context, teamID uint) error {
	stmt := `
		INSERT INTO host_mdm_apple_device_names (host_uuid, status)
		SELECT h.uuid, NULL` + deviceNameEligibleHostsJoins + `
		WHERE ` + deviceNameEligibleHostsWhere + `
			AND h.team_id = ?
		ON DUPLICATE KEY UPDATE status = NULL`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, teamID); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk upsert host device name enforcement")
	}
	return nil
}

func (ds *Datastore) DeleteHostDeviceNameEnforcementForTeam(ctx context.Context, teamID uint) error {
	const stmt = `
		DELETE hmadn
		FROM host_mdm_apple_device_names hmadn
		JOIN hosts h ON h.uuid = hmadn.host_uuid
		WHERE h.team_id = ?`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, teamID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host device name enforcement for team")
	}
	return nil
}

func (ds *Datastore) ListHostsPendingDeviceNameCommand(ctx context.Context, limit int) ([]fleet.HostDeviceNamePending, error) {
	const stmt = `
		SELECT
			h.id AS host_id,
			h.uuid AS host_uuid,
			h.hardware_serial,
			h.platform,
			h.computer_name,
			h.team_id
		FROM host_mdm_apple_device_names hmadn
		JOIN hosts h ON h.uuid = hmadn.host_uuid
		WHERE hmadn.status IS NULL
		LIMIT ?`

	var pending []fleet.HostDeviceNamePending
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &pending, stmt, limit); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list hosts pending device name command")
	}
	return pending, nil
}

func (ds *Datastore) SetHostDeviceNameStatus(ctx context.Context, hostUUID string, status fleet.MDMDeliveryStatus, commandUUID *string, expectedName, detail string) error {
	// commandUUID is bound directly: a non-nil value records the enqueued
	// command; nil clears it so a result from a previously sent command can't
	// match this row and overwrite the outcome recorded here.
	const stmt = `
		UPDATE host_mdm_apple_device_names
		SET status = ?, command_uuid = ?, expected_device_name = ?, detail = ?
		WHERE host_uuid = ?`

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, status, commandUUID, expectedName, detail, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set host device name status")
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		// The row went away between the cron listing it and this write (e.g. the
		// template was cleared); nothing to record. Any command already sent will
		// simply not match a row when its result arrives and be dropped.
		ds.logger.DebugContext(ctx, "host device name status set but no enforcement row updated", "host_uuid", hostUUID)
	}
	return nil
}

func (ds *Datastore) UpdateHostDeviceNameStatusFromCommand(ctx context.Context, commandUUID string, acknowledged bool, detail string) error {
	// The command result is one of exactly two outcomes: acknowledged (the
	// device applied the rename → verifying) or an error (→ failed).
	status := fleet.MDMDeliveryFailed
	if acknowledged {
		status = fleet.MDMDeliveryVerifying
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// The UPDATE is authoritative. A 0-row result means no current row holds
		// this command UUID: it was superseded by a newer command for the same
		// host (the row keeps only the latest) or the row was deleted. Either way
		// the result is stale and callers must treat this not-found as ignorable.
		const updateStmt = `
			UPDATE host_mdm_apple_device_names
			SET status = ?, detail = ?
			WHERE command_uuid = ?`
		res, err := tx.ExecContext(ctx, updateStmt, status, detail, commandUUID)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "update host device name status from command %s", commandUUID)
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return ctxerr.Wrap(ctx, notFound("HostDeviceNameEnforcement").WithName(commandUUID))
		}

		if !acknowledged {
			// Only an acknowledgment renames the host; error results just record
			// the failure on the row.
			return nil
		}

		// Acknowledged: rename the host in Fleet in this same transaction so the
		// row transition and the Fleet-side rename are atomic. Join the row to
		// its host to read the expected name and the fields needed to derive the
		// display name. The row is locked by the UPDATE above, so this can't miss.
		var host struct {
			ID                 uint    `db:"id"`
			HardwareModel      string  `db:"hardware_model"`
			HardwareSerial     string  `db:"hardware_serial"`
			ExpectedDeviceName *string `db:"expected_device_name"`
		}
		const selectStmt = `
			SELECT h.id, h.hardware_model, h.hardware_serial, n.expected_device_name
			FROM host_mdm_apple_device_names n
			JOIN hosts h ON h.uuid = n.host_uuid
			WHERE n.command_uuid = ?`
		if err := sqlx.GetContext(ctx, tx, &host, selectStmt, commandUUID); err != nil {
			return ctxerr.Wrapf(ctx, err, "get host to rename for command %s", commandUUID)
		}
		// A row only carries a command_uuid when SetHostDeviceNameStatus set it
		// together with the resolved expected_device_name, so a row matched here by
		// command_uuid always has a name.
		name := *host.ExpectedDeviceName

		if _, err := tx.ExecContext(ctx,
			`UPDATE hosts SET computer_name = ?, hostname = ? WHERE id = ?`, name, name, host.ID); err != nil {
			return ctxerr.Wrap(ctx, err, "rename host from device name")
		}
		displayName := fleet.HostDisplayName(name, name, host.HardwareModel, host.HardwareSerial)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO host_display_names (host_id, display_name) VALUES (?, ?)
			ON DUPLICATE KEY UPDATE display_name = VALUES(display_name)`, host.ID, displayName); err != nil {
			return ctxerr.Wrap(ctx, err, "update host display name from device name")
		}
		return nil
	})
}

// deviceNameVerifyGracePeriod is how long after an acknowledgment (the row
// entered verifying) a mismatching reported name is ignored rather than
// recorded as drift. A report the agent collected before the device applied the
// rename can arrive after the acknowledgment carrying the old name; the only
// staleness is the agent's collect-to-submit latency (seconds), so a small
// fixed window comfortably covers it. The comparison is entirely against the DB
// clock (updated_at vs NOW()), so there's no cross-machine skew to pad for.
const deviceNameVerifyGracePeriod = 60 * time.Second

func (ds *Datastore) UpdateHostDeviceNameStatusFromReport(ctx context.Context, hostUUID, reportedName string) error {
	// Only rows already awaiting or past verification are reconciled against the
	// device-reported name: a match confirms the rename (verified), a mismatch
	// records drift (failed). Rows in any other state and hosts with no row are
	// left untouched.
	//
	// A mismatch on a row acknowledged within the last deviceNameVerifyGracePeriod
	// is left untouched (false drift; failed rows only recover via an explicit
	// resend). Rows already verified reached that state through a fresh
	// post-rename report, so a mismatch there is genuine drift regardless of age.
	// When the CASEs resolve to the current values, MySQL skips the row write,
	// preserving updated_at (the grace anchor).
	const stmt = `
		UPDATE host_mdm_apple_device_names
		SET
			status = CASE
				WHEN expected_device_name = ? THEN ?
				WHEN status = ? AND updated_at > DATE_SUB(NOW(6), INTERVAL ? SECOND) THEN status
				ELSE ? END,
			detail = CASE
				WHEN expected_device_name = ? THEN ''
				WHEN status = ? AND updated_at > DATE_SUB(NOW(6), INTERVAL ? SECOND) THEN detail
				ELSE ? END
		WHERE host_uuid = ?
			AND status IN (?, ?)`

	const driftDetail = "Host was renamed on the device and no longer matches the fleet's naming template."
	graceSeconds := int(deviceNameVerifyGracePeriod.Seconds())
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt,
		reportedName, fleet.MDMDeliveryVerified, fleet.MDMDeliveryVerifying, graceSeconds, fleet.MDMDeliveryFailed,
		reportedName, fleet.MDMDeliveryVerifying, graceSeconds, driftDetail,
		hostUUID,
		fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerified,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "update host device name status from report")
	}
	return nil
}

func (ds *Datastore) GetHostDeviceNameEnforcement(ctx context.Context, hostUUID string) (*fleet.HostDeviceNameEnforcement, error) {
	const stmt = `
		SELECT host_uuid, status, command_uuid, expected_device_name, COALESCE(detail, '') AS detail, created_at, updated_at
		FROM host_mdm_apple_device_names
		WHERE host_uuid = ?`

	var enforcement fleet.HostDeviceNameEnforcement
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &enforcement, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostDeviceNameEnforcement").WithName(hostUUID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host device name enforcement")
	}
	return &enforcement, nil
}

func (ds *Datastore) ResendHostDeviceName(ctx context.Context, hostUUID string) error {
	// Reset the status to NULL to trigger resending on the next cron run, same as
	// ResendHostMDMProfile. command_uuid is cleared too so a late acknowledgment
	// for the previous command can't match this row and undo the resend.
	const stmt = `UPDATE host_mdm_apple_device_names SET status = NULL, command_uuid = NULL WHERE host_uuid = ?`

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "resend host device name")
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		// this should never happen, log for debugging
		ds.logger.DebugContext(ctx, "resend device name status not updated", "host_uuid", hostUUID)
	}
	return nil
}

func (ds *Datastore) ReconcileHostDeviceNamesForHosts(ctx context.Context, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return reconcileHostDeviceNamesForHostsDB(ctx, tx, hostIDs)
	})
}

// reconcileHostDeviceNamesForHostsDB upserts or deletes host-name enforcement
// rows for the given hosts based on each host's current team template.
//
// A host should have a queued enforcement row when it is eligible and its team
// carries a non-empty name template; otherwise any existing row is removed. The
// template lives in teams.config JSON at $.mdm.name_template (empty string when
// unset, NULL for teams whose config predates the feature).
func reconcileHostDeviceNamesForHostsDB(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	deleteStmt, deleteArgs, err := sqlx.In(`
		DELETE hmadn
		FROM host_mdm_apple_device_names hmadn
		JOIN hosts h ON h.uuid = hmadn.host_uuid
		WHERE h.id IN (?)`, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build reconcile device name delete")
	}
	if _, err := tx.ExecContext(ctx, deleteStmt, deleteArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "reconcile device name delete")
	}

	insertStmt, insertArgs, err := sqlx.In(`
		INSERT INTO host_mdm_apple_device_names (host_uuid, status)
		SELECT h.uuid, NULL`+deviceNameEligibleHostsJoins+`
		JOIN teams t ON t.id = h.team_id
		WHERE h.id IN (?)
			AND `+deviceNameEligibleHostsWhere+`
			AND t.config->>'$.mdm.name_template' IS NOT NULL
			AND t.config->>'$.mdm.name_template' != ''`, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build reconcile device name insert")
	}
	if _, err := tx.ExecContext(ctx, insertStmt, insertArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "reconcile device name insert")
	}
	return nil
}
