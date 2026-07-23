package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// deviceNameEligibleHostsJoins and deviceNameEligibleHostsWhere express the
// shared eligibility predicate for host-name enforcement: the host must be on an
// Apple platform, enrolled in Fleet's own MDM (the nano_enrollments join is the
// Fleet-server signal), and not a BYOD enrollment. BYOD hosts are skipped because
// Apple rejects the DeviceName setting on personal (user) enrollments.
//
// The nano_enrollments join keys on ne.id (its primary key) rather than
// ne.device_id: the device channel's row has id == device_id == the device UDID
// (== hosts.uuid), so ne.id = h.uuid selects it directly via the primary key,
// while user-channel rows (id = "<udid>:<user>") never match. Joining on
// device_id instead would fan out to those user rows before the type filter
// narrowed them back down.
//
// BYOD is excluded two ways, because either signal alone leaves a gap:
//   - ne.type = 'Device' drops Account-Driven User Enrollment, whose device
//     channel is recorded as 'User Enrollment (Device)'. Filtering on the type is
//     required because such hosts enrolled before is_personal_enrollment existed
//     default that column to 0, so the flag alone would let them through.
//   - hm.is_personal_enrollment = 0 drops manual, profile-driven BYOD, which does
//     carry a UDID and so is recorded as a 'Device' enrollment; only the flag
//     distinguishes it from a company-owned device.
const deviceNameEligibleHostsJoins = `
	FROM hosts h
	JOIN nano_enrollments ne ON ne.id = h.uuid
	JOIN host_mdm hm ON hm.host_id = h.id`

const deviceNameEligibleHostsWhere = `
	h.platform IN ('darwin', 'ios', 'ipados')
	AND ne.enabled = 1
	AND ne.type = 'Device'
	AND hm.enrolled = 1
	AND hm.is_personal_enrollment = 0`

// deviceNameNoTeamTemplateExpr is a scalar SQL expression that yields the
// "No team" host name template from the single app_config_json row, or ” when
// unset. AppConfig.MDM.HostNameTemplate is an optjson.String that marshals to
// JSON null when unset, and `->>` would surface that as the literal string
// "null"; gating on JSON_TYPE = 'STRING' resolves null/absent to ” (no template),
// matching the empty-string semantics the Go optjson value uses.
const deviceNameNoTeamTemplateExpr = `(SELECT IF(
	JSON_TYPE(JSON_EXTRACT(json_value, '$.mdm.name_template')) = 'STRING',
	json_value->>'$.mdm.name_template', '') FROM app_config_json LIMIT 1)`

func deviceNameTeamScope(teamID *uint) (filter string, args []any) {
	if teamID != nil && *teamID > 0 {
		return "h.team_id = ?", []any{*teamID}
	}
	return "h.team_id IS NULL", nil
}

func (ds *Datastore) BulkUpsertHostDeviceNameEnforcement(ctx context.Context, teamID *uint) error {
	teamFilter, args := deviceNameTeamScope(teamID)

	// On reset, clear command_uuid (and the resolved name/detail) alongside the
	// status: an existing row may still carry a previously-sent command whose
	// result hasn't arrived. UpdateHostDeviceNameStatusFromCommand matches purely
	// on command_uuid, so leaving it set would let a late acknowledgment of the
	// superseded command match this re-queued row and rename the host to the old
	// name before the cron re-sends. Same guard as ResendHostDeviceName.
	stmt := `
		INSERT INTO host_mdm_apple_device_names (host_uuid, status)
		SELECT h.uuid, NULL` + deviceNameEligibleHostsJoins + `
		WHERE ` + deviceNameEligibleHostsWhere + `
			AND ` + teamFilter + `
		ON DUPLICATE KEY UPDATE status = NULL, command_uuid = NULL, expected_device_name = NULL, detail = NULL`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk upsert host device name enforcement")
	}
	return nil
}

func (ds *Datastore) DeleteHostDeviceNameEnforcementForTeam(ctx context.Context, teamID *uint) error {
	teamFilter, args := deviceNameTeamScope(teamID)

	stmt := `
		DELETE hmadn
		FROM host_mdm_apple_device_names hmadn
		JOIN hosts h ON h.uuid = hmadn.host_uuid
		WHERE ` + teamFilter

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host device name enforcement for team")
	}
	return nil
}

func (ds *Datastore) DeactivateHostDeviceNameCommands(ctx context.Context, hostUUIDs []string) error {
	if len(hostUUIDs) == 0 {
		return nil
	}
	// Deactivate any still-active DeviceName command previously enqueued for these
	// hosts before a fresh one is enqueued. A row is only re-listed as queued
	// (status NULL) after having been reset by a resend, template change, or
	// transfer/enrollment reconcile; if the earlier command never executed (device
	// offline, or replying NotNow), it lingers active in the queue. Commands aren't
	// guaranteed to run in order — a NotNow retry can pick the stale one after the
	// new one — so leaving it active risks the device landing on the old name.
	// Same technique as DeactivateMDMAppleHostSCEPRenewCommands.
	stmt, args, err := sqlx.In(
		`UPDATE nano_enrollment_queue SET active = 0 WHERE active = 1 AND command_uuid LIKE ? AND id IN (?)`,
		fleet.DeviceNameCommandUUIDPrefix+"%", hostUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build deactivate host device name commands")
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deactivate host device name commands")
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

// resendDeviceNamesForSecretChange re-queues host-name enforcement rows for the
// scopes whose template references any of the given changed custom (secret)
// variables, so the device-name cron re-resolves with the new secret value and
// enqueues a fresh DeviceName command.
func (ds *Datastore) resendDeviceNamesForSecretChange(ctx context.Context, changedSecretNames []string) error {
	if len(changedSecretNames) == 0 {
		return nil
	}
	pattern := "FLEET_SECRET_(" + strings.Join(changedSecretNames, "|") + `)\b`

	// Teams whose template references a changed secret.
	var teamIDs []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &teamIDs,
		`SELECT id FROM teams WHERE COALESCE(config->>'$.mdm.name_template', '') REGEXP ?`, pattern); err != nil {
		return ctxerr.Wrap(ctx, err, "select teams using changed secret in device name template")
	}

	// The "No team" (global) template.
	var noTeamMatches bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &noTeamMatches,
		`SELECT COALESCE(`+deviceNameNoTeamTemplateExpr+`, '') REGEXP ?`, pattern); err != nil {
		return ctxerr.Wrap(ctx, err, "check no-team device name template for changed secret")
	}

	for _, teamID := range teamIDs {
		if err := ds.BulkUpsertHostDeviceNameEnforcement(ctx, &teamID); err != nil {
			return err
		}
	}
	if noTeamMatches {
		if err := ds.BulkUpsertHostDeviceNameEnforcement(ctx, nil); err != nil {
			return err
		}
	}
	return nil
}

// resendDeviceNameForCustomHostVital re-queues the host's device-name
// enforcement row if its applicable (team or No-team) name template
// references $FLEET_HOST_VITAL_<vitalID>, so the cron re-resolves with the
// host's newly-set value. Mirrors resendMDMProfilesForCustomHostVital's
// content-match precision: a host whose template doesn't reference this vital
// gets no needless resend. Must run in the same transaction as the value
// write (see SetHostCustomHostVitalValue) so the reconciler never reads a
// stale value.
func resendDeviceNameForCustomHostVital(ctx context.Context, tx sqlx.ExtContext, hostID, vitalID uint) error {
	var tmpl string
	err := sqlx.GetContext(ctx, tx, &tmpl, `
		SELECT COALESCE(
			CASE WHEN h.team_id IS NULL
				THEN `+deviceNameNoTeamTemplateExpr+`
				ELSE (SELECT t.config->>'$.mdm.name_template' FROM teams t WHERE t.id = h.team_id)
			END, '')
		FROM hosts h WHERE h.id = ?`, hostID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return ctxerr.Wrap(ctx, err, "get host name template for custom host vital resend")
	}

	if tmpl == "" || !fleet.ContainsVar(tmpl, fmt.Sprintf("%s%d", fleet.CustomHostVitalPrefix, vitalID)) {
		return nil
	}

	return reconcileHostDeviceNamesForHostsDB(ctx, tx, []uint{hostID})
}

func (ds *Datastore) ReconcileHostDeviceNamesForHosts(ctx context.Context, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return reconcileHostDeviceNamesForHostsDB(ctx, tx, hostIDs)
	})
}

// deleteHostDeviceNameRowsForHostsDB removes enforcement rows for the given
// hosts.
func deleteHostDeviceNameRowsForHostsDB(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint) error {
	stmt, args, err := sqlx.In(`
		DELETE hmadn
		FROM host_mdm_apple_device_names hmadn
		JOIN hosts h ON h.uuid = hmadn.host_uuid
		WHERE h.id IN (?)`, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build reconcile device name delete")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "reconcile device name delete")
	}
	return nil
}

// deviceNameQueueEligibleRows queues (status NULL) an enforcement row for every
// eligible host in hostIDs. It does not resolve a template — callers gate on the
// template themselves — so it only applies the eligibility predicate.
func deviceNameQueueEligibleRows(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint, extraWhere string) error {
	insertStmt, insertArgs, err := sqlx.In(`
		INSERT INTO host_mdm_apple_device_names (host_uuid, status)
		SELECT h.uuid, NULL`+deviceNameEligibleHostsJoins+`
		WHERE h.id IN (?)
			AND `+deviceNameEligibleHostsWhere+extraWhere, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build reconcile device name insert")
	}
	if _, err := tx.ExecContext(ctx, insertStmt, insertArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "reconcile device name insert")
	}
	return nil
}

// reconcileHostDeviceNamesForHostsDB upserts or deletes host-name enforcement
// rows for the given hosts based on each host's current host name template.
//
// A host should have a queued enforcement row when it is eligible and the
// template that governs it is non-empty; otherwise any existing row is removed.
// The governing template is the host's team template (teams.config JSON at
// $.mdm.name_template) for a fleet host, or the global "No team" template
// (app_config_json at $.mdm.name_template) for a host in "No team"
// (team_id IS NULL). In both stores empty string means "unset"; either way
// an empty/NULL template means no row. This keys entirely on each host, so
// transfer-into-No-team and No-team enrollment queue rows automatically.
func reconcileHostDeviceNamesForHostsDB(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	if err := deleteHostDeviceNameRowsForHostsDB(ctx, tx, hostIDs); err != nil {
		return err
	}

	return deviceNameQueueEligibleRows(ctx, tx, hostIDs, `
		AND COALESCE(
			CASE WHEN h.team_id IS NULL
				THEN `+deviceNameNoTeamTemplateExpr+`
				ELSE (SELECT t.config->>'$.mdm.name_template' FROM teams t WHERE t.id = h.team_id)
			END, '') != ''`)
}

func reconcileHostDeviceNamesForTeamDB(ctx context.Context, tx sqlx.ExtContext, teamID *uint, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	// Always clear the batch's rows first: a host moving to a template-less scope
	// (or one that became ineligible) must lose its row regardless of the template.
	if err := deleteHostDeviceNameRowsForHostsDB(ctx, tx, hostIDs); err != nil {
		return err
	}

	// Resolve the destination scope's template once.
	var tmpl string
	if teamID != nil && *teamID > 0 {
		if err := sqlx.GetContext(ctx, tx, &tmpl,
			`SELECT COALESCE(config->>'$.mdm.name_template', '') FROM teams WHERE id = ?`, *teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "resolve team name template")
		}
	} else if err := sqlx.GetContext(ctx, tx, &tmpl,
		`SELECT `+deviceNameNoTeamTemplateExpr); err != nil {
		return ctxerr.Wrap(ctx, err, "resolve no-team name template")
	}

	// No template ⇒ nothing to enforce; the delete above already removed any rows.
	if tmpl == "" {
		return nil
	}

	// The template is known non-empty for the whole (homogeneous) batch, so queue
	// eligible hosts with no per-row template resolution and no teams join.
	return deviceNameQueueEligibleRows(ctx, tx, hostIDs, "")
}
