package inmem

import (
	"fmt"
	"sort"
	"time"

	"github.com/fleetdm/fleet/server/kolide"
)

// NewInvite creates and stores a new invitation in a DB.
func (d *Datastore) NewInvite(invite *kolide.Invite) (*kolide.Invite, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, in := range d.invites {
		if in.Email == invite.Email {
			return nil, alreadyExists("Invite", invite.ID)
		}
	}

	// set time if missing.
	if invite.CreatedAt.IsZero() {
		invite.CreatedAt = time.Now()
	}

	invite.ID = uint(len(d.invites) + 1)
	d.invites[invite.ID] = invite
	return invite, nil
}

// Invites lists all invites in the datastore.
func (d *Datastore) ListInvites(opt kolide.ListOptions) ([]*kolide.Invite, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	for k, _ := range d.invites {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	invites := []*kolide.Invite{}
	for _, k := range keys {
		invites = append(invites, d.invites[uint(k)])
	}

	// Apply ordering
	if opt.OrderKey != "" {
		var fields = map[string]string{
			"id":                 "ID",
			"created_at":         "CreatedAt",
			"updated_at":         "UpdatedAt",
			"detail_update_time": "DetailUpdateTime",
			"email":              "Email",
			"admin":              "Admin",
			"name":               "Name",
			"position":           "Position",
		}
		if err := sortResults(invites, opt, fields); err != nil {
			return nil, err
		}
	}

	// Apply limit/offset
	low, high := d.getLimitOffsetSliceBounds(opt, len(invites))
	invites = invites[low:high]

	return invites, nil
}

func (d *Datastore) Invite(id uint) (*kolide.Invite, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if invite, ok := d.invites[id]; ok {
		return invite, nil
	}
	return nil, alreadyExists("Invite", id)
}

// InviteByEmail retrieves an invite for a specific email address.
func (d *Datastore) InviteByEmail(email string) (*kolide.Invite, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, invite := range d.invites {
		if invite.Email == email {
			return invite, nil
		}
	}
	return nil, notFound("Invite").
		WithMessage(fmt.Sprintf("with email %s", email))
}

// InviteByToken retrieves an invite given the invite token.
func (d *Datastore) InviteByToken(token string) (*kolide.Invite, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, invite := range d.invites {
		if invite.Token == token {
			return invite, nil
		}
	}
	return nil, notFound("Invite").
		WithMessage(fmt.Sprintf("with token %s", token))
}

// DeleteInvite deletes an invitation.
func (d *Datastore) DeleteInvite(id uint) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.invites[id]; !ok {
		return notFound("Invite").WithID(id)
	}
	delete(d.invites, id)
	return nil
}
