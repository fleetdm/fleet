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

func (d *Datastore) SavePasswordResetRequest(req *fleet.PasswordResetRequest) error {
	sqlStatement := `
		UPDATE password_reset_requests SET
			expires_at = ?,
			user_id = ?,
			token = ?
		WHERE id = ?
	`
	result, err := d.db.Exec(sqlStatement, req.ExpiresAt, req.UserID, req.Token, req.ID)
	if err != nil {
		return errors.Wrap(err, "updating password reset requests")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "rows affected updating password reset requests")
	}
	if rows == 0 {
		return notFound("PasswordResetRequest").WithID(req.ID)
	}

	return nil
}

func (d *Datastore) DeletePasswordResetRequest(req *fleet.PasswordResetRequest) error {
	err := d.deleteEntity("password_reset_requests", req.ID)
	if err != nil {
		return errors.Wrap(err, "deleting from password reset request")
	}

	return nil
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

func (d *Datastore) FindPassswordResetByID(id uint) (*fleet.PasswordResetRequest, error) {
	sqlStatement := `
		SELECT * FROM password_reset_requests
		WHERE id = ? LIMIT 1
	`
	passwordResetRequest := &fleet.PasswordResetRequest{}
	err := d.db.Get(&passwordResetRequest, sqlStatement, id)
	if err != nil {
		return nil, errors.Wrap(err, "selecting password reset by id")
	}

	return passwordResetRequest, nil
}

func (d *Datastore) FindPassswordResetsByUserID(id uint) ([]*fleet.PasswordResetRequest, error) {
	sqlStatement := `
		SELECT * FROM password_reset_requests
		WHERE user_id = ?
	`

	passwordResetRequests := []*fleet.PasswordResetRequest{}
	err := d.db.Select(&passwordResetRequests, sqlStatement, id)
	if err != nil {
		return nil, errors.Wrap(err, "finding password resets by user id")
	}

	return passwordResetRequests, nil

}

func (d *Datastore) FindPassswordResetByToken(token string) (*fleet.PasswordResetRequest, error) {
	sqlStatement := `
		SELECT * FROM password_reset_requests
		WHERE token = ? LIMIT 1
	`
	passwordResetRequest := &fleet.PasswordResetRequest{}
	err := d.db.Get(passwordResetRequest, sqlStatement, token)
	if err != nil {
		return nil, errors.Wrap(err, "selecting password reset requests")
	}

	return passwordResetRequest, nil

}

func (d *Datastore) FindPassswordResetByTokenAndUserID(token string, id uint) (*fleet.PasswordResetRequest, error) {
	sqlStatement := `
		SELECT * FROM password_reset_requests
		WHERE user_id = ? AND token = ?
		LIMIT 1
	`
	passwordResetRequest := &fleet.PasswordResetRequest{}
	err := d.db.Get(passwordResetRequest, sqlStatement, id, token)
	if err != nil {
		return nil, errors.Wrap(err, "selecting password reset by token and user id")
	}

	return passwordResetRequest, nil
}
