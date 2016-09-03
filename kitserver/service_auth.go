package kitserver

import (
	"fmt"
	"net/http"

	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

func (svc service) Authenticate(ctx context.Context, username, password string) (*kolide.User, error) {
	user, err := svc.ds.User(username)
	switch err {
	case nil:
	case datastore.ErrNotFound:
		return nil, authError{
			message: fmt.Sprintf("user %s not found", username),
		}
	default:
		return nil, err
	}
	if !user.Enabled {
		return nil, authError{
			message: fmt.Sprintf("account disabled %s", username),
		}
	}
	if err := user.ValidatePassword(password); err != nil {
		return nil, authError{
			message: fmt.Sprintf("invalid password for user %s", username),
		}
	}
	return user, nil
}

func (svc service) NewSessionManager(ctx context.Context, w http.ResponseWriter, r *http.Request) *kolide.SessionManager {
	return &kolide.SessionManager{
		Request:    r,
		Writer:     w,
		Store:      svc.ds,
		JWTKey:     svc.jwtKey,
		CookieName: svc.cookieName,
	}
}
