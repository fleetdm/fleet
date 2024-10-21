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

const PasswordResetRequestDuration = 24 * time.Hour

func (ds *Datastore) NewPasswordResetRequest(ctx context.Context, req *fleet.PasswordResetRequest) (*fleet.PasswordResetRequest, error) {
	sqlStatement := `
		INSERT INTO password_reset_requests
		( user_id, token, expires_at)
		VALUES (?,?, DATE_ADD(CURRENT_TIMESTAMP, INTERVAL ? MINUTE))
	`
	response, err := ds.writer(ctx).ExecContext(ctx, sqlStatement, req.UserID, req.Token, PasswordResetRequestDuration.Minutes())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting password reset requests")
	}

	id, _ := response.LastInsertId()
	req.ID = uint(id) //nolint:gosec // dismiss G115
	return req, nil
}

func (ds *Datastore) DeletePasswordResetRequestsForUser(ctx context.Context, userID uint) error {
	sqlStatement := `
		DELETE FROM password_reset_requests WHERE user_id = ?
	`
	_, err := ds.writer(ctx).ExecContext(ctx, sqlStatement, userID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting password reset request by user")
	}

	return nil
}

func (ds *Datastore) FindPasswordResetByToken(ctx context.Context, token string) (*fleet.PasswordResetRequest, error) {
	sqlStatement := `
		SELECT * FROM password_reset_requests
		WHERE token = ? AND CURRENT_TIMESTAMP < expires_at LIMIT 1
    `
	passwordResetRequest := &fleet.PasswordResetRequest{}
	err := sqlx.GetContext(ctx, ds.reader(ctx), passwordResetRequest, sqlStatement, token)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ctxerr.Wrap(ctx, err, "invalid password reset token")
	} else if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting password reset token")
	}

	return passwordResetRequest, nil
}

func (ds *Datastore) CleanupExpiredPasswordResetRequests(ctx context.Context) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM password_reset_requests
		WHERE CURRENT_TIMESTAMP >= expires_at`)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning up expired password reset requests")
	}

	return nil
}
