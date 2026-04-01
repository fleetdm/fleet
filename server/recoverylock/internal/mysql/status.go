package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/recoverylock/internal/types"
	"github.com/jmoiron/sqlx"
)

// GetHostRecoveryLockPasswordStatus returns the full status for a host.
func (ds *Datastore) GetHostRecoveryLockPasswordStatus(ctx context.Context, hostUUID string) (*types.HostStatus, error) {
	const stmt = `
		SELECT
			status,
			operation_type,
			COALESCE(error_message, '') AS error_message,
			encrypted_password IS NOT NULL AS password_available,
			pending_encrypted_password IS NOT NULL AS has_pending_rotation
		FROM host_recovery_key_passwords
		WHERE host_uuid = ? AND deleted = 0`

	var row struct {
		Status             *string `db:"status"`
		OperationType      string  `db:"operation_type"`
		ErrorMessage       string  `db:"error_message"`
		PasswordAvailable  bool    `db:"password_available"`
		HasPendingRotation bool    `db:"has_pending_rotation"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "getting recovery lock password status")
	}

	// Treat NULL status as pending (retry state after failed command enqueue)
	status := statusPending
	if row.Status != nil {
		status = *row.Status
	}

	return &types.HostStatus{
		Status:             status,
		OperationType:      row.OperationType,
		ErrorMessage:       row.ErrorMessage,
		PasswordAvailable:  row.PasswordAvailable,
		HasPendingRotation: row.HasPendingRotation,
	}, nil
}

// SetRecoveryLockVerified marks a recovery lock as successfully verified.
func (ds *Datastore) SetRecoveryLockVerified(ctx context.Context, hostUUID string) error {
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET status = '%s',
		    error_message = NULL
		WHERE host_uuid = ?
		  AND deleted = 0
	`, statusVerified)

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set recovery lock verified")
	}

	return nil
}

// SetRecoveryLockFailed marks a recovery lock operation as failed.
func (ds *Datastore) SetRecoveryLockFailed(ctx context.Context, hostUUID string, errorMsg string) error {
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET status = '%s',
		    error_message = ?
		WHERE host_uuid = ?
		  AND deleted = 0
	`, statusFailed)

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, errorMsg, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set recovery lock failed")
	}

	return nil
}

// ClearRecoveryLockPendingStatus clears pending status for hosts (for retry).
func (ds *Datastore) ClearRecoveryLockPendingStatus(ctx context.Context, hostUUIDs []string) error {
	if len(hostUUIDs) == 0 {
		return nil
	}

	// Reset status to NULL for hosts that failed to have their commands enqueued.
	// This allows them to be picked up again on the next cron run.
	// Only clears status if it's currently 'pending' to avoid overwriting other statuses.
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET status = NULL
		WHERE host_uuid IN (?)
		  AND status = '%s'
		  AND deleted = 0
	`, statusPending)

	query, args, err := sqlx.In(stmt, hostUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build query for clear recovery lock pending status")
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "clear recovery lock pending status")
	}

	return nil
}

// GetRecoveryLockOperationType returns the current operation type for a host.
func (ds *Datastore) GetRecoveryLockOperationType(ctx context.Context, hostUUID string) (string, error) {
	const stmt = `SELECT operation_type FROM host_recovery_key_passwords WHERE host_uuid = ? AND deleted = 0`

	var opType string
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &opType, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ctxerr.Wrap(ctx, notFound("HostRecoveryLockPassword").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return "", ctxerr.Wrap(ctx, err, "get recovery lock operation type")
	}

	return opType, nil
}

// ResetRecoveryLockForRetry resets a failed recovery lock for retry.
func (ds *Datastore) ResetRecoveryLockForRetry(ctx context.Context, hostUUID string) error {
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET operation_type = '%s', status = '%s', error_message = NULL
		WHERE host_uuid = ?
		  AND deleted = 0
	`, operationInstall, statusVerified)

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "reset recovery lock for retry")
	}

	return nil
}

// GetHostsStatusBulk returns status for multiple hosts.
func (ds *Datastore) GetHostsStatusBulk(ctx context.Context, hostUUIDs []string) (map[string]*types.HostStatus, error) {
	if len(hostUUIDs) == 0 {
		return make(map[string]*types.HostStatus), nil
	}

	const stmt = `
		SELECT
			host_uuid,
			status,
			operation_type,
			COALESCE(error_message, '') AS error_message,
			encrypted_password IS NOT NULL AS password_available,
			pending_encrypted_password IS NOT NULL AS has_pending_rotation
		FROM host_recovery_key_passwords
		WHERE host_uuid IN (?)
		  AND deleted = 0`

	query, args, err := sqlx.In(stmt, hostUUIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build bulk status query")
	}

	var rows []struct {
		HostUUID           string  `db:"host_uuid"`
		Status             *string `db:"status"`
		OperationType      string  `db:"operation_type"`
		ErrorMessage       string  `db:"error_message"`
		PasswordAvailable  bool    `db:"password_available"`
		HasPendingRotation bool    `db:"has_pending_rotation"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get bulk recovery lock status")
	}

	result := make(map[string]*types.HostStatus, len(rows))
	for _, row := range rows {
		status := statusPending
		if row.Status != nil {
			status = *row.Status
		}
		result[row.HostUUID] = &types.HostStatus{
			Status:             status,
			OperationType:      row.OperationType,
			ErrorMessage:       row.ErrorMessage,
			PasswordAvailable:  row.PasswordAvailable,
			HasPendingRotation: row.HasPendingRotation,
		}
	}

	return result, nil
}

// GetHostUUIDsByStatus returns host UUIDs with the given status.
func (ds *Datastore) GetHostUUIDsByStatus(ctx context.Context, status string) ([]string, error) {
	var stmt string
	var args []any

	// NULL status is treated as pending
	if status == statusPending {
		stmt = `
			SELECT host_uuid
			FROM host_recovery_key_passwords
			WHERE (status = ? OR status IS NULL)
			  AND deleted = 0`
		args = []any{status}
	} else {
		stmt = `
			SELECT host_uuid
			FROM host_recovery_key_passwords
			WHERE status = ?
			  AND deleted = 0`
		args = []any{status}
	}

	var hostUUIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostUUIDs, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host UUIDs by status")
	}

	return hostUUIDs, nil
}

// FilterHostUUIDsByStatus filters candidate UUIDs to those with the given status.
func (ds *Datastore) FilterHostUUIDsByStatus(ctx context.Context, status string, candidateUUIDs []string) ([]string, error) {
	if len(candidateUUIDs) == 0 {
		return ds.GetHostUUIDsByStatus(ctx, status)
	}

	var stmt string
	var args []any

	// NULL status is treated as pending
	if status == statusPending {
		stmt = `
			SELECT host_uuid
			FROM host_recovery_key_passwords
			WHERE host_uuid IN (?)
			  AND (status = ? OR status IS NULL)
			  AND deleted = 0`
		query, queryArgs, err := sqlx.In(stmt, candidateUUIDs, status)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "build filter query")
		}
		stmt = query
		args = queryArgs
	} else {
		stmt = `
			SELECT host_uuid
			FROM host_recovery_key_passwords
			WHERE host_uuid IN (?)
			  AND status = ?
			  AND deleted = 0`
		query, queryArgs, err := sqlx.In(stmt, candidateUUIDs, status)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "build filter query")
		}
		stmt = query
		args = queryArgs
	}

	var hostUUIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostUUIDs, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "filter host UUIDs by status")
	}

	return hostUUIDs, nil
}
