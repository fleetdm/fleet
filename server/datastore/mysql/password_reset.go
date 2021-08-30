package mysql

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (d *Datastore) NewPasswordResetRequest(req *fleet.PasswordResetRequest) (*fleet.PasswordResetRequest, error) {
	sqlStatement := `
		INSERT INTO password_reset_requests
		( user_id, token, expires_at)
		VALUES (?,?, NOW())
	`
	response, err := d.db.Exec(sqlStatement, req.UserID, req.Token)
	if err != nil {
		return nil, errors.Wrap(err, "inserting password reset requests")
	}

	id, _ := response.LastInsertId()
	req.ID = uint(id)
	return req, nil

}

func (d *Datastore) DeletePasswordResetRequestsForUser(userID uint) error {
	sqlStatement := `
		DELETE FROM password_reset_requests WHERE user_id = ?
	`
	_, err := d.db.Exec(sqlStatement, userID)
	if err != nil {
		return errors.Wrap(err, "deleting password reset request by user")
	}

	return nil
}
