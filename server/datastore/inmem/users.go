package inmem

import (
	"sort"

	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (d *Datastore) NewUser(user *kolide.User) (*kolide.User, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, in := range d.users {
		if in.Username == user.Username {
			return nil, errors.ErrExists
		}
	}

	user.ID = d.nextID(user)
	d.users[user.ID] = user

	return user, nil
}

func (d *Datastore) User(username string) (*kolide.User, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, user := range d.users {
		if user.Username == username {
			return user, nil
		}
	}

	return nil, errors.ErrNotFound
}

func (d *Datastore) ListUsers(opt kolide.ListOptions) ([]*kolide.User, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	for k, _ := range d.users {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	users := []*kolide.User{}
	for _, k := range keys {
		users = append(users, d.users[uint(k)])
	}

	// Apply ordering
	if opt.OrderKey != "" {
		var fields = map[string]string{
			"id":         "ID",
			"created_at": "CreatedAt",
			"updated_at": "UpdatedAt",
			"username":   "Username",
			"name":       "Name",
			"email":      "Email",
			"admin":      "Admin",
			"enabled":    "Enabled",
			"position":   "Position",
		}
		if err := sortResults(users, opt, fields); err != nil {
			return nil, err
		}
	}

	// Apply limit/offset
	low, high := d.getLimitOffsetSliceBounds(opt, len(users))
	users = users[low:high]

	return users, nil
}

func (d *Datastore) UserByEmail(email string) (*kolide.User, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, user := range d.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, errors.ErrNotFound
}

func (d *Datastore) UserByID(id uint) (*kolide.User, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if user, ok := d.users[id]; ok {
		return user, nil
	}

	return nil, errors.ErrNotFound
}

func (d *Datastore) SaveUser(user *kolide.User) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.users[user.ID]; !ok {
		return errors.ErrNotFound
	}

	d.users[user.ID] = user
	return nil
}
