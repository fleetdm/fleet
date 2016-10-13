package datastore

import (
	"sort"

	"github.com/kolide/kolide-ose/server/kolide"
)

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
func (orm *inmem) Invites(opt kolide.ListOptions) ([]*kolide.Invite, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	for k, _ := range orm.invites {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	invites := []*kolide.Invite{}
	for _, k := range keys {
		invites = append(invites, orm.invites[uint(k)])
	}

	// Apply limit/offset
	low, high := orm.getLimitOffsetSliceBounds(opt, len(invites))
	invites = invites[low:high]

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
