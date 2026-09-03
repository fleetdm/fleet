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

// maxArchivedRecoveryLockPasswordsPerHost bounds a host that never tells Fleet which
// password it holds: every re-enrollment and every retry adds a candidate that the
// proof-based prune has nothing to rule out with. This is only a runaway backstop —
// SetRecoveryLockVerified is what keeps the archive at one row per healthy host.
const maxArchivedRecoveryLockPasswordsPerHost = 25

// recoveryLockArchiveRow is one password Fleet is about to put on the wire. encryptedPassword
// must be the same ciphertext written to host_recovery_key_passwords: the prune identifies
// the row a device confirmed by matching on it.
type recoveryLockArchiveRow struct {
	hostUUID          string
	encryptedPassword []byte
	setCommandUUID    string
}

// archiveRecoveryLockPasswords records passwords before their command is enqueued, then
// trims the affected hosts back to the cap. Archiving at generation time rather than on
// acknowledgment is deliberate: a device can apply a SetRecoveryLock without Fleet ever
// learning it did (lost result, CheckOut mid-command, queue cleared by a re-enrollment),
// and each retry generates a fresh password, so an operation can put several on the device.
func archiveRecoveryLockPasswords(ctx context.Context, tx sqlx.ExtContext, rows []recoveryLockArchiveRow) error {
	if len(rows) == 0 {
		return nil
	}

	args := make([]any, 0, len(rows)*3)
	hostUUIDs := make([]string, 0, len(rows))
	for _, r := range rows {
		args = append(args, r.hostUUID, r.encryptedPassword, r.setCommandUUID)
		hostUUIDs = append(hostUUIDs, r.hostUUID)
	}

	stmt := fmt.Sprintf(`
		INSERT INTO host_recovery_key_password_archive (host_uuid, encrypted_password, set_command_uuid)
		VALUES %s
	`, strings.TrimSuffix(strings.Repeat("(?, ?, ?),", len(rows)), ","))

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "archiving recovery lock passwords")
	}

	// Scoped to the hosts just inserted so this stays an indexed lookup per host rather
	// than a scan of the whole archive.
	trimStmt, trimArgs, err := sqlx.In(`
		DELETE a FROM host_recovery_key_password_archive a
		JOIN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY host_uuid ORDER BY id DESC) AS rn
				FROM host_recovery_key_password_archive
				WHERE host_uuid IN (?)
			) ranked
			WHERE ranked.rn > ?
		) doomed ON doomed.id = a.id
	`, hostUUIDs, maxArchivedRecoveryLockPasswordsPerHost)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build trim query for archived recovery lock passwords")
	}
	if _, err := tx.ExecContext(ctx, trimStmt, trimArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "trimming archived recovery lock passwords")
	}

	return nil
}

// pruneArchivedRecoveryLockPasswords drops every archived password older than the one the
// device just confirmed. A successful verify is proof of device state: the device answered
// that it holds this password, so no earlier one can still be on it. Anything newer is kept
// — those were generated after the confirmed one and may yet have reached the device.
func pruneArchivedRecoveryLockPasswords(ctx context.Context, tx sqlx.ExtContext, hostUUID string, activePassword []byte) error {
	var confirmedID sql.NullInt64
	if err := sqlx.GetContext(ctx, tx, &confirmedID, `
		SELECT MAX(id) FROM host_recovery_key_password_archive
		WHERE host_uuid = ? AND encrypted_password = ?
	`, hostUUID, activePassword); err != nil {
		return ctxerr.Wrap(ctx, err, "find confirmed archived recovery lock password")
	}
	if !confirmedID.Valid {
		// Nothing to anchor the prune on, so nothing can be ruled out. Happens for rows
		// that predate the archive.
		return nil
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM host_recovery_key_password_archive
		WHERE host_uuid = ? AND id < ?
	`, hostUUID, confirmedID.Int64); err != nil {
		return ctxerr.Wrap(ctx, err, "prune archived recovery lock passwords")
	}

	return nil
}

// SetHostsRecoveryLockPasswords is only called in the initial setting of a recovery lock password, and therefore inserts needed values, on duplicate it clears out any old values.
// It sets the password as pending.
func (ds *Datastore) SetHostsRecoveryLockPasswords(ctx context.Context, passwords []fleet.HostRecoveryLockPasswordPayload) error {
	if len(passwords) == 0 {
		return nil
	}

	// Build values for bulk insert.
	// Status is set to 'pending' immediately to prevent the host from being picked up
	// again by the next cron run while the command is being enqueued. If enqueue fails,
	// ClearRecoveryLockPendingStatus should be called to reset the status to NULL.
	var args []any
	archive := make([]recoveryLockArchiveRow, 0, len(passwords))
	for _, p := range passwords {
		encrypted, err := encrypt([]byte(p.Password), ds.serverPrivateKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "encrypting recovery lock password")
		}
		args = append(args, p.HostUUID, encrypted, fleet.MDMDeliveryPending, fleet.MDMOperationTypeInstall, p.PendingSetCommandUUID)
		archive = append(archive, recoveryLockArchiveRow{
			hostUUID:          p.HostUUID,
			encryptedPassword: encrypted,
			setCommandUUID:    p.PendingSetCommandUUID,
		})
	}

	stmt := `
		INSERT INTO host_recovery_key_passwords (host_uuid, pending_encrypted_password, status, operation_type, pending_set_command_uuid)
		VALUES %s
		ON DUPLICATE KEY UPDATE
			pending_encrypted_password = VALUES(pending_encrypted_password),
			status = VALUES(status),
			operation_type = VALUES(operation_type),
			error_message = NULL,
			deleted = 0,
			auto_rotate_at = NULL,
			set_command_uuid = NULL,
			verify_command_uuid = NULL,
			pending_set_command_uuid = VALUES(pending_set_command_uuid),
			pending_verify_command_uuid = NULL
	`

	placeholders := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?),", len(passwords)), ",")
	stmt = fmt.Sprintf(stmt, placeholders)

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "storing recovery lock passwords")
		}
		return archiveRecoveryLockPasswords(ctx, tx, archive)
	})
}

func (ds *Datastore) GetHostRecoveryLockPassword(ctx context.Context, hostUUID string) (*fleet.HostRecoveryLockPassword, error) {
	const stmt = `SELECT encrypted_password, status, updated_at, auto_rotate_at FROM host_recovery_key_passwords WHERE host_uuid = ? AND deleted = 0`

	var row struct {
		EncryptedPassword []byte                   `db:"encrypted_password"`
		Status            *fleet.MDMDeliveryStatus `db:"status"`
		UpdatedAt         time.Time                `db:"updated_at"`
		AutoRotateAt      *time.Time               `db:"auto_rotate_at"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostRecoveryLockPassword").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting recovery lock password")
	}

	var decrypted *string
	if row.EncryptedPassword != nil {
		decryptedPassword, err := decrypt(row.EncryptedPassword, ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "decrypting recovery lock password")
		}
		s := string(decryptedPassword)
		decrypted = &s
	}

	return &fleet.HostRecoveryLockPassword{
		Password:     decrypted,
		Status:       row.Status,
		UpdatedAt:    row.UpdatedAt,
		AutoRotateAt: row.AutoRotateAt,
	}, nil
}

func (ds *Datastore) GetHostRecoveryLockPasswordStatus(ctx context.Context, hostUUID string) (*fleet.HostMDMRecoveryLockPassword, error) {
	const stmt = `
		SELECT
			status,
			operation_type,
			COALESCE(error_message, '') AS detail,
			(encrypted_password IS NOT NULL AND status <=> 'verified') AS password_available
		FROM host_recovery_key_passwords
		WHERE host_uuid = ? AND deleted = 0`

	var row struct {
		Status            *fleet.MDMDeliveryStatus `db:"status"`
		OperationType     fleet.MDMOperationType   `db:"operation_type"`
		Detail            string                   `db:"detail"`
		PasswordAvailable bool                     `db:"password_available"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "getting recovery lock password status")
	}

	// Treat NULL status as pending (retry state after failed command enqueue)
	status := row.Status
	if status == nil {
		status = &fleet.MDMDeliveryPending
	}

	result := &fleet.HostMDMRecoveryLockPassword{
		Detail:            row.Detail,
		PasswordAvailable: row.PasswordAvailable,
	}
	result.SetRawStatus(status, row.OperationType)
	return result, nil
}

func (ds *Datastore) GetHostsForRecoveryLockAction(ctx context.Context) (map[string]bool, error) {
	// Query hosts that:
	// - Have enable_recovery_lock_password = true (from team config or appconfig for no-team hosts)
	// - Are Apple Silicon (ARM CPU)
	// - Are MDM enrolled (enabled = 1 and device enrollment type)
	// - Are NOT personally-owned (BYOD) enrollments. Personal enrollments have the
	//   DeviceLock/DeviceErase access rights stripped (see AppleEnrollmentAccessRights),
	//   so SetRecoveryLock is rejected by the device. Skip them instead of enforcing.
	// - Have no recovery lock password record OR have a password/pending_password with NULL status (command not yet enqueued)
	// Note: hosts with status pending, verified, or failed are NOT included
	// Note: hosts with operation_type='remove' are handled by RestoreRecoveryLockForReenabledHosts
	const stmt = `
		SELECT h.uuid, rkp.encrypted_password IS NOT NULL as has_password
		FROM hosts h
		JOIN nano_enrollments ne ON ne.device_id = h.uuid
		JOIN host_mdm hm ON hm.host_id = h.id
		LEFT JOIN teams t ON t.id = h.team_id
		CROSS JOIN app_config_json ac
		LEFT JOIN host_recovery_key_passwords rkp ON rkp.host_uuid = h.uuid AND rkp.deleted = 0
		WHERE h.platform = 'darwin'
		  AND h.cpu_type LIKE '%arm%'
		  AND ne.enabled = 1
		  AND ne.type IN ('Device', 'User Enrollment (Device)')
		  AND hm.enrolled = 1
		  AND hm.is_personal_enrollment = 0
		  AND (
		      -- Team hosts: check team config
		      (h.team_id IS NOT NULL AND JSON_EXTRACT(t.config, '$.mdm.enable_recovery_lock_password') = true)
		      OR
		      -- No-team hosts: check appconfig
		      (h.team_id IS NULL AND JSON_EXTRACT(ac.json_value, '$.mdm.enable_recovery_lock_password') = true)
		  )
		  AND (rkp.host_uuid IS NULL OR rkp.status IS NULL)
		  AND rkp.pending_set_command_uuid IS NULL
		LIMIT 500
	`

	var rows []struct {
		UUID        string `db:"uuid"`
		HasPassword bool   `db:"has_password"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get hosts for recovery lock action")
	}

	hostUUIDs := make(map[string]bool, len(rows))
	for _, row := range rows {
		hostUUIDs[row.UUID] = row.HasPassword
	}
	return hostUUIDs, nil
}

func (ds *Datastore) RestoreRecoveryLockForReenabledHosts(ctx context.Context) (int64, error) {
	// When recovery lock feature is re-enabled for a host that was in "pending remove" state,
	// we restore it to "verified install" state instead of trying to set a new password.
	// This is because:
	// 1. The device still has the old password (ClearRecoveryLock hasn't completed)
	// 2. Setting a new password would fail (needs current password to change)
	// 3. The existing password in our DB is still valid for the device
	//
	// This handles the edge case where:
	// 1. Feature was disabled → host marked operation_type='remove'
	// 2. ClearRecoveryLock command queued but not yet acknowledged
	// 3. Feature re-enabled → we restore to verified instead of trying to set new password
	//
	// We only restore records in recoverable states (pending or NULL status).
	// Records with status='failed' (e.g., password mismatch) are NOT restored because:
	// - They represent terminal errors that require admin intervention
	// - Restoring them would mask the real problem and clear diagnostic error_message
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords rkp
		JOIN hosts h ON h.uuid = rkp.host_uuid
		LEFT JOIN teams t ON t.id = h.team_id
		CROSS JOIN app_config_json ac
		SET rkp.operation_type = '%s',
		    rkp.status = '%s',
		    rkp.error_message = NULL,
			rkp.retry = 0,
			rkp.pending_set_command_uuid = NULL,
			rkp.pending_verify_command_uuid = NULL,
			rkp.auto_rotate_at = NULL
		WHERE rkp.deleted = 0
		  AND rkp.operation_type = '%s'
		  AND (rkp.status = '%s' OR rkp.status IS NULL)
		  AND (
		      (h.team_id IS NOT NULL AND JSON_EXTRACT(t.config, '$.mdm.enable_recovery_lock_password') = true)
		      OR
		      (h.team_id IS NULL AND JSON_EXTRACT(ac.json_value, '$.mdm.enable_recovery_lock_password') = true)
		  )
	`, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified, fleet.MDMOperationTypeRemove, fleet.MDMDeliveryPending)

	result, err := ds.writer(ctx).ExecContext(ctx, stmt)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "restore recovery lock for re-enabled hosts")
	}

	return result.RowsAffected()
}

func (ds *Datastore) SetRecoveryLockVerified(ctx context.Context, hostUUID string, verifyCommandUUID string) error {
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords SET
			status = '%s',
		    verify_command_uuid = ?,
			encrypted_password = COALESCE(pending_encrypted_password, encrypted_password),
			pending_encrypted_password = NULL,
			pending_set_command_uuid = NULL,
			pending_verify_command_uuid = NULL,
			error_message = NULL,
			auto_rotate_at = NULL,
			retry = 0
		WHERE host_uuid = ?
		  AND pending_verify_command_uuid = ?
		  AND deleted = 0
	`, fleet.MDMDeliveryVerified)

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if _, err := tx.ExecContext(ctx, stmt, verifyCommandUUID, hostUUID, verifyCommandUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "set recovery lock verified")
		}

		// Read back rather than trusting RowsAffected: a result for a command the row is no
		// longer waiting on leaves it untouched, and nothing was confirmed.
		var row struct {
			Status            sql.NullString `db:"status"`
			EncryptedPassword []byte         `db:"encrypted_password"`
		}
		if err := sqlx.GetContext(ctx, tx, &row, `
			SELECT status, encrypted_password FROM host_recovery_key_passwords
			WHERE host_uuid = ? AND deleted = 0
		`, hostUUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return ctxerr.Wrap(ctx, err, "get verified recovery lock password")
		}
		if row.Status.String != string(fleet.MDMDeliveryVerified) || len(row.EncryptedPassword) == 0 {
			return nil
		}

		return pruneArchivedRecoveryLockPasswords(ctx, tx, hostUUID, row.EncryptedPassword)
	})
}

func (ds *Datastore) SetRecoveryLockFailed(ctx context.Context, hostUUID, commandUUID, errorMsg string) error {
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET status = '%s',
		    error_message = ?,
			pending_encrypted_password = NULL,
			pending_set_command_uuid = NULL,
			pending_verify_command_uuid = NULL
		WHERE host_uuid = ?
		  AND (pending_set_command_uuid = ? OR pending_verify_command_uuid = ?)
		  AND deleted = 0
	`, fleet.MDMDeliveryFailed)

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, errorMsg, hostUUID, commandUUID, commandUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set recovery lock failed")
	}

	return nil
}

func (ds *Datastore) ClearRecoveryLockPendingStatus(ctx context.Context, hostUUIDs []string) error {
	if len(hostUUIDs) == 0 {
		return nil
	}

	// Reset status to NULL for hosts that failed to have their commands enqueued.
	// This allows them to be picked up again on the next cron run.
	// Only clears status if it's currently 'pending' to avoid overwriting other statuses.
	// We can safely set pending_password and pending_command to NULL here for the SET path, the Clear path does not enqueue a pending_password but does for a command.
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET status = NULL, pending_encrypted_password = NULL, pending_set_command_uuid = NULL
		WHERE host_uuid IN (?)
		  AND status = '%s'
		  AND deleted = 0
	`, fleet.MDMDeliveryPending)

	query, args, err := sqlx.In(stmt, hostUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build query for clear recovery lock pending status")
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "clear recovery lock pending status")
	}

	return nil
}

func (ds *Datastore) ClaimHostsForRecoveryLockClear(ctx context.Context, clearCommandUUID string) ([]string, error) {
	// Query hosts that need recovery lock cleared where config has it disabled.
	// This includes:
	// 1. New clears: verified passwords (operation_type='install', status='verified')
	// 2. Retries: previous clear attempt failed (operation_type='remove', status=NULL)
	// Also applies the same enrollment/platform filters as GetHostsForRecoveryLockAction
	// to ensure only hosts that can receive MDM commands are claimed.
	selectStmt := fmt.Sprintf(`
		SELECT rkp.host_uuid
		FROM host_recovery_key_passwords rkp
		JOIN hosts h ON h.uuid = rkp.host_uuid
		JOIN nano_enrollments ne ON ne.device_id = h.uuid
		JOIN host_mdm hm ON hm.host_id = h.id
		LEFT JOIN teams t ON t.id = h.team_id
		CROSS JOIN app_config_json ac
		WHERE rkp.deleted = 0
		  AND h.platform = 'darwin'
		  AND h.cpu_type LIKE '%%arm%%'
		  AND ne.enabled = 1
		  AND ne.type IN ('Device', 'User Enrollment (Device)')
		  AND hm.enrolled = 1
		  AND hm.is_personal_enrollment = 0
		  AND (
		      (rkp.operation_type = '%s' AND rkp.status = '%s')
		      OR
		      (rkp.operation_type = '%s' AND rkp.status IS NULL)
		  )
		  AND (
		      (h.team_id IS NOT NULL AND JSON_EXTRACT(t.config, '$.mdm.enable_recovery_lock_password') = false)
		      OR
		      (h.team_id IS NULL AND JSON_EXTRACT(ac.json_value, '$.mdm.enable_recovery_lock_password') = false)
		  )
		LIMIT 500
		FOR UPDATE
	`, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified, fleet.MDMOperationTypeRemove)

	// Update all claimed hosts to remove/pending
	// auto_rotate_at is also nulled: it's meaningful only for install-state
	// rows (see GetHostsForAutoRotation), so leaving a stale view-deadline on
	// a row pending removal would cause the read API to report a rotation
	// time that auto-rotation will never honor.
	updateStmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET operation_type = '%s', status = '%s', auto_rotate_at = NULL,
		    pending_set_command_uuid = ?, pending_verify_command_uuid = NULL
		WHERE host_uuid IN (?)
	`, fleet.MDMOperationTypeRemove, fleet.MDMDeliveryPending)

	var hostUUIDs []string
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := sqlx.SelectContext(ctx, tx, &hostUUIDs, selectStmt); err != nil {
			return ctxerr.Wrap(ctx, err, "select hosts for recovery lock clear")
		}

		if len(hostUUIDs) == 0 {
			return nil
		}

		query, args, err := sqlx.In(updateStmt, clearCommandUUID, hostUUIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build update query")
		}

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "mark hosts pending for clear")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return hostUUIDs, nil
}

// DeleteHostRecoveryLockPassword hard deletes the host's recovery lock password row from the database.
func (ds *Datastore) DeleteHostRecoveryLockPassword(ctx context.Context, hostUUID string, verifyCommandUUID string) error {
	stmt := fmt.Sprintf(`DELETE FROM host_recovery_key_passwords 
		WHERE host_uuid = ? AND deleted = 0 AND pending_verify_command_uuid = ? AND status = '%s' AND operation_type = '%s'`, fleet.MDMDeliveryVerifying, fleet.MDMOperationTypeRemove)

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID, verifyCommandUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host recovery lock password")
	}

	return nil
}

// softDeleteHostRecoveryLockPassword soft-deletes the host's recovery lock password row
// and nulls rotation/view state that would otherwise leak into a future re-enrolled
// password (SetHostsRecoveryLockPasswords' ON DUPLICATE KEY UPDATE does not reset those
// columns on re-animate). Safe to call idempotently — the deleted=0 guard makes repeat
// calls no-ops. Keeps encrypted_password/status/operation_type/error_message for support
// diagnostics. Used by MDM lifecycle hooks (re-enroll, explicit unenroll) — Apple wipes
// the device-side recovery lock whenever the MDM profile is removed, so any of these
// signals invalidates Fleet's stored copy.
func softDeleteHostRecoveryLockPassword(ctx context.Context, tx sqlx.ExtContext, hostUUID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE host_recovery_key_passwords
		SET deleted = 1,
		    pending_encrypted_password = NULL,
		    auto_rotate_at = NULL
		WHERE host_uuid = ? AND deleted = 0`, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "soft-delete host recovery lock password")
	}
	return nil
}

// SoftDeleteRecoveryLockPasswordsForUnenrolledHosts is the cron-driven complement to the
// explicit MDM lifecycle hooks. Catches hosts where MDM was disabled without Fleet receiving
// either a CheckOut (paths handled by MDMTurnOff) or an Authenticate (handled by
// MDMResetEnrollment) — typically when the device user manually removes the MDM profile
// and only osquery refetch eventually reports host_mdm.enrolled=0. Runs each recovery-lock
// cron tick; bounded by the recovery lock table size, not host count.
func (ds *Datastore) SoftDeleteRecoveryLockPasswordsForUnenrolledHosts(ctx context.Context) (int64, error) {
	res, err := ds.writer(ctx).ExecContext(ctx, `
		UPDATE host_recovery_key_passwords rkp
		JOIN hosts h     ON h.uuid = rkp.host_uuid AND h.platform = 'darwin'
		JOIN host_mdm hm ON hm.host_id = h.id AND hm.enrolled = 0
		SET rkp.deleted = 1,
		    rkp.pending_encrypted_password = NULL,
		    rkp.auto_rotate_at = NULL
		WHERE rkp.deleted = 0`)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "soft-delete recovery lock passwords for unenrolled hosts")
	}
	return res.RowsAffected()
}

func (ds *Datastore) InitiateRecoveryLockRotation(ctx context.Context, hostUUID, setCommandUUID string, newPassword string) error {
	encryptedPassword, err := encrypt([]byte(newPassword), ds.serverPrivateKey)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "encrypt pending recovery lock password")
	}

	// Set the pending password and mark status as NULL (to be picked up).
	// Only allow rotation if:
	// - Has an existing password (encrypted_password IS NOT NULL)
	// - Operation type is 'install' (not removing the password)
	// - Current status is 'verified' or 'failed' (not 'pending' or NULL)
	// - No pending rotation already (pending_encrypted_password IS NULL)
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET pending_encrypted_password = ?,
			pending_set_command_uuid = ?,
			pending_verify_command_uuid = NULL,
			retry = 0,
		    status = '%s',
			auto_rotate_at = NULL
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND encrypted_password IS NOT NULL
		  AND operation_type = '%s'
		  AND status IN ('%s', '%s')
		  AND pending_encrypted_password IS NULL
`, fleet.MDMDeliveryPending, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified, fleet.MDMDeliveryFailed)

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		result, err := tx.ExecContext(ctx, stmt, encryptedPassword, setCommandUUID, hostUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "initiate recovery lock rotation")
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			// Determine the specific reason for failure
			var dest struct {
				HasPassword   bool           `db:"has_password"`
				HasPending    bool           `db:"has_pending"`
				Status        sql.NullString `db:"status"`
				OperationType sql.NullString `db:"operation_type"`
			}
			checkStmt := `
				SELECT
					encrypted_password IS NOT NULL AND deleted = 0 AS has_password,
					pending_encrypted_password IS NOT NULL AS has_pending,
					status,
					operation_type
				FROM host_recovery_key_passwords
				WHERE host_uuid = ? AND deleted = 0
			`
			if err := sqlx.GetContext(ctx, tx, &dest, checkStmt, hostUUID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ctxerr.Wrap(ctx, notFound("HostRecoveryLockPassword").
						WithMessage(fmt.Sprintf("for host %s", hostUUID)))
				}
				return ctxerr.Wrap(ctx, err, "check recovery lock rotation eligibility")
			}

			if dest.HasPending {
				return ctxerr.Wrap(ctx, fleet.ErrRecoveryLockRotationPending, fmt.Sprintf("host %s", hostUUID))
			}

			return ctxerr.Wrap(ctx, fleet.ErrRecoveryLockNotEligible, fmt.Sprintf("host %s (status=%v, operation_type=%v)", hostUUID, dest.Status.String, dest.OperationType.String))
		}

		return archiveRecoveryLockPasswords(ctx, tx, []recoveryLockArchiveRow{{
			hostUUID:          hostUUID,
			encryptedPassword: encryptedPassword,
			setCommandUUID:    setCommandUUID,
		}})
	})
}

func (ds *Datastore) ClearRecoveryLockRotation(ctx context.Context, hostUUID string) error {
	// Clear pending rotation (e.g., if command enqueue fails).
	// Only affects rows that were modified by InitiateRecoveryLockRotation, which leaves
	// status 'pending' with both a stored and a pending password. The encrypted_password guard
	// keeps this off an initial install awaiting retry, which is also status NULL with a
	// pending password but has nothing to restore to — that path is ClearRecoveryLockPendingStatus.
	// Restores status to previous state: 'failed' if error_message exists, otherwise 'verified'.
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET pending_encrypted_password = NULL,
		    pending_set_command_uuid = NULL,
		    status = CASE WHEN error_message IS NOT NULL THEN '%s' ELSE '%s' END
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND status = '%s'
		  AND encrypted_password IS NOT NULL
		  AND pending_encrypted_password IS NOT NULL
`, fleet.MDMDeliveryFailed, fleet.MDMDeliveryVerified, fleet.MDMDeliveryPending)

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "clear recovery lock rotation")
	}

	return nil
}

func (ds *Datastore) GetRecoveryLockRotationStatus(ctx context.Context, hostUUID string) (*fleet.HostRecoveryLockRotationStatus, error) {
	const stmt = `
		SELECT
			host_uuid,
			encrypted_password IS NOT NULL AND deleted = 0 AS has_password,
			status,
			operation_type,
			pending_encrypted_password IS NOT NULL AS has_pending_rotation
		FROM host_recovery_key_passwords
		WHERE host_uuid = ?
		  AND deleted = 0
	`

	var row struct {
		HostUUID           string  `db:"host_uuid"`
		HasPassword        bool    `db:"has_password"`
		Status             *string `db:"status"`
		OperationType      string  `db:"operation_type"`
		HasPendingRotation bool    `db:"has_pending_rotation"`
	}

	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostRecoveryLockPassword").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return nil, ctxerr.Wrap(ctx, err, "get recovery lock rotation status")
	}

	return &fleet.HostRecoveryLockRotationStatus{
		HostUUID:           row.HostUUID,
		HasPassword:        row.HasPassword,
		Status:             row.Status,
		OperationType:      row.OperationType,
		HasPendingRotation: row.HasPendingRotation,
	}, nil
}

// MarkRecoveryLockPasswordViewed schedules auto-rotation by setting
// auto_rotate_at to 1 hour from now on the host's install-state recovery lock
// row, overwriting any prior deadline. Returns the scheduled rotation time.
//
// If the row is missing or not in install state (e.g.,
// ClaimHostsForRecoveryLockClear has flipped it to operation_type='remove'
// because the host's current team has the feature disabled), returns a zero
// time and no error: the password is still readable until the device confirms
// ClearRecoveryLock, but auto-rotation does not apply and would be undone by
// the clear anyway. Callers should treat zero as "no rotation scheduled" and
// omit auto_rotate_at from the response rather than reporting a deadline the
// auto-rotation cron (filtered on operation_type='install') will never honor.
func (ds *Datastore) MarkRecoveryLockPasswordViewed(ctx context.Context, hostUUID string) (time.Time, error) {
	rotateAt := time.Now().Add(1 * time.Hour)

	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET auto_rotate_at = ?
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND operation_type = '%s'
	`, fleet.MDMOperationTypeInstall)

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, rotateAt, hostUUID)
	if err != nil {
		return time.Time{}, ctxerr.Wrap(ctx, err, "mark recovery lock password viewed")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return time.Time{}, nil
	}

	return rotateAt, nil
}

func (ds *Datastore) GetHostsForAutoRotation(ctx context.Context) ([]fleet.HostAutoRotationInfo, error) {
	// Return hosts where:
	// - auto_rotate_at is in the past (due for rotation)
	// - status is verified (password is confirmed working)
	// - no pending rotation (pending_encrypted_password IS NULL)
	// - operation_type is install (not in remove state)
	// - not deleted
	// Join with hosts table to get host ID and display name for activity logging.
	stmt := fmt.Sprintf(`
		SELECT
			hrkp.host_uuid,
			h.id AS host_id,
			COALESCE(NULLIF(h.computer_name, ''), h.hostname) AS display_name
		FROM host_recovery_key_passwords hrkp
		JOIN hosts h ON h.uuid = hrkp.host_uuid
		WHERE hrkp.auto_rotate_at IS NOT NULL
		  AND hrkp.auto_rotate_at <= NOW(6)
		  AND hrkp.status = '%s'
		  AND hrkp.pending_encrypted_password IS NULL
		  AND hrkp.operation_type = '%s'
		  AND hrkp.deleted = 0
		LIMIT 100
	`, fleet.MDMDeliveryVerified, fleet.MDMOperationTypeInstall)

	var hosts []fleet.HostAutoRotationInfo
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get hosts for auto rotation")
	}

	return hosts, nil
}

func (ds *Datastore) GetPendingRecoveryLock(ctx context.Context, hostUUID string) (*fleet.HostRecoveryLockPending, error) {
	var pending fleet.HostRecoveryLockPending

	err := sqlx.GetContext(ctx, ds.reader(ctx), &pending, `
		SELECT
			pending_set_command_uuid, pending_verify_command_uuid,
			operation_type,
			encrypted_password IS NOT NULL AS has_current_password,
			retry
		FROM host_recovery_key_passwords
		WHERE host_uuid = ?
		  AND deleted = 0
		LIMIT 1
	`, hostUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get pending recovery lock")
	}

	return &pending, nil
}

func (ds *Datastore) SetRecoveryLockVerifying(ctx context.Context, hostUUID, setCommandUUID, pendingVerifyCommandUUID string) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
		UPDATE host_recovery_key_passwords
		SET status = ?, set_command_uuid = ?, pending_verify_command_uuid = ?,
		pending_set_command_uuid = NULL
		WHERE host_uuid = ?
		  AND pending_set_command_uuid = ?
		  AND deleted = 0
`, fleet.MDMDeliveryVerifying, setCommandUUID, pendingVerifyCommandUUID, hostUUID, setCommandUUID)
	return ctxerr.Wrap(ctx, err, "set recovery lock verifying")
}

func (ds *Datastore) SetRecoveryLockVerifyingLastKnownPassword(ctx context.Context, hostUUID, setCommandUUID, pendingVerifyCommandUUID string) error {
	// The device rejected the pending password, so clear it: the verify going out carries the
	// active password, and that is what SetRecoveryLockVerified must leave in place.
	_, err := ds.writer(ctx).ExecContext(ctx, `
		UPDATE host_recovery_key_passwords
		SET status = ?, set_command_uuid = ?, pending_verify_command_uuid = ?,
		pending_set_command_uuid = NULL,
		pending_encrypted_password = NULL
		WHERE host_uuid = ?
		  AND pending_set_command_uuid = ?
		  AND deleted = 0
`, fleet.MDMDeliveryVerifying, setCommandUUID, pendingVerifyCommandUUID, hostUUID, setCommandUUID)
	return ctxerr.Wrap(ctx, err, "set recovery lock verifying with last known password")
}

func (ds *Datastore) RetryRecoveryLock(ctx context.Context, hostUUID, commandUUID string) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
		UPDATE host_recovery_key_passwords
		SET retry = retry + 1,
		status = NULL,
		pending_set_command_uuid = NULL,
		pending_verify_command_uuid = NULL
		WHERE host_uuid = ?
			AND (pending_set_command_uuid = ? OR pending_verify_command_uuid = ?)
		  	AND deleted = 0
	`, hostUUID, commandUUID, commandUUID)
	return ctxerr.Wrap(ctx, err, "retry recovery lock")
}

func (ds *Datastore) RetryRecoveryLockVerify(ctx context.Context, hostUUID, verifyCommandUUID, newVerifyCommandUUID string) error {
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET retry = retry + 1,
		    pending_verify_command_uuid = ?
		WHERE host_uuid = ?
		  AND pending_verify_command_uuid = ?
		  AND status = '%s'
		  AND deleted = 0
	`, fleet.MDMDeliveryVerifying)

	_, err := ds.writer(ctx).ExecContext(ctx, stmt, newVerifyCommandUUID, hostUUID, verifyCommandUUID)
	return ctxerr.Wrap(ctx, err, "retry recovery lock verify")
}
