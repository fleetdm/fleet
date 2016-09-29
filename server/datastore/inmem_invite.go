package datastore

import "github.com/kolide/kolide-ose/server/kolide"

// NewInvite creates and stores a new invitation in a DB.
func (orm *inmem) NewInvite(invite *kolide.Invite) (*kolide.Invite, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, in := range orm.invites {
		if in.Email == invite.Email {
			return nil, ErrExists
		}
	}

	invite.ID = uint(len(orm.invites) + 1)
	orm.invites[invite.ID] = invite
	return invite, nil
}

// Invites lists all invites in the datastore.
func (orm *inmem) Invites() ([]*kolide.Invite, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	var invites []*kolide.Invite
	for _, invite := range orm.invites {
		invites = append(invites, invite)
	}

	return invites, nil
}

func (orm *inmem) Invite(id uint) (*kolide.Invite, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()
	if invite, ok := orm.invites[id]; ok {
		return invite, nil
	}
	return nil, ErrNotFound
}

// InviteByEmail retrieves an invite for a specific email address.
func (orm *inmem) InviteByEmail(email string) (*kolide.Invite, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, invite := range orm.invites {
		if invite.Email == email {
			return invite, nil
		}
	}
	return nil, ErrNotFound
}

// SaveInvite saves an invitation in the datastore.
func (orm *inmem) SaveInvite(invite *kolide.Invite) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.invites[invite.ID]; !ok {
		return ErrNotFound
	}

	orm.invites[invite.ID] = invite
	return nil
}

// DeleteInvite deletes an invitation.
func (orm *inmem) DeleteInvite(invite *kolide.Invite) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.invites[invite.ID]; !ok {
		return ErrNotFound
	}
	delete(orm.invites, invite.ID)
	return nil
}
