package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/recoverylock/internal/types"
	"github.com/jmoiron/sqlx"
)

// SetHostsRecoveryLockPasswords sets recovery lock passwords for multiple hosts.
// Uses INSERT ... ON DUPLICATE KEY UPDATE for upsert behavior.
func (ds *Datastore) SetHostsRecoveryLockPasswords(ctx context.Context, passwords []types.PasswordPayload) error {
	if len(passwords) == 0 {
		return nil
	}

	// Build values for bulk insert.
	// Status is set to 'pending' immediately to prevent the host from being picked up
	// again by the next cron run while the command is being enqueued. If enqueue fails,
	// ClearRecoveryLockPendingStatus should be called to reset the status to NULL.
	var args []any
	for _, p := range passwords {
		args = append(args, p.HostUUID, p.EncryptedPassword, p.Status, p.OperationType)
	}

	stmt := `
		INSERT INTO host_recovery_key_passwords (host_uuid, encrypted_password, status, operation_type)
		VALUES %s
		ON DUPLICATE KEY UPDATE
			encrypted_password = VALUES(encrypted_password),
			status = VALUES(status),
			operation_type = VALUES(operation_type),
			error_message = NULL,
			deleted = 0
	`

	placeholders := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?),", len(passwords)), ",")
	stmt = fmt.Sprintf(stmt, placeholders)

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "storing recovery lock passwords")
	}

	return nil
}

// GetHostRecoveryLockPassword retrieves the decrypted password for a host.
func (ds *Datastore) GetHostRecoveryLockPassword(ctx context.Context, hostUUID string) (*types.Password, error) {
	const stmt = `SELECT encrypted_password, updated_at, auto_rotate_at FROM host_recovery_key_passwords WHERE host_uuid = ? AND deleted = 0`

	var row struct {
		EncryptedPassword []byte     `db:"encrypted_password"`
		UpdatedAt         time.Time  `db:"updated_at"`
		AutoRotateAt      *time.Time `db:"auto_rotate_at"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostRecoveryLockPassword").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting recovery lock password")
	}

	decrypted, err := ds.decrypt(row.EncryptedPassword)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decrypting recovery lock password")
	}

	return &types.Password{
		Password:     string(decrypted),
		UpdatedAt:    row.UpdatedAt,
		AutoRotateAt: row.AutoRotateAt,
	}, nil
}

// DeleteHostRecoveryLockPassword soft-deletes the password record for a host.
func (ds *Datastore) DeleteHostRecoveryLockPassword(ctx context.Context, hostUUID string) error {
	stmt := fmt.Sprintf(`UPDATE host_recovery_key_passwords SET deleted = 1, status = '%s' WHERE host_uuid = ? AND deleted = 0`, statusVerified)

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "soft delete host recovery lock password")
	}

	return nil
}

// MarkRecoveryLockPasswordViewed marks a password as viewed and schedules auto-rotation.
// Returns the scheduled auto-rotation time.
func (ds *Datastore) MarkRecoveryLockPasswordViewed(ctx context.Context, hostUUID string, autoRotateDuration time.Duration) (*time.Time, error) {
	if autoRotateDuration <= 0 {
		return nil, nil
	}

	rotateAt := time.Now().Add(autoRotateDuration)

	stmt := fmt.Sprintf(`
		UPDATE host_recovery_key_passwords
		SET auto_rotate_at = ?
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND operation_type = '%s'
	`, operationInstall)

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, rotateAt, hostUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "mark recovery lock password viewed")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, ctxerr.Wrap(ctx, notFound("HostRecoveryLockPassword").
			WithMessage(fmt.Sprintf("for host %s", hostUUID)))
	}

	return &rotateAt, nil
}
