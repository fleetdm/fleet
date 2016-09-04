package kolide

import (
	"net/http"

	"golang.org/x/net/context"
)

type AuthService interface {
	Authenticate(ctx context.Context, username, password string) (*User, error)
	NewSessionManager(ctx context.Context, w http.ResponseWriter, r *http.Request) *SessionManager
}
