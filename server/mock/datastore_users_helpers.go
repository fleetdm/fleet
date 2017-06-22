package mock

import "github.com/kolide/fleet/server/kolide"

func UserByEmailWithUser(u *kolide.User) UserByEmailFunc {
	return func(email string) (*kolide.User, error) {
		return u, nil
	}
}

func UserWithEmailNotFound() UserByEmailFunc {
	return func(email string) (*kolide.User, error) {
		return nil, &Error{"not found"}
	}
}

func UserWithID(u *kolide.User) UserByIDFunc {
	return func(id uint) (*kolide.User, error) {
		return u, nil
	}
}
