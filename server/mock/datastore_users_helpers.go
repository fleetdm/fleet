package mock

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func UserByEmailWithUser(u *fleet.User) UserByEmailFunc {
	return func(ctx context.Context, email string) (*fleet.User, error) {
		return u, nil
	}
}

func UserWithEmailNotFound() UserByEmailFunc {
	return func(ctx context.Context, email string) (*fleet.User, error) {
		return nil, &Error{"not found"}
	}
}

func UserWithID(u *fleet.User) UserByIDFunc {
	return func(ctx context.Context, id uint) (*fleet.User, error) {
		return u, nil
	}
}
