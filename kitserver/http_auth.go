package kitserver

import (
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	kitlog "github.com/go-kit/kit/log"

	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
)

func login(ds kolide.UserStore, logger kitlog.Logger) http.HandlerFunc {
	ctx := context.Background()
	logger = kitlog.NewContext(logger).With("method", "login")
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.FormValue("username")
		password := r.FormValue("password")
		if username == "" || password == "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// retrieve user or respond with error
		user, err := getUser(ctx, ds, w, username, password)
		if err != nil {
			logger.Log("err", err, "user", username)
			return
		}
		// create session here

		encodeResponse(ctx, w, getUserResponse{
			ID:                 user.ID,
			Username:           user.Username,
			Name:               user.Name,
			Admin:              user.Admin,
			Enabled:            user.Enabled,
			NeedsPasswordReset: user.NeedsPasswordReset,
		})
		logger.Log("msg", "authenticated", "user", username, "id", user.ID)

	}

}

// gets user from datastore or responds with an authError
func getUser(ctx context.Context, ds kolide.UserStore, w http.ResponseWriter, username, password string) (*kolide.User, error) {
	user, err := ds.User(username)
	switch err {
	case nil:
	case datastore.ErrNotFound:
		encodeResponse(ctx, w, getUserResponse{
			Err: authError{
				message: fmt.Sprintf("user %s not found", username),
			},
		})
		return nil, err
	default:
		encodeResponse(ctx, w, getUserResponse{
			Err: errors.New("unknown error, try again later"),
		})
		return nil, err
	}
	err = user.ValidatePassword(password)
	if err != nil {
		encodeResponse(ctx, w, getUserResponse{
			Err: authError{
				message: fmt.Sprintf("unauthorized: invalid password for user %s", username),
			},
		})
		return nil, err
	}
	return user, nil
}

const noAuthRedirect = "/"

func logout(ds kolide.UserStore, logger kitlog.Logger) http.HandlerFunc {
	logger = kitlog.NewContext(logger).With("method", "logout")
	return func(w http.ResponseWriter, r *http.Request) {
		// delete session first
		var username string
		var user kolide.User
		// TODO

		// redirect
		http.Redirect(w, r, noAuthRedirect, http.StatusFound)
		logger.Log("msg", "loggedout", "user", username, "id", user.ID)
	}
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// all good to pass
		next.ServeHTTP(w, r)
	})
}

type authError struct {
	message string
}

func (e authError) Error() string {
	if e.message == "" {
		return "unauthorized"
	}
	return fmt.Sprintf("unauthorized: %s", e.message)
}

// viewerContext is a struct which represents the ability for an execution
// context to participate in certain actions. Most often, a ViewerContext is
// associated with an application user, but a ViewerContext can represent a
// variety of other execution contexts as well (script, test, etc). The main
// purpose of a ViewerContext is to assist in the authorization of sensitive
// actions.
type viewerContext struct {
	user *kolide.User
}

// IsAdmin indicates whether or not the current user can perform administrative
// actions.
func (vc *viewerContext) IsAdmin() bool {
	if vc.user != nil {
		return vc.user.Admin && vc.user.Enabled
	}
	return false
}

// UserID is a helper that enables quick access to the user ID of the current
// user.
func (vc *viewerContext) UserID() uint {
	if vc.user != nil {
		return vc.user.ID
	}
	return 0
}

// IsLoggedIn determines whether or not the current VC is attached to a user
// account
func (vc *viewerContext) IsLoggedIn() bool {
	return vc.user != nil && vc.user.Enabled
}

// CanPerformActions returns a bool indicating the current user's ability to
// perform the most basic actions on the site
func (vc *viewerContext) CanPerformActions() bool {
	return vc.IsLoggedIn() && !vc.user.NeedsPasswordReset
}

// CanPerformReadActionsOnUser returns a bool indicating the current user's
// ability to perform read actions on the given user
func (vc *viewerContext) CanPerformReadActionOnUser(u *kolide.User) bool {
	return vc.CanPerformActions() || (vc.IsLoggedIn() && vc.IsUserID(u.ID))
}

// CanPerformWriteActionOnUser returns a bool indicating the current user's
// ability to perform write actions on the given user
func (vc *viewerContext) CanPerformWriteActionOnUser(u *kolide.User) bool {
	return vc.CanPerformActions() && (vc.IsUserID(u.ID) || vc.IsAdmin())
}

// IsUserID returns true if the given user id the same as the user which is
// represented by this ViewerContext
func (vc *viewerContext) IsUserID(id uint) bool {
	if vc.UserID() == id {
		return true
	}
	return false
}

// newViewerContext generates a ViewerContext given a user struct
func newViewerContext(user *kolide.User) *viewerContext {
	return &viewerContext{
		user: user,
	}
}

// EmptyVC is a utility which generates an empty ViewerContext. This is often
// used to represent users which are not logged in.
func emptyVC() *viewerContext {
	return &viewerContext{}
}

func vcFromID(ds kolide.UserStore, id uint) (*viewerContext, error) {
	user, err := ds.UserByID(id)
	if err != nil {
		return nil, err
	}
	return &viewerContext{user: user}, nil
}
