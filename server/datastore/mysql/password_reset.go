package mysql

import (
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (d *Datastore) NewPasswordResetRequest(req *kolide.PasswordResetRequest) (*kolide.PasswordResetRequest, error) {
	sqlStatement := `
		INSERT INTO password_reset_requests
		( user_id, token)
		VALUES (?,?)
	`
	response, err := d.db.Exec(sqlStatement, req.UserID, req.Token)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	id, _ := response.LastInsertId()
	req.ID = uint(id)
	return req, nil

}

func (d *Datastore) SavePasswordResetRequest(req *kolide.PasswordResetRequest) error {
	sqlStatement := `
		UPDATE password_reset_requests SET
			expires_at = ?,
			user_id = ?,
			token = ?
		WHERE id = ?
	`
	_, err := d.db.Exec(sqlStatement, req.ExpiresAt, req.UserID, req.Token, req.ID)
	if err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

func (d *Datastore) DeletePasswordResetRequest(req *kolide.PasswordResetRequest) error {

	sqlStatement := `
		DELETE FROM password_reset_requests WHERE id = ?
	`
	_, err := d.db.Exec(sqlStatement, req.ID)
	if err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

func (d *Datastore) DeletePasswordResetRequestsForUser(userID uint) error {
	sqlStatement := `
		DELETE FROM password_reset_requests WHERE user_id = ?
	`
	_, err := d.db.Exec(sqlStatement, userID)
	if err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

func (d *Datastore) FindPassswordResetByID(id uint) (*kolide.PasswordResetRequest, error) {
	sqlStatement := `
		SELECT * FROM password_reset_requests
		WHERE id = ? LIMIT 1
	`
	passwordResetRequest := &kolide.PasswordResetRequest{}
	err := d.db.Get(&passwordResetRequest, sqlStatement, id)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return passwordResetRequest, nil
}

func (d *Datastore) FindPassswordResetsByUserID(id uint) ([]*kolide.PasswordResetRequest, error) {
	sqlStatement := `
		SELECT * FROM password_reset_requests
		WHERE user_id = ?
	`

	passwordResetRequests := []*kolide.PasswordResetRequest{}
	err := d.db.Select(&passwordResetRequests, sqlStatement, id)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return passwordResetRequests, nil

}

func (d *Datastore) FindPassswordResetByToken(token string) (*kolide.PasswordResetRequest, error) {
	sqlStatement := `
		SELECT * FROM password_reset_requests
		WHERE token = ? LIMIT 1
	`
	passwordResetRequest := &kolide.PasswordResetRequest{}
	err := d.db.Get(passwordResetRequest, sqlStatement, token)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return passwordResetRequest, nil

}

func (d *Datastore) FindPassswordResetByTokenAndUserID(token string, id uint) (*kolide.PasswordResetRequest, error) {
	sqlStatement := `
		SELECT * FROM password_reset_requests
		WHERE user_id = ? AND token = ?
		LIMIT 1
	`
	passwordResetRequest := &kolide.PasswordResetRequest{}
	err := d.db.Get(passwordResetRequest, sqlStatement, id, token)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return passwordResetRequest, nil
}
