package mock

import "github.com/fleetdm/fleet/v4/server/fleet"

func UserByEmailWithUser(u *fleet.User) UserByEmailFunc {
	return func(email string) (*fleet.User, error) {
		return u, nil
	}
}

func UserWithEmailNotFound() UserByEmailFunc {
	return func(email string) (*fleet.User, error) {
		return nil, &Error{"not found"}
	}
}

func UserWithID(u *fleet.User) UserByIDFunc {
	return func(id uint) (*fleet.User, error) {
		return u, nil
	}
}
