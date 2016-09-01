package kolide

import (
	"net/http"

	"golang.org/x/net/context"
)

// service a interface stub
type Service interface {
	UserService
	AuthService
}

type UserService interface {
	NewUser(ctx context.Context, p UserPayload) (*User, error)
	User(ctx context.Context, id uint) (*User, error)
	ChangePassword(ctx context.Context, userID uint, old, new string) error
	UpdateAdminRole(ctx context.Context, userID uint, isAdmin bool) error
	UpdateUserStatus(ctx context.Context, userID uint, password string, enabled bool) error
}

type AuthService interface {
	Authenticate(ctx context.Context, username, password string) (*User, error)
	NewSessionManager(ctx context.Context, w http.ResponseWriter, r *http.Request) *SessionManager
}
