package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (d *Datastore) NewPasswordResetRequest(ctx context.Context, req *fleet.PasswordResetRequest) (*fleet.PasswordResetRequest, error) {
	sqlStatement := `
		INSERT INTO password_reset_requests
		( user_id, token, expires_at)
		VALUES (?,?, NOW())
	`
	response, err := d.writer.ExecContext(ctx, sqlStatement, req.UserID, req.Token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting password reset requests")
	}

	id, _ := response.LastInsertId()
	req.ID = uint(id)
	return req, nil

}

func (d *Datastore) DeletePasswordResetRequestsForUser(ctx context.Context, userID uint) error {
	sqlStatement := `
		DELETE FROM password_reset_requests WHERE user_id = ?
	`
	_, err := d.writer.ExecContext(ctx, sqlStatement, userID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting password reset request by user")
	}

	return nil
}
func (d *Datastore) FindPassswordResetByToken(ctx context.Context, token string) (*fleet.PasswordResetRequest, error) {
	sqlStatement := `
               SELECT * FROM password_reset_requests
               WHERE token = ? LIMIT 1
       `
	passwordResetRequest := &fleet.PasswordResetRequest{}
	err := sqlx.GetContext(ctx, d.reader, passwordResetRequest, sqlStatement, token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting password reset requests")
	}

	return passwordResetRequest, nil

}
