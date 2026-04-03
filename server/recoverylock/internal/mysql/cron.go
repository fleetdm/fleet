package mysql

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/recoverylock/internal/types"
	"github.com/jmoiron/sqlx"
)

// GetHostsForRecoveryLockAction returns hosts that need recovery lock set.
// These are hosts with either no password record OR a password with NULL status (command not yet enqueued).
// The eligibleUUIDs parameter is used to filter to hosts that the caller has determined
// are eligible (ARM CPU, MDM enrolled, feature enabled).
func (ds *Datastore) GetHostsForRecoveryLockAction(ctx context.Context, eligibleUUIDs []string) ([]string, error) {
	if len(eligibleUUIDs) == 0 {
		return nil, nil
	}

	// Query hosts from the eligible list that either:
	// - Have no recovery lock password record, OR
	// - Have a password with NULL status (command not yet enqueued for retry)
	// Note: hosts with status pending, verified, or failed are NOT included
	// Note: hosts with operation_type='remove' are handled by RestoreRecoveryLockForReenabledHosts
	//
	// Build the query with the list of eligible UUIDs as a derived table
	// This allows us to use a simple SELECT with values
	var placeholders string
	var args []any
	for i, uuid := range eligibleUUIDs {
		if i > 0 {
			placeholders += " UNION ALL SELECT ?"
		}
		args = append(args, uuid)
	}

	queryStmt := `
		SELECT h.uuid
		FROM (SELECT ? AS uuid` + placeholders + `) h
		LEFT JOIN host_recovery_key_passwords rkp ON rkp.host_uuid = h.uuid AND rkp.deleted = 0
		WHERE rkp.host_uuid IS NULL OR rkp.status IS NULL
		LIMIT 500
	`

	var hostUUIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostUUIDs, queryStmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get hosts for recovery lock action")
	}

	return hostUUIDs, nil
}

// RestoreRecoveryLockForReenabledHosts restores recovery lock for hosts
// where the feature was disabled and then re-enabled.
// This is called with a list of host UUIDs that the caller has determined
// have the feature enabled.
func (ds *Datastore) RestoreRecoveryLockForReenabledHosts(ctx context.Context) (int64, error) {
	// When recovery lock feature is re-enabled for a host that was in "pending remove" state,
	// we restore it to "verified install" state instead of trying to set a new password.
	// This is because:
	// 1. The device still has the old password (ClearRecoveryLock hasn't completed)
	// 2. Setting a new password would fail (needs current password to change)
	// 3. The existing password in our DB is still valid for the device
	//
	// Note: This method operates on all hosts with operation_type='remove' and status='pending' or NULL.
	// The caller must ensure they're calling this at the right time (after determining which hosts
	// have the feature re-enabled). For the bounded context pattern, this will be coordinated
	// through the service layer.
	//
	// We only restore records in recoverable states (pending or NULL status).
	// Records with status='failed' (e.g., password mismatch) are NOT restored because:
	// - They represent terminal errors that require admin intervention
	// - Restoring them would mask the real problem and clear diagnostic error_message
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords rkp
		SET rkp.operation_type = '%s',
		    rkp.status = '%s',
		    rkp.error_message = NULL
		WHERE rkp.deleted = 0
		  AND rkp.operation_type = '%s'
		  AND (rkp.status = '%s' OR rkp.status IS NULL)
	`, operationInstall, statusVerified, operationRemove, statusPending)

	result, err := ds.writer(ctx).ExecContext(ctx, stmt)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "restore recovery lock for re-enabled hosts")
	}

	return result.RowsAffected()
}

// ClaimHostsForRecoveryLockClear claims hosts for recovery lock clear operation.
// Returns the UUIDs of claimed hosts.
// The host eligibility (feature disabled, ARM CPU, MDM enrolled) is determined by the caller
// through DataProviders.
func (ds *Datastore) ClaimHostsForRecoveryLockClear(ctx context.Context) ([]string, error) {
	// Query hosts that need recovery lock cleared.
	// This includes:
	// 1. New clears: verified passwords (operation_type='install', status='verified')
	// 2. Retries: previous clear attempt failed (operation_type='remove', status=NULL)
	//
	// Note: In the bounded context model, the caller filters eligible hosts through
	// DataProviders before calling this method. This method only operates on the
	// host_recovery_key_passwords table.
	selectStmt := fmt.Sprintf(`
		SELECT host_uuid
		FROM host_recovery_key_passwords
		WHERE deleted = 0
		  AND (
		      (operation_type = '%s' AND status = '%s')
		      OR
		      (operation_type = '%s' AND status IS NULL)
		  )
		LIMIT 500
		FOR UPDATE
	`, operationInstall, statusVerified, operationRemove)

	// Update all claimed hosts to remove/pending
	updateStmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET operation_type = '%s', status = '%s'
		WHERE host_uuid IN (?)
	`, operationRemove, statusPending)

	var hostUUIDs []string
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := sqlx.SelectContext(ctx, tx, &hostUUIDs, selectStmt); err != nil {
			return ctxerr.Wrap(ctx, err, "select hosts for recovery lock clear")
		}

		if len(hostUUIDs) == 0 {
			return nil
		}

		query, args, err := sqlx.In(updateStmt, hostUUIDs)
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

// GetHostsForAutoRotation returns hosts with passwords scheduled for auto-rotation.
func (ds *Datastore) GetHostsForAutoRotation(ctx context.Context) ([]types.HostAutoRotationInfo, error) {
	// Return hosts where:
	// - auto_rotate_at is in the past (due for rotation)
	// - status is verified (password is confirmed working)
	// - no pending rotation (pending_encrypted_password IS NULL)
	// - operation_type is install (not in remove state)
	// - not deleted
	//
	// Note: In the bounded context model, host display name is retrieved through
	// DataProviders. This query only returns the host_uuid.
	stmt := fmt.Sprintf(`
		SELECT
			hrkp.host_uuid
		FROM host_recovery_key_passwords hrkp
		WHERE hrkp.auto_rotate_at IS NOT NULL
		  AND hrkp.auto_rotate_at <= NOW(6)
		  AND hrkp.status = '%s'
		  AND hrkp.pending_encrypted_password IS NULL
		  AND hrkp.operation_type = '%s'
		  AND hrkp.deleted = 0
		LIMIT 100
	`, statusVerified, operationInstall)

	var rows []struct {
		HostUUID string `db:"host_uuid"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get hosts for auto rotation")
	}

	// Convert to HostAutoRotationInfo - host info will be enriched by the service layer
	result := make([]types.HostAutoRotationInfo, len(rows))
	for i, row := range rows {
		result[i] = types.HostAutoRotationInfo{
			HostUUID: row.HostUUID,
		}
	}

	return result, nil
}
