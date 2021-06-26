package inmem

import (
	"fmt"
	"sort"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (d *Datastore) NewUser(user *fleet.User) (*fleet.User, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, in := range d.users {
		if in.Email == user.Email {
			return nil, alreadyExists("User", in.ID)
		}
	}

	user.ID = d.nextID(user)
	d.users[user.ID] = user

	return user, nil
}

func (d *Datastore) ListUsers(opt fleet.UserListOptions) ([]*fleet.User, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	for k := range d.users {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	users := []*fleet.User{}
	for _, k := range keys {
		users = append(users, d.users[uint(k)])
	}

	// Apply ordering
	if opt.OrderKey != "" {
		var fields = map[string]string{
			"id":         "ID",
			"created_at": "CreatedAt",
			"updated_at": "UpdatedAt",
			"name":       "Name",
			"email":      "Email",
			"admin":      "Admin",
			"enabled":    "Enabled",
			"position":   "Position",
		}
		if err := sortResults(users, opt.ListOptions, fields); err != nil {
			return nil, err
		}
	}

	// Apply limit/offset
	low, high := d.getLimitOffsetSliceBounds(opt.ListOptions, len(users))
	users = users[low:high]

	return users, nil
}

func (d *Datastore) UserByEmail(email string) (*fleet.User, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, user := range d.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, notFound("User").
		WithMessage(fmt.Sprintf("with email address %s", email))
}

func (d *Datastore) UserByID(id uint) (*fleet.User, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if user, ok := d.users[id]; ok {
		return user, nil
	}

	return nil, notFound("User").WithID(id)
}

func (d *Datastore) SaveUser(user *fleet.User) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.users[user.ID]; !ok {
		return notFound("User").WithID(user.ID)
	}

	d.users[user.ID] = user
	return nil
}
