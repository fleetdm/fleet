package inmem

import (
	"sort"

	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm *Datastore) NewUser(user *kolide.User) (*kolide.User, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, in := range orm.users {
		if in.Username == user.Username {
			return nil, errors.ErrExists
		}
	}

	user.ID = orm.nextID(user)
	orm.users[user.ID] = user

	return user, nil
}

func (orm *Datastore) User(username string) (*kolide.User, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, user := range orm.users {
		if user.Username == username {
			return user, nil
		}
	}

	return nil, errors.ErrNotFound
}

func (orm *Datastore) ListUsers(opt kolide.ListOptions) ([]*kolide.User, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	for k, _ := range orm.users {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	users := []*kolide.User{}
	for _, k := range keys {
		users = append(users, orm.users[uint(k)])
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
	low, high := orm.getLimitOffsetSliceBounds(opt, len(users))
	users = users[low:high]

	return users, nil
}

func (orm *Datastore) UserByEmail(email string) (*kolide.User, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, user := range orm.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, errors.ErrNotFound
}

func (orm *Datastore) UserByID(id uint) (*kolide.User, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if user, ok := orm.users[id]; ok {
		return user, nil
	}

	return nil, errors.ErrNotFound
}

func (orm *Datastore) SaveUser(user *kolide.User) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.users[user.ID]; !ok {
		return errors.ErrNotFound
	}

	orm.users[user.ID] = user
	return nil
}
