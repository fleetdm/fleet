package kolide

import "golang.org/x/net/context"

// service a interface stub
type Service interface {
	UserService
}

type UserService interface {
	NewUser(ctx context.Context, p UserPayload) (*User, error)
	SetPassword(ctx context.Context, userID uint, password string) error
}
