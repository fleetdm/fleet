package datastore

import "github.com/kolide/kolide-ose/server/kolide"

func (orm *inmem) NewUser(user *kolide.User) (*kolide.User, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, in := range orm.users {
		if in.Username == user.Username {
			return nil, ErrExists
		}
	}

	user.ID = uint(len(orm.users) + 1)
	orm.users[user.ID] = user

	return user, nil
}

func (orm *inmem) User(username string) (*kolide.User, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, user := range orm.users {
		if user.Username == username {
			return user, nil
		}
	}

	return nil, ErrNotFound
}

func (orm *inmem) Users() ([]*kolide.User, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	var users []*kolide.User
	for _, user := range orm.users {
		users = append(users, user)
	}

	return users, nil
}

func (orm *inmem) UserByEmail(email string) (*kolide.User, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, user := range orm.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, ErrNotFound
}

func (orm *inmem) UserByID(id uint) (*kolide.User, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if user, ok := orm.users[id]; ok {
		return user, nil
	}

	return nil, ErrNotFound
}

func (orm *inmem) SaveUser(user *kolide.User) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.users[user.ID]; !ok {
		return ErrNotFound
	}

	orm.users[user.ID] = user
	return nil
}
