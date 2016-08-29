package kolide

import "golang.org/x/net/context"

// service a interface stub
type Service interface {
	UserService
}

type UserService interface {
	NewUser(ctx context.Context, p UserPayload) (*User, error)
	User(ctx context.Context, id uint) (*User, error)
	ChangePassword(ctx context.Context, userID uint, old, new string) error
	Authenticate(ctx context.Context, username, password string) (*User, error)
}
