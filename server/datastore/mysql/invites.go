package mysql

import (
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

// NewInvite generates a new invitation
func (d *Datastore) NewInvite(i *kolide.Invite) (*kolide.Invite, error) {

	sql := `
	INSERT INTO invites ( invited_by, email, admin, name, position, token)
	  VALUES ( ?, ?, ?, ?, ?, ?)
	`

	result, err := d.db.Exec(sql, i.InvitedBy, i.Email, i.Admin,
		i.Name, i.Position, i.Token)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	i.ID = uint(id)

	return i, nil

}

// ListInvites lists all invites in the Kolide database. Supply query options
// using the opt parameter. See kolide.ListOptions
func (d *Datastore) ListInvites(opt kolide.ListOptions) ([]*kolide.Invite, error) {

	invites := []*kolide.Invite{}

	sql := appendListOptionsToSQL("SELECT * FROM invites WHERE NOT deleted", opt)
	err := d.db.Select(&invites, sql)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}
	return invites, nil
}

// Invite returns Invite identified by id.
func (d *Datastore) Invite(id uint) (*kolide.Invite, error) {
	invite := &kolide.Invite{}
	err := d.db.Get(invite, "SELECT * FROM invites WHERE id = ? AND NOT deleted", id)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}
	return invite, nil
}

// InviteByEmail finds an Invite with a particular email, if one exists.
func (d *Datastore) InviteByEmail(email string) (*kolide.Invite, error) {
	invite := &kolide.Invite{}
	err := d.db.Get(invite, "SELECT * FROM invites WHERE email = ? AND NOT deleted", email)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}
	return invite, nil
}

// SaveInvite modifies existing Invite
func (d *Datastore) SaveInvite(i *kolide.Invite) error {
	sql := `
	UPDATE invites SET invited_by = ?, email = ?, admin = ?,
	   name = ?, position = ?, token = ?
		 WHERE id = ? AND NOT deleted
	`
	_, err := d.db.Exec(sql, i.InvitedBy, i.Email,
		i.Admin, i.Name, i.Position, i.Token, i.ID,
	)
	if err != nil {
		return errors.DatabaseError(err)
	}

	return nil

}

func (d *Datastore) DeleteInvite(i *kolide.Invite) error {
	i.MarkDeleted(d.clock.Now())
	sql := `
	UPDATE invites SET deleted_at = ?, deleted = ?
		WHERE id = ?
	`
	_, err := d.db.Exec(sql, i.DeletedAt, true, i.ID)
	if err != nil {
		return errors.DatabaseError(err)
	}
	return nil
}
