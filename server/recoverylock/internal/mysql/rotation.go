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

// ErrRecoveryLockRotationPending is returned when a rotation is already in progress.
var ErrRecoveryLockRotationPending = errors.New("recovery lock rotation already pending")

// ErrRecoveryLockNotEligible is returned when a host is not eligible for rotation.
var ErrRecoveryLockNotEligible = errors.New("recovery lock not eligible for rotation")

// InitiateRecoveryLockRotation starts a password rotation.
func (ds *Datastore) InitiateRecoveryLockRotation(ctx context.Context, hostUUID string, newEncryptedPassword []byte) error {
	// Set the pending password and mark status as pending.
	// Only allow rotation if:
	// - Has an existing password (encrypted_password IS NOT NULL)
	// - Operation type is 'install' (not removing the password)
	// - Current status is 'verified' or 'failed' (not 'pending' or NULL)
	// - No pending rotation already (pending_encrypted_password IS NULL)
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET pending_encrypted_password = ?,
		    pending_error_message = NULL,
		    status = '%s'
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND encrypted_password IS NOT NULL
		  AND operation_type = '%s'
		  AND status IN ('%s', '%s')
		  AND pending_encrypted_password IS NULL
	`, statusPending, operationInstall, statusVerified, statusFailed)

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, newEncryptedPassword, hostUUID)
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
		if err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, checkStmt, hostUUID); err != nil {
			if err == sql.ErrNoRows {
				return ctxerr.Wrap(ctx, notFound("HostRecoveryLockPassword").
					WithMessage(fmt.Sprintf("for host %s", hostUUID)))
			}
			return ctxerr.Wrap(ctx, err, "check recovery lock rotation eligibility")
		}

		if dest.HasPending {
			return ctxerr.Wrap(ctx, ErrRecoveryLockRotationPending, fmt.Sprintf("host %s", hostUUID))
		}

		return ctxerr.Wrap(ctx, ErrRecoveryLockNotEligible, fmt.Sprintf("host %s (status=%v, operation_type=%v)", hostUUID, dest.Status.String, dest.OperationType.String))
	}

	return nil
}

// CompleteRecoveryLockRotation completes a successful rotation.
func (ds *Datastore) CompleteRecoveryLockRotation(ctx context.Context, hostUUID string) error {
	// Move pending password to active and clear pending columns.
	// Also clear auto_rotate_at since rotation is now complete.
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET encrypted_password = pending_encrypted_password,
		    pending_encrypted_password = NULL,
		    pending_error_message = NULL,
		    status = '%s',
		    error_message = NULL,
		    auto_rotate_at = NULL
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND pending_encrypted_password IS NOT NULL
	`, statusVerified)

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "complete recovery lock rotation")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ctxerr.Wrap(ctx, notFound("HostRecoveryLockPendingPassword").
			WithMessage(fmt.Sprintf("for host %s", hostUUID)))
	}

	return nil
}

// FailRecoveryLockRotation marks a rotation as failed.
func (ds *Datastore) FailRecoveryLockRotation(ctx context.Context, hostUUID string, errorMsg string) error {
	// Mark as failed but keep pending password for potential retry.
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET status = '%s',
		    pending_error_message = ?
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND pending_encrypted_password IS NOT NULL
	`, statusFailed)

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, errorMsg, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fail recovery lock rotation")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ctxerr.Wrap(ctx, notFound("HostRecoveryLockPendingPassword").
			WithMessage(fmt.Sprintf("for host %s", hostUUID)))
	}

	return nil
}

// ClearRecoveryLockRotation clears a pending rotation.
func (ds *Datastore) ClearRecoveryLockRotation(ctx context.Context, hostUUID string) error {
	// Clear pending rotation (e.g., if command enqueue fails).
	// Only affects rows that were modified by InitiateRecoveryLockRotation
	// (status = pending AND has pending password).
	// Restores status to previous state: 'failed' if error_message exists, otherwise 'verified'.
	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET pending_encrypted_password = NULL,
		    pending_error_message = NULL,
		    status = CASE WHEN error_message IS NOT NULL THEN '%s' ELSE '%s' END
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND status = '%s'
		  AND pending_encrypted_password IS NOT NULL
	`, statusFailed, statusVerified, statusPending)

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "clear recovery lock rotation")
	}

	return nil
}

// GetRecoveryLockRotationStatus returns the rotation status for a host.
func (ds *Datastore) GetRecoveryLockRotationStatus(ctx context.Context, hostUUID string) (*types.RotationStatus, error) {
	const stmt = `
		SELECT
			pending_encrypted_password IS NOT NULL AS has_pending_rotation,
			status,
			COALESCE(pending_error_message, '') AS error_message
		FROM host_recovery_key_passwords
		WHERE host_uuid = ?
		  AND deleted = 0
	`

	var row struct {
		HasPendingRotation bool    `db:"has_pending_rotation"`
		Status             *string `db:"status"`
		ErrorMessage       string  `db:"error_message"`
	}

	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostRecoveryLockPassword").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return nil, ctxerr.Wrap(ctx, err, "get recovery lock rotation status")
	}

	status := statusPending
	if row.Status != nil {
		status = *row.Status
	}

	return &types.RotationStatus{
		HasPendingRotation: row.HasPendingRotation,
		Status:             status,
		ErrorMessage:       row.ErrorMessage,
	}, nil
}

// HasPendingRecoveryLockRotation checks if a rotation is in progress.
func (ds *Datastore) HasPendingRecoveryLockRotation(ctx context.Context, hostUUID string) (bool, error) {
	const stmt = `
		SELECT pending_encrypted_password IS NOT NULL
		FROM host_recovery_key_passwords
		WHERE host_uuid = ?
		  AND deleted = 0
	`

	var hasPending bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &hasPending, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "has pending recovery lock rotation")
	}

	return hasPending, nil
}
