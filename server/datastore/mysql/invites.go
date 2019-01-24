package mysql

import (
	"database/sql"
	"fmt"

	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

// NewInvite generates a new invitation
func (d *Datastore) NewInvite(i *kolide.Invite) (*kolide.Invite, error) {
	var (
		deletedInvite kolide.Invite
		sqlStmt       string
	)
	err := d.db.Get(&deletedInvite, "SELECT * FROM invites WHERE email = ? AND deleted", i.Email)
	switch err {
	case nil:
		sqlStmt = `
		REPLACE INTO invites ( invited_by, email, admin, name, position, token, deleted, sso_enabled)
		  VALUES ( ?, ?, ?, ?, ?, ?, ?, ?)
		`
	case sql.ErrNoRows:
		sqlStmt = `
		INSERT INTO invites ( invited_by, email, admin, name, position, token, deleted, sso_enabled)
		  VALUES ( ?, ?, ?, ?, ?, ?, ?, ?)
		`
	default:
		return nil, errors.Wrap(err, "check for existing invite")
	}

	deleted := false
	result, err := d.db.Exec(sqlStmt, i.InvitedBy, i.Email, i.Admin,
		i.Name, i.Position, i.Token, deleted, i.SSOEnabled)
	if err != nil && isDuplicate(err) {
		return nil, alreadyExists("Invite", 0)
	} else if err != nil {
		return nil, errors.Wrap(err, "create invite")
	}

	id, _ := result.LastInsertId()
	i.ID = uint(id)

	return i, nil

}

// ListInvites lists all invites in the Fleet database. Supply query options
// using the opt parameter. See kolide.ListOptions
func (d *Datastore) ListInvites(opt kolide.ListOptions) ([]*kolide.Invite, error) {

	invites := []*kolide.Invite{}

	query := appendListOptionsToSQL("SELECT * FROM invites WHERE NOT deleted", opt)
	err := d.db.Select(&invites, query)
	if err == sql.ErrNoRows {
		return nil, notFound("Invite")
	} else if err != nil {
		return nil, errors.Wrap(err, "select invite by ID")
	}
	return invites, nil
}

// Invite returns Invite identified by id.
func (d *Datastore) Invite(id uint) (*kolide.Invite, error) {
	var invite kolide.Invite
	err := d.db.Get(&invite, "SELECT * FROM invites WHERE id = ? AND NOT deleted", id)
	if err == sql.ErrNoRows {
		return nil, notFound("Invite").WithID(id)
	} else if err != nil {
		return nil, errors.Wrap(err, "select invite by ID")
	}
	return &invite, nil
}

// InviteByEmail finds an Invite with a particular email, if one exists.
func (d *Datastore) InviteByEmail(email string) (*kolide.Invite, error) {
	var invite kolide.Invite
	err := d.db.Get(&invite, "SELECT * FROM invites WHERE email = ? AND NOT deleted", email)
	if err == sql.ErrNoRows {
		return nil, notFound("Invite").
			WithMessage(fmt.Sprintf("with email %s", email))
	} else if err != nil {
		return nil, errors.Wrap(err, "sqlx get invite by email")
	}
	return &invite, nil
}

// InviteByToken finds an Invite with a particular token, if one exists.
func (d *Datastore) InviteByToken(token string) (*kolide.Invite, error) {
	var invite kolide.Invite
	err := d.db.Get(&invite, "SELECT * FROM invites WHERE token = ? AND NOT deleted", token)
	if err == sql.ErrNoRows {
		return nil, notFound("Invite").
			WithMessage(fmt.Sprintf("with token %s", token))
	} else if err != nil {
		return nil, errors.Wrap(err, "sqlx get invite by token")
	}
	return &invite, nil
}

// SaveInvite modifies existing Invite
func (d *Datastore) SaveInvite(i *kolide.Invite) error {
	sql := `
	UPDATE invites SET invited_by = ?, email = ?, admin = ?,
	   name = ?, position = ?, token = ?, sso_enabled = ?
		 WHERE id = ? AND NOT deleted
	`
	results, err := d.db.Exec(sql, i.InvitedBy, i.Email,
		i.Admin, i.Name, i.Position, i.Token, i.SSOEnabled, i.ID,
	)
	if err != nil {
		return errors.Wrap(err, "save invite")
	}
	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "rows affected updating invite")
	}
	if rowsAffected == 0 {
		return notFound("Invite").WithID(i.ID)
	}

	return nil

}

func (d *Datastore) DeleteInvite(id uint) error {
	return d.deleteEntity("invites", id)
}
